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
	"encoding/binary"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/log"
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
	return e.logTombstone(
		key,
		clock,
		tr,
		func(d *state.Delete) (l, r *log.Cut, err error) {
			if d.Left != nil {
				l, err = e.cutAt(key, dt, clock, d.Left, tr.Start, true)
			}
			if err == nil && d.Right != nil {
				r, err = e.cutAt(key, dt, clock, d.Right, tr.End, false)
			}
			return l, r, err
		},
	)
}

// logTombstone claims tr of key, a key that clock times or clock itself, and logs a
// durable Tombstone of tr with the cuts that cuts returns for the edges of the claim.
// It
// returns false when key holds no data in tr.
func (e *engine) logTombstone(
	key, clock ChannelKey,
	tr telem.TimeRange,
	cuts func(d *state.Delete) (left, right *log.Cut, err error),
) (bool, error) {
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
	t.Left, t.Right, err = cuts(d)
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
	stamps, err := e.stamps(clock, p)
	if err != nil {
		return nil, err
	}
	k, _ := slices.BinarySearch(stamps, at)
	if k == 0 || k == len(stamps) {
		return nil, errors.Newf(
			"stamps of segment %d of key %d do not hold %d inside its pointer",
			edge.Segment.ID,
			key,
			at,
		)
	}
	offset, err := e.sampleOffset(key, dt, edge, k)
	if err != nil {
		return nil, err
	}
	stamp := stamps[k]
	if left {
		stamp = stamps[k-1]
	}
	return &log.Cut{
		Segment:   edge.Segment.ID,
		Offset:    offset,
		Alignment: uint64(p.Alignment.AddSamples(uint32(k))),
		Stamp:     int64(stamp),
	}, nil
}

// sampleOffset returns the stream offset of sample k of the pointer of edge, a pointer
// of key whose samples are of type dt.
func (e *engine) sampleOffset(
	key ChannelKey,
	dt telem.DataType,
	edge *state.Edge,
	k int,
) (uint32, error) {
	if dt.IsVariable() {
		return e.variableOffset(key, edge, k)
	}
	return edge.Pointer.Offset + uint32(dt.Density().Size(int64(k))), nil
}

// stamps returns the stamps that clock holds for the samples of pointer p of a key it
// times: from the segments of clock that the state holds, and from the records of its
// parts that reach the time range of p, both read from one Channel.
func (e *engine) stamps(
	clock ChannelKey,
	p state.Pointer,
) (out []telem.TimeStamp, err error) {
	ch, h, err, unlinkErr := e.load(clock, held{})
	defer func() { err = errors.Join(err, unlinkErr, e.unpin(h)) }()
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, errors.Newf("clock %d holds no stamps", clock)
	}
	pre := state.Preload{Key: clock}
	for _, part := range h.parts {
		if part.First > p.Last || p.First > part.Last {
			continue
		}
		r, err := e.cache.Records(part, clock)
		if err != nil {
			return nil, err
		}
		addLive(&pre, ch.Parts, part, r, p.First, p.Last)
	}
	segs := pre.Segments
	for _, seg := range ch.Segments {
		if seg.Part == 0 {
			segs = append(segs, seg)
		}
	}
	out = make([]telem.TimeStamp, p.Samples)
	from, found := p.Alignment.SampleIndex(), uint32(0)
	var b []byte
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
			b = slices.Grow(b[:0], int(hi-lo)*8)[:int(hi-lo)*8]
			off := int64(q.Offset) + int64(lo-q.Alignment.SampleIndex())*8
			if err := e.readSegment(clock, seg, b, off); err != nil {
				return nil, err
			}
			for i := range hi - lo {
				out[lo-from+i] = telem.TimeStamp(binary.LittleEndian.Uint64(b[8*i:]))
			}
			found += hi - lo
		}
	}
	if found != p.Samples {
		return nil, errors.Newf("clock %d holds %d of the %d stamps from alignment %d",
			clock, found, p.Samples, uint64(p.Alignment))
	}
	return out, nil
}

// variableOffset returns the stream offset of sample k of the pointer of edge, whose
// samples each hold a uint32 length and then that many bytes.
func (e *engine) variableOffset(
	key ChannelKey,
	edge *state.Edge,
	k int,
) (uint32, error) {
	p := edge.Pointer
	b := make([]byte, p.Length)
	if err := e.readSegment(key, edge.Segment, b, int64(p.Offset)); err != nil {
		return 0, err
	}
	var at uint32
	for range k {
		if int(at)+4 > len(b) {
			return 0, errors.Newf(
				"pointer of segment %d of key %d ends inside a sample",
				edge.Segment.ID,
				key,
			)
		}
		at += 4 + binary.LittleEndian.Uint32(b[at:])
	}
	if at >= p.Length {
		return 0, errors.Newf("pointer of segment %d of key %d ends inside a sample",
			edge.Segment.ID, key)
	}
	return p.Offset + at, nil
}

// readSegment reads len(b) stream bytes at off of segment seg of key.
func (e *engine) readSegment(
	key ChannelKey,
	seg state.Segment,
	b []byte,
	off int64,
) error {
	cfg, err := e.readConfig(key, seg)
	if err != nil {
		return err
	}
	h := e.openSegment(key, seg)
	return errors.Join(cfg.ReadAt(h, b, off), h.Close())
}
