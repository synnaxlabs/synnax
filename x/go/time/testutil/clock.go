// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"sync"
	"time"

	xtime "github.com/synnaxlabs/x/time"
)

// Clock is an xtime.Clock whose time moves only when Advance moves it. The zero
// value reads the zero time. Clock is safe for concurrent use.
type Clock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
}

var _ xtime.Clock = (*Clock)(nil)

// NewClock returns a Clock that reads start until Advance moves it.
func NewClock(start time.Time) *Clock { return &Clock{now: start} }

type fakeTimer struct {
	at time.Time
	ch chan time.Time
}

// Now returns the current time.
func (f *Clock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// schedule returns a channel that receives at the given deadline. A deadline at or
// before the current time is delivered right away instead of joining the pending list,
// where only a later Advance would reach it. The channel is buffered so that Advance's
// send never blocks, even if the goroutine reading it has already exited via Stop.
func (f *Clock) schedule(at time.Time) <-chan time.Time {
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

// Pending returns the number of timers scheduled but not yet fired. A test driving a
// consumer that blocks on a timer uses it to wait until the consumer has scheduled,
// because Advance fires only the timers already on the list.
func (f *Clock) Pending() int { f.mu.Lock(); defer f.mu.Unlock(); return len(f.timers) }

// RunAt schedules fn to run from a goroutine at the given deadline, calling it unless
// the returned Timer's Stop method wins the race first.
func (f *Clock) RunAt(t time.Time, fn func()) xtime.Timer {
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
func newFakeFuncTimer(ch <-chan time.Time, fn func()) xtime.Timer {
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
func (f *Clock) Advance(d time.Duration) {
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
