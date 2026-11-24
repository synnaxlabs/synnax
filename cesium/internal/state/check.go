// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state

import (
	"slices"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
)

type checkRequest struct {
	r    *log.Check
	pre  []Preload
	done chan error
}

func (r *checkRequest) fail(err error) { r.done <- err }

func (r *checkRequest) run(s *State) {
	if s.flight != nil && !s.flight.check {
		s.flight.deferred = append(s.flight.deferred, r)
		return
	}
	seg, err := s.checkCheck(r.r, r.pre)
	if err != nil {
		r.done <- err
		return
	}
	s.core.putBack(r.r.Key, []Segment{seg})
	s.core.checking = seg.ID
	s.flight = &flight{
		seg:   seg.ID,
		check: true,
		key:   r.r.Key,
		clock: seg.Clock,
		done:  r.done,
	}
	var e log.Entry
	e.Add(r.r)
	s.cfg.Append(&e)
}

// checkCheck returns the live record of the segment of r. It returns an error that
// wraps ErrStale when the segment left the state, the preload is stale, or a Drop or a
// delete claim of its key or its clock may read it.
func (s *State) checkCheck(r *log.Check, pre []Preload) (Segment, error) {
	if s.flight != nil {
		return Segment{}, errors.Newf("check of segment %d while the check of segment "+
			"%d is in flight", r.Segment, s.flight.seg)
	}
	ch := s.core.channels[r.Key]
	if ch == nil {
		return Segment{}, errors.Wrapf(ErrStale, "key %d of segment %d holds no "+
			"segment", r.Key, r.Segment)
	}
	records, err := s.core.preloadOf(ch, pre)
	if err != nil {
		return Segment{}, err
	}
	var seg Segment
	cur := s.core.segments[r.Segment]
	if cur != nil && cur.key != r.Key {
		return Segment{}, refuse("check of segment %d of key %d, which key %d holds",
			r.Segment, r.Key, cur.key)
	}
	if cur != nil && cur.vis.Part == 0 {
		seg = cur.vis
	} else if i := slices.IndexFunc(records, func(s Segment) bool {
		return s.ID == r.Segment
	}); i >= 0 {
		seg = records[i]
	} else {
		return Segment{}, errors.Wrapf(ErrStale, "segment %d of key %d left the state",
			r.Segment, r.Key)
	}
	switch {
	case seg.Version != 0 || !seg.Sealed:
		return Segment{}, refuse(
			"check of segment %d, which is not a sealed version 0 "+
				"segment",
			r.Segment,
		)
	case seg.Checked:
		return Segment{}, refuse("check of segment %d, which has its checks", r.Segment)
	}
	return seg, s.core.readers(seg.ID, r.Key, seg.Clock, seg.Pointers)
}

// Check checks r, the Check of a sealed version 0 segment whose check file is durable,
// then appends it in an entry. It returns once the state applies the entry, or with the
// error of the log that refused it. pre is the preload of the key of r over the time
// range of the segment. Check returns an error that wraps ErrStale when the segment
// left the state, pre is stale, or a Drop or a delete of its key or its clock is in
// progress. Until the entry applies or fails, a switch keeps the record of the segment,
// and the owner defers each delete and Drop whose check reads the segment, and each
// Replace. It waits for a Replace in flight. The caller runs one Check at a time.
func (s *State) Check(r *log.Check, pre ...Preload) error {
	req := &checkRequest{r: r, pre: pre, done: make(chan error, 1)}
	if err := s.enqueue(req); err != nil {
		return err
	}
	return wait(s, req.done)
}
