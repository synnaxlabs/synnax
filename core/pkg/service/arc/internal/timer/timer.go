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
	"sync"
	"time"
)

// Timer sends on C when the span given to Reset elapses. It never fires early, and
// after Reset or Stop returns, C receives nothing for an earlier deadline. A Timer is
// not safe for concurrent use.
type Timer struct {
	// C receives a value when the timer fires.
	C <-chan struct{}
	// c is C with send access. It holds at most one pending fire.
	c chan struct{}
	// mu guards deadline, sends on c, and changes to the OS timer.
	mu sync.Mutex
	// deadline is the Now reading the timer fires at, or zero when it is stopped.
	deadline time.Duration
	// done closes when the wait goroutine exits.
	done chan struct{}
	// err is the wait goroutine's failure. It is safe to read after done closes.
	err error
	platform
}

// New opens a stopped Timer. The caller must Close it.
func New() (*Timer, error) {
	c := make(chan struct{}, 1)
	t := &Timer{C: c, c: c, done: make(chan struct{})}
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
	t.deadline = Now() + d
	t.drain()
	return t.arm(d)
}

// Stop stops the timer and drops a pending fire.
func (t *Timer) Stop() error {
	if err := t.failure(); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.deadline = 0
	t.drain()
	return t.disarm()
}

// Close stops the timer and releases its OS timer.
func (t *Timer) Close() error {
	t.mu.Lock()
	// The wait goroutine never arms a stopped timer, so close can release it.
	t.deadline = 0
	t.mu.Unlock()
	return t.close()
}

// expire fires the timer when its deadline has come. An earlier wake arms the OS timer
// for the rest of the span. The wait goroutine calls expire on each OS timer wake.
func (t *Timer) expire() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.deadline == 0 {
		return nil
	}
	if left := t.deadline - Now(); left > 0 {
		return t.arm(left)
	}
	t.deadline = 0
	t.fire()
	return nil
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
