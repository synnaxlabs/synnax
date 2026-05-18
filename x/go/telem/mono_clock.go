// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

// MonoClock provides monotonically increasing timestamps. On platforms with
// coarse clock resolution (e.g. Windows), consecutive Now() calls can return
// the same value. MonoClock guarantees each call to Now() returns a strictly
// greater timestamp than the previous one by bumping by 1 nanosecond when
// necessary. A custom time source can be provided for testing; if nil, the
// package-level Now function is used.
type MonoClock struct {
	last   TimeStamp
	Source func() TimeStamp
}

// Now returns a timestamp that is strictly greater than any previous call.
func (c *MonoClock) Now() TimeStamp {
	source := c.Source
	if source == nil {
		source = Now
	}
	now := source()
	if now <= c.last {
		now = c.last + 1
	}
	c.last = now
	return now
}
