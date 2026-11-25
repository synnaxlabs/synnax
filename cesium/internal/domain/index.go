// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/invariants"
	"github.com/synnaxlabs/x/telem"
)

type index struct {
	indexPersist    *indexPersist
	onHistoryChange func(context.Context, telem.TimeRange)
	alamos.Instrumentation
	mu struct {
		pointers []pointer
		// staged holds the domains that a commit in a Group changed and that readers do
		// not see yet. It is empty outside of a commit, and holds one domain for each
		// writer that is in one.
		staged []staged
		sync.RWMutex
	}
	// persistHead is the position of the first pointer that the index file may not
	// hold. Guarded by mu.
	persistHead int
	deleteLock  sync.RWMutex
	totalSize   *atomic.Int64
}

// unprotectedStage records the domain at p.Start as readers see it until g publishes,
// with the history change of the commit. old is the pointer that the commit replaced,
// and is not set for an insert. A nil g makes the domain visible at once. It returns
// the history change to report now.
func (idx *index) unprotectedStage(
	p pointer,
	old *pointer,
	g *Group,
	change historyChange,
) historyChange {
	if g == nil {
		return idx.unprotectedReveal(p.Start).then(change)
	}
	s := staged{
		start:    p.Start,
		group:    g,
		publish:  g.published.Load() + 1,
		inserted: old == nil,
		change:   change,
	}
	if old != nil {
		s.end, s.size = old.End, old.size
	}
	g.commits = append(g.commits, stagedDomain{idx: idx, start: p.Start})
	i := idx.unprotectedStaged(p.Start)
	if i < 0 {
		idx.mu.staged = append(idx.mu.staged, s)
		return historyChange{}
	}
	prev := idx.mu.staged[i]
	s.change = prev.change.then(change)
	// Readers still see the domain as it was before an earlier unpublished commit.
	if old != nil && !prev.visible() {
		s.inserted, s.end, s.size = prev.inserted, prev.end, prev.size
	}
	idx.mu.staged[i] = s
	return historyChange{}
}

// unprotectedStaged returns the position in staged of the domain that starts at start,
// or -1.
func (idx *index) unprotectedStaged(start telem.TimeStamp) int {
	for i := range idx.mu.staged {
		if idx.mu.staged[i].start == start {
			return i
		}
	}
	return -1
}

// unprotectedReveal makes the domain that starts at start visible to readers as the
// index holds it. It returns the history change that waited for that, which the caller
// must report.
func (idx *index) unprotectedReveal(start telem.TimeStamp) historyChange {
	i := idx.unprotectedStaged(start)
	if i < 0 {
		return historyChange{}
	}
	change := idx.mu.staged[i].change
	last := len(idx.mu.staged) - 1
	idx.mu.staged[i] = idx.mu.staged[last]
	idx.mu.staged[last] = staged{}
	idx.mu.staged = idx.mu.staged[:last]
	return change
}

// reveal is unprotectedReveal for a caller that does not hold mu. It reports the
// history change.
func (idx *index) reveal(start telem.TimeStamp) {
	idx.mu.Lock()
	change := idx.unprotectedReveal(start)
	idx.mu.Unlock()
	idx.report(change)
}

// revealPublished is reveal for a domain that g published. It leaves a domain that
// another Group staged since.
func (idx *index) revealPublished(start telem.TimeStamp, g *Group) {
	var change historyChange
	idx.mu.Lock()
	if i := idx.unprotectedStaged(start); i >= 0 && idx.mu.staged[i].group == g &&
		idx.mu.staged[i].visible() {
		change = idx.unprotectedReveal(start)
	}
	idx.mu.Unlock()
	idx.report(change)
}

func (idx *index) report(change historyChange) {
	if change.ctx != nil {
		idx.notifyHistoryChange(change.ctx, change.tr)
	}
}

// unprotectedVisible returns the first domain that readers see, from position i in the
// direction of step, as readers see it.
func (idx *index) unprotectedVisible(i, step int) (int, pointer, bool) {
	for ; i >= 0 && i < len(idx.mu.pointers); i += step {
		ptr := idx.mu.pointers[i]
		if len(idx.mu.staged) == 0 {
			return i, ptr, true
		}
		j := idx.unprotectedStaged(ptr.Start)
		if j < 0 || idx.mu.staged[j].visible() {
			return i, ptr, true
		}
		if s := idx.mu.staged[j]; !s.inserted {
			ptr.End, ptr.size = s.end, s.size
			return i, ptr, true
		}
	}
	return i, pointer{}, false
}

// insert adds a new pointer to the index, in g when g is not nil. When persist is true,
// the caller must persist the returned write.
func (idx *index) insert(
	ctx context.Context,
	p pointer,
	persist bool,
	g *Group,
) (preparedWrite, error) {
	_, span := idx.T.Bench(ctx, "domain/index.insert")
	idx.mu.Lock()
	defer span.End()

	insertAt := 0

	if p.fileKey == 0 {
		idx.mu.Unlock()
		idx.L.DPanic("fileKey must be set")
		return preparedWrite{}, span.Error(
			errors.New("inserted pointer cannot have key 0"),
		)
	}
	if len(idx.mu.pointers) != 0 {
		// Hot path optimization for appending to the end of the index.
		if idx.afterLast(p.Start) {
			insertAt = len(idx.mu.pointers)
		} else if !idx.beforeFirst(p.End) {
			i, overlap := idx.unprotectedSearch(p.TimeRange)
			if overlap {
				err := NewRangeWriteConflictError(
					p.TimeRange,
					idx.mu.pointers[i].TimeRange,
				)
				idx.mu.Unlock()
				return preparedWrite{}, span.Error(err)
			}
			insertAt = i + 1
		}
	}

	appended := insertAt == len(idx.mu.pointers)
	switch {
	case insertAt == 0:
		idx.mu.pointers = append([]pointer{p}, idx.mu.pointers...)
	case insertAt == len(idx.mu.pointers):
		idx.mu.pointers = append(idx.mu.pointers, p)
	default:
		idx.mu.pointers = slices.Insert(idx.mu.pointers, insertAt, p)
	}

	var change historyChange
	// The live stream carries domains added at the end.
	if !appended {
		change = historyChange{ctx: ctx, tr: p.TimeRange}
	}
	idx.checkApart(insertAt, insertAt)
	change = idx.unprotectedStage(p, nil, g, change)
	idx.totalSize.Add(int64(p.size))
	idx.persistHead = min(idx.persistHead, insertAt)
	var write preparedWrite
	if persist {
		write = idx.indexPersist.prepare(idx.persistHead)
	}
	idx.mu.Unlock()
	idx.report(change)
	return write, nil
}

// checkApart panics when a pointer from i to j, or a neighbor of one, ends before it
// starts or after the next pointer starts. It checks only in a build with the
// invariants or race tag. The caller must hold idx.mu.
func (idx *index) checkApart(i, j int) {
	if !invariants.Enabled {
		return
	}
	ptrs := idx.mu.pointers
	for k := max(i-1, 0); k <= min(j+1, len(ptrs)-1); k++ {
		if ptrs[k].End < ptrs[k].Start {
			panic(
				errors.Newf("domain pointer %d ends before it starts: %v", k, ptrs[k]),
			)
		}
		if k > 0 && ptrs[k-1].End > ptrs[k].Start {
			panic(errors.Newf(
				"domain pointers %d and %d are not sorted and apart: %v and %v",
				k-1,
				k,
				ptrs[k-1].TimeRange,
				ptrs[k].TimeRange,
			))
		}
	}
}

// persist writes the pointers that the index file may not hold. The caller must not
// hold mu.
func (idx *index) persist() error {
	idx.mu.Lock()
	write := idx.indexPersist.prepare(idx.persistHead)
	idx.mu.Unlock()
	return idx.run(write)
}

// run runs write. When an earlier write fails, it prepares and runs another write
// that restores the pointers the failed write left out. The caller must not hold mu.
func (idx *index) run(write preparedWrite) error {
	for {
		if err := write.run(); !errors.Is(err, errUncovered) {
			return err
		}
		idx.mu.Lock()
		write = idx.indexPersist.prepare(idx.persistHead)
		idx.mu.Unlock()
	}
}

func (idx *index) overlap(tr telem.TimeRange) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	_, overlap := idx.unprotectedSearch(tr)
	return overlap
}

func (idx *index) timeRange() telem.TimeRange {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if len(idx.mu.pointers) == 0 {
		return telem.TimeRangeZero
	}
	return idx.mu.pointers[0].Start.Range(idx.mu.pointers[len(idx.mu.pointers)-1].End)
}

// update replaces the pointer that starts at p.Start, in g when g is not nil. When
// persist is true, the caller must persist the returned write.
func (idx *index) update(
	ctx context.Context,
	p pointer,
	persist bool,
	g *Group,
) (preparedWrite, error) {
	_, span := idx.T.Bench(ctx, "domain/index.update")
	idx.mu.Lock()

	defer span.End()

	if len(idx.mu.pointers) == 0 {
		// This should be inconceivable since update would not be called with no
		// pointers.
		idx.L.DPanic("cannot update a database with no domains")
		idx.mu.Unlock()
		return preparedWrite{}, span.Error(newRangeNotFoundError(p.TimeRange))
	}
	lastI := len(idx.mu.pointers) - 1
	updateAt := lastI
	if p.Start != idx.mu.pointers[lastI].Start {
		updateAt, _ = idx.unprotectedSearch(p.Start.SpanRange(0))
	}

	ptrs := idx.mu.pointers
	oldP := ptrs[updateAt]
	if oldP.Start != p.Start {
		// This is inconceivable since update would only be called via commit, and
		// commit should find the same pointer the writer has been writing to, which
		// must have the same Start timestamp. Unhandled race conditions might cause the
		// database to reach this inconceivable state.
		idx.L.DPanic("cannot update a pointer with a different start timestamp")
		idx.mu.Unlock()
		return preparedWrite{}, span.Error(newRangeNotFoundError(p.TimeRange))
	}
	overlapsWithNext := updateAt != len(ptrs)-1 &&
		ptrs[updateAt+1].OverlapsWith(p.TimeRange)
	overlapsWithPrev := updateAt != 0 && ptrs[updateAt-1].OverlapsWith(p.TimeRange)
	if overlapsWithPrev {
		err := NewRangeWriteConflictError(p.TimeRange, ptrs[updateAt-1].TimeRange)
		idx.mu.Unlock()
		return preparedWrite{}, span.Error(err)
	} else if overlapsWithNext {
		err := NewRangeWriteConflictError(p.TimeRange, ptrs[updateAt+1].TimeRange)
		idx.mu.Unlock()
		return preparedWrite{}, span.Error(err)
	}
	sizeDelta := int64(p.size) - int64(oldP.size)
	idx.mu.pointers[updateAt] = p
	idx.checkApart(updateAt, updateAt)
	idx.totalSize.Add(sizeDelta)
	var change historyChange
	// The live stream carries commits to the last domain.
	if updateAt != lastI {
		change = historyChange{ctx: ctx, tr: oldP.End.Range(p.End)}
	}
	change = idx.unprotectedStage(p, &oldP, g, change)

	idx.persistHead = min(idx.persistHead, updateAt)
	var write preparedWrite
	if persist {
		write = idx.indexPersist.prepare(idx.persistHead)
	}
	idx.mu.Unlock()
	idx.report(change)
	return write, nil
}

func (idx *index) notifyHistoryChange(ctx context.Context, tr telem.TimeRange) {
	if idx.onHistoryChange != nil {
		idx.onHistoryChange(ctx, tr)
	}
}

func (idx *index) afterLast(ts telem.TimeStamp) bool {
	return ts.After(idx.mu.pointers[len(idx.mu.pointers)-1].End)
}

func (idx *index) beforeFirst(ts telem.TimeStamp) bool {
	return ts.Before(idx.mu.pointers[0].Start)
}

// searchLE returns the position of the domain that contains ts, or else of the closest
// domain before ts, and that domain. ok is false when there is none. One lock covers
// both, so a concurrent insert cannot shift the position off the domain.
func (idx *index) searchLE(
	ctx context.Context,
	ts telem.TimeStamp,
) (i int, ptr pointer, ok bool) {
	_, span := idx.T.Bench(ctx, "domain/index.searchLE")
	idx.read(func() {
		i, _ = idx.unprotectedSearch(ts.SpanRange(0))
		i, ptr, ok = idx.unprotectedVisible(i, -1)
	})
	span.End()
	return i, ptr, ok
}

// searchGE is searchLE for the closest domain after ts.
func (idx *index) searchGE(
	ctx context.Context,
	ts telem.TimeStamp,
) (i int, ptr pointer, ok bool) {
	_, span := idx.T.Bench(ctx, "domain/index.searchGE")
	idx.read(func() {
		var exact bool
		i, exact = idx.unprotectedSearch(ts.SpanRange(0))
		if !exact {
			if i == len(idx.mu.pointers) {
				i = -1
			} else {
				i += 1
			}
		}
		at := i
		i, ptr, ok = idx.unprotectedVisible(i, 1)
		// Readers do not see the part of the domain that holds ts yet.
		if ok && exact && i == at && !ptr.End.After(ts) {
			i, ptr, ok = idx.unprotectedVisible(i+1, 1)
		}
	})
	span.End()
	return i, ptr, ok
}

// unprotectedSearch returns the position in the index of a domain that overlaps with
// the given time range. If there is no domain that contains tr, then the immediate
// previous domain with a smaller start timestamp than the end is returned. False is
// returned as the flag.
// If tr is before all domains, -1 is returned.
// If tr is after all domains, len(idx.mu.pointers) - 1 is returned.
func (idx *index) unprotectedSearch(tr telem.TimeRange) (int, bool) {
	if len(idx.mu.pointers) == 0 {
		return -1, false
	}
	start, end := 0, len(idx.mu.pointers)-1
	for start <= end {
		mid := (start + end) / 2
		ptr := idx.mu.pointers[mid]
		if ptr.OverlapsWith(tr) {
			return mid, true
		}
		if tr.Start.Before(ptr.Start) {
			end = mid - 1
		} else {
			start = mid + 1
		}
	}
	return end, false
}

// next returns the first domain that starts at or after end, the end of the current
// domain, and its position. It finds both under one lock, so a concurrent insert
// cannot shift one off the other. A delete can trim the start of the current domain but
// not its end.
func (idx *index) next(end telem.TimeStamp) (i int, ptr pointer, ok bool) {
	idx.read(func() {
		var exact bool
		i, exact = idx.unprotectedSearch(end.SpanRange(0))
		if !exact || idx.mu.pointers[i].Start != end {
			i++
		}
		i, ptr, ok = idx.unprotectedVisible(i, 1)
	})
	return i, ptr, ok
}

// prev returns the last domain that starts before start, the start of the current
// domain, and its position. A delete can trim the end of the current domain but not its
// start.
func (idx *index) prev(start telem.TimeStamp) (i int, ptr pointer, ok bool) {
	idx.read(func() {
		var exact bool
		if i, exact = idx.unprotectedSearch(start.SpanRange(0)); exact {
			i--
		}
		i, ptr, ok = idx.unprotectedVisible(i, -1)
	})
	return i, ptr, ok
}

func (idx *index) read(f func()) {
	idx.mu.RLock()
	f()
	idx.mu.RUnlock()
}

func (idx *index) close() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.mu.pointers = nil
	idx.mu.staged = nil
	return idx.indexPersist.Close()
}
