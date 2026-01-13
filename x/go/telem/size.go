// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"fmt"
)

// String implements fmt.Stringer.
func (s Size) String() string {
	if s > Terabyte {
		return fmt.Sprintf("%g TB", float64(s)/float64(Terabyte))
	}
	if s > Gigabyte {
		return fmt.Sprintf("%g GB", float64(s)/float64(Gigabyte))
	}
	if s > Megabyte {
		return fmt.Sprintf("%g MB", float64(s)/float64(Megabyte))
	}
	if s > Kilobyte {
		return fmt.Sprintf("%g kB", float64(s)/float64(Kilobyte))
	}
	return fmt.Sprintf("%d B", s)
}

const (
	// Byte is a single byte.
	Byte = Size(1)
	// Kilobyte is 1000 bytes.
	Kilobyte = 1000 * Byte
	// Megabyte is 1000 kilobytes.
	Megabyte = 1000 * Kilobyte
	// Gigabyte is 1000 megabytes.
	Gigabyte = 1000 * Megabyte
	// Terabyte is 1000 gigabytes.
	Terabyte = 1000 * Gigabyte
)
