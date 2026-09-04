// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package debounce

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	xtime "github.com/synnaxlabs/x/time"
	"github.com/synnaxlabs/x/validate"
)

// KeyedConfig configures a Keyed debouncer.
type KeyedConfig[K comparable] struct {
	// Delay is the trailing-edge delay after a key's last trigger before its callback
	// fires.
	//
	// [REQUIRED]
	Delay time.Duration
	// MaxDelay caps the total time from a key's first unprocessed trigger to callback
	// invocation.
	//
	// [OPTIONAL] - defaults to 0.
	MaxDelay time.Duration
	// Clock is the time source used both for MaxDelay tracking and for scheduling the
	// shared timer.
	//
	// [OPTIONAL] - defaults to xtime.Real.
	Clock xtime.Clock
	// Callback is invoked once per fired key on the Keyed's single worker goroutine,
	// so a slow callback delays fires for every other key. Its context is cancelled
	// when the Keyed closes.
	//
	// [REQUIRED]
	Callback func(context.Context, K)
}

var _ config.Config[KeyedConfig[int]] = KeyedConfig[int]{}

// Override returns c with any non-zero fields of other applied on top.
func (c KeyedConfig[K]) Override(other KeyedConfig[K]) KeyedConfig[K] {
	c.Delay = override.Numeric(c.Delay, other.Delay)
	c.MaxDelay = override.Numeric(c.MaxDelay, other.MaxDelay)
	c.Clock = override.Nil(c.Clock, other.Clock)
	c.Callback = override.Nil(c.Callback, other.Callback)
	return c
}

// Validate returns an error if c contains invalid values.
func (c KeyedConfig[K]) Validate() error {
	v := validate.New("debounce.keyed")
	validate.GreaterThan(v, "delay", c.Delay, 0)
	validate.GreaterThanEq(v, "max_delay", c.MaxDelay, 0)
	validate.NotNil(v, "clock", c.Clock)
	validate.NotNil(v, "callback", c.Callback)
	return v.Error()
}

// Keyed coalesces rapid per-key triggers into a single callback invocation per key.
// All keys share one worker goroutine and one timer, so triggering allocates nothing
// beyond the key's pending entry. Unlike Debouncer, a trigger does not cancel an
// in-flight callback for the same key; it schedules another fire after it.
type Keyed[K comparable] struct {
	cfg      KeyedConfig[K]
	shutdown io.Closer
	wake     chan struct{}
	// mu guards closed, timer, and pending.
	mu      sync.Mutex
	closed  bool
	timer   xtime.Timer
	pending map[K]window
}

// window is a key's pending fire. fireAt slides forward with each trigger; capAt
// bounds it, measured from the key's first unprocessed trigger.
type window struct {
	fireAt time.Time
	capAt  time.Time
}

// NewKeyed creates a Keyed from the merged set of configs and starts its worker.
// It returns an error if the resulting KeyedConfig fails validation. Close releases
// the worker.
func NewKeyed[K comparable](configs ...KeyedConfig[K]) (*Keyed[K], error) {
	cfg, err := config.New(KeyedConfig[K]{Clock: xtime.Real}, configs...)
	if err != nil {
		return nil, err
	}
	sCtx, cancel := signal.Isolated()
	k := &Keyed[K]{
		cfg:      cfg,
		shutdown: signal.NewHardShutdown(sCtx, cancel),
		wake:     make(chan struct{}, 1),
		pending:  make(map[K]window),
	}
	sCtx.Go(func(ctx context.Context) error {
		k.run(ctx)
		return nil
	}, signal.WithKey("debounce.keyed"))
	return k, nil
}

// Trigger schedules key's callback after Delay, superseding key's pending fire. A
// no-op after Close.
func (k *Keyed[K]) Trigger(key K) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.closed {
		return
	}
	now := k.cfg.Clock.Now()
	w, ok := k.pending[key]
	if !ok && k.cfg.MaxDelay > 0 {
		w.capAt = now.Add(k.cfg.MaxDelay)
	}
	w.fireAt = now.Add(k.cfg.Delay)
	if !w.capAt.IsZero() && w.fireAt.After(w.capAt) {
		w.fireAt = w.capAt
	}
	k.pending[key] = w
	k.poke()
}

// Forget discards key's pending fire. It does not interrupt an in-flight callback.
func (k *Keyed[K]) Forget(key K) {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.pending, key)
}

// Close discards pending fires, cancels the context of an in-flight callback, and
// waits for the worker to return. Triggers after Close are no-ops.
func (k *Keyed[K]) Close() error {
	k.mu.Lock()
	k.closed = true
	clear(k.pending)
	k.stopTimerLocked()
	k.mu.Unlock()
	return k.shutdown.Close()
}

// poke wakes the worker without blocking; a full wake channel already guarantees a
// pending wakeup.
func (k *Keyed[K]) poke() {
	select {
	case k.wake <- struct{}{}:
	default:
	}
}

func (k *Keyed[K]) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-k.wake:
		}
		for {
			due := k.collect()
			if len(due) == 0 {
				break
			}
			for _, key := range due {
				if ctx.Err() != nil {
					return
				}
				k.cfg.Callback(ctx, key)
			}
		}
	}
}

// collect pops the keys whose deadlines have passed and arms the timer for the
// earliest remaining one.
func (k *Keyed[K]) collect() []K {
	k.mu.Lock()
	defer k.mu.Unlock()
	now := k.cfg.Clock.Now()
	var (
		due  []K
		next time.Time
	)
	for key, w := range k.pending {
		if !w.fireAt.After(now) {
			due = append(due, key)
			delete(k.pending, key)
		} else if next.IsZero() || w.fireAt.Before(next) {
			next = w.fireAt
		}
	}
	k.stopTimerLocked()
	if !next.IsZero() {
		k.timer = k.cfg.Clock.RunAt(next, k.poke)
	}
	return due
}

func (k *Keyed[K]) stopTimerLocked() {
	if k.timer != nil {
		k.timer.Stop()
		k.timer = nil
	}
}
