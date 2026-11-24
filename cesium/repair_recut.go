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
	"slices"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// recut returns t with each cut that names a pointer of the state by another alignment
// number, such as a range that the repair rebuilt with a fresh one, taken again by the
// time range of t against the state. It returns t itself when no cut needs that, or
// when a new cut fails, so that the state skips t.
func (e *engine) recut(t *log.Tombstone) (*log.Tombstone, error) {
	segs := e.state.ReplaySegments(t.Key)
	start, end := telem.TimeStamp(t.Start), telem.TimeStamp(t.End)
	left, lok := edgeAt(segs, start)
	right, rok := edgeAt(segs, end)
	if !stale(t.Left, left, lok) && !stale(t.Right, right, rok) {
		return t, nil
	}
	out := *t
	if stale(t.Left, left, lok) {
		c, err := e.replayCut(t.Key, left, start, true)
		if err != nil || c == nil {
			return t, err
		}
		out.Left = c
	}
	if stale(t.Right, right, rok) {
		c, err := e.replayCut(t.Key, right, end, false)
		if err != nil || c == nil {
			return t, err
		}
		out.Right = c
	}
	return &out, nil
}

// stale reports whether cut names the segment of edge by another alignment number than
// the pointer of edge.
func stale(cut *log.Cut, edge state.Edge, ok bool) bool {
	return cut != nil && ok && cut.Segment == edge.Segment.ID &&
		telem.Alignment(cut.Alignment).DomainIndex() !=
			edge.Pointer.Alignment.DomainIndex()
}

// edgeAt returns the pointer of segs that holds stamp at past its first sample, as a
// Tombstone cuts it, and false when none does.
func edgeAt(segs []state.Segment, at telem.TimeStamp) (state.Edge, bool) {
	for _, seg := range segs {
		for _, p := range seg.Pointers {
			if p.First < at && p.Last >= at {
				return state.Edge{Segment: seg, Pointer: p}, true
			}
		}
	}
	return state.Edge{}, false
}

// replayCut returns the cut of the pointer of edge, a pointer of key, at at, from the
// stamps of its clock in the state in replay mode, or nil when the stamps do not hold
// at inside the pointer.
func (e *engine) replayCut(
	key ChannelKey,
	edge state.Edge,
	at telem.TimeStamp,
	left bool,
) (*log.Cut, error) {
	clock := edge.Segment.Clock
	pre, err := e.records(clock, edge.Pointer.First, edge.Pointer.Last)
	if err != nil {
		return nil, err
	}
	segs := pre.Segments
	for _, seg := range e.state.ReplaySegments(clock) {
		held := func(s state.Segment) bool { return s.ID == seg.ID }
		if !slices.ContainsFunc(segs, held) {
			segs = append(segs, seg)
		}
	}
	r := &stampReader{e: e, clock: clock, samples: int(edge.Pointer.Samples), span: -1}
	if err = r.spansOf(segs, edge.Pointer); err != nil {
		return nil, errors.Join(err, r.close())
	}
	dt := telem.DataTypeCode(edge.Segment.DataType).DataType()
	c, err := e.cutWith(r, key, dt, &edge, at, left)
	if errors.Is(err, errNoCut) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}
