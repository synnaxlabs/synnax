// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/x/slices"
)

// TruncateAndFormatSlice returns a string representation of a slice, showing only the
// first and last few elements if the slice is longer than maxDisplayValues. The
// displayed elements are split evenly between the start and end.
func TruncateAndFormatSlice[T any](
	slice []T,
	maxDisplayValues int,
) string {
	first, last := slices.Truncate(slice, maxDisplayValues)
	if len(last) == 0 {
		return fmt.Sprintf("%v", first)
	}
	firstStr := strings.Trim(fmt.Sprintf("%v", first), "[]")
	lastStr := strings.Trim(fmt.Sprintf("%v", last), "[]")
	return fmt.Sprintf("[%s ... %s]", firstStr, lastStr)
}
