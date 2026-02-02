// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import "fmt"

// String implements fmt.Stringer.
func (s Size) String() string {
	if s > Exabyte {
		return fmt.Sprintf("%g EB", s.Exabytes())
	}
	if s > Petabyte {
		return fmt.Sprintf("%g PB", s.Petabytes())
	}
	if s > Terabyte {
		return fmt.Sprintf("%g TB", s.Terabytes())
	}
	if s > Gigabyte {
		return fmt.Sprintf("%g GB", s.Gigabytes())
	}
	if s > Megabyte {
		return fmt.Sprintf("%g MB", s.Megabytes())
	}
	if s > Kilobyte {
		return fmt.Sprintf("%g kB", s.Kilobytes())
	}
	return fmt.Sprintf("%d B", s)
}

// Kilobytes returns the number of kilobytes in the size.
func (s Size) Kilobytes() float64 { return float64(s) / float64(Kilobyte) }

// Megabytes returns the number of megabytes in the size.
func (s Size) Megabytes() float64 { return float64(s) / float64(Megabyte) }

// Gigabytes returns the number of gigabytes in the size.
func (s Size) Gigabytes() float64 { return float64(s) / float64(Gigabyte) }

// Terabytes returns the number of terabytes in the size.
func (s Size) Terabytes() float64 { return float64(s) / float64(Terabyte) }

// Petabytes returns the number of petabytes in the size.
func (s Size) Petabytes() float64 { return float64(s) / float64(Petabyte) }

// Exabytes returns the number of exabytes in the size.
func (s Size) Exabytes() float64 { return float64(s) / float64(Exabyte) }

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
