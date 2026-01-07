// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/driver/go"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arcstatus "github.com/synnaxlabs/synnax/pkg/service/arc/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
)

// Task implements the godriver.Task interface and manages Arc program execution.
type Task struct {
	// Identity & compiled program
	key  task.Key
	prog arc.Arc // Contains Name, Graph, and compiled Module
	cfg  TaskConfig

	// Context for status updates
	ctx godriver.Context
	// Factory config (injected dependencies for runtime)
	factoryCfg FactoryConfig

	// Runtime state (nil/zero when stopped, initialized on start)
	running   bool
	scheduler *scheduler.Scheduler
	streamer  *streamerSeg
	writer    *writerSeg
	state     *state.State
	closer    io.Closer
	startTime telem.TimeStamp
}

func newTask(key task.Key, prog arc.Arc, cfg TaskConfig, ctx godriver.Context, factoryCfg FactoryConfig) *Task {
	return &Task{
		key:        key,
		prog:       prog,
		cfg:        cfg,
		ctx:        ctx,
		factoryCfg: factoryCfg,
	}
}

func (t *Task) Exec(ctx context.Context, cmd task.Command) error {
	switch cmd.Type {
	case "start":
		return t.start(ctx)
	case "stop":
		return t.stop(ctx, false)
	default:
		return nil
	}
}

func (t *Task) start(ctx context.Context) error {
	if t.running {
		return nil
	}

	mod := t.prog.Module

	// Build state config
	stateCfg, err := NewStateConfig(ctx, t.factoryCfg.Channel, mod)
	if err != nil {
		t.setStatus(status.VariantError, false, err.Error())
		return err
	}
	t.state = state.New(stateCfg.State)

	// Create node factories
	telemFactory := arctelem.NewTelemFactory()
	selectFactory := selector.NewFactory()
	constantFactory := constant.NewFactory()
	opFactory := op.NewFactory()
	stableFactory := stable.NewFactory(stable.FactoryConfig{})
	statusFactory := arcstatus.NewFactory(t.factoryCfg.Status)

	f := node.MultiFactory{
		opFactory,
		telemFactory,
		selectFactory,
		constantFactory,
		stableFactory,
		statusFactory,
	}

	var closers xio.MultiCloser
	if len(mod.WASM) > 0 {
		wasmMod, err := wasm.OpenModule(ctx, wasm.ModuleConfig{
			Module: mod,
			State:  t.state,
		})
		if err != nil {
			t.setStatus(status.VariantError, false, err.Error())
			return err
		}
		closers = append(closers, wasmMod)
		wasmFactory, err := wasm.NewFactory(wasmMod)
		if err != nil {
			t.setStatus(status.VariantError, false, err.Error())
			return err
		}
		f = append(f, wasmFactory)
	}

	// Create nodes
	nodes := make(map[string]node.Node)
	for _, irNode := range mod.Nodes {
		n, err := f.Create(ctx, node.Config{
			Node:   irNode,
			Module: mod,
			State:  t.state.Node(irNode.Key),
		})
		if err != nil {
			t.setStatus(status.VariantError, false, err.Error())
			return err
		}
		nodes[irNode.Key] = n
	}

	// Create scheduler
	t.scheduler = scheduler.New(mod.IR, nodes)

	// Initialize segments
	t.streamer = &streamerSeg{}
	t.writer = &writerSeg{}
	t.startTime = telem.Now()

	// Create stream pipeline
	streamPipeline, requests, err := createStreamPipeline(
		ctx, t, t.factoryCfg.Framer, stateCfg.Reads.Keys(),
	)
	if err != nil {
		t.setStatus(status.VariantError, false, err.Error())
		return err
	}
	t.streamer.requests = requests

	// Create write pipeline if needed
	var writePipeline confluence.Flow
	if len(stateCfg.Writes) > 0 {
		writePipeline, err = createWritePipeline(
			ctx, t.prog.Name, t, t.factoryCfg.Framer, stateCfg.Writes.Keys(),
		)
		if err != nil {
			t.setStatus(status.VariantError, false, err.Error())
			return err
		}
	}

	// Start pipelines
	sCtx, cancel := signal.Isolated()
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
	t.closer = append(closers, signal.NewGracefulShutdown(sCtx, cancel))

	t.running = true
	t.setStatus(status.VariantSuccess, true, "Arc started")
	return nil
}

// processFrame is the core reactive loop: ingests telemetry, schedules nodes, flushes writes.
func (t *Task) processFrame(ctx context.Context, res framer.StreamerResponse) error {
	t.state.Ingest(res.Frame.ToStorage())
	t.scheduler.Next(ctx, telem.Since(t.startTime))
	fr, changed := t.state.FlushWrites(telem.Frame[uint32]{})
	if !changed {
		return nil
	}
	return t.writer.Write(ctx, frame.NewFromStorage(fr))
}

func (t *Task) stop(context.Context, bool) error {
	if !t.running {
		return nil
	}

	c := errors.NewCatcher(errors.WithAggregation())
	if t.streamer != nil {
		c.Exec(t.streamer.Close)
	}
	if t.writer != nil {
		c.Exec(t.writer.Close)
	}
	if t.closer != nil {
		c.Exec(t.closer.Close)
	}

	// Reset runtime state
	t.scheduler = nil
	t.streamer = nil
	t.writer = nil
	t.state = nil
	t.closer = nil
	t.running = false

	if err := c.Error(); err != nil {
		t.setStatus(status.VariantError, false, err.Error())
		return err
	}
	t.setStatus(status.VariantSuccess, false, "Arc stopped")
	return nil
}

func (t *Task) Stop(ctx context.Context, willReconfigure bool) error {
	return t.stop(ctx, willReconfigure)
}

func (t *Task) Key() task.Key { return t.key }

func (t *Task) setStatus(variant status.Variant, running bool, message string) {
	stat := task.Status{
		Key:     task.OntologyID(t.key).String(),
		Variant: variant,
		Message: message,
		Details: task.StatusDetails{
			Task:    t.key,
			Running: running,
		},
	}
	_ = t.ctx.SetStatus(stat)
}

// Pipeline helper types and functions

type streamerSeg struct {
	confluence.UnarySink[framer.StreamerResponse]
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

func retrieveChannels(
	ctx context.Context,
	channelSvc *channel.Service,
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
	t *Task,
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
	t.streamer.Sink = t.processFrame
	plumber.SetSink[framer.StreamerResponse](p, runtimeAddr, t.streamer)
	streamer.InFrom(confluence.NewStream[framer.StreamerRequest]())
	plumber.MustConnect[framer.StreamerResponse](p, streamerAddr, runtimeAddr, 10)
	requests := confluence.NewStream[framer.StreamerRequest]()
	streamer.InFrom(requests)
	return p, requests, nil
}

func createWritePipeline(
	ctx context.Context,
	name string,
	t *Task,
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
	t.writer.Sink = t.writer.sink
	plumber.SetSegment(p, writerAddr, w)
	plumber.SetSegment[framer.WriterResponse, framer.WriterRequest](p, runtimeAddr, t.writer)
	plumber.MustConnect[framer.WriterResponse](p, writerAddr, runtimeAddr, 10)
	plumber.MustConnect[framer.WriterRequest](p, runtimeAddr, writerAddr, 10)
	return p, nil
}
