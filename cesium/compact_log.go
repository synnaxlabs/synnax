// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"context"
	"sync"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	xtime "github.com/synnaxlabs/x/time"
	"go.uber.org/zap"
)

// jobBlocks is the most blocks of an old segment that one job of a rewrite reads.
const jobBlocks = 16

var errStopped = errors.New("compactor stopped")

// compactor rewrites the candidates of the state into new segments that hold only their
// live ranges, one segment at a time. Each job of a rewrite runs on the executor: it
// reads and checks at most jobBlocks blocks of the old segment and copies their live
// bytes. The jobs count their reads in limit, the budget of the background reads.
type compactor struct {
	e     *engine
	clock xtime.Clock
	limit *limiter
	wake  chan struct{}
	// started is closed once the orphan pass is done, which would remove the file of a
	// rewrite in progress.
	started  chan struct{}
	stop     chan struct{}
	exit     chan struct{}
	stopping sync.Once
	// failed holds the segments that failed a block check, which stay out of the
	// rewrites until the next open.
	failed set.Set[uint64]
	// The fields below belong to the rewrite in progress.
	rw      rewrite
	buf     []byte
	step    func()
	stepped chan error
	fire    func()
	fired   chan struct{}
}

// rewrite is the copy of an old segment into a new one.
type rewrite struct {
	old state.Segment
	cfg segment.Config
	src *segment.Handle
	w   *segment.Writer
	// ptrs are the pointers of the copy, at their offsets in the new segment.
	ptrs []log.Pointer
	// ptr is the pointer of the next live byte, done the bytes of it copied, and read
	// the stream bytes read.
	ptr  int
	done int64
	read int64
}

func newCompactor(e *engine, clock xtime.Clock, limit *limiter) *compactor {
	c := &compactor{
		e:       e,
		clock:   clock,
		limit:   limit,
		wake:    make(chan struct{}, 1),
		started: make(chan struct{}),
		stop:    make(chan struct{}),
		exit:    make(chan struct{}),
		failed:  set.New[uint64](),
		buf:     make([]byte, jobBlocks*(segment.BlockSize+segment.VariableTrailer)),
		stepped: make(chan error, 1),
		fired:   make(chan struct{}, 1),
	}
	c.step = func() { c.stepped <- c.copyBlocks() }
	c.fire = func() { c.fired <- struct{}{} }
	return c
}

func (c *compactor) signal() {
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

// start lets run rewrite, once the orphan pass is done.
func (c *compactor) start() {
	close(c.started)
	c.signal()
}

// run rewrites the candidates each time it is signaled, until close or until ctx is
// done. It returns the error of ctx or of the state, or nil at close.
func (c *compactor) run(ctx context.Context) error {
	defer close(c.exit)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.stop:
		return nil
	case <-c.started:
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stop:
			return nil
		case <-c.wake:
		}
		cands, err := c.e.state.Candidates()
		for i := 0; err == nil && i < len(cands); i++ {
			if !c.failed.Contains(cands[i].Segment) {
				err = c.rewrite(ctx, cands[i])
			}
		}
		if errors.Is(err, errStopped) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// close stops run, which drops a rewrite in progress, and waits for it to return.
func (c *compactor) close() {
	c.stopping.Do(func() { close(c.stop) })
	<-c.exit
}

// rewrite copies the live ranges of cand into a new segment, then replaces cand with
// it, and logs a failure. It removes the new file when the copy fails or the Replace is
// stale. After any other Replace error, the log can hold the Replace, so the file stays
// for the orphan pass of the next open. It returns only an error of ctx, of close, or
// of a failed state.
func (c *compactor) rewrite(ctx context.Context, cand state.Candidate) error {
	key := cand.Key
	old, ok := c.e.pinSegment(key, cand.Segment)
	if !ok {
		return nil
	}
	pin := held{segments: []uint64{old.ID}}
	defer func() {
		if err := c.e.unpin(pin); err != nil {
			c.e.ins.L.Error("unlink of a rewritten segment failed", zap.Error(err))
		}
	}()
	id, err := c.e.state.NewID()
	if err != nil {
		if c.e.state.Err() != nil {
			return err
		}
		c.e.ins.L.Warn("rewrite of a segment failed", zap.Uint32("key", cand.Key),
			zap.Uint64("segment", old.ID), zap.Error(err))
		return nil
	}
	r, err := c.copyLive(ctx, key, old, id)
	copied := err == nil
	if copied {
		if err = c.e.state.Replace(r); err == nil {
			if err = c.e.reap(); err != nil {
				c.e.ins.L.Error("unlink of a rewritten segment failed", zap.Error(err))
			}
			return nil
		}
	}
	if name := segmentName(key, id, 0); !copied || errors.Is(err, state.ErrStale) {
		if rmErr := c.e.fs.Remove(name); rmErr != nil {
			c.e.ins.L.Error(
				"removal of a failed rewrite failed",
				zap.String("file", name),
				zap.Error(rmErr),
			)
		}
	}
	fields := []zap.Field{
		zap.Uint32("key", cand.Key), zap.Uint64("segment", old.ID),
		zap.Error(err),
	}
	switch {
	case errors.Is(err, errStopped) || ctx.Err() != nil:
		return err
	case errors.Is(err, place.ErrCorrupt):
		c.failed.Add(old.ID)
		c.e.ins.L.Error("segment failed a block check and is not rewritten", fields...)
	case errors.Is(err, state.ErrStale):
		c.e.ins.L.Debug("segment changed during its rewrite", fields...)
	default:
		c.e.ins.L.Warn("rewrite of a segment failed", fields...)
	}
	return nil
}

// copyLive copies the live ranges of old, a segment of key, into the new segment id,
// and returns the Replace of old by it. The file of the new segment is synced.
func (c *compactor) copyLive(
	ctx context.Context,
	key ChannelKey,
	old state.Segment,
	id uint64,
) (*log.Replace, error) {
	src := c.e.openSegment(key, old)
	dst, err := c.e.budget.Create(c.e.fs, segmentName(key, id, 0))
	if err != nil {
		return nil, errors.Join(err, src.Close())
	}
	cfg := c.e.segmentConfig(old)
	newCfg := cfg
	newCfg.ID = id
	ptrs := make([]log.Pointer, len(old.Pointers))
	for i, p := range old.Pointers {
		ptrs[i] = log.Pointer{
			Length:    p.Length,
			Samples:   p.Samples,
			Alignment: uint64(p.Alignment),
			First:     int64(p.First),
			Last:      int64(p.Last),
		}
	}
	c.rw = rewrite{
		old:  old,
		cfg:  cfg,
		src:  src,
		w:    segment.NewWriter(dst, newCfg),
		ptrs: ptrs,
	}
	for c.rw.ptr < len(ptrs) && err == nil {
		read := c.rw.read
		c.e.exec(c.step)
		if err = <-c.stepped; err == nil {
			err = c.pace(ctx, c.rw.read-read)
		}
	}
	if err == nil {
		err = c.e.budget.Sync([]*segment.Handle{dst})
	}
	length, tailCRC := c.rw.w.Commit()
	c.rw = rewrite{}
	if err = errors.Join(err, dst.Close(), src.Close()); err != nil {
		return nil, err
	}
	return &log.Replace{
		Old:      old.ID,
		New:      id,
		Length:   length,
		TailCRC:  tailCRC,
		Pointers: ptrs,
	}, nil
}

// copyBlocks is one job of the rewrite in progress: it reads and checks the blocks from
// the one of the next live byte, and copies the live bytes in them.
func (c *compactor) copyBlocks() error {
	rw := &c.rw
	last := rw.old.Pointers[len(rw.old.Pointers)-1]
	b := (int64(rw.old.Pointers[rw.ptr].Offset) + rw.done) / segment.BlockSize
	blocks := min(jobBlocks, int64(last.Offset+last.Length-1)/segment.BlockSize-b+1)
	slot := rw.cfg.FileOffset(segment.BlockSize)
	n, err := rw.cfg.ReadChecked(
		rw.src,
		c.buf[:blocks*slot],
		b,
		int64(rw.old.Length),
		rw.old.TailCRC,
	)
	if err != nil {
		return err
	}
	rw.read += int64(n)
	from, to := b*segment.BlockSize, b*segment.BlockSize+int64(n)
	for rw.ptr < len(rw.ptrs) {
		p := rw.old.Pointers[rw.ptr]
		at := int64(p.Offset) + rw.done
		if at >= to {
			break
		}
		if rw.done == 0 {
			rw.ptrs[rw.ptr].Offset = uint32(rw.w.Len())
		}
		end := min(int64(p.Offset+p.Length), to)
		if err = rw.w.Copy(c.buf[at-from : end-from]); err != nil {
			return err
		}
		if rw.done += end - at; rw.done == int64(p.Length) {
			rw.ptr, rw.done = rw.ptr+1, 0
		}
	}
	return nil
}

// pace counts n bytes that a job read in the limiter, and waits until the reads fit its
// rate. It returns the error of ctx, or errStopped once close runs.
func (c *compactor) pace(ctx context.Context, n int64) error {
	at := c.limit.reserve(n)
	if !at.After(c.clock.Now()) {
		return nil
	}
	t := c.clock.RunAt(at, c.fire)
	var err error
	select {
	case <-c.fired:
		return nil
	case <-ctx.Done():
		err = ctx.Err()
	case <-c.stop:
		err = errStopped
	}
	if !t.Stop() {
		<-c.fired
	}
	return err
}
