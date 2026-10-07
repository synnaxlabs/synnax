// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package timer provides a timer that wakes closer to its deadline than a time.Timer.
// A time.Timer wakes at most once per millisecond on Linux, and up to 1 ms late on
// macOS.
package timer

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// SpinAll is a spin that lasts the whole of each wait.
const SpinAll = time.Duration(math.MaxInt64)

// Timer sends on C when the span given to Reset elapses. It never fires early, and
// after Reset or Stop returns, C receives nothing for an earlier deadline. A Timer is
// not safe for concurrent use.
type Timer struct {
	// spin is how long the timer spins on Now before each deadline instead of waiting
	// on the OS timer.
	spin time.Duration
	// waitSpin is spin for the current wait. The spin reads it without mu.
	waitSpin atomic.Int64
	// C receives a value when the timer fires.
	C <-chan struct{}
	// c is C with send access. It holds at most one pending fire.
	c chan struct{}
	// mu guards writes to deadline, sends on c, and changes to the OS timer.
	mu sync.Mutex
	// deadline is the Now reading the timer fires at, or zero when it is stopped. The
	// spin reads it without mu, so Reset and Stop do not wait on the spin.
	deadline atomic.Int64
	// done closes when the wait goroutine exits.
	done chan struct{}
	// err is the wait goroutine's failure. It is safe to read after done closes.
	err error
	platform
}

// New opens a stopped Timer that spins on Now for the last spin of each wait, which
// holds a CPU core for that span. A spin never takes more than half of a wait, unless
// that half is under DefaultSpin, and SpinAll spins for all of it. The caller must
// Close the Timer.
func New(spin time.Duration) (*Timer, error) {
	c := make(chan struct{}, 1)
	t := &Timer{C: c, c: c, done: make(chan struct{}), spin: spin}
	if err := t.open(); err != nil {
		return nil, err
	}
	go func() {
		// The last fire wakes the receiver after done closes, so its next Reset or
		// Stop returns err.
		defer t.fire()
		defer close(t.done)
		t.err = t.wait()
	}()
	return t, nil
}

// Reset stops the timer and starts it again, so it fires after d. A d of zero or less
// fires right away.
func (t *Timer) Reset(d time.Duration) error {
	if d <= 0 {
		if err := t.Stop(); err != nil {
			return err
		}
		t.fire()
		return nil
	}
	if err := t.failure(); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.deadline.Store(int64(Now() + d))
	t.waitSpin.Store(int64(t.spinFor(d)))
	t.drain()
	return t.arm(t.lead(d))
}

// Stop stops the timer and drops a pending fire.
func (t *Timer) Stop() error {
	if err := t.failure(); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.deadline.Store(0)
	t.drain()
	return t.disarm()
}

// Close stops the timer and releases its OS timer.
func (t *Timer) Close() error {
	t.mu.Lock()
	// The wait goroutine never arms a stopped timer, so close can release it.
	t.deadline.Store(0)
	t.mu.Unlock()
	return t.close()
}

// expire fires the timer when its deadline has come, spinning through the last spin
// of the wait. An earlier wake arms the OS timer for the rest of the span. The wait
// goroutine calls expire on each OS timer wake.
func (t *Timer) expire() error {
	t.spinToDeadline()
	t.mu.Lock()
	defer t.mu.Unlock()
	deadline := time.Duration(t.deadline.Load())
	if deadline == 0 {
		return nil
	}
	if left := deadline - Now(); left > 0 {
		return t.arm(t.lead(left))
	}
	t.deadline.Store(0)
	t.fire()
	return nil
}

// spinToDeadline spins until the deadline, unless the timer is stopped or the deadline
// is more than the spin of the current wait away.
func (t *Timer) spinToDeadline() {
	for {
		deadline := time.Duration(t.deadline.Load())
		left := deadline - Now()
		if deadline == 0 || left <= 0 || left > time.Duration(t.waitSpin.Load()) {
			return
		}
	}
}

// spinFor returns the spin of a wait of d. Half of a wait never cuts the spin under
// DefaultSpin, the least spin that outlasts a late wake of the OS timer.
func (t *Timer) spinFor(d time.Duration) time.Duration {
	if t.spin == SpinAll {
		return SpinAll
	}
	return min(t.spin, max(d/2, DefaultSpin))
}

// lead returns the span to arm the OS timer for, so that it wakes the spin of the
// current wait before a deadline d away. The span is always positive, because arming
// zero disarms.
func (t *Timer) lead(d time.Duration) time.Duration {
	return max(d-time.Duration(t.waitSpin.Load()), time.Nanosecond)
}

// failure returns the wait goroutine's failure, or nil while it runs.
func (t *Timer) failure() error {
	select {
	case <-t.done:
		return t.err
	default:
		return nil
	}
}

func (t *Timer) fire() {
	select {
	case t.c <- struct{}{}:
	default:
	}
}

func (t *Timer) drain() {
	select {
	case <-t.c:
	default:
	}
}
