package ranger

import (
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/telem"
	"sync"
)

type index struct {
	mu struct {
		sync.RWMutex
		pointers []pointer
	}
	observer observe.Observer[indexUpdate]
}

type indexUpdate struct {
	afterIndex int
}

var _ observe.Observable[indexUpdate] = (*index)(nil)

// OnChange implements the Observable interface.
func (idx *index) OnChange(f func(update indexUpdate)) { idx.observer.OnChange(f) }

// insert adds a new pointer to the index.
func (idx *index) insert(p pointer) error {
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
				return ErrRangeOverlap
			}
			insertAt = i + 1
		}
	}
	idx.mu.RUnlock()
	idx.insertAt(insertAt, p)
	return nil
}

func (idx *index) overlap(tr telem.TimeRange) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	_, overlap := idx.unprotectedSearch(tr)
	return overlap
}

// update the pointer with the same start timestamp as p.
func (idx *index) update(p pointer) error {
	idx.mu.RLock()
	if len(idx.mu.pointers) == 0 {
		idx.mu.RUnlock()
		return RangeNotFound
	}
	lastI := len(idx.mu.pointers) - 1
	updateAt := lastI
	if p.Start != idx.mu.pointers[lastI].Start {
		updateAt, _ = idx.unprotectedSearch(p.Start.SpanRange(0))
	}
	idx.mu.RUnlock()
	return idx.updateAt(updateAt, p)
}

func (idx *index) afterLast(ts telem.TimeStamp) bool {
	return ts.After(idx.mu.pointers[len(idx.mu.pointers)-1].End)
}

func (idx *index) beforeFirst(ts telem.TimeStamp) bool {
	return ts.Before(idx.mu.pointers[0].Start)
}

func (idx *index) insertAt(i int, p pointer) {
	idx.modifyAfter(i, func() {
		if i == 0 {
			idx.mu.pointers = append([]pointer{p}, idx.mu.pointers...)
		} else if i == len(idx.mu.pointers) {
			idx.mu.pointers = append(idx.mu.pointers, p)
		} else {
			idx.mu.pointers = append(idx.mu.pointers[:i], append([]pointer{p}, idx.mu.pointers[i:]...)...)
		}
	})
}

func (idx *index) updateAt(i int, p pointer) (err error) {
	idx.modifyAfter(i, func() {
		oldP := idx.mu.pointers[i]
		if oldP.Start != p.Start {
			err = RangeNotFound
		} else if i != len(idx.mu.pointers)-1 && idx.mu.pointers[i+1].OverlapsWith(p.TimeRange) ||
			i != 0 && idx.mu.pointers[i-1].OverlapsWith(p.TimeRange) {
			err = ErrRangeOverlap
		} else {
			idx.mu.pointers[i] = p
		}
	})
	return
}

func (idx *index) modifyAfter(i int, f func()) {
	update := indexUpdate{afterIndex: i}
	defer func() {
		idx.observer.Notify(update)
	}()
	idx.mu.Lock()
	defer idx.mu.Unlock()
	f()
}

func (idx *index) searchLE(ts telem.TimeStamp) (i int) {
	idx.read(func() {
		i, _ = idx.unprotectedSearch(ts.SpanRange(0))
	})
	return
}

func (idx *index) searchGE(ts telem.TimeStamp) (i int) {
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
	return
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
	defer idx.mu.RUnlock()
	if i < 0 || i >= len(idx.mu.pointers) {
		return pointer{}, false
	}
	return idx.mu.pointers[i], true
}

func (idx *index) read(f func()) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	f()
}

func (idx *index) close() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.mu.pointers = nil
	return nil
}
