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
