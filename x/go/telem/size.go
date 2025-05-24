// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import "strconv"

// Size represents the size of an element in bytes.
type Size int64

// String implements fmt.Stringer.
func (s Size) String() string { return strconv.Itoa(int(s)) + " B" }

const (
	// ByteSize is a single byte.
	ByteSize = Size(1)
	// Kilobyte is 1000 bytes.
	Kilobyte = 1000 * ByteSize
	// Megabyte is 1000 kilobytes.
	Megabyte = 1000 * Kilobyte
	// Gigabyte is 1000 megabytes.
	Gigabyte = 1000 * Megabyte
)

// Offset is a number of bytes to offset.
type Offset = Size
