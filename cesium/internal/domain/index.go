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
		sync.RWMutex
	}
	// persistHead is the position of the first pointer that the index file may not
	// hold. Guarded by mu.
	persistHead int
	deleteLock  sync.RWMutex
	totalSize   *atomic.Int64
}

// insert adds a new pointer to the index. When persist is true, the caller must persist
// the returned write.
func (idx *index) insert(
	ctx context.Context,
	p pointer,
	persist bool,
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
	if insertAt == 0 {
		idx.mu.pointers = append([]pointer{p}, idx.mu.pointers...)
	} else if insertAt == len(idx.mu.pointers) {
		idx.mu.pointers = append(idx.mu.pointers, p)
	} else {
		idx.mu.pointers = slices.Insert(idx.mu.pointers, insertAt, p)
	}

	idx.totalSize.Add(int64(p.size))
	idx.persistHead = min(idx.persistHead, insertAt)
	var write preparedWrite
	if persist {
		write = idx.indexPersist.prepare(idx.persistHead)
	}
	idx.mu.Unlock()
	// The live stream carries domains added at the end.
	if !appended {
		idx.notifyHistoryChange(ctx, p.TimeRange)
	}
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

// update replaces the pointer that starts at p.Start. When persist is true, the caller
// must persist the returned write.
func (idx *index) update(
	ctx context.Context,
	p pointer,
	persist bool,
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
	} else {
		sizeDelta := int64(p.size) - int64(oldP.size)
		idx.mu.pointers[updateAt] = p
		idx.totalSize.Add(sizeDelta)
	}

	idx.persistHead = min(idx.persistHead, updateAt)
	var write preparedWrite
	if persist {
		write = idx.indexPersist.prepare(idx.persistHead)
	}
	idx.mu.Unlock()
	// The live stream carries commits to the last domain.
	if updateAt != lastI {
		idx.notifyHistoryChange(ctx, oldP.End.Range(p.End))
	}
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
		ptr, ok = idx.unprotectedGet(i)
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
		ptr, ok = idx.unprotectedGet(i)
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
		ptr, ok = idx.unprotectedGet(i)
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
		ptr, ok = idx.unprotectedGet(i)
	})
	return i, ptr, ok
}

func (idx *index) unprotectedGet(i int) (pointer, bool) {
	if i < 0 || i >= len(idx.mu.pointers) {
		return pointer{}, false
	}
	return idx.mu.pointers[i], true
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
	return idx.indexPersist.Close()
}
