package array

import (
	"github.com/synnaxlabs/x/compare"
)

// Searchable implements binary search on a sorted array. The underlying array must
// be sorted in ascending order.
type Searchable[V any] struct{ Array[V] }

// Search searches the array for the 'insert position' of the given value and
// returns the value at that position and the index of that position.
func (s *Searchable[V]) Search(f compare.UnaryFunc[V]) (v V, i int) {
	start := 0
	end := s.Len() - 1
	for start <= end {
		mid := (start + end) / 2
		v := s.Get(mid)
		c := f(v)
		if c == compare.Equal {
			return v, mid
		}
		if c == compare.Less {
			start = mid + 1
		} else {
			end = mid - 1
		}
	}
	if end == -1 {
		return s.Get(0), -1
	}
	if end == s.Len()-1 {
		return s.Get(end), s.Len()
	}
	return s.Get(end), end
}
