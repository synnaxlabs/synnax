// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

// Rate represents a rate in Hz.
type Rate float64

// Period returns a TimeSpan representing the period of the Rate.
func (r Rate) Period() TimeSpan { return TimeSpan(float64(Second) / float64(r)) }

// SampleCount returns an integer representing the number of samples in the provided Span.
func (r Rate) SampleCount(t TimeSpan) int { return int(t.Seconds() * float64(r)) }

const (
	// Hertz is a data rate of 1 Hz.
	Hertz Rate = 1
	// Kilohertz is a data rate of 1 kHz.
	Kilohertz = 1000 * Hertz
	// Megahertz is a data rate of 1 MHz.
	Megahertz = 1000 * Kilohertz
)
