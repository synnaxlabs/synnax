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
	"math"
	"time"

	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

const InfiniteRetries = math.MaxInt

type Config struct {
	// BaseInterval is the interval of time waited on the first time Wait is called on
	// the breaker. This interval keeps growing at an exponential rate set by Scale.
	// Default: 1s.
	BaseInterval time.Duration
	// Scale is the multiplicative rate by which the timeout interval grows with each
	// call to Wait. For example, if set at 2, the second call to Wait will wait 2x
	// longer than the first, the third will wait 4x, etc.
	// Default: 1.
	Scale float32
	// MaxRetries is the number set for how many calls to Wait is allowed. Once a breaker
	// goes beyond this number, it no can no longer Wait and returns false.
	// Default: 0.
	MaxRetries int
}

func (c Config) Override(o Config) Config {
	c.BaseInterval = override.Numeric(c.BaseInterval, o.BaseInterval)
	c.MaxRetries = override.Numeric(c.MaxRetries, o.MaxRetries)
	c.Scale = override.Numeric(c.Scale, o.Scale)
	return c
}

func (c Config) Validate() error {
	v := validate.New("breaker")
	validate.GreaterThanEq(v, "base_interval", c.BaseInterval, 0)
	validate.GreaterThanEq(v, "max_retries", c.MaxRetries, 0)
	validate.GreaterThanEq(v, "scale", c.Scale, 1)
	return v.Error()
}

var (
	_             config.Config[Config] = Config{}
	defaultConfig                       = Config{Scale: 1}
)

type Breaker struct {
	ctx context.Context
	Config
	currInterval time.Duration
	retryCount   int
}

// NewBreaker creates a new breaker on the given context and configuration. If the context
// is canceled while the breaker is waiting, the breaker stops waiting immediately.
func NewBreaker(ctx context.Context, configs ...Config) (Breaker, error) {
	cfg, err := config.New(defaultConfig, configs...)
	if err != nil {
		return Breaker{}, err
	}
	b := Breaker{Config: cfg, ctx: ctx}
	b.Reset()
	return b, nil
}

// Wait returns a boolean indicating whether the breaker can wait again. A breaker
// cannot wait if its context is canceled or if it reached its maximum retry count.
// Wait waits an exponentially increasing amount of time each time it is called.
func (b *Breaker) Wait() bool {
	if b.MaxRetries != InfiniteRetries && b.retryCount == b.MaxRetries {
		return false
	}

	ch := time.After(b.currInterval)
	select {
	case <-ch:
	case <-b.ctx.Done():
		return false
	}
	b.currInterval = time.Duration(float32(b.currInterval) * b.Scale)
	b.retryCount++
	return true
}

// Reset resets the breaker to the base interval and to have 0 retries.
func (b *Breaker) Reset() {
	b.currInterval = b.BaseInterval
	b.retryCount = 0
}
