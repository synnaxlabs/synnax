// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package restart implements the restart policy for a supervised process. It decides,
// after a process exits, whether to leave it stopped, restart it with backoff, or give
// up because the process is crash-looping.
package restart

import (
	"context"
	"time"

	"github.com/synnaxlabs/x/breaker"
)

// Action is the decision a Policy returns after a supervised process exits.
type Action int

const (
	// Stop indicates the process should not be restarted because its exit was expected
	// (the supervisor initiated the shutdown).
	Stop Action = iota
	// Restart indicates the process should be restarted. The backoff interval has
	// already elapsed by the time Decide returns Restart.
	Restart
	// GiveUp indicates the process has exited unexpectedly too many times in quick
	// succession (or the Policy's context was canceled) and should not be restarted
	// again.
	GiveUp
)

// DefaultHealthyUptime is used when Config.HealthyUptime is not set.
const DefaultHealthyUptime = time.Minute

// Config configures a Policy.
type Config struct {
	// BaseInterval is the backoff before the first restart after an unexpected exit. It
	// grows by Scale on each consecutive unexpected exit.
	BaseInterval time.Duration
	// Scale is the multiplicative growth of the backoff per consecutive unexpected
	// exit. Must be >= 1.
	Scale float32
	// MaxRetries caps the number of consecutive restarts before Decide returns GiveUp.
	// The counter resets after a process runs longer than HealthyUptime.
	MaxRetries int
	// HealthyUptime is the minimum run time after which an exit is treated as a healthy
	// run rather than a crash-loop iteration: an exit after at least this much uptime
	// resets the consecutive-failure counter, so a single failure following a long
	// healthy run does not count toward MaxRetries. Defaults to DefaultHealthyUptime.
	HealthyUptime time.Duration
}

// Policy decides whether a supervised process should be restarted after it exits. It is
// not safe for concurrent use; a single supervisor goroutine should own it across the
// lifetime of the process it manages.
type Policy struct {
	breaker       breaker.Breaker
	healthyUptime time.Duration
}

// New constructs a Policy. The provided context bounds the backoff waits in Decide: if
// it is canceled while Decide is waiting, Decide stops waiting and returns GiveUp.
func New(ctx context.Context, cfg Config) (*Policy, error) {
	b, err := breaker.NewBreaker(ctx, breaker.Config{
		BaseInterval: cfg.BaseInterval,
		Scale:        cfg.Scale,
		MaxRetries:   cfg.MaxRetries,
	})
	if err != nil {
		return nil, err
	}
	healthy := cfg.HealthyUptime
	if healthy <= 0 {
		healthy = DefaultHealthyUptime
	}
	return &Policy{breaker: b, healthyUptime: healthy}, nil
}

// Decide reports what the supervisor should do after the managed process exits.
// expected is true when the supervisor initiated the shutdown (a graceful stop), in
// which case Decide returns Stop immediately. Otherwise the exit is treated as a
// failure: if uptime is at least HealthyUptime the failure counter is reset first, then
// Decide waits out the backoff and returns Restart, or returns GiveUp once MaxRetries
// consecutive failures are reached (or the context is canceled).
func (p *Policy) Decide(expected bool, uptime time.Duration) Action {
	if expected {
		return Stop
	}
	if uptime >= p.healthyUptime {
		p.breaker.Reset()
	}
	if p.breaker.Wait() {
		return Restart
	}
	return GiveUp
}
