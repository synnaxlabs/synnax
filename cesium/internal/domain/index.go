// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/telem"
	"sync"
)

type index struct {
	alamos.Instrumentation
	mu struct {
		sync.RWMutex
		pointers   []pointer
		tombstones map[uint16][]pointer
	}
	observe.Observer[indexUpdate]
	persistHead int
}

type indexUpdate struct {
	afterIndex int
}

var _ observe.Observable[indexUpdate] = (*index)(nil)

// insert adds a new pointer to the index.
func (idx *index) insert(ctx context.Context, p pointer, persist bool) error {
	_, span := idx.T.Bench(ctx, "insert")
	idx.mu.Lock()

	defer span.End()

	insertAt := 0

	if p.fileKey == 0 {
		panic("fileKey must be set")
	}
	if len(idx.mu.pointers) != 0 {
		// Hot path optimization for appending to the end of the index.
		if idx.afterLast(p.Start) {
			insertAt = len(idx.mu.pointers)
		} else if !idx.beforeFirst(p.End) {
			i, overlap := idx.unprotectedSearch(p.TimeRange)
			if overlap {
				idx.mu.Unlock()
				return span.Error(ErrDomainOverlap)
			}
			insertAt = i + 1
		}
	}

	if insertAt == 0 {
		idx.mu.pointers = append([]pointer{p}, idx.mu.pointers...)
	} else if insertAt == len(idx.mu.pointers) {
		idx.mu.pointers = append(idx.mu.pointers, p)
	} else {
		idx.mu.pointers = append(idx.mu.pointers[:insertAt], append([]pointer{p}, idx.mu.pointers[insertAt:]...)...)
	}

	idx.persistHead = min(idx.persistHead, insertAt)

	if persist {
		update := indexUpdate{afterIndex: idx.persistHead}
		idx.persistHead = len(idx.mu.pointers) - 1
		idx.mu.Unlock()
		idx.Observer.Notify(ctx, update)

		return nil
	}

	idx.mu.Unlock()
	return nil
}

func (idx *index) insertTombstone(ctx context.Context, p pointer) {
	_, span := idx.T.Bench(ctx, "insert tombstone")
	idx.mu.Lock()
	defer func() {
		idx.mu.Unlock()
		span.End()
	}()

	idx.mu.tombstones[p.fileKey] = append(idx.mu.tombstones[p.fileKey], p)
}

func (idx *index) overlap(tr telem.TimeRange) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	_, overlap := idx.unprotectedSearch(tr)
	return overlap
}

func (idx *index) update(ctx context.Context, p pointer, persist bool) error {
	_, span := idx.T.Bench(ctx, "update")
	idx.mu.Lock()

	defer span.End()

	if len(idx.mu.pointers) == 0 {
		idx.mu.Unlock()
		idx.L.DPanic(RangeNotFound.Error())
		return span.Error(RangeNotFound)
	}
	lastI := len(idx.mu.pointers) - 1
	updateAt := lastI
	if p.Start != idx.mu.pointers[lastI].Start {
		updateAt, _ = idx.unprotectedSearch(p.Start.SpanRange(0))
	}

	ptrs := idx.mu.pointers
	oldP := ptrs[updateAt]
	if oldP.Start != p.Start {
		idx.mu.Unlock()
		idx.L.DPanic(RangeNotFound.Error())
		return span.Error(RangeNotFound)
	}
	overlapsWithNext := updateAt != len(ptrs)-1 && ptrs[updateAt+1].OverlapsWith(p.TimeRange)
	overlapsWithPrev := updateAt != 0 && ptrs[updateAt-1].OverlapsWith(p.TimeRange)
	if overlapsWithPrev || overlapsWithNext {
		idx.mu.Unlock()
		return span.Error(ErrDomainOverlap)
	} else {
		idx.mu.pointers[updateAt] = p
	}

	idx.persistHead = min(idx.persistHead, updateAt)

	if persist {
		update := indexUpdate{afterIndex: idx.persistHead}
		idx.persistHead = len(idx.mu.pointers) - 1
		idx.mu.Unlock()
		idx.Observer.Notify(ctx, update)

		return nil
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

func (idx *index) persist(ctx context.Context) {
	idx.mu.RLock()
	update := indexUpdate{afterIndex: idx.persistHead}
	idx.Observer.Notify(ctx, update)
	idx.mu.RUnlock()
}

func (idx *index) searchLE(ctx context.Context, ts telem.TimeStamp) (i int) {
	_, span := idx.T.Bench(ctx, "searchLE")
	idx.read(func() {
		i, _ = idx.unprotectedSearch(ts.SpanRange(0))
	})
	span.End()
	return
}

func (idx *index) searchGE(ctx context.Context, ts telem.TimeStamp) (i int) {
	_, span := idx.T.Bench(ctx, "searchGE")
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
	_, span := idx.T.Bench(ctx, "searchGE")
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
	idx.mu.pointers = nil
	idx.mu.Unlock()
	return nil
}
