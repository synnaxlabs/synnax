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

// replacing is the Replace or the Kept that the owner appended and has yet to apply or
// see refused.
type replacing struct {
	old        uint64
	key, clock uint32
	done       chan error
	// deferred holds the requests whose check reads the old segment, which run again
	// once the Replace applies or fails.
	deferred []request
}

type replaceRequest struct {
	r       *log.Replace
	version uint16
	done    chan error
}

func (r *replaceRequest) fail(err error) { r.done <- err }

func (r *replaceRequest) run(s *State) {
	old := s.core.segments[r.r.Old]
	if err := s.checkReplace(old, r.r, r.version); err != nil {
		r.done <- err
		return
	}
	s.rep = &replacing{old: r.r.Old, key: old.key, clock: old.vis.Clock, done: r.done}
	var e log.Entry
	e.Add(&log.Create{
		Segment:  r.r.New,
		Key:      old.key,
		Clock:    old.vis.Clock,
		Version:  r.version,
		DataType: old.vis.DataType,
	})
	e.Add(r.r)
	s.cfg.Append(&e)
}

type keepRequest struct {
	segment uint64
	done    chan error
}

func (r *keepRequest) fail(err error) { r.done <- err }

func (r *keepRequest) run(s *State) {
	seg := s.core.segments[r.segment]
	if err := s.checkKeep(seg, r.segment); err != nil {
		r.done <- err
		return
	}
	s.rep = &replacing{old: r.segment, key: seg.key, clock: seg.vis.Clock, done: r.done}
	var e log.Entry
	e.Add(&log.Kept{Segment: r.segment})
	s.cfg.Append(&e)
}

// checkKeep returns ErrStale when seg, the segment of id, is no longer a candidate of
// encoding, or a Drop or a delete claim of its key may read it.
func (s *State) checkKeep(seg *segState, id uint64) error {
	switch {
	case s.rep != nil:
		return errors.Newf("kept of segment %d while the rewrite of segment %d is in "+
			"flight", id, s.rep.old)
	case seg == nil || !s.core.encodable(seg):
		return errors.Wrapf(ErrStale, "segment %d is no candidate of encoding", id)
	}
	return s.checkReaders(seg)
}

// checkReplace returns ErrStale when the durable pointers of old differ from those of
// the copy of r, or a Drop or a delete claim of the key or the clock of old may read
// old. A copy of version 2 needs old to be a candidate of encoding.
func (s *State) checkReplace(old *segState, r *log.Replace, version uint16) error {
	switch {
	case s.rep != nil:
		return errors.Newf("replace of segment %d while the replace of segment %d is "+
			"in flight", r.Old, s.rep.old)
	case old == nil || !copied(old.dur.pointers, r.Pointers):
		return errors.Wrapf(ErrStale, "segment %d changed after its copy", r.Old)
	case !s.core.candidates.Contains(r.Old):
		return errors.Newf("replace of segment %d, which is no candidate of a rewrite",
			r.Old)
	case version == progressionVersion && !s.core.encodable(old):
		return errors.Newf("replace of segment %d by progressions, which is no "+
			"candidate of encoding", r.Old)
	case version != old.vis.Version && version != progressionVersion:
		return errors.Newf("replace of segment %d of version %d by version %d",
			r.Old, old.vis.Version, version)
	}
	return s.checkReaders(old)
}

// checkReaders returns ErrStale when a Drop or a delete claim of the key or the clock
// of old may read old.
func (s *State) checkReaders(old *segState) error {
	for _, key := range [2]uint32{old.key, old.vis.Clock} {
		if _, ok := s.core.dropUsers[key]; ok {
			return errors.Wrapf(
				ErrStale,
				"drop of key %d reads segment %d",
				key,
				old.vis.ID,
			)
		}
		ch := s.core.channels[key]
		if ch == nil {
			continue
		}
		for _, cl := range ch.claims {
			if cl.w == nil && overlaps(old.dur.pointers, cl.start, cl.end) {
				return errors.Wrapf(ErrStale, "delete of key %d from %d reads segment "+
					"%d", key, cl.start, old.vis.ID)
			}
		}
	}
	return nil
}

// Replace checks r, the Replace of a sealed segment by the copy of its live ranges in a
// new segment of version from NewID, then appends a Create of the new segment and r in
// one entry. A copy of version 2 holds the stamps of a candidate of encoding as
// progressions. It returns once the state applies the entry, or with the error of the
// log that refused it. It returns an error that wraps ErrStale when the durable
// pointers of the old segment differ from those of r, or a Drop or a delete of its key
// or its clock is in progress. Until the entry applies or fails, the owner defers each
// delete and Drop whose check reads the old segment. The caller runs one Replace or
// Keep at a time.
func (s *State) Replace(r *log.Replace, version uint16) error {
	req := &replaceRequest{r: r, version: version, done: make(chan error, 1)}
	if err := s.enqueue(req); err != nil {
		return err
	}
	return wait(s, req.done)
}

// Keep checks that segment is a candidate of encoding, then appends a Kept of it, for
// an encoding that failed the half rule. It returns once the state applies the entry,
// or with the error of the log that refused it. It returns an error that wraps ErrStale
// when segment is no candidate of encoding, or a Drop or a delete of its key or its
// clock is in progress. Until the entry applies or fails, the owner defers each delete
// and Drop whose check reads the segment. The caller runs one Replace or Keep at a
// time.
func (s *State) Keep(segment uint64) error {
	req := &keepRequest{segment: segment, done: make(chan error, 1)}
	if err := s.enqueue(req); err != nil {
		return err
	}
	return wait(s, req.done)
}

// deferred holds r until the Replace in flight applies or fails, when the check of r
// reads the old segment: a segment of key, or one that key times. It reports whether it
// held r.
func (s *State) deferred(r request, key uint32) bool {
	if s.rep == nil || (s.rep.key != key && s.rep.clock != key) {
		return false
	}
	s.rep.deferred = append(s.rep.deferred, r)
	return true
}

// endReplace ends the Replace in flight with err, and runs the requests that it
// deferred.
func (s *State) endReplace(err error) {
	rep := s.rep
	s.rep = nil
	rep.done <- err
	for _, r := range rep.deferred {
		r.run(s)
	}
}

// replaced ends the Replace in flight once records, a durable entry, hold it.
func (s *State) replaced(records []log.Record) {
	for _, rec := range records {
		if rewrites(rec, s.rep.old) {
			s.endReplace(nil)
			return
		}
	}
}

// failWaiting fails with err the parked requests, the Replace in flight, and the
// requests that the Replace deferred.
func (s *State) failWaiting(err error) {
	s.failParked(err)
	if s.rep == nil {
		return
	}
	rep := s.rep
	s.rep = nil
	rep.done <- err
	for _, r := range rep.deferred {
		r.fail(err)
	}
}

// rewrites reports whether rec is the Replace or the Kept of segment.
func rewrites(rec log.Record, segment uint64) bool {
	switch rec := rec.(type) {
	case *log.Replace:
		return rec.Old == segment
	case *log.Kept:
		return rec.Segment == segment
	}
	return false
}
