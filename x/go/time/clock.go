// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package time

import (
	"time"
)

// Timer is a single-shot timer scheduled by a Clock. Stop prevents the timer from
// firing and returns true if the call stopped the timer, or false if the timer has
// already fired or been stopped.
type Timer interface {
	// Stop prevents the timer from firing and returns true if the call stopped the
	// timer, or false if the timer has already fired or been stopped.
	Stop() bool
}

// Clock provides a time source and timer scheduler.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
	// RunAt schedules fn to be called at the deadline and returns a Timer that can be
	// used to cancel the call. fn runs immediately if the deadline has already passed.
	// If Stop is called before fn would have run, fn is not called.
	RunAt(time.Time, func()) Timer
}

type real struct{}

// Real is a Clock that uses the system's real-time clock.
var Real Clock = real{}

func (real) Now() time.Time { return time.Now() }

func (real) RunAt(t time.Time, f func()) Timer {
	return time.AfterFunc(time.Until(t), f)
}
