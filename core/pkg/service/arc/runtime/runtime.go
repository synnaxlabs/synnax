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
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/runtime/constant"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/op"
	"github.com/synnaxlabs/arc/runtime/scheduler"
	"github.com/synnaxlabs/arc/runtime/selector"
	"github.com/synnaxlabs/arc/runtime/stable"
	"github.com/synnaxlabs/arc/runtime/state"
	arctelem "github.com/synnaxlabs/arc/runtime/telem"
	"github.com/synnaxlabs/arc/runtime/wasm"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	rstatus "github.com/synnaxlabs/synnax/pkg/service/arc/status"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
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
	Channel channel.Service
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

func (w *writerSeg) sink(_ context.Context, res framer.WriterResponse) error {
	return nil
}

func (w *writerSeg) Write(ctx context.Context, fr framer.Frame) error {
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
	scheduler *scheduler.Scheduler
	streamer  *streamerSeg
	writer    *writerSeg
	state     *state.State
	closer    io.Closer
}

func (r *Runtime) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	c.Exec(r.streamer.Close)
	if r.writer != nil {
		c.Exec(r.writer.Close)
	}
	c.Exec(r.closer.Close)
	return c.Error()
}

func (r *Runtime) processFrame(ctx context.Context, res framer.StreamerResponse) error {
	r.state.Ingest(res.Frame.ToStorage())
	r.scheduler.Next(ctx)
	fr, changed := r.state.FlushWrites(telem.Frame[uint32]{})
	if !changed {
		return nil
	}
	return r.writer.Write(ctx, core.NewFrameFromStorage(fr))
}

func retrieveChannels(
	ctx context.Context,
	channelSvc channel.Readable,
	keys []channel.Key,
) ([]channel.Channel, error) {
	channels := make([]channel.Channel, 0, len(keys))
	if err := channelSvc.NewRetrieve().
		WhereKeys(keys...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	indexes := lo.FilterMap(channels, func(item channel.Channel, index int) (channel.Key, bool) {
		return item.Index(), !item.Virtual
	})
	indexChannels := make([]channel.Channel, 0, len(indexes))
	if err := channelSvc.NewRetrieve().
		WhereKeys(indexes...).
		Entries(&indexChannels).Exec(ctx, nil); err != nil {
		return nil, err
	}
	return slices.Concat(channels, indexChannels), nil
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
	r.streamer.Sink = r.processFrame
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
			ControlSubject: control.Subject{Name: name},
			Start:          telem.Now(),
			Keys:           writeChannelKeys,
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

func Open(ctx context.Context, cfgs ...Config) (*Runtime, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	stateCfg, err := NewStateConfig(ctx, cfg.Channel, cfg.Module)
	if err != nil {
		return nil, err
	}
	progState := state.New(stateCfg.State)

	telemFactory := arctelem.NewTelemFactory()
	selectFactory := selector.NewFactory()
	constantFactory := constant.NewFactory()
	opFactory := op.NewFactory()
	stableFactory := stable.NewFactory(stable.FactoryConfig{})
	statusFactory := rstatus.NewFactory(cfg.Status)

	f := node.MultiFactory{
		opFactory,
		telemFactory,
		selectFactory,
		constantFactory,
		stableFactory,
		statusFactory,
	}
	var closers xio.MultiCloser
	if len(cfg.Module.WASM) > 0 {
		wasmMod, err := wasm.OpenModule(ctx, wasm.ModuleConfig{
			Module: cfg.Module,
			State:  progState,
		})
		closers = append(closers, wasmMod)
		if err != nil {
			return nil, err
		}
		wasmFactory, err := wasm.NewFactory(wasmMod)
		if err != nil {
			return nil, err
		}
		f = append(f, wasmFactory)
	}
	nodes := make(map[string]node.Node)
	for _, irNode := range cfg.Module.Nodes {
		n, err := f.Create(ctx, node.Config{
			Node:   irNode,
			Module: cfg.Module,
			State:  progState.Node(irNode.Key),
		})
		if err != nil {
			return nil, err
		}
		nodes[irNode.Key] = n
	}

	// Create scheduler with time wheel
	sched := scheduler.New(ctx, cfg.Module.IR, nodes)
	r := &Runtime{
		scheduler: sched,
		state:     progState,
		streamer:  &streamerSeg{},
		writer:    &writerSeg{},
	}

	streamPipeline, requests, err := createStreamPipeline(
		ctx,
		r,
		cfg.Framer,
		stateCfg.Reads.Keys(),
	)
	if err != nil {
		return nil, err
	}
	r.streamer.requests = requests
	var writePipeline confluence.Flow
	if len(stateCfg.Writes) > 0 {
		writePipeline, err = createWritePipeline(
			ctx,
			cfg.Name,
			r,
			cfg.Framer,
			stateCfg.Writes.Keys(),
		)
		if err != nil {
			return nil, err
		}
	}

	sCtx, cancel := signal.Isolated()
	r.scheduler.Init(ctx)
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
	r.closer = append(closers, signal.NewGracefulShutdown(sCtx, cancel))
	return r, err
}
