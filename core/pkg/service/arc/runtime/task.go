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
	"fmt"
	"io"
	"math"
	stdtime "time"

	"github.com/synnaxlabs/arc/runtime/constant"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/op"
	"github.com/synnaxlabs/arc/runtime/scheduler"
	"github.com/synnaxlabs/arc/runtime/selector"
	"github.com/synnaxlabs/arc/runtime/stable"
	"github.com/synnaxlabs/arc/runtime/stage"
	"github.com/synnaxlabs/arc/runtime/state"
	arctelem "github.com/synnaxlabs/arc/runtime/telem"
	arctime "github.com/synnaxlabs/arc/runtime/time"
	"github.com/synnaxlabs/arc/runtime/wasm"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arcstatus "github.com/synnaxlabs/synnax/pkg/service/arc/status"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
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
	"go.uber.org/zap"
)

const (
	streamerAddr          address.Address = "streamer"
	writerAddr            address.Address = "writer"
	writerResponsesAddr   address.Address = "writer_responses"
	runtimeAddr           address.Address = "runtime"
)

// taskImpl implements the driver.Task interface and manages Arc program execution.
type taskImpl struct {
	ctx        driver.Context
	factoryCfg FactoryConfig
	task       task.Task
	cfg        TaskConfig
	prog       arc.Arc

	closer io.Closer
}

func (t *taskImpl) Exec(ctx context.Context, cmd task.Command) error {
	switch cmd.Type {
	case "start":
		return t.start(ctx)
	case "stop":
		return t.stop()
	default:
		return errors.Newf("invalid command %s received for arc task", cmd)
	}
}

func (t *taskImpl) isRunning() bool {
	return t.closer != nil
}

func (t *taskImpl) start(ctx context.Context) error {
	if t.isRunning() {
		return nil
	}
	drt := dataRuntime{}
	stateCfg, err := NewStateConfig(ctx, t.factoryCfg.Channel, t.prog.Module)
	if err != nil {
		t.setStatus(status.VariantError, false, err.Error())
		return err
	}
	drt.state = state.New(stateCfg.State)
	timeFactory := arctime.NewFactory()
	f := node.MultiFactory{
		arctelem.NewTelemFactory(),
		selector.NewFactory(),
		constant.NewFactory(),
		op.NewFactory(),
		stage.NewFactory(),
		timeFactory,
		stable.NewFactory(stable.FactoryConfig{}),
		arcstatus.NewFactory(t.factoryCfg.Status),
	}
	var closers xio.MultiCloser
	var wasmMod *wasm.Module

	if len(t.prog.Module.WASM) > 0 {
		var err error
		wasmMod, err = wasm.OpenModule(ctx, wasm.ModuleConfig{
			Module: t.prog.Module,
			State:  drt.state,
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

	nodes := make(map[string]node.Node)
	for _, irNode := range t.prog.Module.Nodes {
		n, err := f.Create(ctx, node.Config{
			Node:   irNode,
			Module: t.prog.Module,
			State:  drt.state.Node(irNode.Key),
		})
		if err != nil {
			t.setStatus(status.VariantError, false, err.Error())
			return err
		}
		nodes[irNode.Key] = n
	}

	tolerance := arctime.CalculateTolerance(timeFactory.BaseInterval)
	drt.scheduler = scheduler.New(t.prog.Module.IR, nodes, tolerance)

	drt.scheduler.SetErrorHandler(scheduler.ErrorHandlerFunc(func(nodeKey string, err error) {
		t.factoryCfg.L.Warn("runtime error in arc node",
			zap.String("node", nodeKey),
			zap.Uint64("task", uint64(t.task.Key)),
			zap.Error(err),
		)
		t.setRuntimeError(nodeKey, err)
	}))

	drt.startTime = telem.Now()

	pipeline := plumber.New()

	var runtime confluence.Segment[framer.StreamerResponse, framer.WriterRequest] = &drt
	if hasIntervals := timeFactory.BaseInterval != telem.TimeSpan(math.MaxInt64); hasIntervals {
		runtime = &tickerRuntime{dataRuntime: drt, interval: timeFactory.BaseInterval}
	}
	plumber.SetSegment(pipeline, runtimeAddr, runtime)

	var (
		streamerRequests    = confluence.NewStream[framer.StreamerRequest]()
		streamerCloseSignal io.Closer
	)
	if len(stateCfg.Reads) > 0 {
		streamer, err := t.factoryCfg.Framer.NewStreamer(
			ctx,
			framer.StreamerConfig{Keys: stateCfg.Reads.Keys()},
		)
		if err != nil {
			t.setStatus(status.VariantError, false, err.Error())
			return err
		}
		plumber.SetSegment(pipeline, streamerAddr, streamer)
		plumber.MustConnect[framer.StreamerResponse](pipeline, streamerAddr, runtimeAddr, 10)
		streamer.InFrom(streamerRequests)
		streamerCloseSignal = xio.NoFailCloserFunc(streamerRequests.Close)
	} else {
		streamerResponses := confluence.NewStream[framer.StreamerResponse]()
		runtime.InFrom(streamerResponses)
		streamerCloseSignal = xio.NoFailCloserFunc(streamerResponses.Close)
	}

	if len(stateCfg.Writes) > 0 {
		wrt, err := t.factoryCfg.Framer.NewStreamWriter(
			ctx,
			framer.WriterConfig{
				ControlSubject: control.Subject{Name: t.prog.Name},
				Start:          drt.startTime,
				Keys:           stateCfg.Writes.Keys(),
			},
		)
		if err != nil {
			t.setStatus(status.VariantError, false, err.Error())
			return err
		}
		plumber.SetSegment(pipeline, writerAddr, wrt)
		plumber.MustConnect[framer.WriterRequest](pipeline, runtimeAddr, writerAddr, 10)
		writerResponses := &confluence.UnarySink[framer.WriterResponse]{
			Sink: func(ctx context.Context, res framer.WriterResponse) error {
				if !res.Authorized {
					t.factoryCfg.L.Warn("unauthorized writer response",
						zap.Uint64("task", uint64(t.task.Key)),
						zap.Int("seqNum", res.SeqNum),
						zap.Stringer("command", res.Command),
						zap.Error(res.Err),
					)
				} else if res.Err != nil {
					t.factoryCfg.L.Error("unexpected writer response error",
						zap.Uint64("task", uint64(t.task.Key)),
						zap.Int("seqNum", res.SeqNum),
						zap.Error(res.Err),
					)
				}
				return nil
			},
		}
		plumber.SetSink(pipeline, writerResponsesAddr, writerResponses)
		plumber.MustConnect[framer.WriterResponse](pipeline, writerAddr, writerResponsesAddr, 10)
	}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(t.factoryCfg.Instrumentation))
	t.closer = append(
		closers,
		signal.NewGracefulShutdown(sCtx, cancel),
		streamerCloseSignal,
	)
	pipeline.Flow(sCtx, confluence.CloseOutputInletsOnExit(), confluence.RecoverWithErrOnPanic())
	t.setStatus(status.VariantSuccess, true, "Task started successfully")
	return nil
}

func (t *taskImpl) stop() error {
	if !t.isRunning() {
		return nil
	}
	err := t.closer.Close()
	t.closer = nil
	if err != nil {
		t.setStatus(status.VariantError, false, err.Error())
		return err
	}
	t.setStatus(status.VariantSuccess, false, "Task stopped successfully")
	return nil
}

func (t *taskImpl) Stop(bool) error {
	return t.stop()
}

func (t *taskImpl) Key() task.Key { return t.task.Key }

func (t *taskImpl) setStatus(variant status.Variant, running bool, message string) {
	stat := task.Status{
		Key:     task.OntologyID(t.task.Key).String(),
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
		Details: task.StatusDetails{Task: t.task.Key, Running: running},
	}
	if err := t.ctx.SetStatus(stat); err != nil {
		t.factoryCfg.L.Error(
			"failed to set status for taskImpl",
			zap.Uint64("key", uint64(t.task.Key)),
			zap.String("name", t.task.Name),
			zap.Error(err),
		)
	}
}

func (t *taskImpl) setRuntimeError(nodeKey string, err error) {
	nodeType := nodeKey
	if n, ok := t.prog.Module.Nodes.Find(nodeKey); ok {
		nodeType = n.Type
	}
	stat := task.Status{
		Key:         task.OntologyID(t.task.Key).String(),
		Variant:     status.VariantWarning,
		Message:     fmt.Sprintf("Runtime error in %s", nodeType),
		Description: err.Error(),
		Time:        telem.Now(),
		Details:     task.StatusDetails{Task: t.task.Key, Running: true},
	}
	if setErr := t.ctx.SetStatus(stat); setErr != nil {
		t.factoryCfg.L.Error("failed to set error status", zap.Error(setErr))
	}
}

type dataRuntime struct {
	confluence.AbstractLinear[framer.StreamerResponse, framer.WriterRequest]
	startTime telem.TimeStamp
	scheduler *scheduler.Scheduler
	state     *state.State
}

func (d *dataRuntime) next(
	ctx context.Context,
	res framer.StreamerResponse,
	reason node.RunReason,
) error {
	d.state.Ingest(res.Frame.ToStorage())
	d.scheduler.Next(ctx, telem.Since(d.startTime), reason)
	d.state.ClearReads()
	if fr, changed := d.state.Flush(telem.Frame[uint32]{}); changed && d.Out != nil {
		req := framer.WriterRequest{
			Frame:   frame.NewFromStorage(fr),
			Command: writer.CommandWrite,
		}
		return signal.SendUnderContext(ctx, d.Out.Inlet(), req)
	}
	return nil
}

func (d *dataRuntime) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	if d.Out != nil {
		o.AttachClosables(d.Out)
	}
	signal.GoRange(sCtx, d.In.Outlet(), func(ctx context.Context, res framer.StreamerResponse) error {
		return d.next(ctx, res, node.ReasonChannelInput)
	}, o.Signal...)
}

type tickerRuntime struct {
	dataRuntime
	interval telem.TimeSpan
}

func (r *tickerRuntime) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	if r.Out != nil {
		o.AttachClosables(r.Out)
	}
	sCtx.Go(func(ctx context.Context) error {
		var (
			runReason node.RunReason
			ticker    = stdtime.NewTicker(r.interval.Duration())
			res       framer.StreamerResponse
			ok        bool
		)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				runReason = node.ReasonTimerTick
			case res, ok = <-r.In.Outlet():
				if !ok {
					return nil
				}
				runReason = node.ReasonChannelInput
			}
			if err := r.next(ctx, res, runReason); err != nil {
				return err
			}
		}
	}, o.Signal...)
}
