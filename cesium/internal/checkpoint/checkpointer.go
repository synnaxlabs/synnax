// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint

import (
	"context"
	"sync"

	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/validate"
)

// ErrDiskFull is the error of a switch that would keep more than MaxLogs logs, because
// heads keep failing.
var ErrDiskFull = errors.New("disk full")

// MaxLogs is the most logs that the root keeps while heads fail.
const MaxLogs = 16

// Config is the configuration of a checkpointer.
type Config struct {
	// FS is the root of the database.
	FS xfs.FS
	// DB is the identity of the database.
	DB uint64
	// LogSize is the size of each new log file.
	LogSize int64
	// Snapshot returns the durable state that a head holds.
	Snapshot func() (*state.Snapshot, error)
	// Prepared reports that the file of a log is ready, or why it is not.
	Prepared func(next uint64, err error)
	// Executor runs each step of the checkpointer. It must run every function it gets.
	Executor func(f func())
}

// Validate returns an error that wraps validate.ErrValidation when a field of c is not
// set.
func (c Config) Validate() error {
	v := validate.New("checkpoint.Config")
	v.NotNil("fs", c.FS)
	v.Positive("log_size", c.LogSize)
	v.NotNil("snapshot", c.Snapshot)
	v.NotNil("prepared", c.Prepared)
	v.NotNil("executor", c.Executor)
	return v.Error()
}

// Checkpointer is the owner of heads and log files. It makes the file of the next log
// when the log asks, and writes the head of a log once the log switches to it, then
// removes what the switch no longer needs. Each step runs on its executor, and its
// calls return at once. A Checkpointer is safe for concurrent use.
type Checkpointer struct {
	cfg  Config
	wake chan struct{}
	exit chan struct{}
	mu   struct {
		sync.Mutex
		prepare []uint64
		heads   []uint64
		err     error
		closed  bool
	}
	step    func()
	stepped chan bool
}

// New returns a checkpointer of cfg.
func New(cfg Config) (*Checkpointer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	c := &Checkpointer{
		cfg:     cfg,
		wake:    make(chan struct{}, 1),
		exit:    make(chan struct{}),
		stepped: make(chan bool, 1),
	}
	c.step = c.run
	return c, nil
}

func (c *Checkpointer) signal() {
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

// Prepare queues switch step 1 for the log of generation next: the file of the log,
// and the removal of a leftover head of that generation. It reports the result to the
// Prepared hook.
func (c *Checkpointer) Prepare(next uint64) {
	c.mu.Lock()
	c.mu.prepare = append(c.mu.prepare, next)
	c.mu.Unlock()
	c.signal()
}

// Switched queues switch steps 4 to 6 for the log of generation: the head of the
// durable state, then the removal of the heads and logs that the switch no longer
// needs. A failed head leaves the older head and its logs in place.
func (c *Checkpointer) Switched(generation uint64) {
	c.mu.Lock()
	c.mu.heads = append(c.mu.heads, generation)
	c.mu.Unlock()
	c.signal()
}

// Err returns the error of the last head, or nil once a later head succeeds. It wraps
// ErrDiskFull when MaxLogs logs pile up.
func (c *Checkpointer) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.mu.err
}

// Run runs one step of the checkpointer each time work is queued, until Close or until
// ctx is done. It returns the error of ctx, or nil at Close.
func (c *Checkpointer) Run(ctx context.Context) error {
	defer close(c.exit)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.wake:
		}
		c.cfg.Executor(c.step)
		if <-c.stepped {
			return nil
		}
	}
}

// run is one step: it prepares each queued log, then writes each queued head.
func (c *Checkpointer) run() {
	c.mu.Lock()
	prepare, heads, closed := c.mu.prepare, c.mu.heads, c.mu.closed
	c.mu.prepare, c.mu.heads = nil, nil
	c.mu.Unlock()
	for _, next := range prepare {
		c.cfg.Prepared(next, Prepare(c.cfg.FS, next, c.cfg.LogSize))
	}
	for _, generation := range heads {
		err := c.head(generation)
		c.mu.Lock()
		c.mu.err = err
		c.mu.Unlock()
	}
	c.stepped <- closed
}

// head writes the head of generation, then cleans the root.
func (c *Checkpointer) head(generation uint64) error {
	s, err := c.cfg.Snapshot()
	if err == nil {
		err = Write(c.cfg.FS, generation, Encode(c.cfg.DB, generation, s))
	}
	if err == nil {
		err = Clean(c.cfg.FS, generation)
	}
	if err != nil {
		if n, listErr := countLogs(c.cfg.FS); listErr != nil || n > MaxLogs {
			err = errors.Join(err, listErr, errors.Wrapf(ErrDiskFull,
				"%d logs since the last head", n))
		}
	}
	return err
}

func countLogs(fs xfs.FS) (int, error) {
	entries, err := fs.List("")
	if err != nil {
		return 0, err
	}
	var n int
	for _, e := range entries {
		if _, ok := generationOf(e.Name(), ".log"); ok {
			n++
		}
	}
	return n, nil
}

// Close stops the checkpointer after its queued work, waits for Run to return, and
// returns Err.
func (c *Checkpointer) Close() error {
	c.mu.Lock()
	c.mu.closed = true
	c.mu.Unlock()
	c.signal()
	<-c.exit
	return c.Err()
}
