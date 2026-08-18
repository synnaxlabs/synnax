// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

// Gigabytes returns the number of gigabytes in the size.
func (s Size) Gigabytes() float64 { return float64(s) / float64(Gigabyte) }

const (
	// Byte is a single byte.
	Byte = Size(1)
	// Kilobyte is 1,000 bytes.
	Kilobyte = 1000 * Byte
	// Megabyte is 1,000 kilobytes.
	Megabyte = 1000 * Kilobyte
	// Gigabyte is 1,000 megabytes.
	Gigabyte = 1000 * Megabyte
	// Terabyte is 1,000 gigabytes.
	Terabyte = 1000 * Gigabyte
	// Petabyte is 1,000 terabytes.
	Petabyte = 1000 * Terabyte
	// Exabyte is 1,000 petabytes.
	Exabyte = 1000 * Petabyte
)
