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

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/unsafe"
	"github.com/synnaxlabs/x/validate"
	"github.com/tetratelabs/wazero"
)

type Config struct {
	Module  arc.Module
	Channel channel.Readable
	Framer  *framer.Service
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Module = override.Nil(c.Module, other.Module)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("arc.runtime")
	validate.NotNil(v, "module", c.Module)
	validate.NotNil(v, "Framer", c.Framer)
	validate.NotNil(v, "Channel", c.Channel)
	return v.Error()
}

type Runtime struct {
	// module is the arc program module
	module *arc.Module
	// wasm is the webassembly runtime for the program.
	wasm wazero.Runtime
	confluence.UnarySink[framer.StreamerResponse]
	// requests are used to manage the life cycle of the telemetry frame streamer.
	requests confluence.Inlet[framer.StreamerRequest]
	// values are the current telemetry values for each requested channel in the program.
	values map[channel.Key]telem.Series
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

func (r *Runtime) createOnOutput(nodeKey string) stage.OutputHandler {
	return func(ctx context.Context, value stage.Value) {
		for _, edge := range r.module.Edges {
			if edge.Source.Node == nodeKey {
				value.Param = edge.Target.Param
				r.nodes[edge.Target.Node].Next(ctx, value)
			}
		}
	}
}

func (r *Runtime) processOutput(ctx context.Context, value framer.StreamerResponse) error {
	for i, ser := range value.Frame.RawSeries() {
		if value.Frame.ShouldExcludeRaw(i) {
			continue
		}
		r.values[value.Frame.RawKeyAt(i)] = ser
	}
	keys := value.Frame.KeysSlice()
	for _, nodes := range r.nodes {
		if len(lo.Intersect(nodes.ReadChannels(), keys)) > 0 {
			nodes.Next(ctx, stage.Value{})
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
		keys.Add(unsafe.ReinterpretSlice[uint32, channel.Key](node.Channels.Read.Values())...)
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
) (confluence.Flow, error) {
	p := plumber.New()
	streamer, err := frameSvc.NewStreamer(
		ctx,
		framer.StreamerConfig{Keys: readChannelKeys},
	)
	if err != nil {
		return nil, err
	}
	plumber.SetSegment(p, streamerAddr, streamer)
	r.Sink = r.processOutput
	plumber.SetSink[framer.StreamerResponse](p, "runtime", r)
	streamer.InFrom(confluence.NewStream[framer.StreamerRequest]())
	plumber.MustConnect[framer.StreamerResponse](p, streamerAddr, runtimeAddr, 10)
	return p, nil
}

func Open(ctx context.Context, cfgs ...Config) (*Runtime, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	r := &Runtime{
		wasm:   wazero.NewRuntime(ctx),
		module: cfg.Module,
		values: make(map[channel.Key]telem.Series),
		nodes:  make(map[string]stage.Stage),
	}
	readChannels, err := retrieveChannels(ctx, cfg.Channel, cfg.Module.Nodes)
	if err != nil {
		return nil, err
	}
	p, err := createStreamPipeline(
		ctx,
		r,
		cfg.Framer,
		channel.KeysFromChannels(readChannels),
	)
	if err != nil {
		return nil, err
	}
	sCtx, cancel := signal.Isolated()
	for _, node := range r.nodes {
		node.Flow(sCtx)
	}
	p.Flow(sCtx)
	r.close = signal.NewGracefulShutdown(sCtx, cancel)
	return r, err
}
