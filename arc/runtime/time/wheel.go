// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package time

import (
	"context"
	"sync"
	"time"

	"github.com/synnaxlabs/x/telem"
)

// Wheel manages all interval-based scheduling with a single ticker.
// This provides predictable, efficient execution of periodic tasks.
type Wheel struct {
	mu          sync.RWMutex
	resolution  time.Duration
	ticker      *time.Ticker
	intervals   map[string]*Interval
	onTick      func(key string)
	startTime   time.Time
	currentTick uint64
	stop        chan struct{}
}

// Interval represents a periodic timer registered with the time wheel.
type Interval struct {
	Key          string
	Period       time.Duration
	InitialDelay time.Duration
	NextTick     uint64 // When this interval should fire next
	TickCount    uint64 // How many times it has fired
	Enabled      bool
}

// NewWheel creates a centralized time wheel scheduler.
// resolution is the base tick rate - should be the GCD of all expected periods.
// onTick is called whenever any interval fires.
func NewWheel(resolution time.Duration, onTick func(string)) *Wheel {
	return &Wheel{
		resolution: resolution,
		intervals:  make(map[string]*Interval),
		onTick:     onTick,
		stop:       make(chan struct{}),
	}
}

// Start begins the time wheel ticker.
func (w *Wheel) Start(ctx context.Context) {
	w.startTime = time.Now()
	w.ticker = time.NewTicker(w.resolution)

	go w.run(ctx)
}

func (w *Wheel) run(ctx context.Context) {
	defer w.ticker.Stop()

	for {
		select {
		case <-w.ticker.C:
			w.tick(ctx)
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		}
	}
}

func (w *Wheel) tick(ctx context.Context) {
	w.mu.Lock()
	w.currentTick++
	currentTick := w.currentTick

	// Collect intervals that should fire this tick
	var toFire []string
	for key, interval := range w.intervals {
		if interval.Enabled && currentTick >= interval.NextTick {
			toFire = append(toFire, key)

			// Schedule next firing
			periodTicks := uint64(interval.Period / w.resolution)
			interval.NextTick = currentTick + periodTicks
			interval.TickCount++
		}
	}
	w.mu.Unlock()

	// Fire callbacks outside the lock
	for _, key := range toFire {
		w.onTick(key)
	}
}

// Register adds a new interval to the time wheel.
func (w *Wheel) Register(key string, period, initialDelay time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()

	delayTicks := uint64(initialDelay / w.resolution)
	periodTicks := uint64(period / w.resolution)

	w.intervals[key] = &Interval{
		Key:          key,
		Period:       period,
		InitialDelay: initialDelay,
		NextTick:     w.currentTick + delayTicks + periodTicks,
		TickCount:    0,
		Enabled:      true,
	}
}

// Unregister removes an interval from the time wheel.
func (w *Wheel) Unregister(key string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.intervals, key)
}

// GetState returns the current state for an interval.
func (w *Wheel) GetState(key string) (tick uint64, timestamp telem.TimeStamp, elapsed telem.TimeSpan, ok bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	interval, exists := w.intervals[key]
	if !exists {
		return 0, 0, 0, false
	}

	now := time.Now()
	return interval.TickCount,
		telem.TimeStamp(now.UnixNano()),
		telem.TimeSpan(now.Sub(w.startTime).Nanoseconds()),
		true
}

// Enable enables an interval.
func (w *Wheel) Enable(key string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if interval, exists := w.intervals[key]; exists {
		interval.Enabled = true
	}
}

// Disable disables an interval without removing it.
func (w *Wheel) Disable(key string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if interval, exists := w.intervals[key]; exists {
		interval.Enabled = false
	}
}

// Stop halts the time wheel.
func (w *Wheel) Stop() {
	close(w.stop)
}
