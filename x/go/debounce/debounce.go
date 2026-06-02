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

	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	xtime "github.com/synnaxlabs/x/time"
	"github.com/synnaxlabs/x/validate"
)

// Config configures a Debouncer.
type Config struct {
	// Delay is the trailing-edge delay after the last trigger before the callback
	// fires.
	//
	// [REQUIRED]
	Delay time.Duration
	// MaxDelay caps the total time from the first unprocessed trigger to callback
	// invocation.
	//
	// [OPTIONAL] - defaults to 0.
	MaxDelay time.Duration
	// Clock is the time source used both for MaxDelay tracking and for scheduling the
	// trailing-edge timer.
	//
	// [OPTIONAL] - defaults to xtime.Real.
	Clock xtime.Clock
	// Callback is invoked in a new goroutine each time the debouncer fires. Its context
	// is cancelled if Trigger or Stop is called while it is still running.
	//
	// [REQUIRED]
	Callback func(context.Context)
}

var _ config.Config[Config] = Config{}

// Override returns c with any non-zero fields of other applied on top.
func (c Config) Override(other Config) Config {
	c.Delay = override.Numeric(c.Delay, other.Delay)
	c.MaxDelay = override.Numeric(c.MaxDelay, other.MaxDelay)
	c.Clock = override.Nil(c.Clock, other.Clock)
	c.Callback = override.Nil(c.Callback, other.Callback)
	return c
}

// Validate returns an error if c contains invalid values.
func (c Config) Validate() error {
	v := validate.New("debounce")
	validate.GreaterThan(v, "delay", c.Delay, 0)
	validate.GreaterThanEq(v, "max_delay", c.MaxDelay, 0)
	validate.NotNil(v, "clock", c.Clock)
	validate.NotNil(v, "callback", c.Callback)
	return v.Error()
}

// Debouncer coalesces rapid triggers into a single invocation of a callback. Each
// Trigger resets a trailing-edge timer. When the timer fires, the callback runs in a
// new goroutine with a cancellable context. If Trigger or Stop is called while the
// callback is still running, its context is cancelled.
//
// MaxDelay caps the total delay from the first unprocessed trigger, ensuring sustained
// rapid triggers still produce periodic invocations.
type Debouncer struct {
	cfg        Config
	mu         sync.Mutex
	timer      xtime.Timer
	timerGen   uint64
	cancel     context.CancelFunc
	firstAt    time.Time
	firstAtSet bool
}

// New creates a Debouncer from the merged set of configs. It returns an error if the
// resulting Config fails validation.
func New(configs ...Config) (*Debouncer, error) {
	cfg, err := config.New(Config{Clock: xtime.Real}, configs...)
	if err != nil {
		return nil, err
	}
	return &Debouncer{cfg: cfg}, nil
}

// Trigger resets the debounce timer. If the timer is already running, the pending fire
// is superseded. If a previous callback is in-flight, its context is cancelled.
func (d *Debouncer) Trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cancelLocked()
	d.stopTimerLocked()
	d.timerGen++
	gen := d.timerGen
	now := d.cfg.Clock.Now()
	if !d.firstAtSet {
		d.firstAt = now
		d.firstAtSet = true
	}
	delay, maxDelay := d.cfg.Delay, d.cfg.MaxDelay
	if maxDelay > 0 {
		if maxRemaining := maxDelay - now.Sub(d.firstAt); maxRemaining < delay {
			delay = maxRemaining
		}
	}
	if delay <= 0 {
		d.firstAtSet = false
		d.fireLocked()
		return
	}
	d.timer = d.cfg.Clock.AfterFunc(delay, func() { d.onTimerFire(gen) })
}

// Stop supersedes any pending timer and cancels any in-flight callback.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopTimerLocked()
	d.timerGen++
	d.cancelLocked()
	d.firstAtSet = false
}

// onTimerFire is invoked by Clock.AfterFunc when the timer elapses. The gen check
// guards against a timer that fired between a concurrent Trigger/Stop calling
// timer.Stop (which returned false) and the firing goroutine acquiring d.mu.
func (d *Debouncer) onTimerFire(gen uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timerGen != gen {
		return
	}
	d.firstAtSet = false
	d.fireLocked()
}

func (d *Debouncer) stopTimerLocked() {
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
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
	go d.cfg.Callback(ctx)
}
