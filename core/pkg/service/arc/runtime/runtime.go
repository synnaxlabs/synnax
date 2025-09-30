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
	"fmt"
	"io"
	"sync"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/std"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/value"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/tetratelabs/wazero"
)

// Config is the configuration for an arc runtime.
type Config struct {
	Name string
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
	c.Name = override.String(c.Name, other.Name)
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

type streamerSeg struct {
	confluence.UnarySink[framer.StreamerResponse]
	// requests are used to manage the life cycle of the telemetry frame streamerSeg.
	requests confluence.Inlet[framer.StreamerRequest]
	// values are the current telemetry values for each requested channel in the program.
	mu struct {
		sync.RWMutex
		values map[channel.Key]telem.Series
	}
}

func (s *streamerSeg) Close() error {
	s.requests.Close()
	confluence.Drain(s.In)
	return nil
}

type writerSeg struct {
	confluence.UnarySink[framer.WriterResponse]
	confluence.AbstractUnarySource[framer.WriterRequest]
}

func (w *writerSeg) sink(ctx context.Context, res framer.WriterResponse) error {
	fmt.Println(res)
	return nil
}

func (w *writerSeg) Write(ctx context.Context, fr core.Frame) error {
	return signal.SendUnderContext(
		ctx,
		w.Out.Inlet(),
		framer.WriterRequest{Frame: fr, Command: writer.Write},
	)
}

func (w *writerSeg) Close() error {
	if w.Out != nil {
		w.Out.Close()
		confluence.Drain(w.In)
	}
	return nil
}

type Runtime struct {
	// module is the arc program module
	module arc.Module
	// wasm is the webassembly runtime for the program.
	wasm     wazero.Runtime
	streamer *streamerSeg
	writer   *writerSeg
	// nodes are all nodes in the program
	nodes map[string]stage.Stage
	// close is a shutdown handler that stops internal processes
	close io.Closer
}

func (r *Runtime) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	c.Exec(r.streamer.Close)
	c.Exec(r.writer.Close)
	c.Exec(r.close.Close)
	return c.Error()
}

func (r *Runtime) Get(key channel.Key) telem.Series {
	r.streamer.mu.RLock()
	defer r.streamer.mu.RUnlock()
	return r.streamer.mu.values[key]
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
		r.streamer.mu.values[res.Frame.RawKeyAt(i)] = ser
	}
	keys := res.Frame.KeysSlice()
	for _, node := range r.nodes {
		if len(lo.Intersect(node.ReadChannels(), keys)) > 0 {
			node.Next(ctx, "", value.Value{})
		}
	}
	return nil
}

func retrieveReadChannels(
	ctx context.Context,
	channelSvc channel.Readable,
	nodes map[string]stage.Stage,
) ([]channel.Channel, error) {
	keys := make(set.Set[channel.Key])
	for _, node := range nodes {
		keys.Add(node.ReadChannels()...)
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

func retrieveWriteChannels(
	ctx context.Context,
	channelSvc channel.Readable,
	nodes map[string]stage.Stage,
) ([]channel.Channel, error) {
	keys := make(set.Set[channel.Key])
	for _, node := range nodes {
		keys.Add(node.WriteChannels()...)
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
	streamerAddr address.Address = "streamerSeg"
	writerAddr   address.Address = "writer"
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
	r.streamer.Sink = r.processOutput
	plumber.SetSink[framer.StreamerResponse](p, runtimeAddr, r.streamer)
	streamer.InFrom(confluence.NewStream[framer.StreamerRequest]())
	plumber.MustConnect[framer.StreamerResponse](p, streamerAddr, runtimeAddr, 10)
	requests := confluence.NewStream[framer.StreamerRequest]()
	streamer.InFrom(requests)
	return p, requests, nil
}

func createWritePipeline(
	ctx context.Context,
	name string,
	r *Runtime,
	frameSvc *framer.Service,
	writeChannelKeys []channel.Key,
) (confluence.Flow, error) {
	p := plumber.New()
	w, err := frameSvc.NewStreamWriter(
		ctx,
		framer.WriterConfig{
			ControlSubject:   control.Subject{Name: name},
			Start:            telem.Now(),
			Keys:             writeChannelKeys,
			EnableAutoCommit: config.True(),
		},
	)
	if err != nil {
		return nil, err
	}
	r.writer.Sink = r.writer.sink
	plumber.SetSegment(p, writerAddr, w)
	plumber.SetSegment[framer.WriterResponse, framer.WriterRequest](p, runtimeAddr, r.writer)
	plumber.MustConnect[framer.WriterResponse](p, writerAddr, runtimeAddr, 10)
	plumber.MustConnect[framer.WriterRequest](p, runtimeAddr, writerAddr, 10)
	return p, nil
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
		Write:       r.writer.Write,
		Channel:     cfg.Channel,
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
	r.writer = &writerSeg{}
	r.streamer = &streamerSeg{}
	r.streamer.mu.values = make(map[channel.Key]telem.Series)
	if len(cfg.Module.WASM) != 0 {
		r.wasm = wazero.NewRuntime(ctx)
	}

	for _, nodeSpec := range cfg.Module.Nodes {
		n, err := r.create(ctx, cfg, nodeSpec)
		if err != nil {
			return nil, err
		}
		n.OnOutput(r.createOnOutput(nodeSpec.Key))
		r.nodes[n.Key()] = n
	}
	readChannels, err := retrieveReadChannels(ctx, cfg.Channel, r.nodes)
	if err != nil {
		return nil, err
	}
	writeChannels, err := retrieveWriteChannels(ctx, cfg.Channel, r.nodes)
	if err != nil {
		return nil, err
	}
	streamPipeline, requests, err := createStreamPipeline(
		ctx,
		r,
		cfg.Framer,
		channel.KeysFromChannels(readChannels),
	)
	if err != nil {
		return nil, err
	}
	r.streamer.requests = requests
	var writePipeline confluence.Flow
	if len(writeChannels) > 0 {
		writePipeline, err = createWritePipeline(
			ctx,
			cfg.Name,
			r,
			cfg.Framer,
			channel.KeysFromChannels(writeChannels),
		)
		if err != nil {
			return nil, err
		}
	}
	sCtx, cancel := signal.Isolated()
	for _, node := range r.nodes {
		node.Flow(sCtx)
	}
	streamPipeline.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
	)
	if writePipeline != nil {
		writePipeline.Flow(
			sCtx,
			confluence.CloseOutputInletsOnExit(),
			confluence.RecoverWithErrOnPanic(),
		)
	}
	r.close = signal.NewGracefulShutdown(sCtx, cancel)
	return r, err
}
