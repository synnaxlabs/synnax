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
	"cmp"
	"context"
	"encoding/binary"
	"slices"
	"sort"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// deleteLog deletes tr from each of keys on the pointer log layout: the data keys
// first, then the indexes, since the state refuses a clock range that a data key still
// uses. Each key with data in tr gets one durable Tombstone and a HistoryChange. It
// refuses a range that starts after it ends. The caller holds db.mu.
func (db *DB) deleteLog(
	ctx context.Context,
	keys []ChannelKey,
	tr telem.TimeRange,
) error {
	if !tr.Valid() {
		return errors.Wrapf(
			validate.ErrValidation,
			"delete start %d cannot be after delete end %d",
			tr.Start,
			tr.End,
		)
	}
	var data, indexes []Channel
	for _, k := range lo.Uniq(keys) {
		ch, ok := db.mu.channels[k]
		if !ok || ch.Virtual {
			return channel.NewNotFoundError(k)
		}
		if ch.IsIndex {
			indexes = append(indexes, ch)
		} else {
			data = append(data, ch)
		}
	}
	for _, ch := range append(data, indexes...) {
		clock := ch.Index
		if ch.IsIndex {
			clock = ch.Key
		}
		changed, err := db.engine.deleteRange(ch.Key, ch.DataType, clock, tr)
		if err != nil {
			return channel.Wrap(ch.Key, err)
		}
		if changed {
			db.historyChanges.Notify(ctx, HistoryChange{Key: ch.Key, TimeRange: tr})
		}
	}
	return db.engine.reap()
}

// deleteRange logs a durable Tombstone of tr for key, whose samples are of type dt and
// timed by clock, and returns false when key holds no data in tr. A claim of the state
// keeps commits out of tr from the cuts to the ack of the Tombstone.
func (e *engine) deleteRange(
	key ChannelKey,
	dt telem.DataType,
	clock ChannelKey,
	tr telem.TimeRange,
) (bool, error) {
	if tr.Start == tr.End {
		return false, nil
	}
	// A delete of a clock checks the records of the keys that it times.
	timed := key == clock
	var (
		d   *state.Delete
		pre state.Preload
		err error
	)
	for {
		if pre, err = e.preload(key, tr.Start, tr.End-1, timed); err != nil {
			return false, err
		}
		d, err = e.state.BeginDelete(key, tr.Start, tr.End, pre)
		if !errors.Is(err, state.ErrStale) {
			break
		}
	}
	if err != nil {
		return false, err
	}
	if d.Empty {
		return false, d.Release()
	}
	t := &log.Tombstone{Key: key, Start: int64(tr.Start), End: int64(tr.End)}
	if d.Left != nil {
		t.Left, err = e.cutAt(key, dt, clock, d.Left, tr.Start, true)
	}
	if err == nil && d.Right != nil {
		t.Right, err = e.cutAt(key, dt, clock, d.Right, tr.End, false)
	}
	if err == nil {
		err = d.Check(t)
	}
	if err == nil {
		var entry log.Entry
		entry.Add(t)
		_, err = e.log.Append(&entry).Wait()
	}
	return err == nil, errors.Join(err, d.Release())
}

// cutAt returns the cut of the pointer of edge at the first sample whose stamp is at
// or after at: a left cut, which keeps the samples before it, or a right cut, which
// keeps it and the samples after it. It reads the stamps of the pointer from clock.
func (e *engine) cutAt(
	key ChannelKey,
	dt telem.DataType,
	clock ChannelKey,
	edge *state.Edge,
	at telem.TimeStamp,
	left bool,
) (*log.Cut, error) {
	p := edge.Pointer
	c, err := e.openStamps(clock, p)
	if err != nil {
		return nil, err
	}
	k, stamp, err := c.search(at, left)
	if err = errors.Join(err, c.close()); err != nil {
		return nil, err
	}
	if k == 0 || k == int(p.Samples) {
		return nil, errors.Newf(
			"stamps of segment %d of key %d do not hold %d inside its pointer",
			edge.Segment.ID,
			key,
			at,
		)
	}
	var offset uint32
	if dt.IsVariable() {
		if offset, err = e.variableOffset(key, edge, k); err != nil {
			return nil, err
		}
	} else {
		offset = p.Offset + uint32(dt.Density().Size(int64(k)))
	}
	return &log.Cut{
		Segment:   edge.Segment.ID,
		Offset:    offset,
		Alignment: uint64(p.Alignment.AddSamples(uint32(k))),
		Stamp:     int64(stamp),
	}, nil
}

// stampSpan is the run of stamps of a stampReader that one pointer of the clock holds:
// those of samples lo to below hi, the first at stream offset off of seg.
type stampSpan struct {
	seg    state.Segment
	lo, hi int
	off    int64
}

// stampReader reads the stamps of samples of a clock for a search by time. It reads and
// checks one block of the clock at a time, or decodes the stamps of one block of a
// version 2 segment, and keeps the last one, so a search reads a few blocks, not every
// stamp. It keeps what the load of the clock pinned, and a reader of the pool of the
// engine, until close.
type stampReader struct {
	e       *engine
	clock   ChannelKey
	samples int
	spans   []stampSpan
	held    held
	rd      *segment.Reader
	// h is the handle of segment hSeg. buf holds stream bytes of span from stream
	// offset from, or nothing when span is -1.
	h    *segment.Handle
	hSeg uint64
	span int
	from int64
	buf  []byte
}

// openStamps returns the reader of the stamps that clock holds for the samples of
// pointer p of a key it times: from the segments of clock that the state holds, and
// from the records of its parts that reach the time range of p, both read from one
// Channel. The caller closes it.
func (e *engine) openStamps(clock ChannelKey, p state.Pointer) (*stampReader, error) {
	ch, h, err, unlinkErr := e.load(clock, held{})
	r := &stampReader{
		e:       e,
		clock:   clock,
		samples: int(p.Samples),
		span:    -1,
		held:    h,
		rd:      e.reader(),
	}
	if err = errors.Join(err, unlinkErr); err != nil {
		return nil, errors.Join(err, r.close())
	}
	if ch == nil {
		return nil, errors.Join(
			errors.Newf("clock %d holds no stamps", clock),
			r.close(),
		)
	}
	pre := state.Preload{Key: clock}
	for _, part := range h.parts {
		if part.First > p.Last || p.First > part.Last {
			continue
		}
		rec, err := e.cache.Records(part, clock)
		if err != nil {
			return nil, errors.Join(err, r.close())
		}
		addLive(&pre, ch.Parts, part, rec, p.First, p.Last)
	}
	segs := pre.Segments
	for _, seg := range ch.Segments {
		if seg.Part == 0 {
			segs = append(segs, seg)
		}
	}
	from, found := p.Alignment.SampleIndex(), uint32(0)
	for _, seg := range segs {
		for _, q := range seg.Pointers {
			if q.Alignment.DomainIndex() != p.Alignment.DomainIndex() {
				continue
			}
			lo, hi := max(from, q.Alignment.SampleIndex()),
				min(from+p.Samples, q.Alignment.SampleIndex()+q.Samples)
			if lo >= hi {
				continue
			}
			r.spans = append(r.spans, stampSpan{
				seg: seg,
				lo:  int(lo - from),
				hi:  int(hi - from),
				off: int64(q.Offset) + int64(lo-q.Alignment.SampleIndex())*8,
			})
			found += hi - lo
		}
	}
	if found != p.Samples {
		return nil, errors.Join(
			errors.Newf("clock %d holds %d of the %d stamps from alignment %d",
				clock, found, p.Samples, uint64(p.Alignment)),
			r.close(),
		)
	}
	slices.SortFunc(
		r.spans,
		func(a, b stampSpan) int { return cmp.Compare(a.lo, b.lo) },
	)
	return r, nil
}

// search returns k, the index of the first stamp at or after at, by a binary search
// that reads one block at each step that leaves the block it holds. It also returns
// the stamp at k-1 when left is true, and the stamp at k otherwise, or zero when that
// index holds no stamp.
func (r *stampReader) search(
	at telem.TimeStamp,
	left bool,
) (int, telem.TimeStamp, error) {
	var err error
	n := r.samples
	k := sort.Search(n, func(i int) bool {
		if err != nil {
			return true
		}
		var stamp telem.TimeStamp
		stamp, err = r.at(i)
		return err != nil || stamp >= at
	})
	if err != nil {
		return 0, 0, err
	}
	i := k
	if left {
		i = k - 1
	}
	if i < 0 || i >= n {
		return k, 0, nil
	}
	stamp, err := r.at(i)
	return k, stamp, err
}

// at returns the stamp of sample i.
func (r *stampReader) at(i int) (telem.TimeStamp, error) {
	s, _ := slices.BinarySearchFunc(r.spans, i, func(sp stampSpan, i int) int {
		if i >= sp.hi {
			return -1
		}
		if i < sp.lo {
			return 1
		}
		return 0
	})
	off := r.spans[s].off + int64(i-r.spans[s].lo)*8
	if s != r.span || off < r.from || off+8 > r.from+int64(len(r.buf)) {
		if err := r.load(s, off); err != nil {
			return 0, err
		}
	}
	return telem.TimeStamp(binary.LittleEndian.Uint64(r.buf[off-r.from:])), nil
}

// load reads the bytes of span s in the block of stream offset off, and the rest of
// the stamp at off when it crosses the end of that block.
func (r *stampReader) load(s int, off int64) error {
	sp := r.spans[s]
	if r.h == nil || r.hSeg != sp.seg.ID {
		if err := r.closeHandle(); err != nil {
			return err
		}
		r.h = r.e.budget.Open(r.e.fs, segmentName(r.clock, sp.seg.ID))
		r.hSeg = sp.seg.ID
	}
	start := off - off%segment.BlockSize
	from := max(start, sp.off)
	to := min(max(start+segment.BlockSize, off+8), sp.off+int64(sp.hi-sp.lo)*8)
	r.buf = slices.Grow(r.buf[:0], int(to-from))[:to-from]
	r.span = -1
	c := r.e.committed(r.clock, sp.seg)
	var err error
	if sp.seg.Version == segment.ProgressionVersion {
		err = r.rd.Stamps(r.h, c, r.buf, from/stampSize)
	} else {
		err = r.rd.Read(r.h, c, r.buf, from)
	}
	if err != nil {
		return err
	}
	r.span, r.from = s, from
	return nil
}

// close puts the reader of r back in the pool, closes the handle of the segment that r
// reads, and unpins what the load of the clock pinned. It is called once.
func (r *stampReader) close() error {
	r.rd.Reset()
	r.e.readers.Put(r.rd)
	return errors.Join(r.closeHandle(), r.e.unpin(r.held))
}

// closeHandle closes the handle of the segment that r reads.
func (r *stampReader) closeHandle() error {
	if r.h == nil {
		return nil
	}
	h := r.h
	r.h = nil
	return h.Close()
}

// variableOffset returns the stream offset of sample k of the pointer of edge, whose
// samples each hold a uint32 length and then that many bytes. It reads the trailers of
// a few blocks, not the whole pointer.
func (e *engine) variableOffset(
	key ChannelKey,
	edge *state.Edge,
	k int,
) (uint32, error) {
	p := edge.Pointer
	h := e.budget.Open(e.fs, segmentName(key, edge.Segment.ID))
	at, err := e.segmentConfig(edge.Segment).SeekFrom(
		h,
		make([]byte, segment.BlockSize),
		int64(p.Offset),
		uint32(k),
		int64(edge.Segment.Length),
	)
	if err = errors.Join(err, h.Close()); err != nil {
		return 0, err
	}
	if at >= int64(p.Offset)+int64(p.Length) {
		return 0, errors.Newf("pointer of segment %d of key %d ends inside a sample",
			edge.Segment.ID, key)
	}
	return uint32(at), nil
}
