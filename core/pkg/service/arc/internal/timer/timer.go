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

// Timer sends on C when the span given to Reset elapses. It never fires early, but a
// receive on C can be a stale fire from an earlier Reset, so the receiver must check
// its own clock. A Timer is not safe for concurrent use.
type Timer struct {
	// C receives a value when the timer fires.
	C <-chan struct{}
	// c is C with send access. It holds at most one pending fire.
	c chan struct{}
	platform
}

// New opens a stopped Timer. The caller must Close it.
func New() (*Timer, error) {
	c := make(chan struct{}, 1)
	t := &Timer{C: c, c: c}
	if err := t.open(); err != nil {
		return nil, err
	}
	return t, nil
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
