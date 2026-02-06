// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package debounce provides a trailing-edge debouncer with max delay cap and
// cancellation of in-flight work.
package debounce

import (
	"context"
	"sync"
	"time"
)

// Debouncer coalesces rapid triggers into a single invocation of a callback.
// Each Trigger resets a trailing-edge timer. When the timer fires, the callback
// runs in a new goroutine with a cancellable context. If Trigger or Stop is
// called while the callback is still running, its context is cancelled.
//
// MaxDelay caps the total delay from the first unprocessed trigger, ensuring
// sustained rapid triggers still produce periodic invocations.
type Debouncer struct {
	// Delay is the trailing-edge delay after the last trigger before the
	// callback fires.
	Delay time.Duration
	// MaxDelay caps the total time from the first unprocessed trigger to
	// callback invocation. Zero means no cap.
	MaxDelay time.Duration
	fn       func(ctx context.Context)
	mu       sync.Mutex
	timer    *time.Timer
	cancel   context.CancelFunc
	firstAt  time.Time
}

// New creates a Debouncer that will call fn when the trailing-edge timer fires.
func New(delay, maxDelay time.Duration, fn func(ctx context.Context)) *Debouncer {
	return &Debouncer{Delay: delay, MaxDelay: maxDelay, fn: fn}
}

// Trigger resets the debounce timer. If the timer is already running, it is
// stopped and restarted. If a previous callback is in-flight, its context is
// cancelled.
func (d *Debouncer) Trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cancelLocked()
	if d.timer != nil {
		d.timer.Stop()
	}
	now := time.Now()
	if d.firstAt.IsZero() {
		d.firstAt = now
	}
	delay := d.Delay
	if d.MaxDelay > 0 {
		if maxRemaining := d.MaxDelay - now.Sub(d.firstAt); maxRemaining < delay {
			delay = maxRemaining
		}
	}
	if delay <= 0 {
		d.firstAt = time.Time{}
		d.fireLocked()
		return
	}
	d.timer = time.AfterFunc(delay, func() {
		d.mu.Lock()
		d.firstAt = time.Time{}
		d.fireLocked()
		d.mu.Unlock()
	})
}

// Stop cancels any pending timer and any in-flight callback.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	d.cancelLocked()
	d.firstAt = time.Time{}
}

func (d *Debouncer) cancelLocked() {
	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
}

func (d *Debouncer) fireLocked() {
	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	go d.fn(ctx)
}
