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
	"slices"
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

var (
	errStopped = errors.New("compactor stopped")
	// errNoFit stops the first pass of an encoding that takes more than its limit.
	errNoFit = errors.New("progressions do not fit")
)

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
	// sink takes the live bytes of a pass in order.
	sink func([]byte) error
	// ptrs are the pointers of the copy, at their offsets in the new segment.
	ptrs []log.Pointer
	// ptr is the pointer of the next live byte, done the bytes of it copied, out the
	// live bytes of the pass so far, and read the stream bytes read.
	ptr  int
	done int64
	out  int64
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
// it, and logs a failure. A candidate of encoding whose progressions do not fit, and
// that no rewrite frees enough of, is marked Kept in place of a copy. It removes the
// new file when the copy fails or the Replace is stale. After any other Replace error,
// the log can hold the Replace, so the file stays for the orphan pass of the next open.
// It returns only an error of ctx, of close, or of a failed state.
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
	r, version, err := c.copyLive(ctx, key, old, id, cand)
	if err == nil {
		if err = c.apply(key, old.ID, r, version); err == nil {
			return nil
		}
	}
	if name := segmentName(key, id); r != nil && errors.Is(err, state.ErrStale) {
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

// apply hands the result of a rewrite of segment old of key to the state: a Replace by
// a copy of version, or a Kept mark when r is nil.
func (c *compactor) apply(
	key ChannelKey,
	old uint64,
	r *log.Replace,
	version uint16,
) error {
	if r == nil {
		err := c.e.state.Keep(old)
		if err == nil {
			c.e.ins.L.Debug("clock segment kept at version 1", zap.Uint32("key", key),
				zap.Uint64("segment", old))
		}
		return err
	}
	if err := c.e.state.Replace(r, version); err != nil {
		return err
	}
	if err := c.e.reap(); err != nil {
		c.e.ins.L.Error("unlink of a rewritten segment failed", zap.Error(err))
	}
	return nil
}

// copyLive copies the live ranges of old, a segment of key, into the new segment id,
// and returns the Replace of old by it with the format version of the new segment. A
// candidate of encoding is first written as version 2, when that takes at most half of
// its live bytes. When it does not fit and no rewrite frees enough of old, copyLive
// returns a nil Replace: the caller marks old Kept. The file of the new segment is
// synced.
func (c *compactor) copyLive(
	ctx context.Context,
	key ChannelKey,
	old state.Segment,
	id uint64,
	cand state.Candidate,
) (*log.Replace, uint16, error) {
	src := c.e.budget.Open(c.e.fs, segmentName(key, old.ID))
	ptrs := make([]log.Pointer, len(old.Pointers))
	var live int64
	for i, p := range old.Pointers {
		ptrs[i] = log.Pointer{
			Length:    p.Length,
			Samples:   p.Samples,
			Alignment: uint64(p.Alignment),
			First:     int64(p.First),
			Last:      int64(p.Last),
		}
		live += int64(p.Length)
	}
	c.rw = rewrite{old: old, cfg: c.e.segmentConfig(old), src: src, ptrs: ptrs}
	defer func() { c.rw = rewrite{} }()
	if cand.Encodable {
		encode := func(w *segment.Writer) (bool, error) { return c.encode(ctx, w, live/2) }
		r, err := c.write(key, id, segment.ProgressionVersion, encode)
		if r != nil || err != nil || !cand.Reclaimable {
			return r, segment.ProgressionVersion, errors.Join(err, src.Close())
		}
	}
	r, err := c.write(key, id, old.Version, func(w *segment.Writer) (bool, error) {
		if err := c.pass(ctx, w.Copy); err != nil {
			return false, err
		}
		return true, w.Pad()
	})
	return r, old.Version, errors.Join(err, src.Close())
}

// write creates the file of the new segment id of key at version, fills it with fill,
// and syncs it. It returns the Replace of the old segment of the rewrite in progress by
// it, or a nil Replace when fill reports that the result does not fit. A file that is
// not synced and replaced is removed.
func (c *compactor) write(
	key ChannelKey,
	id uint64,
	version uint16,
	fill func(w *segment.Writer) (bool, error),
) (*log.Replace, error) {
	name := segmentName(key, id)
	dst, err := c.e.budget.Create(c.e.fs, name)
	if err != nil {
		return nil, err
	}
	cfg := c.rw.cfg
	cfg.ID, cfg.Version = id, version
	w := segment.NewWriter(dst, cfg)
	fits, err := fill(w)
	if err == nil && fits {
		err = c.e.budget.Sync([]*segment.Handle{dst})
	}
	length, tailCRC := w.Commit()
	if err = errors.Join(err, dst.Close()); err != nil || !fits {
		return nil, errors.Join(err, c.e.fs.Remove(name))
	}
	return &log.Replace{
		Old:      c.rw.old.ID,
		New:      id,
		Length:   length,
		TailCRC:  tailCRC,
		Pointers: slices.Clone(c.rw.ptrs),
	}, nil
}

// encode writes the stamps of the live pointers of the clock segment of the rewrite in
// progress to w as version 2, in two passes. It reports false when the result would
// take more than limit bytes, or the stamps do not rise.
func (c *compactor) encode(
	ctx context.Context,
	w *segment.Writer,
	limit int64,
) (bool, error) {
	enc := segment.NewEncoder(w, limit)
	err := c.pass(ctx, func(p []byte) error {
		if err := enc.Add(p); err != nil {
			return err
		}
		if !enc.Fits() {
			return errNoFit
		}
		return nil
	})
	if errors.Is(err, errNoFit) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if fits, err := enc.Rewind(); err != nil || !fits {
		return false, err
	}
	if err = c.pass(ctx, enc.Add); err != nil {
		return false, err
	}
	return true, enc.Close()
}

// pass reads the live bytes of the old segment of the rewrite in progress, in jobs of
// the executor whose reads count in the limit, and hands them to sink in order. It
// returns the error of a read, of sink, or of pace.
func (c *compactor) pass(ctx context.Context, sink func([]byte) error) error {
	c.rw.sink, c.rw.ptr, c.rw.done, c.rw.out = sink, 0, 0, 0
	var err error
	for c.rw.ptr < len(c.rw.ptrs) && err == nil {
		read := c.rw.read
		c.e.exec(c.step)
		if err = <-c.stepped; err == nil {
			err = c.pace(ctx, c.rw.read-read)
		}
	}
	return err
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
			rw.ptrs[rw.ptr].Offset = uint32(rw.out)
		}
		end := min(int64(p.Offset+p.Length), to)
		if err = rw.sink(c.buf[at-from : end-from]); err != nil {
			return err
		}
		rw.out += end - at
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
