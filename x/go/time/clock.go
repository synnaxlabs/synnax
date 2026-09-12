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

// schedule returns a channel that receives at the given deadline. A deadline at or
// before the current time is delivered right away instead of joining the pending list,
// where only a later Advance would reach it. The channel is buffered so that Advance's
// send never blocks, even if the goroutine reading it has already exited via Stop.
func (f *Fake) schedule(at time.Time) <-chan time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	ch := make(chan time.Time, 1)
	if !at.After(f.now) {
		ch <- f.now
		close(ch)
		return ch
	}
	f.timers = append(f.timers, &fakeTimer{at: at, ch: ch})
	return ch
}

// RunAt schedules fn to run from a goroutine at the given deadline, calling it unless
// the returned Timer's Stop method wins the race first.
func (f *Fake) RunAt(t time.Time, fn func()) Timer {
	return newFakeFuncTimer(f.schedule(t), fn)
}

// fakeFuncTimer coordinates the race between Stop and the timer goroutine: whoever
// calls claim first wins. Stop winning suppresses fn and releases the goroutine; the
// goroutine winning runs fn.
type fakeFuncTimer struct {
	mu      sync.Mutex
	claimed bool
	stop    chan struct{}
}

// newFakeFuncTimer calls fn once ch receives. Stop releases the goroutine when the
// timer is cancelled before Advance crosses its deadline; otherwise a never-fired timer
// would leak its goroutine.
func newFakeFuncTimer(ch <-chan time.Time, fn func()) Timer {
	t := &fakeFuncTimer{stop: make(chan struct{})}
	go func() {
		select {
		case <-ch:
			if t.claim() {
				fn()
			}
		case <-t.stop:
		}
	}()
	return t
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

// Stop cancels the timer, returning true if it had not already fired or been stopped.
// On the winning call it releases the RunAt goroutine.
func (t *fakeFuncTimer) Stop() bool {
	if !t.claim() {
		return false
	}
	close(t.stop)
	return true
}

// Advance advances the clock by the given duration. It fires any timers whose deadline
// has been crossed and removes them from the list of pending timers. Channel sends
// happen after f.mu is released so that RunAt callbacks can safely call back into the
// clock without deadlocking.
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
