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
		// lastID is the last pointer id handed out.
		lastID uint64
		// rewrites counts the GC rewrites of each file.
		rewrites map[uint16]uint64
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
// direction of step, as readers see it, and the GC rewrite count of its file.
func (idx *index) unprotectedVisible(i, step int) (int, pointer, uint64, bool) {
	for ; i >= 0 && i < len(idx.mu.pointers); i += step {
		ptr := idx.mu.pointers[i]
		rewrites := idx.mu.rewrites[ptr.fileKey]
		if len(idx.mu.staged) == 0 {
			return i, ptr, rewrites, true
		}
		j := idx.unprotectedStaged(ptr.Start)
		if j < 0 || idx.mu.staged[j].visible() {
			return i, ptr, rewrites, true
		}
		if s := idx.mu.staged[j]; !s.inserted {
			ptr.End, ptr.size = s.end, s.size
			return i, ptr, rewrites, true
		}
	}
	return i, pointer{}, 0, false
}

// locate returns the domain in the index that is ptr, and the GC rewrite count of its
// file. found is false if a delete removed the domain or cut any bytes of ptr from it.
func (idx *index) locate(ptr pointer) (current pointer, rewrites uint64, found bool) {
	idx.read(func() {
		rewrites = idx.mu.rewrites[ptr.fileKey]
		i, exact := idx.unprotectedSearch(ptr.Start.SpanRange(0))
		if !exact {
			return
		}
		if p := idx.mu.pointers[i]; p.id == ptr.id && p.size >= ptr.size {
			current, found = p, true
		}
	})
	return current, rewrites, found
}

func (idx *index) unprotectedNextID() uint64 {
	idx.mu.lastID++
	return idx.mu.lastID
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

	p.id = idx.unprotectedNextID()
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
	p.id = oldP.id
	idx.mu.pointers[updateAt] = p
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
) (i int, ptr pointer, rewrites uint64, ok bool) {
	_, span := idx.T.Bench(ctx, "domain/index.searchLE")
	idx.read(func() {
		i, _ = idx.unprotectedSearch(ts.SpanRange(0))
		i, ptr, rewrites, ok = idx.unprotectedVisible(i, -1)
	})
	span.End()
	return i, ptr, rewrites, ok
}

// searchGE is searchLE for the closest domain after ts.
func (idx *index) searchGE(
	ctx context.Context,
	ts telem.TimeStamp,
) (i int, ptr pointer, rewrites uint64, ok bool) {
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
		i, ptr, rewrites, ok = idx.unprotectedVisible(i, 1)
		// Readers do not see the part of the domain that holds ts yet.
		if ok && exact && i == at && !ptr.End.After(ts) {
			i, ptr, rewrites, ok = idx.unprotectedVisible(i+1, 1)
		}
	})
	span.End()
	return i, ptr, rewrites, ok
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

// next returns the first domain that starts at or after the start of cur, a domain the
// caller has seen, and its position. It skips a domain that overlaps cur and has its
// id: cur after it grows, and what a delete left of it. It finds both under one lock,
// so a concurrent insert cannot shift one off the other. After next returns a domain
// written inside cur, the next step can return what remains of cur, which the caller
// has read.
func (idx *index) next(
	cur pointer,
) (i int, ptr pointer, rewrites uint64, ok bool) {
	idx.read(func() {
		i, _ = slices.BinarySearchFunc(
			idx.mu.pointers,
			cur.Start,
			func(p pointer, start telem.TimeStamp) int {
				if p.Start >= start {
					return 1
				}
				return -1
			},
		)
		i, ptr, rewrites, ok = idx.unprotectedVisible(i, 1)
		for ok && ptr.Start < cur.End && ptr.id == cur.id {
			i, ptr, rewrites, ok = idx.unprotectedVisible(i+1, 1)
		}
	})
	return i, ptr, rewrites, ok
}

// prev returns the last domain that ends at or before the end of cur, a domain the
// caller has seen, and its position. It skips a domain that overlaps cur and has its
// id. After prev returns a domain written inside cur, the next step can return what
// remains of cur, which the caller has read.
func (idx *index) prev(
	cur pointer,
) (i int, ptr pointer, rewrites uint64, ok bool) {
	idx.read(func() {
		i, _ = slices.BinarySearchFunc(
			idx.mu.pointers,
			cur.End,
			func(p pointer, end telem.TimeStamp) int {
				if p.End > end {
					return 1
				}
				return -1
			},
		)
		i, ptr, rewrites, ok = idx.unprotectedVisible(i-1, -1)
		for ok && ptr.End > cur.Start && ptr.id == cur.id {
			i, ptr, rewrites, ok = idx.unprotectedVisible(i-1, -1)
		}
	})
	return i, ptr, rewrites, ok
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
