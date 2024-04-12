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
	afterIndex int
}

var _ observe.Observable[indexUpdate] = (*index)(nil)

// insert adds a new pointer to the index.
func (idx *index) insert(ctx context.Context, p pointer) error {
	_, span := idx.T.Bench(ctx, "insert")
	defer span.End()
	idx.mu.RLock()
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
				idx.mu.RUnlock()
				return span.EndWith(ErrDomainOverlap)
			}
			insertAt = i + 1
		}
	}
	idx.mu.RUnlock()
	idx.insertAt(ctx, insertAt, p)
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

func (idx *index) update(ctx context.Context, p pointer) (err error) {
	_, span := idx.T.Bench(ctx, "update")
	idx.mu.RLock()
	if len(idx.mu.pointers) == 0 {
		idx.mu.RUnlock()
		return span.EndWith(RangeNotFound)
	}
	lastI := len(idx.mu.pointers) - 1
	updateAt := lastI
	if p.Start != idx.mu.pointers[lastI].Start {
		updateAt, _ = idx.unprotectedSearch(p.Start.SpanRange(0))
	}
	idx.mu.RUnlock()
	return span.EndWith(idx.updateAt(ctx, updateAt, p))
}

func (idx *index) afterLast(ts telem.TimeStamp) bool {
	return ts.After(idx.mu.pointers[len(idx.mu.pointers)-1].End)
}

func (idx *index) beforeFirst(ts telem.TimeStamp) bool {
	return ts.Before(idx.mu.pointers[0].Start)
}

func (idx *index) insertAt(ctx context.Context, i int, p pointer) {
	idx.modifyAfter(ctx, i, func() {
		if i == 0 {
			idx.mu.pointers = append([]pointer{p}, idx.mu.pointers...)
		} else if i == len(idx.mu.pointers) {
			idx.mu.pointers = append(idx.mu.pointers, p)
		} else {
			idx.mu.pointers = append(idx.mu.pointers[:i], append([]pointer{p}, idx.mu.pointers[i:]...)...)
		}
	})
}

func (idx *index) updateAt(ctx context.Context, i int, p pointer) (err error) {
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
	})
	return
}

func (idx *index) modifyAfter(ctx context.Context, i int, f func()) {
	update := indexUpdate{afterIndex: i}
	defer func() {
		idx.mu.Unlock()
		idx.Observer.Notify(ctx, update)
	}()
	idx.mu.Lock()
	f()
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
