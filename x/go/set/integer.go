// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package set

import (
	"slices"

	"golang.org/x/exp/constraints"
)

// intRange is a set that includes every integer between start and end (start inclusive,
// end exclusive).
type intRange[T constraints.Integer] struct {
	// start is the first integer in intRange.
	start T
	// end is the integer after the last integer in intRange.
	end T
}

// size returns how many integers intRange contains.
func (r intRange[T]) size() T {
	return r.end - r.start
}

// Integer is a set of integers that is memory-optimized to limit the space required for
// consecutive number of integers.
type Integer[T constraints.Integer] []intRange[T]

// NewInteger creates a new Integer set and inserts the provided nums.
func NewInteger[T constraints.Integer](nums []T) *Integer[T] {
	s := &Integer[T]{}
	s.Insert(nums...)
	return s
}

// Insert inserts the integers nums into the set s.
func (s *Integer[T]) Insert(nums ...T) {
	if len(nums) == 0 {
		return
	}
	r := intRange[T]{
		start: nums[0],
		end:   nums[0],
	}
	for _, n := range nums {
		if r.end == n {
			r.end++
		} else {
			s.insert(r)
			r.start = n
			r.end = n + 1
		}
	}
	s.insert(r)
}

// Remove removes all integers in nums from s. Remove is idempotent, so you can call
// remove from a set s even if some or all integers in nums do not exist in s.
func (s *Integer[T]) Remove(nums ...T) {
	if len(nums) == 0 {
		return
	}

	toRemove := make([]T, 0, len(nums))
	for _, num := range nums {
		if s.Contains(num) {
			toRemove = append(toRemove, num)
		}
	}
	if len(toRemove) == 0 {
		return
	}

	r := intRange[T]{
		start: toRemove[0],
		end:   toRemove[0],
	}

	for _, num := range toRemove {
		if r.end == num {
			r.end++
		} else {
			s.remove(r)
			r.start = num
			r.end = num + 1
		}
	}
	s.remove(r)
}

// NumLessThan returns the total number of integers in s less than num.
func (s Integer[T]) NumLessThan(num T) T {
	counter := T(0)
	for i := 0; i < len(s) && s[i].start < num; i++ {
		if s[i].end > num {
			counter += num - s[i].start
		} else {
			counter += s[i].size()
		}
	}
	return counter
}

// NumGreaterThan returns the total number of integers in s greater than num.
func (s Integer[T]) NumGreaterThan(num T) T {
	counter := T(0)
	for i := len(s) - 1; i >= 0 && s[i].end > num; i-- {
		if s[i].start <= num {
			counter += s[i].end - num - 1
		} else {
			counter += s[i].size()
		}
	}
	return counter
}

// Size returns the total unique numbers in Integer s.
func (s Integer[T]) Size() T {
	counter := T(0)
	for _, r := range s {
		counter += r.size()
	}
	return counter
}

// Contains return true if num is in s and false otherwise.
func (s Integer[T]) Contains(num T) bool {
	i, _ := slices.BinarySearchFunc(s, num, func(a intRange[T], b T) int {
		if a.start > b {
			return 1
		}
		return -1
	})

	if i == 0 || s[i-1].end <= num {
		return false
	}
	return true
}

// Copy returns a deep copy of s.
func (s Integer[T]) Copy() Integer[T] {
	newSet := make(Integer[T], len(s))
	copy(newSet, s)
	return newSet
}

// delete removes the element s[i] from s.
func (s *Integer[T]) delete(i int) {
	*s = slices.Delete(*s, i, i+1)
}

// insert inserts r into s.
func (s *Integer[T]) insert(r intRange[T]) {
	if r.size() == 0 {
		return
	}
	if len(*s) == 0 {
		*s = Integer[T]{r}
		return
	}

	// c[i] is the first intRange in s whose start is later than r.start.
	i, _ := slices.BinarySearchFunc(*s, r, func(a, b intRange[T]) int {
		if a.start > b.start {
			return 1
		}
		return -1
	})

	// Check if the intRange to the left overlaps with r. If so, we remove the overlap
	// from r and insert it again.
	if i != 0 {
		endOfBefore := (*s)[i-1].end
		if endOfBefore > r.start {
			r.start = endOfBefore
			if r.start >= r.end {
				return
			}
			s.insert(r)
			return
		}
	}

	// Check if the intRange to the right overlaps with r. If so, remove overlap from r
	// and insert it again.
	if i != len(*s) {
		startOfNext := (*s)[i].start
		if r.end > startOfNext {
			s.insert(intRange[T]{
				start: startOfNext,
				end:   r.end,
			})
			r.end = startOfNext
			if r.end <= r.start {
				return
			}
			s.insert(r)
			return
		}
	}

	// At this point, r does not overlap at all with any of the intRanges in c.

	// Check if we need to compress c.
	if i != 0 && (*s)[i-1].end == r.start {
		// Compress on the left.
		(*s)[i-1].end = r.end
		// Compress on the right, if needed.
		if i != len(*s) && r.end == (*s)[i].start {
			(*s)[i-1].end = (*s)[i].end
			s.delete(i)
		}
		return
	} else if i != len(*s) && r.end == (*s)[i].start {
		// Compress on the right.
		(*s)[i].start = r.start
		return
	}

	*s = slices.Insert(*s, i, r)
}

// remove removes the intRange r from s.
func (s *Integer[T]) remove(r intRange[T]) {
	if len(*s) == 0 || r.size() == 0 {
		return
	}
	// s[i] is the first intRange that partially intersects with r or is completely to
	// the right of r.
	i, _ := slices.BinarySearchFunc(*s, r, func(a, b intRange[T]) int {
		if a.end > b.start {
			return 1
		}
		return -1
	})

	// r is not in the set s.
	if i == len(*s) || r.end <= (*s)[i].start {
		return
	}

	if r.start < (*s)[i].start {
		r.start = (*s)[i].start
	}

	if r.end > (*s)[i].end {
		s.remove(intRange[T]{
			start: (*s)[i].end,
			end:   r.end,
		})
		r.end = (*s)[i].end
	}

	// r is fully in s[i].

	if r.start == (*s)[i].start {
		if r.end == (*s)[i].end {
			s.delete(i)
			return
		}
		(*s)[i].start = r.end
		return
	}
	if r.end == (*s)[i].end {
		(*s)[i].end = r.start
		return
	}

	// r is in the middle of s, so we need to split s. s[i] will be to the left of r,
	// newintRange will be to the right of r.
	newRange := intRange[T]{
		start: r.end,
		end:   (*s)[i].end,
	}
	(*s)[i].end = r.start
	s.insert(newRange)
}
