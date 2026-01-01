// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package version

// Counter is a simple monotonic counter that tracks a version.
type Counter int64

// Increment increments the Counter.
func (c Counter) Increment() Counter { return c + 1 }

// NewerThan returns true if the Counter is higher than other.
func (c Counter) NewerThan(other Counter) bool { return c > other }

// OlderThan returns true if the counter is lower than other.
func (c Counter) OlderThan(other Counter) bool { return c < other }

// EqualTo returns true if the counter is equal to another Counter.
func (c Counter) EqualTo(other Counter) bool { return c == other }
