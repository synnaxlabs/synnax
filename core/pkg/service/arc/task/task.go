// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"context"
	"fmt"
	"io"
	stdtime "time"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/scheduler"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/stl/constant"
	stlcontrol "github.com/synnaxlabs/arc/stl/control"
	stlerrors "github.com/synnaxlabs/arc/stl/errors"
	"github.com/synnaxlabs/arc/stl/math"
	"github.com/synnaxlabs/arc/stl/op"
	"github.com/synnaxlabs/arc/stl/selector"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stable"
	"github.com/synnaxlabs/arc/stl/stateful"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/stl/variable"
	"github.com/synnaxlabs/arc/stl/wasm"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/internal/taskreporter"
	"github.com/synnaxlabs/synnax/pkg/service/arc/internal/timer"
	"github.com/synnaxlabs/synnax/pkg/service/arc/ranges"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	arcstatus "github.com/synnaxlabs/synnax/pkg/service/arc/status"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/tetratelabs/wazero"
	"go.uber.org/zap"
)

const (
	writerAddr          address.Address = "writer"
	writerResponsesAddr address.Address = "writer_responses"
	runtimeAddr         address.Address = "runtime"
)

// impl implements the driver.Task interface and manages Arc program execution.
type impl struct {
	factoryCfg FactoryConfig
	task       task.Task
	cfg        Config
	prog       arc.AutomationArc
	// status is the authority on this instance's current status.
	status *driver.StatusHandler

	// closer shuts down the runtime, nil once released.
	closer io.Closer
	// sCtx is the runtime's context. A runtime that fails cancels it.
	sCtx signal.Context
}

var _ driver.Task = (*impl)(nil)

// Exec implements driver.Task. Start and stop answer the command with a status,
// failures included, so Exec returns an error only when that status was not written.
func (t *impl) Exec(ctx context.Context, cmd task.Command) error {
	var err, answerErr error
	switch cmd.Type {
	case "start":
		err, answerErr = t.start(ctx, cmd.Key)
	case "stop":
		err, answerErr = t.stop(ctx, cmd.Key, true)
	default:
		return driver.ErrUnsupportedCommand
	}
	if answerErr != nil {
		return errors.Combine(err, answerErr)
	}
	if err != nil {
		t.factoryCfg.L.Error("failed to execute command",
			zap.Stringer("task", t.task.Key),
			zap.Stringer("command", cmd),
			zap.Error(err),
		)
	}
	return nil
}

// isRunning reports whether the runtime is up. A runtime that failed on its own
// cancels sCtx without releasing the closer, so the closer alone is not enough.
func (t *impl) isRunning() bool { return t.closer != nil && t.sCtx.Err() == nil }

// release shuts the runtime down and clears it, returning the error that ended it.
func (t *impl) release() error {
	err := t.closer.Close()
	t.closer = nil
	return err
}

// start returns the error that kept the runtime from starting, and the error of the
// status write that answers cmdKey.
func (t *impl) start(ctx context.Context, cmdKey string) (err, answerErr error) {
	if t.isRunning() {
		return nil, t.status.Ack(ctx, cmdKey, true)
	}
	if t.closer != nil {
		if err := t.release(); err != nil {
			t.factoryCfg.L.Warn("arc runtime failed",
				zap.Stringer("task", t.task.Key),
				zap.Error(err),
			)
		}
	}
	flow, err := t.open(ctx)
	if err != nil {
		return err, t.status.Send(
			ctx,
			cmdKey,
			status.VariantError,
			false,
			"Failed to start task",
			err.Error(),
		)
	}
	answerErr = t.status.Send(
		ctx, cmdKey, status.VariantSuccess, true, "Started task", "",
	)
	// The runtime flows after the start status, so that status never replaces one from
	// the first cycle. A writer that never flows keeps its channels, so the runtime
	// flows before a release too.
	flow()
	if answerErr != nil {
		// The caller sees the start fail, so the runtime must not outlive it.
		return t.release(), answerErr
	}
	return nil, nil
}

// open builds the Arc runtime and returns the function that flows it, which the caller
// must call before it releases the runtime. It writes no statuses: start owns them.
func (t *impl) open(ctx context.Context) (flow func(), err error) {
	drt, deps, err := t.newDataRuntime(ctx)
	if err != nil {
		return nil, err
	}
	closers := make(xio.MultiCloser, 0, 4)
	defer func() {
		if err != nil {
			err = errors.Join(err, closers.Close())
		}
	}()
	wasmRT := t.newWASMRuntime(ctx, &closers)
	f, err := t.newFactory(ctx, wasmRT, &drt, &closers)
	if err != nil {
		return nil, err
	}
	if err = t.newScheduler(f, &drt); err != nil {
		return nil, err
	}
	drt.startReading = timer.Now()
	drt.writeKeys = deps.Writes.Slice()
	pipeline := plumber.New()
	ticker, err := t.newTicker(drt)
	if err != nil {
		return nil, err
	}
	pipeline.SetSegment(runtimeAddr, ticker)
	in, err := t.openInput(ctx, deps.Reads)
	if err != nil {
		return nil, err
	}
	ticker.InFrom(in.responses)
	sCtx, cancel := signal.Isolated(
		signal.WithInstrumentation(t.factoryCfg.Instrumentation),
	)
	defer func() {
		if err != nil {
			err = errors.Join(err, signal.NewHardShutdown(sCtx, cancel).Close())
		}
	}()
	opts := []confluence.Option{
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
		confluence.CancelOnFail(),
	}
	if err = in.flow(ctx, sCtx, opts...); err != nil {
		return nil, err
	}
	if err = t.connectWriter(ctx, pipeline, deps.Writes); err != nil {
		return nil, err
	}
	t.sCtx = sCtx
	closers = append(
		closers,
		signal.NewGracefulShutdown(sCtx, cancel),
		in.closeSignal,
	)
	t.closer, closers = closers, nil
	return func() { pipeline.Flow(sCtx, opts...) }, nil
}

// newWASMRuntime returns the WASM runtime of the program and adds its closer to
// closers. It returns nil when the program has no WASM.
func (t *impl) newWASMRuntime(
	ctx context.Context,
	closers *xio.MultiCloser,
) wazero.Runtime {
	if len(t.prog.Program.WASM) == 0 {
		return nil
	}
	wasmRT := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
	*closers = append(*closers, xio.CloserFunc(func() error {
		return wasmRT.Close(ctx)
	}))
	return wasmRT
}

func (t *impl) newDataRuntime(
	ctx context.Context,
) (dataRuntime, runtime.Dependencies, error) {
	var deps runtime.Dependencies
	if err := t.factoryCfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
		deps, err = runtime.NewDependencies(
			ctx,
			snap,
			t.factoryCfg.Channel,
			*t.prog.Program,
		)
		return err
	}); err != nil {
		return dataRuntime{}, runtime.Dependencies{}, err
	}
	drt := dataRuntime{
		clock: telem.MonoClock{Source: t.factoryCfg.Now},
		errorLogs: &errorLogs{
			nodes: make(map[string]*errorLog, len(t.prog.Program.Nodes)),
		},
	}
	drt.state.nodes = node.New(t.prog.Program.IR)
	drt.state.channel = channels.NewProgramState(deps.ChannelDigests)
	drt.state.series = series.NewProgramState()
	drt.state.strings = strings.NewProgramState()
	drt.state.authority = &stlcontrol.ProgramState{}
	return drt, deps, nil
}

// hosts are the STL modules that bind to the WASM runtime.
type hosts struct {
	time     *time.Host
	channels *channels.Host
	stateful *stateful.Host
	strings  *strings.Host
	math     *math.Host
	errors   *stlerrors.Host
}

func (t *impl) newHosts(
	ctx context.Context,
	wasmRT wazero.Runtime,
	drt *dataRuntime,
) (h hosts, err error) {
	report := t.reporter()
	if h.time, err = time.NewHost(
		ctx,
		wasmRT,
		drt.state.strings,
		func(ctx context.Context, message, description string) {
			report(ctx, status.VariantWarning, message, description)
		},
	); err != nil {
		return hosts{}, err
	}
	if h.channels, err = channels.NewHost(
		ctx,
		wasmRT,
		drt.state.channel,
		drt.state.strings,
	); err != nil {
		return hosts{}, err
	}
	if h.stateful, err = stateful.NewHost(
		ctx,
		wasmRT,
		drt.state.series,
		drt.state.strings,
	); err != nil {
		return hosts{}, err
	}
	if _, err = series.NewHost(ctx, wasmRT, drt.state.series); err != nil {
		return hosts{}, err
	}
	if h.strings, err = strings.NewHost(
		ctx,
		wasmRT,
		drt.state.strings,
		nil,
	); err != nil {
		return hosts{}, err
	}
	if h.math, err = math.NewHost(ctx, wasmRT); err != nil {
		return hosts{}, err
	}
	if h.errors, err = stlerrors.NewHost(ctx, wasmRT, nil); err != nil {
		return hosts{}, err
	}
	return h, nil
}

// newFactory returns the factory that creates the nodes of the program. It sets the
// time module and loop budget of drt, and adds the closer of the WASM guest to closers.
func (t *impl) newFactory(
	ctx context.Context,
	wasmRT wazero.Runtime,
	drt *dataRuntime,
	closers *xio.MultiCloser,
) (node.CompoundFactory, error) {
	h, err := t.newHosts(ctx, wasmRT, drt)
	if err != nil {
		return nil, err
	}
	drt.timeMod = h.time
	statusMod, err := arcstatus.NewModule(ctx, arcstatus.ModuleConfig{
		Status:   t.factoryCfg.Status,
		Strings:  drt.state.strings,
		Runtime:  wasmRT,
		Reporter: t.reporter(),
	})
	if err != nil {
		return nil, err
	}
	rangesMod, err := ranges.NewModule(ctx, ranges.ModuleConfig{
		DB:       t.factoryCfg.DB,
		Ranger:   t.factoryCfg.Ranger,
		Strings:  drt.state.strings,
		Runtime:  wasmRT,
		Reporter: t.reporter(),
	})
	if err != nil {
		return nil, err
	}
	// One more slot holds the WASM guest when the program has one.
	f := make(node.CompoundFactory, 0, 13)
	f = append(
		f,
		h.channels,
		h.stateful,
		h.time,
		selector.NewHost(),
		constant.NewHost(),
		variable.NewHost(),
		op.NewHost(),
		stable.NewHost(),
		statusMod,
		rangesMod,
		stlcontrol.NewHost(drt.state.authority),
		h.math,
	)
	if len(t.prog.Program.WASM) == 0 {
		return f, nil
	}
	guest, err := t.instantiate(ctx, wasmRT, h, drt, closers)
	if err != nil {
		return nil, err
	}
	return append(f, guest), nil
}

// instantiate instantiates the WASM guest of the program and adds its closer to
// closers.
func (t *impl) instantiate(
	ctx context.Context,
	wasmRT wazero.Runtime,
	h hosts,
	drt *dataRuntime,
	closers *xio.MultiCloser,
) (*wasm.Module, error) {
	guest, err := wasmRT.Instantiate(ctx, t.prog.Program.WASM)
	if err != nil {
		return nil, err
	}
	h.strings.SetMemory(guest.Memory())
	h.errors.SetMemory(guest.Memory())
	*closers = append(*closers, xio.CloserFunc(func() error {
		return guest.Close(ctx)
	}))
	if drt.budget, err = runtime.NewBudget(
		guest,
		t.factoryCfg.LoopBudget,
	); err != nil {
		return nil, err
	}
	return &wasm.Module{
		Module:        guest,
		Memory:        guest.Memory(),
		Strings:       drt.state.strings,
		NodeKeySetter: h.stateful,
	}, nil
}

// newScheduler creates the nodes of the program with f and sets the scheduler of drt.
func (t *impl) newScheduler(f node.CompoundFactory, drt *dataRuntime) error {
	nodes := make(map[string]node.Node)
	for _, irNode := range t.prog.Program.Nodes {
		n, err := f.Create(node.Config{
			Node:    irNode,
			Program: *t.prog.Program,
			State:   drt.state.nodes.Node(irNode.Key),
		})
		if err != nil {
			return err
		}
		nodes[irNode.Key] = n
		drt.errorLogs.nodes[irNode.Key] = &errorLog{}
	}
	sched, err := scheduler.New(t.prog.Program.IR, nodes)
	if err != nil {
		return err
	}
	logs := drt.errorLogs
	sched.SetErrorHandler(
		scheduler.ErrorHandlerFunc(
			func(ctx context.Context, nodeKey string, err error) {
				text := err.Error()
				skipped, ok := logs.nodes[nodeKey].take(text, logs.now)
				if ok {
					t.factoryCfg.L.Warn("runtime error in arc node",
						zap.String("node", nodeKey),
						zap.Stringer("task", t.task.Key),
						zap.Int("skipped", skipped),
						zap.Error(err),
					)
				}
				t.setRuntimeError(ctx, nodeKey, text)
			},
		),
	)
	drt.scheduler = sched
	return nil
}

func (t *impl) newTicker(drt dataRuntime) (*tickerRuntime, error) {
	// The ticker's t=0 startup tick fires entry nodes; an input-driven program
	// would otherwise not fire them until the first input is received.
	spin, ok := spins[t.cfg.Performance]
	if !ok {
		return nil, errors.Newf("unknown performance %q", t.cfg.Performance)
	}
	return &tickerRuntime{
		dataRuntime: drt,
		spin:        spin,
		overrun: func(ctx context.Context, err error) {
			t.setStatus(
				ctx,
				driver.NoCommand,
				status.VariantError,
				false,
				"Program ran too long",
				err.Error(),
			)
		},
	}, nil
}

// input streams the channels that the runtime reads.
type input struct {
	// streamer is nil when the runtime reads no channels.
	streamer  framer.Streamer
	responses *confluence.Stream[framer.StreamerResponse]
	// closeSignal tells the input to shut down.
	closeSignal io.Closer
}

func (t *impl) openInput(
	ctx context.Context,
	reads set.Set[channel.Key],
) (input, error) {
	var (
		requests  = confluence.NewStream[framer.StreamerRequest]()
		responses = confluence.NewStream[framer.StreamerResponse](10)
	)
	if len(reads) == 0 {
		return input{
			responses:   responses,
			closeSignal: xio.NoFailCloserFunc(responses.Close),
		}, nil
	}
	streamer, err := t.factoryCfg.Framer.NewStreamer(
		ctx,
		framer.StreamerConfig{Keys: reads.Slice(), SendOpenAck: true},
	)
	if err != nil {
		return input{}, err
	}
	streamer.InFrom(requests)
	streamer.OutTo(responses)
	return input{
		streamer:    streamer,
		responses:   responses,
		closeSignal: xio.NoFailCloserFunc(requests.Close),
	}, nil
}

// flow flows the streamer and waits for its open ack, so the task reads every input
// written after start returns. The runtime flows after this, so it never sees the ack.
func (in input) flow(
	ctx context.Context,
	sCtx signal.Context,
	opts ...confluence.Option,
) error {
	if in.streamer == nil {
		return nil
	}
	in.streamer.Flow(sCtx, opts...)
	select {
	case <-ctx.Done():
	case _, ok := <-in.responses.Outlet():
		if !ok {
			return errors.New("streamer exited before it opened")
		}
	}
	return ctx.Err()
}

// connectWriter adds a writer of the channels in writes to pipeline, fed by the
// runtime. It adds nothing when writes is empty.
func (t *impl) connectWriter(
	ctx context.Context,
	pipeline *plumber.Pipeline,
	writes set.Set[channel.Key],
) error {
	if len(writes) == 0 {
		return nil
	}
	// Critical: ToSlice is extracted from a map, so we need to convert it to a slice
	// ONCE in order go guarantee stable order.
	writeKeys := writes.Slice()
	writerCfg := framer.WriterConfig{
		ControlSubject: control.Subject{
			Name: t.prog.Name,
			Key:  t.task.Key.String(),
		},
		Keys: writeKeys,
	}
	if authorities := buildAuthorities(
		t.prog.Program.Authorities,
		writeKeys,
	); len(authorities) > 0 {
		writerCfg.Authorities = authorities
	}
	wrt, err := t.factoryCfg.Framer.NewStreamWriter(ctx, writerCfg)
	if err != nil {
		return err
	}
	pipeline.SetSegment(writerAddr, wrt)
	pipeline.MustConnect[framer.WriterRequest](runtimeAddr, writerAddr, 10)
	pipeline.SetSink(writerResponsesAddr, &confluence.UnarySink[framer.WriterResponse]{
		Sink: t.handleWriterResponse,
	})
	pipeline.MustConnect[framer.WriterResponse](writerAddr, writerResponsesAddr, 10)
	return nil
}

func (t *impl) handleWriterResponse(
	ctx context.Context,
	res framer.WriterResponse,
) error {
	if res.Err != nil {
		t.factoryCfg.L.Error("unexpected writer response error",
			zap.Stringer("task", t.task),
			zap.Int("seqNum", res.SeqNum),
			zap.Error(res.Err),
		)
		t.setStatus(
			ctx,
			driver.NoCommand,
			status.VariantError,
			false,
			"Failed to write channel data",
			t.factoryCfg.Channel.NameErrors(ctx, res.Err, nil).Error(),
		)
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
}

func (t *impl) Stop(sendStatus bool) error {
	/// TODO until we fix our usage of contexts in general:
	// https://linear.app/synnax/issue/SY-4002/refactor-usages-of-contextcontext
	return errors.Combine(t.stop(context.TODO(), driver.NoCommand, sendStatus))
}

// stop releases the runtime. It guards on the closer rather than isRunning so that a
// runtime that already failed is still released, and its failure still reported. It
// returns the error that ended the runtime, and the error of the status write that
// answers cmdKey.
func (t *impl) stop(
	ctx context.Context,
	cmdKey string,
	sendStatus bool,
) (err, answerErr error) {
	if t.closer == nil {
		if sendStatus {
			return nil, t.status.Ack(ctx, cmdKey, false)
		}
		return nil, nil
	}
	err = t.release()
	if !sendStatus {
		return err, nil
	}
	if err != nil {
		return err, t.status.Send(
			ctx, cmdKey, status.VariantError, false, "Failed to stop task", err.Error(),
		)
	}
	return nil, t.status.Send(
		ctx, cmdKey, status.VariantSuccess, false, "Stopped task", "",
	)
}

func (t *impl) reporter() taskreporter.Reporter {
	return func(
		ctx context.Context,
		variant status.Variant,
		message string,
		description string,
	) {
		t.setStatus(
			ctx,
			driver.NoCommand,
			variant,
			t.isRunning(),
			message,
			description,
		)
	}
}

func (t *impl) setStatus(
	ctx context.Context,
	cmdKey string,
	variant status.Variant,
	running bool,
	message string,
	description string,
) {
	if err := t.status.Send(
		ctx,
		cmdKey,
		variant,
		running,
		message,
		description,
	); err != nil {
		t.factoryCfg.L.Error(
			"failed to set status for Arc task",
			zap.Stringer("key", t.task.Key),
			zap.String("name", t.task.Name),
			zap.Error(err),
		)
	}
}

func (t *impl) setRuntimeError(ctx context.Context, nodeKey, text string) {
	nodeType := nodeKey
	if n, ok := t.prog.Program.Nodes.Find(nodeKey); ok {
		nodeType = n.Type
	}
	if setErr := t.status.Warn(ctx, driver.Warning{
		Message:     "Node failed",
		Description: fmt.Sprintf("The %s node failed: %s", nodeType, text),
	}); setErr != nil {
		t.factoryCfg.L.Error("failed to set error status", zap.Error(setErr))
	}
}

type state struct {
	nodes     *node.ProgramState
	channel   *channels.ProgramState
	series    *series.ProgramState
	strings   *strings.ProgramState
	authority *stlcontrol.ProgramState
}

type dataRuntime struct {
	confluence.AbstractLinear[framer.StreamerResponse, framer.WriterRequest]
	// startReading is the timer.Now reading the runtime started at. Every deadline
	// counts from it.
	startReading stdtime.Duration
	// clock stamps every cycle. It is the runtime's only clock: nodes and host
	// functions read the stamp it produces instead of sampling their own.
	clock     telem.MonoClock
	scheduler *scheduler.Scheduler
	// timeMod receives each cycle's stamp so the `now` WASM binding, which guest
	// code calls without a node Context, returns the same value as everything else.
	timeMod   *time.Host
	writeKeys channel.Keys
	state     state
	// budget limits the loop iterations and function calls of each cycle.
	budget runtime.Budget
	// errorLogs limits the log lines of each node.
	errorLogs *errorLogs
}

func (d *dataRuntime) next(
	ctx context.Context,
	res framer.StreamerResponse,
	reason node.RunReason,
) error {
	d.state.channel.Ingest(res.Frame.ToStorage())
	cycle := node.Cycle{
		Now:     d.clock.Now(),
		Elapsed: d.elapsed(),
		Reason:  reason,
	}
	d.timeMod.SetNow(cycle.Now)
	d.errorLogs.now = cycle.Now
	d.budget.Reset()
	d.clock.Advance(d.scheduler.Next(ctx, cycle))
	if err := d.budget.Err(); err != nil {
		return err
	}
	d.state.channel.ClearReads()
	if d.Out != nil {
		if err := d.flushAuthorityChanges(ctx); err != nil {
			return err
		}
	}
	d.state.series.Clear()
	d.state.strings.Clear()
	fr, highest, changed := d.state.channel.Flush(telem.Frame[uint32]{}, cycle.Now)
	d.clock.Advance(highest)
	if changed && d.Out != nil {
		req := framer.WriterRequest{
			Frame:   frame.NewFromStorage(fr),
			Command: framer.WriterCommandWrite,
		}
		return signal.SendUnderContext(ctx, d.Out.Inlet(), req)
	}
	return nil
}

// elapsed returns the time since the runtime started.
func (d *dataRuntime) elapsed() telem.TimeSpan {
	return telem.TimeSpan(timer.Now() - d.startReading)
}

func (d *dataRuntime) flushAuthorityChanges(ctx context.Context) error {
	changes := d.state.authority.Flush()
	if len(changes) == 0 {
		return nil
	}
	cfg := framer.WriterConfig{}
	for _, change := range changes {
		if change.Channel != nil {
			cfg.Keys = append(cfg.Keys, channel.Key(*change.Channel))
			cfg.Authorities = append(
				cfg.Authorities,
				control.Authority(change.Authority),
			)
		} else {
			cfg.Keys = append(cfg.Keys, d.writeKeys...)
			for range d.writeKeys {
				cfg.Authorities = append(
					cfg.Authorities,
					control.Authority(change.Authority),
				)
			}
		}
	}
	req := framer.WriterRequest{Command: framer.WriterCommandSetAuthority, Config: cfg}
	return signal.SendUnderContext(ctx, d.Out.Inlet(), req)
}

// spins maps each performance level to how long the ticker's timer spins before a
// deadline.
var spins = map[Performance]stdtime.Duration{
	PerformanceAuto:   timer.DefaultSpin,
	PerformanceLow:    0,
	PerformanceMedium: stdtime.Millisecond,
	PerformanceHigh:   timer.SpinAll,
}

type tickerRuntime struct {
	dataRuntime
	// spin is how long the timer spins before each deadline.
	spin stdtime.Duration
	// overrun reports a cycle that used its whole loop budget.
	overrun func(context.Context, error)
}

func (r *tickerRuntime) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	if r.Out != nil {
		o.AttachClosables(r.Out)
	}
	sCtx.Go(func(ctx context.Context) (err error) {
		t, err := timer.New(r.spin)
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, t.Close()) }()
		return r.run(ctx, t)
	}, o.Signal...)
}

// run runs a cycle on each wake of t and each channel input, until ctx ends or the
// input closes.
func (r *tickerRuntime) run(ctx context.Context, t *timer.Timer) error {
	// Fire right away, so timer nodes set their first deadline even when no streaming
	// input is connected.
	if err := t.Reset(0); err != nil {
		return err
	}
	var (
		runReason node.RunReason
		res       framer.StreamerResponse
		ok        bool
	)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			runReason = node.ReasonTimerTick
		case res, ok = <-r.In.Outlet():
			if !ok {
				return nil
			}
			runReason = node.ReasonChannelInput
			// A due timer fires on this cycle, not on a later timer wake.
			if r.elapsed() >= r.scheduler.NextDeadline() {
				runReason = node.ReasonTimerTick
			}
		}
		if err := r.next(ctx, res, runReason); err != nil {
			if errors.Is(err, runtime.ErrLoopBudget) {
				r.overrun(ctx, err)
			}
			return err
		}
		if err := r.arm(t); err != nil {
			return err
		}
	}
}

// arm sets t to wake at the next deadline of the scheduler.
func (r *tickerRuntime) arm(t *timer.Timer) error {
	deadline := r.scheduler.NextDeadline()
	if deadline == telem.TimeSpanMax {
		// No active timers, so the loop wakes only on channel input.
		return t.Stop()
	}
	return t.Reset((deadline - r.elapsed()).Duration())
}

const DefaultAuthority = control.AuthorityAbsolute

// buildAuthorities constructs a per-channel authority slice from the static
// Authorities in the IR. It maps channel keys to authority values and
// returns the authorities array aligned with writeKeys.
func buildAuthorities(
	auth ir.Authorities,
	writeKeys channel.Keys,
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
			if wk == channel.Key(key) {
				authorities[i] = control.Authority(value)
				break
			}
		}
	}
	return authorities
}
