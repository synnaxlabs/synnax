// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slices

import (
	"fmt"
	"iter"
)

// Truncate returns a limited number of elements from the slice if the number is larger
// than maxDisplayValues, splitting the values evenly between maxDisplayValues/2 in first
// and maxDisplayValues/2 in last. If len(slice) is less than maxDisplayValues,
// returns the entire slice as first, and last is nil.
func Truncate[T any](slice []T, maxDisplayValues int) (first, last []T) {
	if len(slice) <= maxDisplayValues || maxDisplayValues <= 0 {
		first = slice
		return
	}
	startCount := maxDisplayValues / 2
	endCount := maxDisplayValues - startCount
	first = slice[:startCount]
	last = slice[len(slice)-endCount:]
	return first, last
}

// ConvertNegativeIndex adds compatibility for working with negative indexes in slices,
// automatically converting the given index into the correct adjusted positive index
// for the slice.
//
// Panics if the negative index is out of bounds.
func ConvertNegativeIndex(i int, len int) int {
	if i < 0 {
		adjusted := i + len
		if adjusted > len || adjusted < 0 {
			panic(fmt.Sprintf("index out of range [%v] with length %v", i, len))
		}
		return adjusted
	}
	return i
}

func IterEndlessly[T any](values []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		i := 0
		for {
			if !yield(values[i]) {
				return
			}
			i = (i + 1) % len(values)
		}
	}
}
