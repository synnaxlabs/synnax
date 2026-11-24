// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"context"
	"sync"
	"time"

	"github.com/synnaxlabs/x/telem"
	xtime "github.com/synnaxlabs/x/time"
)

// limiter is the one budget of the background reads of an engine. Each owner of such
// reads counts the bytes it reads, so together they read at most rate bytes a second on
// clock. It is safe for concurrent use.
type limiter struct {
	clock xtime.Clock
	rate  float64
	mu    sync.Mutex
	// next is when the reads counted so far fit the rate.
	next time.Time
}

func newLimiter(clock xtime.Clock, rate telem.Size) *limiter {
	return &limiter{clock: clock, rate: float64(rate)}
}

// reserve counts n bytes that the caller read, and returns when the reads counted so
// far fit the rate. The caller waits until then before it reads again.
func (l *limiter) reserve(n int64) time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now := l.clock.Now(); l.next.Before(now) {
		l.next = now
	}
	l.next = l.next.Add(time.Duration(float64(n) / l.rate * float64(time.Second)))
	return l.next
}

// pacer waits on a clock for one owner of background reads: for the limiter after each
// job, or until a set time. It stops waiting once stop closes.
type pacer struct {
	clock xtime.Clock
	limit *limiter
	stop  <-chan struct{}
	fire  func()
	fired chan struct{}
}

func newPacer(clock xtime.Clock, limit *limiter, stop <-chan struct{}) *pacer {
	p := &pacer{clock: clock, limit: limit, stop: stop, fired: make(chan struct{}, 1)}
	p.fire = func() { p.fired <- struct{}{} }
	return p
}

// pace counts n bytes that a job read in the limiter, and waits until the reads fit its
// rate. It returns the error of ctx, or errStopped once stop closes.
func (p *pacer) pace(ctx context.Context, n int64) error {
	return p.wait(ctx, p.limit.reserve(n))
}

// wait waits until at on the clock. It returns the error of ctx, or errStopped once
// stop closes.
func (p *pacer) wait(ctx context.Context, at time.Time) error {
	if !at.After(p.clock.Now()) {
		return nil
	}
	t := p.clock.RunAt(at, p.fire)
	var err error
	select {
	case <-p.fired:
		return nil
	case <-ctx.Done():
		err = ctx.Err()
	case <-p.stop:
		err = errStopped
	}
	if !t.Stop() {
		<-p.fired
	}
	return err
}
