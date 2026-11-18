// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package relay implements the central distributing layer mechanism for streaming
// telemetry throughout a Synnax deployment. The documentation in this header will
// focus on implementation
package relay

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for opening the telemetry relay. See its fields for more
// information.
type Config struct {
	// TS is the underlying time-series database engine. The relay streams stored
	// channels from it.
	//
	// [REQUIRED]
	TS *ts.DB
	// FreeWrites is the pipeline for moving data for free virtual channels. Free
	// virtual channels are not leased to any node and their data is not stored; their
	// writes are propagated straight through the relay. Most free channels carry
	// metadata change notices.
	//
	// [REQUIRED]
	FreeWrites confluence.Outlet[Response]
	// Instrumentation is used for logging, tracing, etc.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// StreamerBufferSize sets the channel buffer size, in frames, between the relay and
	// each streamer. The relay drops a frame for a streamer whose buffer is full, so a
	// slow streamer never delays the relay or other streamers.
	//
	// [OPTIONAL] - Defaults to 100.
	StreamerBufferSize int
	// ResponseBufferSize sets the channel buffer size for the main response streaming
	// pipe. All written frames will be moved through this pipe, so the value should be
	// relatively large.
	//
	// [OPTIONAL] - Defaults to 1000 (equivalent 72 kB of data).
	ResponseBufferSize int
	// DemandBufferSize sets the channel buffer size for channel demands to the relay.
	// This value should be relatively small.
	//
	// [OPTIONAL] - Defaults to 50.
	DemandBufferSize int
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening a relay. This
	// configuration is not valid on its own and must be overridden with the required
	// fields. See Config for more information.
	DefaultConfig = Config{
		StreamerBufferSize: 100,
		// 72 B * 1000 = 72 kB
		ResponseBufferSize: 1000,
		DemandBufferSize:   50,
	}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.TS = override.Nil(c.TS, other.TS)
	c.FreeWrites = override.Nil(c.FreeWrites, other.FreeWrites)
	c.StreamerBufferSize = override.Numeric(
		c.StreamerBufferSize,
		other.StreamerBufferSize,
	)
	c.ResponseBufferSize = override.Numeric(
		c.ResponseBufferSize,
		other.ResponseBufferSize,
	)
	c.DemandBufferSize = override.Numeric(c.DemandBufferSize, other.DemandBufferSize)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("relay")
	v.NotNil("ts", c.TS)
	v.NotNil("free_writes", c.FreeWrites)
	v.Positive("streamer_buffer_size", c.StreamerBufferSize)
	v.Positive("response_buffer_size", c.ResponseBufferSize)
	v.Positive("demand_buffer_size", c.DemandBufferSize)
	return v.Error()
}

// Relay is the central mechanism for streaming real-time telemetry within the
// distribution layer. It moves frames from writers to the streamers that demand them
// while Run runs.
type Relay struct {
	ins     alamos.Instrumentation
	demands confluence.Inlet[demand]
	tapper  confluence.Segment[demand, Response]
	delta   *delta
	// stopped closes when Run returns.
	stopped chan struct{}
	ran     atomic.Bool
	cfg     Config
}

// Open opens a Relay. It starts no goroutines: streamers wait for Run to start.
func Open(configs ...Config) (*Relay, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	r := &Relay{
		cfg:     cfg,
		ins:     cfg.Instrumentation,
		tapper:  newTapper(cfg),
		delta:   newDelta(cfg.Instrumentation),
		stopped: make(chan struct{}),
	}
	demands := confluence.NewStream[demand](cfg.DemandBufferSize)
	demands.SetOutletAddress("demands")
	r.tapper.InFrom(demands)
	r.demands = demands
	writes := confluence.NewStream[Response](cfg.ResponseBufferSize)
	writes.SetInletAddress("delta")
	writes.SetOutletAddress("taps")
	r.delta.InFrom(writes)
	r.tapper.OutTo(writes)
	return r, nil
}

// Run moves frames to streamers until ctx ends. It then stops the taps into storage
// and free writes, drops every streamer, and returns. A streamer that runs after Run
// returns fails. A second Run returns an error.
func (r *Relay) Run(ctx context.Context) error {
	if !r.ran.CompareAndSwap(false, true) {
		return errors.New("run called twice")
	}
	defer close(r.stopped)
	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(r.ins))
	defer cancel()
	r.delta.Flow(
		sCtx,
		confluence.WithAddress("delta"),
		confluence.RecoverWithErrOnPanic(),
		confluence.WithRetryOnPanic(),
	)
	r.tapper.Flow(
		sCtx,
		confluence.WithAddress("tapper"),
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
		confluence.WithRetryOnPanic(),
	)
	return errors.Skip(sCtx.Wait(), context.Canceled)
}

var (
	errStopped = errors.New("relay stopped")
	errDropped = errors.New("relay dropped the streamer")
)

// sendDemand sends d to the tapper. It fails when ctx ends or Run has returned first.
func (r *Relay) sendDemand(ctx context.Context, d demand) error {
	select {
	case r.demands.Inlet() <- d:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-r.stopped:
		return errStopped
	}
}

// removeDemand tells the tapper that the entity at addr demands no channels. It
// returns once the tapper takes the demand, or once Run has returned.
func (r *Relay) removeDemand(addr address.Address) {
	select {
	case r.demands.Inlet() <- demand{addr: addr, removed: true}:
	case <-r.stopped:
	}
}

// connectToDelta connects a subscriber to the delta and returns its responses, which
// close when the delta drops the subscriber. It fails when ctx ends or Run has
// returned first. Call the returned disconnect only before the responses close.
func (r *Relay) connectToDelta(
	ctx context.Context,
	accepts func(Response) bool,
) (<-chan Response, observe.Disconnect, error) {
	s := &subscriber{
		responses: make(chan Response, r.cfg.StreamerBufferSize),
		accepts:   accepts,
		addr:      address.Newf("%s_%s", r.ins.Meta.Path, address.Rand().String()),
	}
	select {
	case r.delta.connections <- s:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-r.stopped:
		return nil, nil, errStopped
	}
	return s.responses, func() {
		// NOTE: This area is a source of concurrency bugs. BE CAREFUL. We need to make
		// sure we drain the frames in a SEPARATE goroutine. This prevents deadlocks
		// inside the relay.
		var wg sync.WaitGroup
		wg.Go(func() {
			for range s.responses {
			}
		})
		select {
		case r.delta.disconnections <- s:
		case <-r.stopped:
		}
		wg.Wait()
	}, nil
}
