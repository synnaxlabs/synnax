// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package breaker

import (
	"context"
	"time"

	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	xtime "github.com/synnaxlabs/x/time"
	"github.com/synnaxlabs/x/validate"
)

// InfiniteRetries lifts the retry limit: a breaker configured with it keeps waiting for
// as long as its context is live.
const InfiniteRetries = -1

// Config configures a Breaker.
type Config struct {
	// BaseInterval is the interval of time waited on the first time Wait is called on
	// the breaker. This interval keeps growing at an exponential rate set by Scale.
	//
	// Default: 1s.
	BaseInterval time.Duration
	// Scale is the multiplicative rate by which the timeout interval grows with each
	// call to Wait. For example, if set at 2, the second call to Wait will wait 2x
	// longer than the first, the third will wait 4x, etc.
	//
	// Default: 1.
	Scale float64
	// MaxRetries is how many calls to Wait are allowed. Once a breaker reaches this
	// number, Wait no longer waits and returns false. Set it to InfiniteRetries to
	// lift the limit.
	//
	// Default: 0.
	MaxRetries int
	// Clock is the time source Wait blocks against.
	//
	// Default: xtime.Real.
	Clock xtime.Clock
}

// Override returns c with any non-zero fields of o applied on top.
func (c Config) Override(o Config) Config {
	c.BaseInterval = override.Numeric(c.BaseInterval, o.BaseInterval)
	c.MaxRetries = override.Numeric(c.MaxRetries, o.MaxRetries)
	c.Scale = override.Numeric(c.Scale, o.Scale)
	c.Clock = override.Nil(c.Clock, o.Clock)
	return c
}

// Validate returns an error if c contains invalid values.
func (c Config) Validate() error {
	v := validate.New("breaker")
	v.GreaterThanEq("base_interval", c.BaseInterval, 0)
	v.GreaterThanEq("max_retries", c.MaxRetries, InfiniteRetries)
	v.GreaterThanEq("scale", c.Scale, 1)
	v.NotNil("clock", c.Clock)
	return v.Error()
}

var _ config.Config[Config] = Config{}

// Breaker waits an exponentially growing interval between retries, giving up once it
// reaches MaxRetries or its context is canceled. A Breaker is not safe for concurrent
// use.
type Breaker struct {
	ctx          context.Context
	cfg          Config
	currInterval time.Duration
	retryCount   int
}

// NewBreaker creates a new breaker on the given context and configuration. If the
// context is canceled while the breaker is waiting, the breaker stops waiting
// immediately.
func NewBreaker(ctx context.Context, configs ...Config) (Breaker, error) {
	cfg, err := config.New(Config{Scale: 1, Clock: xtime.Real}, configs...)
	if err != nil {
		return Breaker{}, err
	}
	b := Breaker{cfg: cfg, ctx: ctx}
	b.Reset()
	return b, nil
}

// Wait returns a boolean indicating whether the breaker can wait again. A breaker
// cannot wait if its context is canceled or if it reached its maximum retry count. Wait
// waits an exponentially increasing amount of time each time it is called.
func (b *Breaker) Wait() bool {
	if b.cfg.MaxRetries != InfiniteRetries && b.retryCount == b.cfg.MaxRetries {
		return false
	}

	fired := make(chan struct{})
	deadline := b.cfg.Clock.Now().Add(b.currInterval)
	timer := b.cfg.Clock.RunAt(deadline, func() { close(fired) })
	select {
	case <-fired:
	case <-b.ctx.Done():
		timer.Stop()
		return false
	}
	b.currInterval = time.Duration(float64(b.currInterval) * b.cfg.Scale)
	b.retryCount++
	return true
}

// Reset resets the breaker to the base interval and to have 0 retries.
func (b *Breaker) Reset() { b.currInterval = b.cfg.BaseInterval; b.retryCount = 0 }
