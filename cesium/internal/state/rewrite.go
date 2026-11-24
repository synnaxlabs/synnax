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
	"cmp"
	"slices"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
)

// Candidate is a segment that passes the candidate test of a rewrite.
type Candidate struct {
	// Key is the key whose samples the segment holds.
	Key uint32
	// Segment is the identity of the segment.
	Segment uint64
}

type candidatesRequest struct {
	candidates []Candidate
	done       chan error
}

func (r *candidatesRequest) fail(err error) { r.done <- err }

func (r *candidatesRequest) run(s *State) {
	r.candidates = make([]Candidate, 0, len(s.core.candidates))
	for id := range s.core.candidates {
		r.candidates = append(
			r.candidates,
			Candidate{Key: s.core.segments[id].key, Segment: id},
		)
	}
	slices.SortFunc(r.candidates, func(a, b Candidate) int {
		return cmp.Compare(a.Segment, b.Segment)
	})
	r.done <- nil
}

// Candidates returns the segments that pass the candidate test of a rewrite, in rising
// order of identity.
func (s *State) Candidates() ([]Candidate, error) {
	r := &candidatesRequest{done: make(chan error, 1)}
	if err := s.enqueue(r); err != nil {
		return nil, err
	}
	if err := wait(s, r.done); err != nil {
		return nil, err
	}
	return r.candidates, nil
}

type idRequest struct {
	id   uint64
	done chan error
}

func (r *idRequest) fail(err error) { r.done <- err }

func (r *idRequest) run(s *State) {
	if s.nextID >= s.core.limits.Segments {
		s.park(r)
		return
	}
	r.id = s.nextID
	s.nextID++
	s.reserveAhead()
	r.done <- nil
}

// NewID gives out the identity of the new segment of a rewrite.
func (s *State) NewID() (uint64, error) {
	r := &idRequest{done: make(chan error, 1)}
	if err := s.enqueue(r); err != nil {
		return 0, err
	}
	if err := wait(s, r.done); err != nil {
		return 0, err
	}
	return r.id, nil
}

// flight is the Replace or the Check that the owner appended and has yet to apply or
// see refused. The compactor and the check pass each run one at a time.
type flight struct {
	// seg is the old segment of a Replace, or the segment of a Check.
	seg        uint64
	check      bool
	key, clock uint32
	done       chan error
	// deferred holds the requests that wait for the entry: a Replace or a Check, and
	// each request whose check reads seg. They run again once the entry applies or
	// fails.
	deferred []request
}

type replaceRequest struct {
	r    *log.Replace
	done chan error
}

func (r *replaceRequest) fail(err error) { r.done <- err }

func (r *replaceRequest) run(s *State) {
	if s.flight != nil && s.flight.check {
		s.flight.deferred = append(s.flight.deferred, r)
		return
	}
	old := s.core.segments[r.r.Old]
	if err := s.checkReplace(old, r.r); err != nil {
		r.done <- err
		return
	}
	s.flight = &flight{seg: r.r.Old, key: old.key, clock: old.vis.Clock, done: r.done}
	var e log.Entry
	e.Add(&log.Create{
		Segment:  r.r.New,
		Key:      old.key,
		Clock:    old.vis.Clock,
		Version:  s.cfg.Version,
		DataType: old.vis.DataType,
	})
	e.Add(r.r)
	s.cfg.Append(&e)
}

// checkReplace returns ErrStale when the durable pointers of old differ from those of
// the copy of r, or a Drop or a delete claim of the key or the clock of old may read
// old.
func (s *State) checkReplace(old *segState, r *log.Replace) error {
	switch {
	case s.flight != nil:
		return errors.Newf("replace of segment %d while the replace of segment %d is "+
			"in flight", r.Old, s.flight.seg)
	case old == nil || !copied(old.dur.pointers, r.Pointers):
		return errors.Wrapf(ErrStale, "segment %d changed after its copy", r.Old)
	case !s.core.candidates.Contains(r.Old):
		return errors.Newf("replace of segment %d, which is no candidate of a rewrite",
			r.Old)
	}
	return s.core.readers(r.Old, old.key, old.vis.Clock, old.dur.pointers)
}

// readers returns an error that wraps ErrStale when a Drop or a delete claim of key or
// clock may read segment id, whose durable pointers are ptrs.
func (c *core) readers(id uint64, key, clock uint32, ptrs []Pointer) error {
	for _, k := range [2]uint32{key, clock} {
		if _, ok := c.dropUsers[k]; ok {
			return errors.Wrapf(ErrStale, "drop of key %d reads segment %d", k, id)
		}
		ch := c.channels[k]
		if ch == nil {
			continue
		}
		for _, cl := range ch.claims {
			if cl.w == nil && overlaps(ptrs, cl.start, cl.end) {
				return errors.Wrapf(ErrStale, "delete of key %d from %d reads segment "+
					"%d", k, cl.start, id)
			}
		}
	}
	return nil
}

// Replace checks r, the Replace of a sealed segment by the copy of its live ranges in a
// new segment from NewID, then appends a Create of the new segment and r in one entry.
// It returns once the state applies the entry, or with the error of the log that
// refused it. It returns an error that wraps ErrStale when the durable pointers of the
// old segment differ from those of r, or a Drop or a delete of its key or its clock is
// in progress. Until the entry applies or fails, the owner defers each delete and Drop
// whose check reads the old segment, and each Check. It waits for a Check in flight.
// The caller runs one Replace at a time.
func (s *State) Replace(r *log.Replace) error {
	req := &replaceRequest{r: r, done: make(chan error, 1)}
	if err := s.enqueue(req); err != nil {
		return err
	}
	return wait(s, req.done)
}

// deferred holds r until the Replace or the Check in flight applies or fails, when the
// check of r reads its segment: a segment of key, or one that key times. It reports
// whether it held r.
func (s *State) deferred(r request, key uint32) bool {
	if s.flight == nil || (s.flight.key != key && s.flight.clock != key) {
		return false
	}
	s.flight.deferred = append(s.flight.deferred, r)
	return true
}

// endFlight ends the Replace or the Check in flight with err, and runs the requests
// that it deferred.
func (s *State) endFlight(err error) {
	f := s.flight
	s.flight, s.core.checking = nil, 0
	f.done <- err
	for _, r := range f.deferred {
		r.run(s)
	}
}

// landed ends the Replace or the Check in flight once records, a durable entry, hold
// it.
func (s *State) landed(records []log.Record) {
	for _, rec := range records {
		switch r := rec.(type) {
		case *log.Replace:
			if !s.flight.check && r.Old == s.flight.seg {
				s.endFlight(nil)
				return
			}
		case *log.Check:
			if s.flight.check && r.Segment == s.flight.seg {
				s.endFlight(nil)
				return
			}
		}
	}
}

// failWaiting fails with err the parked requests, the Replace or the Check in flight,
// and the requests that it deferred.
func (s *State) failWaiting(err error) {
	s.failParked(err)
	if s.flight == nil {
		return
	}
	f := s.flight
	s.flight, s.core.checking = nil, 0
	f.done <- err
	for _, r := range f.deferred {
		r.fail(err)
	}
}
