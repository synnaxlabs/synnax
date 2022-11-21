package ranger

import (
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"sync"
)

type index struct {
	mu struct {
		sync.RWMutex
		pointers []pointer
	}
	persist *indexPersist
}

func openIndex(fs xfs.FS) (*index, error) {
	persist, err := openIndexPersist(fs)
	if err != nil {
		return nil, err
	}
	pointers, err := persist.load()
	if err != nil {
		return nil, err
	}
	idx := &index{persist: persist}
	idx.mu.pointers = pointers
	return idx, nil
}

// insert adds a new pointer to the index.
func (idx *index) insert(p pointer) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if len(idx.mu.pointers) == 0 {
		idx.mu.pointers = append(idx.mu.pointers, p)
		return idx.persist.persist(0, idx.mu.pointers)
	}

	if p.bounds.Start.After(idx.mu.pointers[len(idx.mu.pointers)-1].bounds.End) {
		idx.mu.pointers = append(idx.mu.pointers, p)
		return idx.persist.persist(len(idx.mu.pointers)-1, idx.mu.pointers)
	}

	if p.bounds.Start.Before(idx.mu.pointers[0].bounds.Start) {
		idx.mu.pointers = append([]pointer{p}, idx.mu.pointers...)
		return idx.persist.persist(0, idx.mu.pointers)
	}

	start, end := 0, len(idx.mu.pointers)-1
	for start <= end {
		mid := (start + end) / 2
		ptr := idx.mu.pointers[mid]
		if ptr.bounds.OverlapsWith(p.bounds) {
			return ErrRangeOverlap
		}
		if p.bounds.Start.Before(ptr.bounds.Start) {
			end = mid - 1
		} else {
			start = mid + 1
		}
	}
	idx.mu.pointers = append(idx.mu.pointers[:end+1], idx.mu.pointers[end:]...)
	idx.mu.pointers[end] = p
	return idx.persist.persist(end, idx.mu.pointers)
}

// update the pointer with the same start timestamp as p.
func (idx *index) update(p pointer) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if len(idx.mu.pointers) == 0 {
		return RangeNotFound
	}

	lastI := len(idx.mu.pointers) - 1

	if p.bounds.Start == idx.mu.pointers[lastI].bounds.Start {
		return idx.updateAt(len(idx.mu.pointers)-1, p)
	}

	start, end := 0, len(idx.mu.pointers)-1
	for start <= end {
		mid := (start + end) / 2
		ptr := idx.mu.pointers[mid]
		if ptr.bounds.Start == p.bounds.Start {
			return idx.updateAt(mid, p)
		}
		if p.bounds.Start.Before(ptr.bounds.Start) {
			end = mid - 1
		} else {
			start = mid + 1
		}
	}
	return RangeNotFound
}

func (idx *index) updateAt(i int, p pointer) error {
	if i != len(idx.mu.pointers)-1 {
		fut := idx.mu.pointers[i+1]
		if fut.bounds.OverlapsWith(p.bounds) {
			return ErrRangeOverlap
		}
	}
	if i != 0 {
		past := idx.mu.pointers[i-1]
		if past.bounds.OverlapsWith(p.bounds) {
			return ErrRangeOverlap
		}
	}
	idx.mu.pointers[i] = p
	return idx.persist.persist(i, idx.mu.pointers)
}

func (idx *index) searchLE(ts telem.TimeStamp) (int, pointer) {
	i, exact := idx.search(ts)
	if exact {
		return i, idx.mu.pointers[i]
	}
	if i == 0 {
		return -1, pointer{}
	}
	return i - 1, idx.mu.pointers[i-1]
}

func (idx *index) searchGE(ts telem.TimeStamp) (int, pointer) {
	i, _ := idx.search(ts)
	if i == len(idx.mu.pointers) {
		return -1, pointer{}
	}
	return i, idx.mu.pointers[i]
}

// searchGE searches for the first pointer whose bounds overlap with the given timestamp.
// If no such pointer exists, returns the insertion index of the pointer that would be
// inserted if the timestamp were to be inserted.
func (idx *index) search(ts telem.TimeStamp) (int, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if len(idx.mu.pointers) == 0 {
		return -1, false
	}
	start, end := 0, len(idx.mu.pointers)
	for start < end {
		mid := (start + end) / 2
		ptr := idx.mu.pointers[mid]
		if ptr.bounds.ContainsStamp(ts) {
			return mid, true
		}
		if ts.Before(ptr.bounds.Start) {
			end = mid - 1
		} else {
			start = mid + 1
		}
	}
	if end == len(idx.mu.pointers) {
		return end, false
	}
	return end, idx.mu.pointers[end].bounds.ContainsStamp(ts)
}

func (idx *index) get(i int) (pointer, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	if i < 0 || i >= len(idx.mu.pointers) {
		return pointer{}, false
	}
	return idx.mu.pointers[i], true
}
