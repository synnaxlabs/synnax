// Copyright 2025 Synnax Labs, Inc.
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
	"io"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for opening the telemetry relay. See its fields for
// more information.
type Config struct {
	// Instrumentation is used for logging, tracing, etc.
	// [OPTIONAL]
	alamos.Instrumentation
	// Transport is the network transport used to move telemetry streams. This is used
	// to both send telemetry streams from the host node to peer nodes, and to stream
	// telemetry from peer nodes to the host node.
	//
	// Remote nodes are one of the three available data sources for the relay. Data
	// for channels whose leaseholder is not the host node will be streamed from remote
	// nodes.
	// [REQUIRED]
	Transport Transport
	// HostResolver is used to retrieve information about the host node.
	// [REQUIRED]
	HostResolver cluster.HostResolver
	// TS is the underlying time-series database engine that serves as one of the three
	// main data sources for the relay.
	//
	// This is the second of the three available data sources for the relay. Data for
	// channels whose leaseholder is the host node will be streamed from the time-series
	// engine's streaming mechanism.
	// [REQUIRED]
	TS *ts.DB
	// FreeWrites is the pipeline for moving data for free virtual channels. Free virtual
	// channels are not leased to any node, and their data is not stored in the cluster
	// and is propagated through the cluster using a separate mechanism. This is mostly
	// used for signaling changes in the cluster meta-data through aspen based key-value
	// gossip.
	// [REQUIRED]
	FreeWrites confluence.Outlet[Response]
	// Channel is used for retrieving channel information from the cluster.
	//
	// [REQUIRED]
	Channel *channel.Service
	// SlowConsumerTimeout sets the maximum amount of time that the relay will wait for
	// a streamer to receive a response before dropping the frame.
	SlowConsumerTimeout time.Duration
	// ResponseBufferSize sets the channel buffer size for the main response streaming
	// pipe. All written frames will be moved through this pipe, so the value should be
	// relatively large.
	// [OPTIONAL: Default is 1000 (equivalent 72 kB of data)]
	ResponseBufferSize int
	// DemandBufferSize sets the channel buffer size for channel demands to the relay.
	// This value should be relatively small.
	// [OPTIONAL: Default is 50]
	DemandBufferSize int
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening a relay. This configuration
	// is not valid on its own and must be overridden with the required fields. See
	// Config for more information.
	DefaultConfig = Config{
		SlowConsumerTimeout: time.Millisecond * 20,
		// 72 B * 1000 = 72 kB
		ResponseBufferSize: 1000,
		DemandBufferSize:   50,
	}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.TS = override.Nil(c.TS, other.TS)
	c.FreeWrites = override.Nil(c.FreeWrites, other.FreeWrites)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.SlowConsumerTimeout = override.Numeric(c.SlowConsumerTimeout, other.SlowConsumerTimeout)
	c.ResponseBufferSize = override.Numeric(c.ResponseBufferSize, other.ResponseBufferSize)
	c.DemandBufferSize = override.Numeric(c.DemandBufferSize, other.DemandBufferSize)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("relay")
	validate.NotNil(v, "transport", c.Transport)
	validate.NotNil(v, "host_provider", c.HostResolver)
	validate.NotNil(v, "ts", c.TS)
	validate.NotNil(v, "free_writers", c.FreeWrites)
	validate.NotNil(v, "channel", c.Channel)
	validate.Positive(v, "slow_consumer_timeout", c.SlowConsumerTimeout)
	validate.Positive(v, "response_buffer_size", c.ResponseBufferSize)
	validate.Positive(v, "demand_buffer_size", c.DemandBufferSize)
	return v.Error()
}

// Relay is the central mechanism for streaming real-time telemetry within the
// distribution layer. It moves
type Relay struct {
	cfg      Config
	ins      alamos.Instrumentation
	delta    *confluence.DynamicDeltaMultiplier[Response]
	demands  confluence.Inlet[demand]
	shutdown io.Closer
}

func Open(configs ...Config) (*Relay, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}

	r := &Relay{cfg: cfg, ins: cfg.Instrumentation}

	tpr := newTapper(cfg)
	demands := confluence.NewStream[demand](cfg.DemandBufferSize)
	demands.SetOutletAddress("peer_demands")
	demands.Acquire(1)
	tpr.InFrom(demands)
	r.demands = demands

	r.delta = confluence.NewDynamicDeltaMultiplier[Response](
		cfg.SlowConsumerTimeout,
		cfg.Instrumentation,
	)
	writes := confluence.NewStream[Response](cfg.ResponseBufferSize)
	writes.SetInletAddress("delta")
	writes.SetOutletAddress("taps")
	r.delta.InFrom(writes)
	tpr.OutTo(writes)

	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	r.shutdown = signal.NewGracefulShutdown(sCtx, cancel)

	r.delta.Flow(
		sCtx,
		confluence.WithAddress("delta"),
		confluence.RecoverWithErrOnPanic(),
		confluence.WithRetryOnPanic(),
	)
	tpr.Flow(
		sCtx,
		confluence.WithAddress("tapper"),
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
		confluence.WithRetryOnPanic(),
	)

	startServer(cfg, r.NewStreamer)

	return r, nil
}

func (r *Relay) Close() error {
	r.demands.Close()
	err := r.shutdown.Close()
	return err
}

func (r *Relay) connectToDelta(buf int) (confluence.Outlet[Response], observe.Disconnect) {
	var (
		data = confluence.NewStream[Response](buf)
		addr = address.Newf("%s_%s", r.ins.Meta.Path, address.Rand().String())
	)
	data.SetInletAddress(addr)
	r.delta.Connect(data)
	return data, func() {
		// NOTE: This area is a source of concurrency bugs. BE CAREFUL. We need to make
		// sure we drain the frames in a SEPARATE goroutine. This prevents deadlocks
		// inside the relay.
		c := make(chan struct{})
		go func() {
			confluence.Drain(data)
			close(c)
		}()
		r.delta.Disconnect(data)
		<-c
	}
}
