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
	"slices"

	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// readCount is a publish count that reads registered, with the number of them.
type readCount struct {
	count uint64
	reads int
}

// register registers the publish count of the state for a read, which then loads with
// State.Load, and returns it. Until unregister, cover keeps each dead segment whose
// Count is past the count. A segment that free took before the call has a Count at most
// one past it, and Load waits out that step.
func (e *engine) register() uint64 {
	e.cover.Lock()
	defer e.cover.Unlock()
	count := e.state.Count()
	if n := len(e.cover.reads); n > 0 && e.cover.reads[n-1].count == count {
		e.cover.reads[n-1].reads++
	} else {
		e.cover.reads = append(e.cover.reads, readCount{count: count, reads: 1})
	}
	return count
}

// unregister drops a read of count, which register returned, and unlinks each dead
// segment that the log covers and that no read may use after it.
func (e *engine) unregister(count uint64) error {
	e.cover.Lock()
	i, found := slices.BinarySearchFunc(
		e.cover.reads,
		count,
		func(r readCount, c uint64) int { return cmp.Compare(r.count, c) },
	)
	if !found {
		e.cover.Unlock()
		panic(errors.Newf("cesium: unregister of count %d, which no read holds", count))
	}
	var free []state.Dead
	if e.cover.reads[i].reads--; e.cover.reads[i].reads == 0 {
		e.cover.reads = slices.Delete(e.cover.reads, i, i+1)
		if i == 0 {
			free = e.free()
		}
	}
	e.cover.Unlock()
	return e.unlink(free)
}

// free takes out of cover and returns each dead segment that the log covers and that no
// read may use, as no read holds a count below that of the segment. The caller holds
// e.cover.
func (e *engine) free() []state.Dead {
	var free []state.Dead
	waiting := e.cover.dead[:0]
	for _, d := range e.cover.dead {
		if d.Sequence < e.cover.covered &&
			(len(e.cover.reads) == 0 || e.cover.reads[0].count >= d.Count) {
			free = append(free, d)
		} else {
			waiting = append(waiting, d)
		}
	}
	e.cover.dead = waiting
	return free
}

// waiting returns the file names of the dead segments of key that wait in cover. Reaps
// run one at a time, so a dead segment that a reap took from the state is in cover or
// on its way to an unlink.
func (e *engine) waiting(key ChannelKey) set.Set[string] {
	e.reaping.Lock()
	defer e.reaping.Unlock()
	e.cover.Lock()
	defer e.cover.Unlock()
	names := set.New[string]()
	for _, d := range e.cover.dead {
		if d.Key == key {
			names.Add(segmentName(key, d.Segment))
		}
	}
	return names
}

// unlink closes the read handles of each segment of dead and removes its file, and
// tries every removal.
func (e *engine) unlink(dead []state.Dead) error {
	errs := make([]error, 0, len(dead))
	for _, d := range dead {
		errs = append(
			errs,
			e.closeHandle(d.Key, d.Segment),
			e.fs.Remove(segmentName(d.Key, d.Segment)),
		)
	}
	return errors.Join(errs...)
}
