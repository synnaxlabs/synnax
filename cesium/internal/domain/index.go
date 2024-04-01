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
}

type indexUpdate struct {
	afterIndex  int
	unprotected bool
}

var _ observe.Observable[indexUpdate] = (*index)(nil)

// insert adds a new pointer to the index.
func (idx *index) insert(ctx context.Context, p pointer, withLock bool) error {
	_, span := idx.T.Bench(ctx, "insert")
	defer span.End()
	if withLock {
		idx.mu.Lock()
	}
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
				if withLock {
					idx.mu.Unlock()
				}
				return span.EndWith(ErrDomainOverlap)
			}
			insertAt = i + 1
		}
	}

	// If we want to run this function with lock, then we have already locked it, therefore
	// we must call insertAt with no lock.
	// If we want to run this function with no lock, then we still should not lock
	idx.insertAt(ctx, insertAt, p, false)

	if withLock {
		idx.mu.Unlock()
	}
	return nil
}

func (idx *index) insertTombstone(ctx context.Context, p pointer, withLock bool) {
	_, span := idx.T.Bench(ctx, "insert tombstone")
	defer span.End()
	if withLock {
		idx.mu.Lock()
		defer idx.mu.Unlock()
	}

	if idx.mu.tombstones == nil {

	}

	idx.mu.tombstones[p.fileKey] = append(idx.mu.tombstones[p.fileKey], p)
}

func (idx *index) overlap(tr telem.TimeRange, withLock bool) bool {
	if withLock {
		idx.mu.RLock()
		defer idx.mu.RUnlock()
	}
	_, overlap := idx.unprotectedSearch(tr)
	return overlap
}

func (idx *index) update(ctx context.Context, p pointer, withLock bool) (err error) {
	_, span := idx.T.Bench(ctx, "update")
	if withLock {
		idx.mu.Lock()
		defer idx.mu.Unlock()
	}
	if len(idx.mu.pointers) == 0 {
		return span.EndWith(RangeNotFound)
	}
	lastI := len(idx.mu.pointers) - 1
	updateAt := lastI
	if p.Start != idx.mu.pointers[lastI].Start {
		updateAt, _ = idx.unprotectedSearch(p.Start.SpanRange(0))
	}
	return span.EndWith(idx.updateAt(ctx, updateAt, p, false))
}

func (idx *index) afterLast(ts telem.TimeStamp) bool {
	return ts.After(idx.mu.pointers[len(idx.mu.pointers)-1].End)
}

func (idx *index) beforeFirst(ts telem.TimeStamp) bool {
	return ts.Before(idx.mu.pointers[0].Start)
}

func (idx *index) insertAt(ctx context.Context, i int, p pointer, withLock bool) {
	idx.modifyAfter(ctx, i, func() {
		if i == 0 {
			idx.mu.pointers = append([]pointer{p}, idx.mu.pointers...)
		} else if i == len(idx.mu.pointers) {
			idx.mu.pointers = append(idx.mu.pointers, p)
		} else {
			idx.mu.pointers = append(idx.mu.pointers[:i], append([]pointer{p}, idx.mu.pointers[i:]...)...)
		}
	}, withLock)
}

// updateAt updates the i-th pointer in the index
func (idx *index) updateAt(ctx context.Context, i int, p pointer, withLock bool) (err error) {
	ptrs := idx.mu.pointers
	idx.modifyAfter(ctx, i, func() {
		oldP := ptrs[i]
		if oldP.Start != p.Start {
			err = RangeNotFound
			return
		}
		overlapsWithNext := i != len(ptrs)-1 && ptrs[i+1].OverlapsWith(p.TimeRange)
		overlapsWithPrev := i != 0 && ptrs[i-1].OverlapsWith(p.TimeRange)
		if overlapsWithPrev || overlapsWithNext {
			err = ErrDomainOverlap
		} else {
			idx.mu.pointers[i] = p
		}
	}, withLock)
	return
}

// modifyAfter updates the i-th pointer and notifies an index update
func (idx *index) modifyAfter(ctx context.Context, i int, f func(), withLock bool) {
	update := indexUpdate{afterIndex: i, unprotected: true}
	defer func() {
		idx.Observer.Notify(ctx, update)
	}()
	if withLock {
		idx.mu.Lock()
	}
	f()
	if withLock {
		idx.mu.Unlock()
	}
}

func (idx *index) searchLE(ctx context.Context, ts telem.TimeStamp, withLock bool) (i int) {
	_, span := idx.T.Bench(ctx, "searchLE")
	if withLock {
		idx.mu.RLock()
	}
	i, _ = idx.unprotectedSearch(ts.SpanRange(0))
	if withLock {
		idx.mu.RUnlock()
	}
	span.End()
	return
}

func (idx *index) searchGE(ctx context.Context, ts telem.TimeStamp, withLock bool) (i int) {
	_, span := idx.T.Bench(ctx, "searchGE")
	if withLock {
		idx.mu.RLock()
	}
	var exact bool
	i, exact = idx.unprotectedSearch(ts.SpanRange(0))
	if !exact {
		if i == len(idx.mu.pointers) {
			i = -1
		} else {
			i += 1
		}
	}
	if withLock {
		idx.mu.RUnlock()
	}
	span.End()
	return
}

// unprotectedSearch binary searches for a pointer overlapping tr
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

// get gets the i-th pointer in the domain, if it is out of bounds, it returns false
func (idx *index) get(i int, withLock bool) (pointer, bool) {
	if withLock {
		idx.mu.RLock()
	}
	if i < 0 || i >= len(idx.mu.pointers) {
		if withLock {
			idx.mu.RUnlock()
		}
		return pointer{}, false
	}
	v := idx.mu.pointers[i]
	if withLock {
		idx.mu.RUnlock()
	}
	return v, true
}

func (idx *index) read(f func()) {
	idx.mu.RLock()
	f()
	idx.mu.RUnlock()
}

func (idx *index) close(withLock bool) error {
	if withLock {
		idx.mu.Lock()
	}
	idx.mu.pointers = nil
	if withLock {
		idx.mu.Unlock()
	}
	return nil
}
