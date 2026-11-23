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

// errDiskFull is the error of the file of a next log when the root holds maxLogs logs,
// because heads keep failing.
var errDiskFull = errors.New("disk full")

// maxLogs is the most logs that the root keeps while heads fail.
const maxLogs = 16

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
	// Parts are the parts that the head of recovery names, or nil for a new database.
	Parts *Index
	// PartSize is the size of the sealed segment records in a head at which a switch
	// moves them to a part. Zero means 8 MiB.
	PartSize int
	// Live is the generation of the head that recovery opened. Before its first switch,
	// the checkpointer cleans the root against it, as Clean does, so a part that a
	// crash left is gone before a switch writes a part of its identity. Zero means no
	// such clean.
	Live uint64
}

// validate returns an error that wraps validate.ErrValidation when a field of c is not
// set.
func (c Config) validate() error {
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
// executor, and its calls return at once. A Checkpointer is safe for concurrent use.
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
	// index holds the parts that the last head names. Only a step touches it.
	index *Index
	// cleaned is true once the clean against the head of recovery ran.
	cleaned bool
}

// New returns a checkpointer of cfg.
func New(cfg Config) (*Checkpointer, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	c := &Checkpointer{
		cfg:     cfg,
		wake:    make(chan struct{}, 1),
		exit:    make(chan struct{}),
		stepped: make(chan bool, 1),
		index:   cfg.Parts,
	}
	if c.index == nil {
		c.index = &Index{}
	}
	if c.cfg.PartSize == 0 {
		c.cfg.PartSize = defaultPartSize
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

// Prepare queues switch step 1 for the log of generation next: the file of the log, and
// the removal of a leftover head of that generation. It reports the result to the
// Prepared hook: an error that ends in "disk full" when the root holds 16 logs below
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

// err returns the error of the last head, or nil once a later head succeeds.
func (c *Checkpointer) err() error {
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
	if !c.cleaned {
		c.cleaned = true
		if c.cfg.Live != 0 {
			err := clean(c.cfg.FS, c.cfg.Live, c.index.ids())
			c.mu.Lock()
			c.mu.err = err
			c.mu.Unlock()
		}
	}
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

// write writes the parts that the switch of h needs, then the head, then cleans the
// root. On an error, the head of the attempt is gone and the root synced, and write
// removes the parts of the attempt and keeps the index of the last head.
func (c *Checkpointer) write(h head) error {
	pl := c.plan(h.snapshot)
	// The identities of a failed attempt stay used: its parts may stay on disk.
	defer func() { c.index.limit = max(c.index.limit, pl.index.limit) }()
	var written []uint64
	for _, w := range pl.writes {
		p, err := writePart(c.cfg.FS, c.cfg.DB, w.id, h.generation, w.channels)
		if err != nil {
			return c.undo(written, err)
		}
		written = append(written, w.id)
		pl.index.parts = append(pl.index.parts, index(p, w.channels))
	}
	refs := make([]Part, len(pl.index.parts))
	for i, p := range pl.index.parts {
		refs[i] = p.Part
	}
	s := *h.snapshot
	s.Channels, s.Limits.Parts = pl.head, pl.index.limit
	if gone, err := writeFile(
		c.cfg.FS,
		Name(h.generation),
		encodeHead(c.cfg.DB, h.generation, &s, refs),
	); err != nil {
		// A head that may stay names the parts.
		if !gone {
			return err
		}
		return c.undo(written, err)
	}
	c.index = pl.index
	return clean(c.cfg.FS, h.generation, c.index.ids())
}

// undo removes the parts of a failed switch, then syncs the root.
func (c *Checkpointer) undo(written []uint64, err error) error {
	for _, id := range written {
		err = errors.Join(err, c.cfg.FS.Remove(partName(id)))
	}
	return errors.Join(err, c.cfg.FS.SyncDir(""))
}

// prepare makes the file of the log of generation next, unless the root holds maxLogs
// logs below next. Only a head removes logs.
func (c *Checkpointer) prepare(next uint64) error {
	n, err := countLogs(c.cfg.FS, next)
	if err != nil {
		return err
	}
	if n >= maxLogs {
		return errors.Wrapf(errDiskFull, "%d logs since the oldest head", n)
	}
	return prepareLog(c.cfg.FS, next, c.cfg.LogSize)
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
// returns the error of the last head, or nil once a later head succeeds.
func (c *Checkpointer) Close() error {
	c.mu.Lock()
	c.mu.closed = true
	c.mu.Unlock()
	c.signal()
	<-c.exit
	return c.err()
}
