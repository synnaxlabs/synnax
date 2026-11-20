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

// ErrDiskFull is the error of the file of a next log when the root holds MaxLogs logs,
// because heads keep failing.
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
	v.NotNil("prepared", c.Prepared)
	v.NotNil("executor", c.Executor)
	return v.Error()
}

// Checkpointer is the owner of heads and log files. It makes the file of the next log
// when the log asks, and writes the head of a log when the state hands it the snapshot
// of the switch, then removes what the switch no longer needs. Each step runs on its
// executor, and its
// calls return at once. A Checkpointer is safe for concurrent use.
type Checkpointer struct {
	cfg  Config
	wake chan struct{}
	exit chan struct{}
	mu   struct {
		sync.Mutex
		prepare []uint64
		heads   []head
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
// Prepared hook: an error that wraps ErrDiskFull when the root holds MaxLogs logs below
// next.
func (c *Checkpointer) Prepare(next uint64) {
	c.mu.Lock()
	c.mu.prepare = append(c.mu.prepare, next)
	c.mu.Unlock()
	c.signal()
}

type head struct {
	generation uint64
	snapshot   *state.Snapshot
}

// Head queues switch steps 4 to 6 for the log of generation: the head of s, the durable
// state at the switch, then the removal of the heads and logs that the switch no longer
// needs. A failed head leaves the older head and its logs in place.
func (c *Checkpointer) Head(generation uint64, s *state.Snapshot) {
	c.mu.Lock()
	c.mu.heads = append(c.mu.heads, head{generation: generation, snapshot: s})
	c.mu.Unlock()
	c.signal()
}

// Err returns the error of the last head, or nil once a later head succeeds.
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
		c.cfg.Prepared(next, c.prepare(next))
	}
	for _, h := range heads {
		err := c.write(h)
		c.mu.Lock()
		c.mu.err = err
		c.mu.Unlock()
	}
	c.stepped <- closed
}

// write writes the head h, then cleans the root.
func (c *Checkpointer) write(h head) error {
	err := Write(c.cfg.FS, h.generation, Encode(c.cfg.DB, h.generation, h.snapshot))
	if err != nil {
		return err
	}
	return Clean(c.cfg.FS, h.generation)
}

// prepare makes the file of the log of generation next, unless the root holds MaxLogs
// logs below next. Only a head removes logs.
func (c *Checkpointer) prepare(next uint64) error {
	n, err := countLogs(c.cfg.FS, next)
	if err != nil {
		return err
	}
	if n >= MaxLogs {
		return errors.Wrapf(ErrDiskFull, "%d logs since the oldest head", n)
	}
	return Prepare(c.cfg.FS, next, c.cfg.LogSize)
}

// countLogs returns the number of logs in fs below generation below.
func countLogs(fs xfs.FS, below uint64) (int, error) {
	entries, err := fs.List("")
	if err != nil {
		return 0, err
	}
	var n int
	for _, e := range entries {
		if g, ok := generationOf(e.Name(), ".log"); ok && g < below {
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
