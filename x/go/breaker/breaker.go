// Copyright 2024 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"time"
)

type Config struct {
	BaseInterval time.Duration
	Scale        float32
	MaxRetries   int
}

func (c Config) Override(o Config) Config {
	c.BaseInterval = override.Numeric(c.BaseInterval, o.BaseInterval)
	c.MaxRetries = override.Numeric(c.MaxRetries, o.MaxRetries)
	c.Scale = override.Numeric(c.Scale, o.Scale)
	return c
}

func (c Config) Validate() error {
	v := validate.New("breaker")
	validate.Positive(v, "BaseInterval", c.BaseInterval)
	validate.Positive(v, "MaxRetries", c.MaxRetries)
	validate.GreaterThanEq(v, "Scale", c.Scale, 1)
	return v.Error()
}

var (
	_             config.Config[Config] = Config{}
	defaultConfig                       = Config{
		BaseInterval: 1 * telem.Millisecond.Duration(),
		MaxRetries:   5,
		Scale:        2,
	}
)

type Breaker struct {
	Config
	ctx          context.Context
	currInterval time.Duration
	retryCount   int
}

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
	if b.retryCount == b.MaxRetries {
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

func (b *Breaker) Reset() {
	b.currInterval = b.BaseInterval
	b.retryCount = 0
}
