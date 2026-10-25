// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package history publishes the history changes of channels stored on a Core to the
// sy_history_change channel.
package history

import (
	"cmp"
	"context"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	xtime "github.com/synnaxlabs/x/time"
	"github.com/synnaxlabs/x/validate"
)

const channelName = "sy_history_change"

// Config configures the history change publisher.
type Config struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// Framer reports the history changes to publish.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Signals opens the publisher that writes to the sy_history_change channel.
	//
	// [REQUIRED]
	Signals *signals.Provider
	// FlushInterval is how long changes collect before they publish. Changes to one
	// channel within the interval merge into one change that spans them all.
	//
	// [OPTIONAL] - Defaults to 100ms.
	FlushInterval telem.TimeSpan
	// Clock times the flush interval.
	//
	// [OPTIONAL] - Defaults to xtime.Real.
	Clock xtime.Clock
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for Publish.
	DefaultConfig = Config{FlushInterval: 100 * telem.Millisecond, Clock: xtime.Real}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("framer.history")
	v.NotNil("framer", c.Framer)
	v.NotNil("signals", c.Signals)
	v.Positive("flush_interval", c.FlushInterval)
	v.NotNil("clock", c.Clock)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.FlushInterval = override.Numeric(c.FlushInterval, other.FlushInterval)
	c.Clock = override.Nil(c.Clock, other.Clock)
	return c
}

// Publish writes the history changes of channels stored on this Core to the
// sy_history_change channel. Each sample is a JSON list of framer.HistoryChange with at
// most one change per channel. The returned io.Closer stops publishing.
func Publish(ctx context.Context, cfgs ...Config) (io.Closer, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	c := &coalescer{
		Observer: observe.New[[]framer.HistoryChange](),
		wake:     make(chan struct{}, 1),
		interval: cfg.FlushInterval.Duration(),
		clock:    cfg.Clock,
	}
	c.mu.changes = make(map[channel.Key]telem.TimeRange)
	publisher, err := cfg.Signals.PublishJSON(
		ctx,
		signals.JSONPublisherConfig[[]framer.HistoryChange]{
			Observable: c,
			SetName:    channelName,
		},
	)
	if err != nil {
		return nil, err
	}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	sCtx.Go(c.run, signal.WithKey("flush"))
	disconnect := cfg.Framer.OnHistoryChange(c.add)
	return xio.MultiCloser{
		publisher,
		signal.NewHardShutdown(sCtx, cancel),
		xio.NoFailCloserFunc(disconnect),
	}, nil
}

// coalescer merges history changes per channel. add runs under storage locks, so it
// only records the change and wakes run, which publishes after the flush interval.
type coalescer struct {
	observe.Observer[[]framer.HistoryChange]
	wake     chan struct{}
	interval time.Duration
	clock    xtime.Clock
	mu       struct {
		changes map[channel.Key]telem.TimeRange
		sync.Mutex
	}
}

func (c *coalescer) add(_ context.Context, change framer.HistoryChange) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tr, ok := c.mu.changes[change.Key]
	if ok {
		tr = tr.Union(change.TimeRange)
	} else {
		tr = change.TimeRange
	}
	c.mu.changes[change.Key] = tr
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

func (c *coalescer) run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.wake:
		}
		due := make(chan struct{})
		timer := c.clock.RunAt(c.clock.Now().Add(c.interval), func() { close(due) })
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-due:
		}
		if changes := c.drain(); len(changes) > 0 {
			c.Notify(ctx, changes)
		}
	}
}

func (c *coalescer) drain() []framer.HistoryChange {
	c.mu.Lock()
	defer c.mu.Unlock()
	changes := make([]framer.HistoryChange, 0, len(c.mu.changes))
	for key, tr := range c.mu.changes {
		changes = append(changes, framer.HistoryChange{Key: key, TimeRange: tr})
	}
	clear(c.mu.changes)
	// Every pending wake came from a change drained here.
	select {
	case <-c.wake:
	default:
	}
	slices.SortFunc(changes, func(a, b framer.HistoryChange) int {
		return cmp.Compare(a.Key, b.Key)
	})
	return changes
}
