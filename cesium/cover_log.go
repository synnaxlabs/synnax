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
	"slices"

	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

// reap unlinks the file of each segment that left the state once the log covers the
// record that took it out and no read may use it.
func (e *engine) reap() error {
	e.reaping.Lock()
	dead, err := e.state.TakeDead()
	free := e.uncover(0, dead)
	e.reaping.Unlock()
	return errors.Join(err, e.unlink(free))
}

// covered unlinks, off the step of the log, the file of each dead segment whose record
// the log now covers and that no read may use, and asks Run to take the segments that
// left the state since the last reap.
func (e *engine) covered(sequence uint64) {
	e.unlinkLater(e.uncover(sequence, nil))
	if e.state.HasDead() {
		e.askReap()
	}
}

// askReap asks Run for a reap, and returns at once.
func (e *engine) askReap() {
	select {
	case e.reaps <- struct{}{}:
	default:
	}
}

// uncover raises the covered sequence to covered, adds dead to the segments that wait,
// and returns those whose record the log covers and that no read may use.
func (e *engine) uncover(covered uint64, dead []state.Dead) []state.Dead {
	e.cover.Lock()
	defer e.cover.Unlock()
	if covered > e.cover.covered {
		e.cover.covered = covered
		close(e.cover.advanced)
		e.cover.advanced = make(chan struct{})
	}
	e.cover.dead = append(e.cover.dead, dead...)
	return e.free()
}

// waitCovered returns once the log covers the entry of sequence, or with the error of
// ctx.
func (e *engine) waitCovered(ctx context.Context, sequence uint64) error {
	for {
		e.cover.Lock()
		covered, advanced := e.cover.covered, e.cover.advanced
		e.cover.Unlock()
		if sequence < covered {
			return nil
		}
		select {
		case <-advanced:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// readCount is a publish count that reads registered, with the number of them.
type readCount struct {
	count uint64
	reads int
}

// register registers the publish count of the state for a read, which then loads with
// State.Load, and returns it. Until unregister, cover keeps each dead segment whose
// Count is past the count. A segment that free took before the call has a Count at most
// one past it, and Load waits out that request.
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

// unregister drops a read of count, which register returned, and unlinks on the
// executor each dead segment that the log covers and that no read may use after it.
func (e *engine) unregister(count uint64) {
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
	e.unlinkLater(free)
}

// free takes out of cover and returns each dead segment that the log covers and that no
// read may use: no read holds it or a count below that of the segment. The caller holds
// e.cover, and passes the result to unlink.
func (e *engine) free() []state.Dead {
	var free []state.Dead
	waiting := e.cover.dead[:0]
	for _, d := range e.cover.dead {
		if d.Sequence < e.cover.covered && e.cover.held[d.Segment] == 0 &&
			(len(e.cover.reads) == 0 || e.cover.reads[0].count >= d.Count) {
			free = append(free, d)
			e.cover.unlinking[d.Key]++
		} else {
			waiting = append(waiting, d)
		}
	}
	e.cover.dead = waiting
	return free
}

// waitUnlinked returns once no unlink of a segment of key is in flight. Reaps run one
// at a time, so each dead segment of key that a reap took from the state is then in
// cover, or its file is gone.
func (e *engine) waitUnlinked(key ChannelKey) {
	for {
		e.reaping.Lock()
		e.cover.Lock()
		n, unlinked := e.cover.unlinking[key], e.cover.unlinked
		e.cover.Unlock()
		e.reaping.Unlock()
		if n == 0 {
			return
		}
		<-unlinked
	}
}

// hold keeps the files of segs, the segments of a Channel that a read froze, until
// unhold.
func (e *engine) hold(segs []state.Segment) {
	e.cover.Lock()
	defer e.cover.Unlock()
	for _, seg := range segs {
		e.cover.held[seg.ID]++
	}
}

// unhold drops a hold of segs, and unlinks on the executor each dead segment that the
// log covers and that no read may use after it.
func (e *engine) unhold(segs []state.Segment) {
	e.cover.Lock()
	for _, seg := range segs {
		if e.cover.held[seg.ID]--; e.cover.held[seg.ID] == 0 {
			delete(e.cover.held, seg.ID)
		}
	}
	free := e.free()
	e.cover.Unlock()
	e.unlinkLater(free)
}

// unlinkLater unlinks dead, which free took, on the executor, and logs the error.
func (e *engine) unlinkLater(dead []state.Dead) {
	if len(dead) == 0 {
		return
	}
	e.exec(func() {
		if err := e.unlink(dead); err != nil {
			e.ins.L.Error(
				"failed to unlink the file of a dead segment",
				zap.Error(err),
			)
		}
	})
}

// unlink closes the read handles of each segment of dead, which free took, and removes
// its file. It tries every removal, and then ends the unlink of each segment.
func (e *engine) unlink(dead []state.Dead) error {
	if len(dead) == 0 {
		return nil
	}
	errs := make([]error, 0, 2*len(dead))
	for _, d := range dead {
		errs = append(
			errs,
			e.closeHandle(d.Key, d.Segment),
			e.fs.Remove(segmentName(d.Key, d.Segment)),
		)
	}
	e.cover.Lock()
	for _, d := range dead {
		if e.cover.unlinking[d.Key]--; e.cover.unlinking[d.Key] == 0 {
			delete(e.cover.unlinking, d.Key)
		}
	}
	close(e.cover.unlinked)
	e.cover.unlinked = make(chan struct{})
	e.cover.Unlock()
	return errors.Join(errs...)
}
