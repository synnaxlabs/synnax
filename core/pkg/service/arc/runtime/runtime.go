// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime

import (
	"context"
	"io"
	"sync"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/std"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/value"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/unsafe"
	"github.com/synnaxlabs/x/validate"
	"github.com/tetratelabs/wazero"
)

// Config is the configuration for an arc runtime.
type Config struct {
	// Module is the compiled arc module that needs to be executed.
	//
	// [REQUIRED]
	Module arc.Module
	// Channel is used for retrieving channel information from the cluster.
	//
	// [REQUIRED]
	Channel channel.Readable
	// Framer is used for reading from and writing telemetry to the cluster.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Status is used for updating statuses.
	//
	// [REQUIRED]
	Status *status.Service
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening a runtime. This
	// configuration is not valid on its own. Fields must be set according to the
	// Config documentation.
	DefaultConfig = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Module = override.Zero(c.Module, other.Module)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("arc.runtime")
	validate.NotNil(v, "module", c.Module)
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "status", c.Status)
	return v.Error()
}

type Runtime struct {
	// module is the arc program module
	module arc.Module
	// wasm is the webassembly runtime for the program.
	wasm wazero.Runtime
	confluence.UnarySink[framer.StreamerResponse]
	// requests are used to manage the life cycle of the telemetry frame streamer.
	requests confluence.Inlet[framer.StreamerRequest]
	// values are the current telemetry values for each requested channel in the program.
	mu struct {
		sync.RWMutex
		values map[channel.Key]telem.Series
	}
	// nodes are all nodes in the program
	nodes map[string]stage.Stage
	// close is a shutdown handler that stops internal processes
	close io.Closer
}

func (r *Runtime) Close() error {
	r.requests.Close()
	defer confluence.Drain(r.In)
	return r.close.Close()
}

func (r *Runtime) Get(key channel.Key) telem.Series {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mu.values[key]
}

func (r *Runtime) createOnOutput(nodeKey string) stage.OutputHandler {
	return func(ctx context.Context, sourceParam string, value value.Value) {
		for _, edge := range r.module.Edges {
			if edge.Source.Node == nodeKey && edge.Source.Param == sourceParam {
				r.nodes[edge.Target.Node].Next(ctx, edge.Target.Param, value)
			}
		}
	}
}

func (r *Runtime) processOutput(ctx context.Context, res framer.StreamerResponse) error {
	for i, ser := range res.Frame.RawSeries() {
		if res.Frame.ShouldExcludeRaw(i) {
			continue
		}
		r.mu.values[res.Frame.RawKeyAt(i)] = ser
	}
	keys := res.Frame.KeysSlice()
	for _, node := range r.nodes {
		if len(lo.Intersect(node.ReadChannels(), keys)) > 0 {
			node.Next(ctx, "", value.Value{})
		}
	}
	return nil
}

func retrieveChannels(
	ctx context.Context,
	channelSvc channel.Readable,
	nodes []arc.Node,
) ([]channel.Channel, error) {
	keys := make(set.Set[channel.Key])
	for _, node := range nodes {
		keys.Add(unsafe.ReinterpretSlice[uint32, channel.Key](node.Channels.Read.Keys())...)
	}
	channels := make([]channel.Channel, 0, len(keys))
	if err := channelSvc.NewRetrieve().
		WhereKeys(keys.Keys()...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	return channels, nil
}

var (
	streamerAddr address.Address = "streamer"
	runtimeAddr  address.Address = "runtime"
)

func createStreamPipeline(
	ctx context.Context,
	r *Runtime,
	frameSvc *framer.Service,
	readChannelKeys []channel.Key,
) (confluence.Flow, confluence.Inlet[framer.StreamerRequest], error) {
	p := plumber.New()
	streamer, err := frameSvc.NewStreamer(
		ctx,
		framer.StreamerConfig{Keys: readChannelKeys},
	)
	if err != nil {
		return nil, nil, err
	}
	plumber.SetSegment(p, streamerAddr, streamer)
	r.Sink = r.processOutput
	plumber.SetSink[framer.StreamerResponse](p, "runtime", r)
	streamer.InFrom(confluence.NewStream[framer.StreamerRequest]())
	plumber.MustConnect[framer.StreamerResponse](p, streamerAddr, runtimeAddr, 10)
	requests := confluence.NewStream[framer.StreamerRequest]()
	streamer.InFrom(requests)
	return p, requests, nil
}

func (r *Runtime) create(ctx context.Context, cfg Config, arcNode arc.Node) (stage.Stage, error) {
	_, ok := cfg.Module.GetStage(arcNode.Type)
	if ok {
		return nil, errors.Newf("unsupported module type: %s", arcNode.Type)
	}
	return std.Create(ctx, std.Config{
		Node:        arcNode,
		Status:      cfg.Status,
		ChannelData: r,
	})
}

func Open(ctx context.Context, cfgs ...Config) (*Runtime, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	r := &Runtime{
		module: cfg.Module,
		nodes:  make(map[string]stage.Stage),
	}
	r.mu.values = make(map[channel.Key]telem.Series)
	if len(cfg.Module.WASM) != 0 {
		r.wasm = wazero.NewRuntime(ctx)
	}
	readChannels, err := retrieveChannels(ctx, cfg.Channel, cfg.Module.Nodes)
	if err != nil {
		return nil, err
	}

	for _, nodeSpec := range cfg.Module.Nodes {
		n, err := r.create(ctx, cfg, nodeSpec)
		if err != nil {
			return nil, err
		}
		n.OnOutput(r.createOnOutput(nodeSpec.Key))
		r.nodes[n.Key()] = n
	}
	p, requests, err := createStreamPipeline(
		ctx,
		r,
		cfg.Framer,
		channel.KeysFromChannels(readChannels),
	)
	r.requests = requests
	if err != nil {
		return nil, err
	}
	sCtx, cancel := signal.Isolated()
	for _, node := range r.nodes {
		node.Flow(sCtx)
	}
	p.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
	)
	r.close = signal.NewGracefulShutdown(sCtx, cancel)
	return r, err
}
