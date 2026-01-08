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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

type index struct {
	indexPersist *indexPersist
	alamos.Instrumentation
	mu struct {
		pointers []pointer
		sync.RWMutex
	}
	persistHead int
	deleteLock  sync.RWMutex
}

// insert adds a new pointer to the index.
func (idx *index) insert(ctx context.Context, p pointer, persist bool) error {
	_, span := idx.T.Bench(ctx, "domain/index.insert")
	idx.mu.Lock()
	defer span.End()

	insertAt := 0

	if p.fileKey == 0 {
		idx.mu.Unlock()
		idx.L.DPanic("fileKey must be set")
		return span.Error(errors.New("inserted pointer cannot have key 0"))
	}
	if len(idx.mu.pointers) != 0 {
		// Hot path optimization for appending to the end of the index.
		if idx.afterLast(p.Start) {
			insertAt = len(idx.mu.pointers)
		} else if !idx.beforeFirst(p.End) {
			i, overlap := idx.unprotectedSearch(p.TimeRange)
			if overlap {
				idx.mu.Unlock()
				return span.Error(NewErrRangeWriteConflict(p.TimeRange, idx.mu.pointers[i].TimeRange))
			}
			insertAt = i + 1
		}
	}

	if insertAt == 0 {
		idx.mu.pointers = append([]pointer{p}, idx.mu.pointers...)
	} else if insertAt == len(idx.mu.pointers) {
		idx.mu.pointers = append(idx.mu.pointers, p)
	} else {
		idx.mu.pointers = slices.Insert(idx.mu.pointers, insertAt, p)
	}

	idx.persistHead = min(idx.persistHead, insertAt)

	idx.mu.Unlock()
	if !persist {
		return nil
	}

	persistPointers := idx.indexPersist.prepare(idx.persistHead)
	return persistPointers()
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

func (idx *index) update(ctx context.Context, p pointer, persist bool) error {
	_, span := idx.T.Bench(ctx, "domain/index.update")
	idx.mu.Lock()

	defer span.End()

	if len(idx.mu.pointers) == 0 {
		// This should be inconceivable since update would not be called with no
		// pointers.
		idx.L.DPanic("cannot update a database with no domains")
		idx.mu.Unlock()
		return span.Error(NewErrRangeNotFound(p.TimeRange))
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
		return span.Error(NewErrRangeNotFound(p.TimeRange))
	}
	overlapsWithNext := updateAt != len(ptrs)-1 && ptrs[updateAt+1].OverlapsWith(p.TimeRange)
	overlapsWithPrev := updateAt != 0 && ptrs[updateAt-1].OverlapsWith(p.TimeRange)
	if overlapsWithPrev {
		idx.mu.Unlock()
		return span.Error(NewErrRangeWriteConflict(p.TimeRange, ptrs[updateAt-1].TimeRange))
	} else if overlapsWithNext {
		idx.mu.Unlock()
		return span.Error(NewErrRangeWriteConflict(p.TimeRange, ptrs[updateAt+1].TimeRange))
	} else {
		idx.mu.pointers[updateAt] = p
	}

	idx.persistHead = min(idx.persistHead, updateAt)

	if persist {
		persistPointers := idx.indexPersist.prepare(idx.persistHead)
		idx.mu.Unlock()
		return persistPointers()
	}

	idx.mu.Unlock()
	return nil
}

func (idx *index) afterLast(ts telem.TimeStamp) bool {
	return ts.After(idx.mu.pointers[len(idx.mu.pointers)-1].End)
}

func (idx *index) beforeFirst(ts telem.TimeStamp) bool {
	return ts.Before(idx.mu.pointers[0].Start)
}

func (idx *index) searchLE(ctx context.Context, ts telem.TimeStamp) (i int) {
	_, span := idx.T.Bench(ctx, "domain/index.searchLE")
	idx.read(func() {
		i, _ = idx.unprotectedSearch(ts.SpanRange(0))
	})
	span.End()
	return
}

func (idx *index) searchGE(ctx context.Context, ts telem.TimeStamp) (i int) {
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
	})
	span.End()
	return
}

func (idx *index) getGE(ctx context.Context, ts telem.TimeStamp) (ptr pointer, ok bool) {
	_, span := idx.T.Bench(ctx, "domain/index.getGE")
	idx.mu.RLock()
	defer func() {
		span.End()
		idx.mu.RUnlock()
	}()
	var exact bool
	i, exact := idx.unprotectedSearch(ts.SpanRange(0))
	if !exact {
		if i == len(idx.mu.pointers) {
			return pointer{}, false
		} else {
			i += 1
		}
	}

	if i < 0 || i >= len(idx.mu.pointers) {
		return pointer{}, false
	}

	return idx.mu.pointers[i], true
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

func (idx *index) get(i int) (pointer, bool) {
	idx.mu.RLock()
	if i < 0 || i >= len(idx.mu.pointers) {
		idx.mu.RUnlock()
		return pointer{}, false
	}
	v := idx.mu.pointers[i]
	idx.mu.RUnlock()
	return v, true
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
