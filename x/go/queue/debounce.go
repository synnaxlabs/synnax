// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package queue

import (
	"context"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"time"
)

type DebounceConfig struct {
	// FlushInterval is the time between flushes.
	FlushInterval time.Duration
	// FlushThreshold is the maximum number of values to store in Debounce.
	// Debounce will flush when this threshold is reached, regardless of the FlushInterval.
	FlushThreshold int
}

var _ config.Config[DebounceConfig] = DebounceConfig{}

func (cfg DebounceConfig) Override(other DebounceConfig) DebounceConfig {
	cfg.FlushInterval = override.Numeric(cfg.FlushInterval, other.FlushInterval)
	cfg.FlushThreshold = override.Numeric(cfg.FlushThreshold, other.FlushThreshold)
	return cfg
}

func (cfg DebounceConfig) Validate() error {
	v := validate.New("queue.Debounce")
	validate.Positive(v, "FlushInterval", cfg.FlushInterval)
	validate.NonNegative(v, "FlushThreshold", cfg.FlushThreshold)
	return v.Error()
}

// Debounce is a simple, goroutine safe queue that flushes data to a channel on a timer or queue size threshold.
type Debounce[V confluence.Value] struct {
	Config DebounceConfig
	confluence.LinearTransform[[]V, []V]
}

// Flow starts the queue.
func (d *Debounce[V]) Flow(ctx signal.Context, opts ...confluence.Option) {
	fo := confluence.NewOptions(opts)
	ctx.Go(func(ctx context.Context) error {
		var (
			t = time.NewTicker(d.Config.FlushInterval)
		)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			values, ok := d.fill(t.C)
			if !ok {
				return nil
			}
			if len(values) == 0 {
				continue
			}
			d.Out.Inlet() <- values
		}
	}, fo.Signal...)
}

func (d *Debounce[V]) fill(C <-chan time.Time) ([]V, bool) {
	ops := make([]V, 0, d.Config.FlushThreshold)
	for {
		select {
		case values, ok := <-d.In.Outlet():
			if !ok {
				return ops, false
			}
			ops = append(ops, values...)
			if len(ops) >= d.Config.FlushThreshold {
				return ops, true
			}
		case <-C:
			return ops, true
		}
	}
}
