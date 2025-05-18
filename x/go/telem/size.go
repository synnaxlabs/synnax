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

const (
	ByteSize = Size(1)
	Kilobyte = 1024 * ByteSize
	Megabyte = 1024 * Kilobyte
	Gigabyte = 1024 * Megabyte
)

type Offset = Size

// String implements fmt.Stringer.
func (s Size) String() string { return strconv.Itoa(int(s)) + "B" }
