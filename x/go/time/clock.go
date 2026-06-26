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
	"sync"
	"time"
)

// Timer is a single-shot timer scheduled via Clock.AfterFunc. Stop prevents the timer
// from firing and returns true if the call stopped the timer, or false if the timer
// has already fired or been stopped.
type Timer interface {
	// Stop prevents the timer from firing and returns true if the call stopped the timer,
	// or false if the timer has already fired or been stopped.
	Stop() bool
}

// Clock provides a time source and timer scheduler.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
	// After returns a channel that will receive the current time after the duration
	// elapses.
	After(time.Duration) <-chan time.Time
	// AfterFunc schedules fn to be called after the duration elapses and returns a
	// Timer that can be used to cancel the call. If Stop is called before fn would have
	// run, fn is not called.
	AfterFunc(time.Duration, func()) Timer
}

type real struct{}

// Real is a Clock that uses the system's real-time clock.
var Real Clock = real{}

func (real) Now() time.Time { return time.Now() }

func (real) After(d time.Duration) <-chan time.Time { return time.After(d) }

func (real) AfterFunc(d time.Duration, f func()) Timer { return time.AfterFunc(d, f) }

// Fake is a Clock that uses a fake time source and timer scheduler for testing.
type Fake struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
}

var _ Clock = &Fake{}

type fakeTimer struct {
	at time.Time
	ch chan time.Time
}

// Now returns the current time.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// After returns a channel that will receive the current time after the duration elapses.
func (f *Fake) After(d time.Duration) <-chan time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	ch := make(chan time.Time)
	t := &fakeTimer{at: f.now.Add(d), ch: ch}
	f.timers = append(f.timers, t)
	return ch
}

// AfterFunc waits on a channel registered with After in a goroutine, calling fn unless
// the returned Timer's Stop method wins the race first. The goroutine always drains the
// channel, so Advance can still complete its send when a timer is stopped.
func (f *Fake) AfterFunc(d time.Duration, fn func()) Timer {
	t := &fakeFuncTimer{}
	ch := f.After(d)
	go func() {
		<-ch
		if t.claim() {
			fn()
		}
	}()
	return t
}

// fakeFuncTimer coordinates the race between Stop and the AfterFunc goroutine: whoever
// calls claim first wins. Stop winning suppresses fn; the goroutine winning runs fn.
type fakeFuncTimer struct {
	mu      sync.Mutex
	claimed bool
}

func (t *fakeFuncTimer) claim() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.claimed {
		return false
	}
	t.claimed = true
	return true
}

func (t *fakeFuncTimer) Stop() bool { return t.claim() }

// Advance advances the clock by the given duration. It fires any timers whose deadline
// has been crossed and removes them from the list of pending timers. Channel sends
// happen after f.mu is released so that timer receivers (including AfterFunc callbacks)
// can safely call back into the clock without deadlocking.
func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	f.now = f.now.Add(d)
	now := f.now
	var fire []*fakeTimer
	remaining := f.timers[:0]
	for _, t := range f.timers {
		if !t.at.After(now) {
			fire = append(fire, t)
		} else {
			remaining = append(remaining, t)
		}
	}
	f.timers = remaining
	f.mu.Unlock()
	for _, t := range fire {
		t.ch <- now
		close(t.ch)
	}
}
