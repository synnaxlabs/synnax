// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slices

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
