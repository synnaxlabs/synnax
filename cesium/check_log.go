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
	"bytes"
	"cmp"
	"context"
	"encoding/binary"
	"maps"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	xtime "github.com/synnaxlabs/x/time"
	"go.uber.org/zap"
)

// checker gives each sealed version 0 segment its check file and turns its checks on,
// newest last stamp first. Each job of a segment runs on the executor: it reads at most
// jobBlocks blocks of the old file, each two or three times, and counts the reads in
// the budget of the background reads. In a segment of a clock, it also cuts each
// pointer at its first stamp that does not rise inside the range of the pointer, with
// the samples that the stamps time.
type checker struct {
	e     *engine
	clock xtime.Clock
	pacer *pacer
	// retry is the time from a round that a claim or a change cut short to the next.
	retry time.Duration
	// started is closed once the orphan pass is done, which would remove a check file
	// in progress.
	started  chan struct{}
	stop     chan struct{}
	exit     chan struct{}
	stopping sync.Once
	// failed holds the segments whose block reads never agreed, which stay unchecked
	// until the next open.
	failed set.Set[uint64]
	// keys holds the keys that the next round reads for targets.
	keys set.Set[ChannelKey]
	// reads holds the three reads of a block.
	reads [3][]byte
	// The fields below belong to the segment in progress.
	sg      pass
	step    func()
	stepped chan error
}

// pass is the check of one segment.
type pass struct {
	cfg    segment.Config
	h      *segment.Handle
	length int64
	// next is the next block to read, and read the bytes read so far.
	next, read int64
	sums       []uint32
	// rise is the rise check of a clock segment, or nil.
	rise *rise
}

// target is an unchecked version 0 segment of key, and the last stamp it holds.
type target struct {
	key  ChannelKey
	seg  uint64
	last telem.TimeStamp
}

func newChecker(
	e *engine,
	clock xtime.Clock,
	limit *limiter,
	retry time.Duration,
) *checker {
	c := &checker{
		e:       e,
		clock:   clock,
		retry:   retry,
		started: make(chan struct{}),
		stop:    make(chan struct{}),
		exit:    make(chan struct{}),
		failed:  set.New[uint64](),
		stepped: make(chan error, 1),
	}
	for i := range c.reads {
		c.reads[i] = make([]byte, segment.BlockSize)
	}
	c.pacer = newPacer(clock, limit, c.stop)
	c.step = func() { c.stepped <- c.readBlocks() }
	return c
}

// start lets run check the version 0 segments of keys, once the orphan pass is done.
func (c *checker) start(keys set.Set[ChannelKey]) {
	c.keys = keys
	close(c.started)
}

// run checks the targets in rounds until close or until ctx is done: once at the start,
// and again a while after a round that a claim or a change cut short. It returns the
// error of ctx or of the state, or nil at close.
func (c *checker) run(ctx context.Context) error {
	defer close(c.exit)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.stop:
		return nil
	case <-c.started:
	}
	for {
		again, err := c.round(ctx)
		if errors.Is(err, errStopped) {
			return nil
		}
		if err != nil {
			return err
		}
		c.e.ins.L.Debug("finished a round of the check pass")
		if !again {
			break
		}
		if err = c.pacer.wait(ctx, c.clock.Now().Add(c.retry)); err != nil {
			if errors.Is(err, errStopped) {
				return nil
			}
			return err
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.stop:
		return nil
	}
}

// close stops run, which drops a check in progress, and waits for it to return.
func (c *checker) close() {
	c.stopping.Do(func() { close(c.stop) })
	<-c.exit
}

// round checks each target of the keys of c, and reports whether a claim or a change
// cut it short. The next round reads only the keys of the targets that it cut short.
func (c *checker) round(ctx context.Context) (again bool, err error) {
	targets, err := c.targets()
	if err != nil {
		if c.e.state.Err() != nil {
			return false, err
		}
		c.e.ins.L.Warn("list of the check pass failed", zap.Error(err))
		return true, nil
	}
	next := set.New[ChannelKey]()
	for _, t := range targets {
		retry, err := c.check(ctx, t)
		if err != nil {
			return false, err
		}
		if retry {
			next.Add(t.key)
		}
	}
	c.keys = next
	return len(next) > 0, nil
}

// targets returns the unchecked version 0 segments of the keys of c, newest last stamp
// first.
func (c *checker) targets() ([]target, error) {
	var targets []target
	for _, key := range slices.Sorted(maps.Keys(c.keys)) {
		segs, h, err := c.e.loadLive(key)
		for _, seg := range segs {
			if seg.Version != 0 || seg.Checked || c.failed.Contains(seg.ID) {
				continue
			}
			_, last := bounds(seg)
			targets = append(targets, target{key: key, seg: seg.ID, last: last})
		}
		if err = errors.Join(err, c.e.unpin(h)); err != nil {
			return nil, err
		}
	}
	slices.SortFunc(
		targets,
		func(a, b target) int { return cmp.Compare(b.last, a.last) },
	)
	return targets, nil
}

// loadLive pins the segments and parts of key, and returns the live records of key: the
// segments of its Channel that no part holds, and the live records of its parts. The
// caller unpins h, also after an error.
func (e *engine) loadLive(key ChannelKey) (segs []state.Segment, h held, err error) {
	ch, h, err, unlinkErr := e.load(key, held{})
	if err = errors.Join(err, unlinkErr); err != nil || ch == nil {
		return nil, h, err
	}
	pre := state.Preload{Key: key}
	for _, part := range h.parts {
		r, err := e.cache.Records(part, key)
		if err != nil {
			return nil, h, err
		}
		addLive(&pre, ch.Parts, part, r, telem.TimeStampMin, telem.TimeStampMax)
	}
	for _, seg := range ch.Segments {
		if seg.Part == 0 {
			pre.Segments = append(pre.Segments, seg)
		}
	}
	return pre.Segments, h, nil
}

// check checks t, and logs a failure. It reports whether a claim or a change cut the
// check short. It returns only an error of ctx, of close, or of a failed state.
func (c *checker) check(ctx context.Context, t target) (retry bool, err error) {
	segs, h, err := c.e.loadLive(t.key)
	defer func() {
		if unpinErr := c.e.unpin(h); unpinErr != nil {
			c.e.ins.L.Error("unlink of a checked segment failed", zap.Error(unpinErr))
		}
	}()
	fields := []zap.Field{zap.Uint32("key", t.key), zap.Uint64("segment", t.seg)}
	if err != nil {
		c.e.ins.L.Warn("check of a segment failed", append(fields, zap.Error(err))...)
		return true, c.fatal(err)
	}
	i := slices.IndexFunc(segs, func(s state.Segment) bool { return s.ID == t.seg })
	if i < 0 {
		return false, nil
	}
	seg := segs[i]
	bad, err := c.read(ctx, t.key, seg)
	switch {
	case errors.Is(err, errStopped) || ctx.Err() != nil:
		return false, err
	case errors.Is(err, place.ErrCorrupt):
		c.failed.Add(seg.ID)
		c.e.ins.L.Error(
			"segment failed its block reads and stays unchecked",
			append(fields, zap.Error(err))...,
		)
		return false, nil
	case err != nil:
		c.e.ins.L.Warn("check of a segment failed", append(fields, zap.Error(err))...)
		return false, nil
	}
	return c.finish(t.key, seg, bad, fields)
}

// finish logs the cuts of bad, the stamps of seg that fail the rise check, then the
// Check of seg, a segment of key, and logs a failure with fields. It reports whether a
// claim or a change cut it short. It returns only an error of a failed state.
func (c *checker) finish(
	key ChannelKey,
	seg state.Segment,
	bad []badStamp,
	fields []zap.Field,
) (retry bool, err error) {
	for _, b := range bad {
		if err = c.cut(key, b); err != nil {
			break
		}
	}
	if err == nil {
		err = c.logCheck(key, seg)
	}
	if reapErr := c.e.reap(); reapErr != nil {
		c.e.ins.L.Error("unlink of a cut segment failed", zap.Error(reapErr))
	}
	switch {
	case err == nil:
		return false, nil
	case errors.Is(err, state.ErrConflict):
		c.e.ins.L.Debug("a claim holds a cut of the check pass", fields...)
		return true, nil
	case errors.Is(err, state.ErrStale):
		c.e.ins.L.Debug(
			"dropped the check of a segment that changed during its pass",
			append(fields, zap.Error(err))...,
		)
		return true, nil
	}
	c.e.ins.L.Warn("check of a segment failed", append(fields, zap.Error(err))...)
	return false, c.fatal(err)
}

// fatal returns err when the state failed, and nil otherwise.
func (c *checker) fatal(err error) error {
	if c.e.state.Err() != nil {
		return err
	}
	return nil
}

// read reads each block of seg, a segment of key, in jobs on the executor, and leaves
// the sums of its blocks in c.sg. It returns the stamps of a clock segment that fail
// the rise check, or an error that wraps place.ErrCorrupt when no two reads of a block
// agree.
func (c *checker) read(
	ctx context.Context,
	key ChannelKey,
	seg state.Segment,
) ([]badStamp, error) {
	c.sg = pass{
		cfg:    c.e.segmentConfig(seg),
		h:      c.e.openSegment(key, seg),
		length: int64(seg.Length),
	}
	if seg.Clock == key {
		c.sg.rise = newRise(seg.Pointers)
	}
	var err error
	for c.sg.next*segment.BlockSize < c.sg.length && err == nil {
		read := c.sg.read
		c.e.exec(c.step)
		if err = <-c.stepped; err == nil {
			err = c.pacer.pace(ctx, c.sg.read-read)
		}
	}
	err = errors.Join(err, c.sg.h.Close())
	var bad []badStamp
	if c.sg.rise != nil {
		bad = c.sg.rise.bad
	}
	c.sg.h, c.sg.rise = nil, nil
	return bad, err
}

// readBlocks is one job of the segment in progress: it reads at most jobBlocks blocks
// from the next one, and adds the sum of each.
func (c *checker) readBlocks() error {
	sg := &c.sg
	for end := sg.next + jobBlocks; sg.next < end; sg.next++ {
		off := sg.next * segment.BlockSize
		if off >= sg.length {
			return nil
		}
		block, err := c.agree(off, int(min(segment.BlockSize, sg.length-off)))
		if err != nil {
			return err
		}
		sg.sums = append(sg.sums, sg.cfg.BlockSum(sg.next, block))
		if sg.rise != nil {
			sg.rise.feed(block, off)
		}
	}
	return nil
}

// agree reads the size bytes at file offset off of the segment in progress two times,
// and a third time when they differ, and returns the bytes that two reads agree on. It
// returns an error that wraps place.ErrCorrupt when no two of them agree.
func (c *checker) agree(off int64, size int) ([]byte, error) {
	sg := &c.sg
	read := func(i int) ([]byte, error) {
		p := c.reads[i][:size]
		n, err := sg.h.ReadAt(p, off)
		sg.read += int64(n)
		if n < size {
			return nil, errors.Wrapf(err, "read of block %d of segment %d",
				off/segment.BlockSize, sg.cfg.ID)
		}
		return p, nil
	}
	a, err := read(0)
	if err != nil {
		return nil, err
	}
	b, err := read(1)
	switch {
	case err != nil:
		return nil, err
	case bytes.Equal(a, b):
		return a, nil
	}
	third, err := read(2)
	switch {
	case err != nil:
		return nil, err
	case bytes.Equal(third, a):
		return a, nil
	case bytes.Equal(third, b):
		return b, nil
	}
	return nil, errors.Wrapf(place.ErrCorrupt, "no two of three reads of block %d of "+
		"segment %d agree", off/segment.BlockSize, sg.cfg.ID)
}

// cut cuts the pointer of b, a pointer of clock, at its first stamp that fails the rise
// check, with the samples of each key that clock times there. The cuts are by position,
// since the stamps there do not rise. The timed keys go first, since the state refuses
// a cut of a range of clock that a key still uses. It returns an error that wraps
// state.ErrConflict when another claim reaches the range.
func (c *checker) cut(clock ChannelKey, b badStamp) error {
	tr := telem.TimeRange{Start: b.prev + 1, End: b.p.Last + 1}
	if b.i == 0 {
		tr.Start = b.p.First
	}
	at := b.p.Alignment.AddSamples(b.i)
	timed, err := c.e.timedKeys(clock, tr)
	if err != nil {
		return err
	}
	for _, key := range append(timed, clock) {
		if _, err := c.e.logTombstone(key, clock, tr, func(
			d *state.Delete,
		) (left, right *log.Cut, err error) {
			if d.Right != nil {
				return nil, nil, errors.Newf("segment %d of key %d holds samples past "+
					"the end of the cut range of clock %d", d.Right.Segment.ID, key, clock)
			}
			if d.Left == nil {
				return nil, nil, nil
			}
			left, err = c.e.cutBefore(key, d.Left, at, b.prev)
			return left, nil, err
		}); err != nil {
			return err
		}
	}
	return nil
}

// timedKeys returns the keys other than clock that clock times in tr: those of the
// segments that the state holds, and those of the live part records that reach tr.
func (e *engine) timedKeys(clock ChannelKey, tr telem.TimeRange) ([]ChannelKey, error) {
	users, err := e.clockUsers(clock, tr.Start, tr.End-1)
	if err != nil {
		return nil, err
	}
	keys := set.New[ChannelKey]()
	for _, u := range users {
		if u.Key != clock {
			keys.Add(u.Key)
		}
	}
	for _, key := range e.state.Keys() {
		ch := e.state.Channel(key)
		if key != clock && ch != nil && slices.ContainsFunc(
			ch.Segments,
			func(s state.Segment) bool { return s.Clock == clock },
		) {
			keys.Add(key)
		}
	}
	return slices.Sorted(maps.Keys(keys)), nil
}

// cutBefore returns the left cut of the pointer of edge, a pointer of key, that keeps
// its samples before alignment at, the last of them stamped prev.
func (e *engine) cutBefore(
	key ChannelKey,
	edge *state.Edge,
	at telem.Alignment,
	prev telem.TimeStamp,
) (*log.Cut, error) {
	p := edge.Pointer
	k := int64(at.SampleIndex()) - int64(p.Alignment.SampleIndex())
	if at.DomainIndex() != p.Alignment.DomainIndex() || k <= 0 ||
		k >= int64(p.Samples) {
		return nil, errors.Newf("alignment %d lies outside the pointer of segment %d "+
			"of key %d at alignment %d", uint64(at), edge.Segment.ID, key,
			uint64(p.Alignment))
	}
	dt := telem.DataTypeCode(edge.Segment.DataType).DataType()
	offset, err := e.sampleOffset(key, dt, edge, int(k))
	if err != nil {
		return nil, err
	}
	return &log.Cut{
		Segment:   edge.Segment.ID,
		Offset:    offset,
		Alignment: uint64(at),
		Stamp:     int64(prev),
	}, nil
}

// logCheck writes the check file of seg, a segment of key, with the sums of the
// segment in progress, makes it durable, and turns the checks of seg on. It removes
// the file when the state refuses the Check as stale. After any other error, the log
// can hold the Check, so the file stays for the orphan pass of the next open.
func (c *checker) logCheck(key ChannelKey, seg state.Segment) error {
	name := checkName(key, seg.ID)
	if err := c.e.writeChecks(key, name, c.sg.cfg.EncodeChecks(c.sg.sums)); err != nil {
		return err
	}
	first, last := bounds(seg)
	pre, err := c.e.records(key, first, last)
	if err == nil {
		err = c.e.state.Check(&log.Check{Segment: seg.ID, Key: key}, pre)
	}
	if errors.Is(err, state.ErrStale) {
		if rmErr := c.e.fs.Remove(name); rmErr != nil {
			c.e.ins.L.Error(
				"removal of a dropped check file failed",
				zap.String("file", name),
				zap.Error(rmErr),
			)
		}
	}
	return err
}

// bounds returns the first and the last stamp of the pointers of seg, which a sealed
// segment always holds.
func bounds(seg state.Segment) (first, last telem.TimeStamp) {
	first, last = telem.TimeStampMax, telem.TimeStampMin
	for _, p := range seg.Pointers {
		first, last = min(first, p.First), max(last, p.Last)
	}
	return first, last
}

// writeChecks writes b to the check file name of key, then syncs the file and the
// directory of key.
func (e *engine) writeChecks(key ChannelKey, name string, b []byte) error {
	f, err := e.fs.Open(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	if err = errors.Join(err, f.Close()); err != nil {
		return err
	}
	return e.fs.SyncDir(keyToDirName(key))
}

// badStamp is the first stamp i of pointer p of a clock that lies outside the range of
// p or does not rise, and prev the stamp before it in p.
type badStamp struct {
	p    state.Pointer
	i    uint32
	prev telem.TimeStamp
}

// rise checks that the stamps of each pointer of a clock segment rise inside the range
// of the pointer. It takes the bytes of the segment in stream order.
type rise struct {
	ptrs []state.Pointer
	// p is the pointer of the next stamp, i its sample in p, and prev the stamp before
	// it.
	p    int
	i    uint32
	prev telem.TimeStamp
	// part holds the bytes of a stamp that a block end split.
	part []byte
	bad  []badStamp
}

func newRise(ptrs []state.Pointer) *rise {
	r := &rise{ptrs: slices.Clone(ptrs), part: make([]byte, 0, 8)}
	slices.SortFunc(r.ptrs, func(a, b state.Pointer) int {
		return cmp.Compare(a.Offset, b.Offset)
	})
	return r
}

// feed checks the stamps in b, the bytes of the segment from stream offset from.
func (r *rise) feed(b []byte, from int64) {
	to := from + int64(len(b))
	for r.p < len(r.ptrs) {
		ptr := r.ptrs[r.p]
		at := int64(ptr.Offset) + 8*int64(r.i) + int64(len(r.part))
		if at >= to {
			return
		}
		n := min(8-int64(len(r.part)), to-at)
		r.part = append(r.part, b[at-from:at-from+n]...)
		if len(r.part) < 8 {
			return
		}
		r.check(telem.TimeStamp(binary.LittleEndian.Uint64(r.part)))
		r.part = r.part[:0]
	}
}

// check checks stamp, sample i of pointer p.
func (r *rise) check(stamp telem.TimeStamp) {
	ptr := r.ptrs[r.p]
	if stamp < ptr.First || stamp > ptr.Last || (r.i > 0 && stamp <= r.prev) {
		r.bad = append(r.bad, badStamp{p: ptr, i: r.i, prev: r.prev})
		r.p, r.i = r.p+1, 0
		return
	}
	r.prev, r.i = stamp, r.i+1
	if r.i == ptr.Samples {
		r.p, r.i = r.p+1, 0
	}
}
