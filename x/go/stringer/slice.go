// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stringer

import (
	"fmt"
	"strings"
)

func TruncateSlice[T any](slice []T, maxDisplayValues int) (first []T, last []T) {
	if len(slice) <= maxDisplayValues || maxDisplayValues <= 0 {
		first = slice
		return
	}
	startCount := maxDisplayValues / 2
	endCount := maxDisplayValues - startCount
	first = slice[:startCount]
	last = slice[len(slice)-endCount:]
	return
}

// TruncateAndFormatSlice returns a string representation of a slice, showing only the first and last few elements
// if the slice is longer than maxDisplayValues. The displayed elements are split evenly between the start and end.
func TruncateAndFormatSlice[T any](
	slice []T,
	maxDisplayValues int,
) string {
	first, last := TruncateSlice(slice, maxDisplayValues)
	if len(last) == 0 {
		return fmt.Sprintf("%v", first)
	}
	firstStr := strings.Trim(fmt.Sprintf("%v", first), "[]")
	lastStr := strings.Trim(fmt.Sprintf("%v", last), "[]")
	return fmt.Sprintf("[%s ... %s]", firstStr, lastStr)
}
