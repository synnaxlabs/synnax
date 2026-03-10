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

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/scheduler"
	"github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/stl/constant"
	stlcontrol "github.com/synnaxlabs/arc/stl/control"
	stlerrors "github.com/synnaxlabs/arc/stl/errors"
	stlmath "github.com/synnaxlabs/arc/stl/math"
	stlop "github.com/synnaxlabs/arc/stl/op"
	"github.com/synnaxlabs/arc/stl/selector"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stable"
	"github.com/synnaxlabs/arc/stl/stage"
	"github.com/synnaxlabs/arc/stl/stat"
	"github.com/synnaxlabs/arc/stl/stateful"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/stl/wasm"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
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
	"github.com/tetratelabs/wazero"
	"go.uber.org/zap"
)

const (
	streamerAddr        address.Address = "streamer"
	writerAddr          address.Address = "writer"
	writerResponsesAddr address.Address = "writer_responses"
	runtimeAddr         address.Address = "runtime"
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
	stateCfg, err := NewStateConfig(ctx, t.factoryCfg.Channel, t.prog.Program)
	if err != nil {
		t.setStatus(status.VariantError, false, err.Error())
		return err
	}

	drt.state.nodes = node.New(stateCfg.IR)
	drt.state.channel = channel.NewProgramState(stateCfg.ChannelDigests)
	drt.state.series = series.NewProgramState()
	drt.state.strings = stlstrings.NewProgramState()
	drt.state.control = &stlcontrol.ProgramState{}

	var wasmRT wazero.Runtime
	if len(t.prog.Program.WASM) > 0 {
		wasmRT = wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
	}

	timeMod, err := time.NewModule(ctx, wasmRT)
	if err != nil {
		t.setStatus(status.VariantError, false, err.Error())
		return err
	}
	channelMod, err := channel.NewModule(ctx, drt.state.channel, drt.state.strings, wasmRT)
	if err != nil {
		t.setStatus(status.VariantError, false, err.Error())
		return err
	}
	statefulMod, err := stateful.NewModule(ctx, drt.state.series, drt.state.strings, wasmRT)
	if err != nil {
		t.setStatus(status.VariantError, false, err.Error())
		return err
	}
	if _, err := series.NewModule(ctx, drt.state.series, wasmRT); err != nil {
		t.setStatus(status.VariantError, false, err.Error())
		return err
	}
	stringsMod, err := stlstrings.NewModule(ctx, drt.state.strings, wasmRT, nil)
	if err != nil {
		t.setStatus(status.VariantError, false, err.Error())
		return err
	}
	if _, err := stlmath.NewModule(ctx, wasmRT); err != nil {
		t.setStatus(status.VariantError, false, err.Error())
		return err
	}
	errorsMod, err := stlerrors.NewModule(ctx, nil, wasmRT)
	if err != nil {
		t.setStatus(status.VariantError, false, err.Error())
		return err
	}

	f := node.CompoundFactory{
		channelMod,
		statefulMod,
		timeMod,
		selector.NewModule(),
		constant.NewModule(),
		stlop.NewModule(),
		stage.NewModule(),
		stable.NewModule(),
		arcstatus.NewModule(t.factoryCfg.Status),
		stlcontrol.NewModule(drt.state.control),
		&stat.Module{},
	}

	var closers xio.MultiCloser
	if len(t.prog.Program.WASM) > 0 {
		guest, err := wasmRT.Instantiate(ctx, t.prog.Program.WASM)
		if err != nil {
			t.setStatus(status.VariantError, false, err.Error())
			return err
		}
		stringsMod.SetMemory(guest.Memory())
		errorsMod.SetMemory(guest.Memory())
		closers = append(closers, xio.CloserFunc(func() error {
			return errors.Join(guest.Close(ctx), wasmRT.Close(ctx))
		}))
		f = append(f, &wasm.Module{
			Module:        guest,
			Memory:        guest.Memory(),
			Strings:       drt.state.strings,
			NodeKeySetter: statefulMod,
		})
	}

	nodes := make(map[string]node.Node)
	for _, irNode := range t.prog.Program.Nodes {
		n, err := f.Create(ctx, node.Config{
			Node:    irNode,
			Program: t.prog.Program,
			State:   drt.state.nodes.Node(irNode.Key),
		})
		if err != nil {
			t.setStatus(status.VariantError, false, err.Error())
			return err
		}
		nodes[irNode.Key] = n
	}

	tolerance := time.CalculateTolerance(timeMod.BaseInterval)
	drt.scheduler = scheduler.New(t.prog.Program.IR, nodes, tolerance)

	drt.scheduler.SetErrorHandler(scheduler.ErrorHandlerFunc(func(nodeKey string, err error) {
		t.factoryCfg.L.Warn("runtime error in arc node",
			zap.String("node", nodeKey),
			zap.Uint64("task", uint64(t.task.Key)),
			zap.Error(err),
		)
		t.setRuntimeError(nodeKey, err)
	}))

	drt.startTime = telem.Now()
	drt.writeKeys = stateCfg.Writes.Keys()

	pipeline := plumber.New()

	var runtime confluence.Segment[framer.StreamerResponse, framer.WriterRequest] = &drt
	if hasIntervals := timeMod.BaseInterval != telem.TimeSpan(math.MaxInt64); hasIntervals {
		runtime = &tickerRuntime{dataRuntime: drt}
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
		// Critical: Keys is extracted from a map, so we need to convert it to a
		// slice ONCE in order go guarantee stable order.
		writeKeys := stateCfg.Writes.Keys()
		writerCfg := framer.WriterConfig{
			ControlSubject: control.Subject{
				Name: t.prog.Name,
				Key:  t.task.Key.String(),
			},
			Start: drt.startTime,
			Keys:  writeKeys,
		}
		if authorities := buildAuthorities(
			t.prog.Program.Authorities,
			writeKeys,
		); len(authorities) > 0 {
			writerCfg.Authorities = authorities
		}
		wrt, err := t.factoryCfg.Framer.NewStreamWriter(ctx, writerCfg)
		if err != nil {
			t.setStatus(status.VariantError, false, err.Error())
			return err
		}
		plumber.SetSegment(pipeline, writerAddr, wrt)
		plumber.MustConnect[framer.WriterRequest](pipeline, runtimeAddr, writerAddr, 10)
		writerResponses := &confluence.UnarySink[framer.WriterResponse]{
			Sink: func(ctx context.Context, res framer.WriterResponse) error {
				if res.Err != nil {
					t.factoryCfg.L.Error("unexpected writer response error",
						zap.Stringer("task", t.task),
						zap.Int("seqNum", res.SeqNum),
						zap.Error(res.Err),
					)
					t.setStatus(status.VariantError, false, res.Err.Error())
					return res.Err
				} else if !res.Authorized {
					t.factoryCfg.L.Warn("unauthorized writer response",
						zap.Stringer("task", t.task),
						zap.Int("seqNum", res.SeqNum),
						zap.Stringer("command", res.Command),
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
	pipeline.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
		confluence.CancelOnFail(),
	)
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
	if n, ok := t.prog.Program.Nodes.Find(nodeKey); ok {
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

type state struct {
	nodes   *node.ProgramState
	channel *channel.ProgramState
	series  *series.ProgramState
	strings *stlstrings.ProgramState
	control *stlcontrol.ProgramState
}

type dataRuntime struct {
	confluence.AbstractLinear[framer.StreamerResponse, framer.WriterRequest]
	startTime telem.TimeStamp
	scheduler *scheduler.Scheduler
	writeKeys distchannel.Keys
	state     state
}

func (d *dataRuntime) next(
	ctx context.Context,
	res framer.StreamerResponse,
	reason node.RunReason,
) error {
	d.state.channel.Ingest(res.Frame.ToStorage())
	d.scheduler.Next(ctx, telem.Since(d.startTime), reason)
	d.state.channel.ClearReads()
	if d.Out != nil {
		if err := d.flushAuthorityChanges(ctx); err != nil {
			return err
		}
	}
	d.state.series.Clear()
	d.state.strings.Clear()
	if fr, changed := d.state.channel.Flush(telem.Frame[uint32]{}); changed && d.Out != nil {
		req := framer.WriterRequest{
			Frame:   frame.NewFromStorage(fr),
			Command: writer.CommandWrite,
		}
		return signal.SendUnderContext(ctx, d.Out.Inlet(), req)
	}
	return nil
}

func (d *dataRuntime) flushAuthorityChanges(ctx context.Context) error {
	changes := d.state.control.Flush()
	if len(changes) == 0 {
		return nil
	}
	cfg := writer.Config{}
	for _, change := range changes {
		if change.Channel != nil {
			cfg.Keys = append(cfg.Keys, distchannel.Key(*change.Channel))
			cfg.Authorities = append(cfg.Authorities, control.Authority(change.Authority))
		} else {
			cfg.Keys = append(cfg.Keys, d.writeKeys...)
			for range d.writeKeys {
				cfg.Authorities = append(cfg.Authorities, control.Authority(change.Authority))
			}
		}
	}
	req := framer.WriterRequest{Command: writer.CommandSetAuthority, Config: cfg}
	return signal.SendUnderContext(ctx, d.Out.Inlet(), req)
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
}

func (r *tickerRuntime) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	if r.Out != nil {
		o.AttachClosables(r.Out)
	}
	sCtx.Go(func(ctx context.Context) error {
		var (
			runReason node.RunReason
			// Fire immediately so timer nodes seed their first deadline
			// even when no streaming input is connected.
			timer = stdtime.NewTimer(0)
			res   framer.StreamerResponse
			ok    bool
		)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
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
			// Drain the timer channel before resetting to avoid stale
			// values from a simultaneous fire during the select.
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			deadline := r.scheduler.NextDeadline()
			elapsed := telem.Since(r.startTime)
			if deadline == telem.TimeSpanMax {
				// No active timers. Timer stays stopped (from the
				// drain above). We'll only wake on channel input.
			} else if deadline > elapsed {
				timer.Reset((deadline - elapsed).Duration())
			} else {
				timer.Reset(0)
			}
		}
	}, o.Signal...)
}

const DefaultAuthority = control.AuthorityAbsolute

// buildAuthorities constructs a per-channel authority slice from the static
// Authorities in the IR. It maps channel keys to authority values and
// returns the authorities array aligned with writeKeys.
func buildAuthorities(
	auth ir.Authorities,
	writeKeys distchannel.Keys,
) []control.Authority {
	if auth.Default == nil && len(auth.Channels) == 0 {
		return []control.Authority{DefaultAuthority}
	}
	authorities := make([]control.Authority, len(writeKeys))
	for i := range writeKeys {
		if auth.Default != nil {
			authorities[i] = control.Authority(*auth.Default)
		} else {
			authorities[i] = DefaultAuthority
		}
	}
	for key, value := range auth.Channels {
		for i, wk := range writeKeys {
			if wk == distchannel.Key(key) {
				authorities[i] = control.Authority(value)
				break
			}
		}
	}
	return authorities
}
