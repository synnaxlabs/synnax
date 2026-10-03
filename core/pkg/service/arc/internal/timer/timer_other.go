// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !linux

package timer

import "time"

// platform wakes on a time.Timer.
type platform struct {
	// coarse fires the timer after the span given to Reset.
	coarse *time.Timer
}

func (t *Timer) open() error { return nil }

// Reset stops the timer and starts it again, so it fires after d. A d of zero or less
// fires right away.
func (t *Timer) Reset(d time.Duration) error {
	if err := t.Stop(); err != nil {
		return err
	}
	t.coarse = time.AfterFunc(d, t.fire)
	return nil
}

// Stop stops the timer and drops a pending fire.
func (t *Timer) Stop() error {
	if t.coarse != nil {
		t.coarse.Stop()
	}
	t.drain()
	return nil
}

// Close stops the timer.
func (t *Timer) Close() error { return t.Stop() }
