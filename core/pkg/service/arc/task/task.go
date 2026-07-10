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
	"sync"
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
	"github.com/synnaxlabs/arc/stl/wasm"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arccontrol "github.com/synnaxlabs/synnax/pkg/service/arc/control"
	"github.com/synnaxlabs/synnax/pkg/service/arc/internal/taskreporter"
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
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/tetratelabs/wazero"
	"go.uber.org/zap"
)

const (
	streamerAddr        address.Address = "streamer"
	writerAddr          address.Address = "writer"
	writerResponsesAddr address.Address = "writer_responses"
	runtimeAddr         address.Address = "runtime"
	controlStreamerAddr address.Address = "control_streamer"
	controlSinkAddr     address.Address = "control_sink"
)

// impl implements the driver.Task interface and manages Arc program execution.
type impl struct {
	factoryCfg FactoryConfig
	task       task.Task
	cfg        Config
	prog       arc.Arc

	closer      io.Closer
	status      statusState
	tickErrored bool
}

var _ driver.Task = (*impl)(nil)

func (t *impl) Exec(ctx context.Context, cmd task.Command) error {
	switch cmd.Type {
	case "start":
		return t.start(ctx)
	case "stop":
		return t.Stop()
	default:
		return driver.ErrUnsupportedCommand
	}
}

func (t *impl) isRunning() bool { return t.closer != nil }

func (t *impl) start(ctx context.Context) (err error) {
	if t.isRunning() {
		return nil
	}
	t.resetStatus()
	drt := dataRuntime{task: t}
	deps, err := runtime.NewDependencies(ctx, t.factoryCfg.Channel, *t.prog.Program)
	if err != nil {
		t.setTerminal(ctx, status.VariantError, err.Error())
		return err
	}

	drt.state.nodes = node.New(t.prog.Program.IR)
	drt.state.channel = channels.NewProgramState(deps.ChannelDigests)
	drt.state.series = series.NewProgramState()
	drt.state.strings = strings.NewProgramState()
	drt.state.authority = &stlcontrol.ProgramState{}

	var closers xio.MultiCloser
	defer func() {
		if err != nil {
			err = errors.Join(err, closers.Close())
		}
	}()

	var wasmRT wazero.Runtime
	if len(t.prog.Program.WASM) > 0 {
		wasmRT = wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
		closers = append(closers, xio.CloserFunc(func() error {
			return wasmRT.Close(ctx)
		}))
	}

	timeMod, err := time.NewHost(ctx, wasmRT)
	if err != nil {
		t.setTerminal(ctx, status.VariantError, err.Error())
		return err
	}
	channelMod, err := channels.NewHost(ctx, wasmRT, drt.state.channel, drt.state.strings)
	if err != nil {
		t.setTerminal(ctx, status.VariantError, err.Error())
		return err
	}
	statefulMod, err := stateful.NewHost(ctx, wasmRT, drt.state.series, drt.state.strings)
	if err != nil {
		t.setTerminal(ctx, status.VariantError, err.Error())
		return err
	}
	if _, err = series.NewHost(ctx, wasmRT, drt.state.series); err != nil {
		t.setTerminal(ctx, status.VariantError, err.Error())
		return err
	}
	stringsMod, err := strings.NewHost(ctx, wasmRT, drt.state.strings, nil)
	if err != nil {
		t.setTerminal(ctx, status.VariantError, err.Error())
		return err
	}
	mathMod, err := math.NewHost(ctx, wasmRT)
	if err != nil {
		t.setTerminal(ctx, status.VariantError, err.Error())
		return err
	}
	errorsMod, err := stlerrors.NewHost(ctx, wasmRT, nil)
	if err != nil {
		t.setTerminal(ctx, status.VariantError, err.Error())
		return err
	}
	statusMod, err := arcstatus.NewModule(ctx, arcstatus.ModuleConfig{
		Status:   t.factoryCfg.Status,
		Strings:  drt.state.strings,
		Runtime:  wasmRT,
		Reporter: t.reporter(),
	})
	if err != nil {
		t.setTerminal(ctx, status.VariantError, err.Error())
		return err
	}
	rangesMod, err := ranges.NewModule(ctx, ranges.ModuleConfig{
		Ranger:   t.factoryCfg.Ranger,
		Strings:  drt.state.strings,
		Runtime:  wasmRT,
		Reporter: t.reporter(),
	})
	if err != nil {
		t.setTerminal(ctx, status.VariantError, err.Error())
		return err
	}

	f := node.CompoundFactory{
		channelMod,
		statefulMod,
		timeMod,
		selector.NewHost(),
		constant.NewHost(),
		op.NewHost(),
		stable.NewHost(),
		statusMod,
		rangesMod,
		stlcontrol.NewHost(drt.state.authority),
		mathMod,
	}

	if len(t.prog.Program.WASM) > 0 {
		guest, guestErr := wasmRT.Instantiate(ctx, t.prog.Program.WASM)
		if guestErr != nil {
			t.setTerminal(ctx, status.VariantError, guestErr.Error())
			return guestErr
		}
		stringsMod.SetMemory(guest.Memory())
		errorsMod.SetMemory(guest.Memory())
		closers = append(closers, xio.CloserFunc(func() error {
			return guest.Close(ctx)
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
		n, nodeErr := f.Create(ctx, node.Config{
			Node:    irNode,
			Program: *t.prog.Program,
			State:   drt.state.nodes.Node(irNode.Key),
		})
		if nodeErr != nil {
			t.setTerminal(ctx, status.VariantError, nodeErr.Error())
			return nodeErr
		}
		nodes[irNode.Key] = n
	}

	tolerance := time.CalculateTolerance(timeMod.BaseInterval)
	drt.scheduler = scheduler.New(t.prog.Program.IR, nodes, tolerance)

	drt.scheduler.SetErrorHandler(scheduler.ErrorHandlerFunc(func(ctx context.Context, nodeKey string, err error) {
		t.factoryCfg.L.Warn("runtime error in arc node",
			zap.String("node", nodeKey),
			zap.Uint64("task", uint64(t.task.Key)),
			zap.Error(err),
		)
		t.setRuntimeError(ctx, nodeKey, err)
		t.tickErrored = true
	}))

	drt.startTime = telem.Now()
	drt.writeKeys = deps.Writes.Slice()

	var warner *controlWarner
	if len(deps.Writes) > 0 {
		drt.subject = control.Subject{Name: t.prog.Name, Key: t.task.Key.String()}
	}

	pipeline := plumber.New()

	// The ticker's t=0 startup tick fires entry nodes; an input-driven program
	// would otherwise not fire them until the first input is received.
	ticker := &tickerRuntime{dataRuntime: drt}
	plumber.SetSegment(pipeline, runtimeAddr, ticker)

	var (
		streamerRequests    = confluence.NewStream[framer.StreamerRequest]()
		streamerCloseSignal io.Closer
		digestCloseSignal   io.Closer
	)
	if len(deps.Reads) > 0 {
		var streamer framer.Streamer
		streamer, err = t.factoryCfg.Framer.NewStreamer(
			ctx,
			framer.StreamerConfig{Keys: deps.Reads.Slice()},
		)
		if err != nil {
			t.setTerminal(ctx, status.VariantError, err.Error())
			return err
		}
		plumber.SetSegment(pipeline, streamerAddr, streamer)
		plumber.MustConnect[framer.StreamerResponse](pipeline, streamerAddr, runtimeAddr, 10)
		streamer.InFrom(streamerRequests)
		streamerCloseSignal = xio.NoFailCloserFunc(streamerRequests.Close)
	} else {
		streamerResponses := confluence.NewStream[framer.StreamerResponse]()
		ticker.InFrom(streamerResponses)
		streamerCloseSignal = xio.NoFailCloserFunc(streamerResponses.Close)
	}

	if len(deps.Writes) > 0 {
		// Critical: ToSlice is extracted from a map, so we need to convert it to a
		// slice ONCE in order go guarantee stable order.
		writeKeys := deps.Writes.Slice()

		// Resolve the control-conflict inputs before opening the writer so a retrieval
		// failure cannot leak the writer's acquired control gate.
		declared := declaredWriteKeys(t.prog).Unique()
		digestKeys, digestErr := controlDigestKeys(ctx, t.factoryCfg.Channel, declared)
		if digestErr != nil {
			t.setTerminal(ctx, status.VariantError, digestErr.Error())
			return digestErr
		}
		var writeChannels []channel.Channel
		if len(digestKeys) > 0 {
			writeChannels, err = retrieveWriteChannels(ctx, t.factoryCfg.Channel, declared)
			if err != nil {
				t.setTerminal(ctx, status.VariantError, err.Error())
				return err
			}
		} else if len(declared) > 0 {
			t.factoryCfg.L.Warn(
				"no control digest channels resolved; control warnings disabled",
				zap.Uint64("key", uint64(t.task.Key)),
			)
		}

		// Open the digest streamer before the writer so the writer, which acquires the
		// control gate, is the last fallible open before Flow.
		var digestStreamer framer.Streamer
		if len(digestKeys) > 0 {
			digestStreamer, err = t.factoryCfg.Framer.NewStreamer(ctx, framer.StreamerConfig{Keys: digestKeys})
			if err != nil {
				t.setTerminal(ctx, status.VariantError, err.Error())
				return err
			}
		}

		writerCfg := framer.WriterConfig{
			ControlSubject: drt.subject,
			Start:          drt.startTime,
			Keys:           writeKeys,
		}
		if authorities := buildAuthorities(
			t.prog.Program.Authorities,
			writeKeys,
		); len(authorities) > 0 {
			writerCfg.Authorities = authorities
		}
		var wrt framer.StreamWriter
		wrt, err = t.factoryCfg.Framer.NewStreamWriter(ctx, writerCfg)
		if err != nil {
			t.setTerminal(ctx, status.VariantError, err.Error())
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
					t.setTerminal(ctx, status.VariantError, res.Err.Error())
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

		if digestStreamer != nil {
			states := arccontrol.New()
			plumber.SetSegment(pipeline, controlStreamerAddr, digestStreamer)
			warner = newControlWarner(t, states, writeChannels, drt.subject)
			plumber.SetSink(pipeline, controlSinkAddr, warner.sink())
			plumber.MustConnect[framer.StreamerResponse](pipeline, controlStreamerAddr, controlSinkAddr, 10)
			digestRequests := confluence.NewStream[framer.StreamerRequest]()
			digestStreamer.InFrom(digestRequests)
			digestCloseSignal = xio.NoFailCloserFunc(digestRequests.Close)
		}
	}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(t.factoryCfg.Instrumentation))
	// Establish the running baseline before launching the warner so a first-frame
	// conflict cannot be clobbered by a later baseline write.
	t.setRunning(ctx, true)
	if warner != nil {
		sCtx.Go(func(ctx context.Context) error {
			warner.run(ctx)
			return nil
		})
	}
	closers = append(closers, signal.NewGracefulShutdown(sCtx, cancel), streamerCloseSignal)
	if digestCloseSignal != nil {
		closers = append(closers, digestCloseSignal)
	}
	if warner != nil {
		closers = append(closers, xio.NoFailCloserFunc(warner.stop))
	}
	t.closer = closers
	closers = nil
	pipeline.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
		confluence.CancelOnFail(),
	)
	return nil
}

func (t *impl) Stop() error {
	if !t.isRunning() {
		return nil
	}
	err := t.closer.Close()
	t.closer = nil
	/// TODO until we fix our usage of contexts in general:
	// https://linear.app/synnax/issue/SY-4002/refactor-usages-of-contextcontext
	ctx := context.TODO()
	if err != nil {
		t.setTerminal(ctx, status.VariantError, err.Error())
		return err
	}
	t.setTerminal(ctx, status.VariantSuccess, "Task stopped successfully")
	return nil
}

// statusContribution is one source's contribution to the composed task status.
type statusContribution struct {
	variant     status.Variant
	message     string
	description string
}

// statusState composes one task status from the conditions that can hold at once, so one
// writer clearing its condition never erases another's. Safe for concurrent use.
type statusState struct {
	// mu guards every field and is held across the status write to serialize writers.
	mu sync.Mutex
	// running reports whether the task has started and not yet stopped.
	running bool
	// terminal is the final status after the task stops or fails; it outranks all others.
	terminal *statusContribution
	// runtimeErr, when set, is the latest node runtime error. Sticky until stop.
	runtimeErr *statusContribution
	// conflicts is the current set of out-ranked write channels.
	conflicts []controlConflict
	// reported, when set, is the latest status reported by an stl module.
	reported *statusContribution
	// lastRendered is the last status written, used to skip identical writes.
	lastRendered *task.Status
}

// top returns the highest-precedence active condition, or nil when only the running
// baseline applies. Callers must hold mu.
func (s *statusState) top() *statusContribution {
	switch {
	case s.terminal != nil:
		return s.terminal
	case s.runtimeErr != nil:
		return s.runtimeErr
	case len(s.conflicts) > 0:
		msg, desc := controlWarning(s.conflicts)
		return &statusContribution{variant: status.VariantWarning, message: msg, description: desc}
	case s.reported != nil:
		return s.reported
	default:
		return nil
	}
}

// renderStatus composes the current status. Callers must hold t.status.mu.
func (t *impl) renderStatus() task.Status {
	s := &t.status
	stat := task.Status{
		Key:     task.OntologyID(t.task.Key).String(),
		Name:    t.task.Name,
		Time:    telem.Now(),
		Details: task.StatusDetails{Task: t.task.Key, Running: s.running && s.terminal == nil},
	}
	if c := s.top(); c != nil {
		stat.Variant, stat.Message, stat.Description = c.variant, c.message, c.description
	} else {
		stat.Variant, stat.Message = status.VariantSuccess, "Task started successfully"
	}
	return stat
}

// statusEqual reports whether two rendered statuses are equal ignoring Time.
func statusEqual(a, b task.Status) bool {
	return a.Variant == b.Variant &&
		a.Message == b.Message &&
		a.Description == b.Description &&
		a.Details.Running == b.Details.Running
}

// updateStatus applies mutate then writes the recomposed status, skipping the write when
// the render is unchanged. It holds the lock across the write so writers cannot interleave.
func (t *impl) updateStatus(ctx context.Context, mutate func(s *statusState)) {
	t.status.mu.Lock()
	defer t.status.mu.Unlock()
	mutate(&t.status)
	stat := t.renderStatus()
	if t.status.lastRendered != nil && statusEqual(*t.status.lastRendered, stat) {
		return
	}
	if err := status.NewWriter[task.StatusDetails](t.factoryCfg.Status, nil).Set(ctx, &stat); err != nil {
		t.factoryCfg.L.Error(
			"failed to set status for Arc task",
			zap.Uint64("key", uint64(t.task.Key)),
			zap.String("name", t.task.Name),
			zap.Error(err),
		)
		return
	}
	t.status.lastRendered = &stat
}

// resetStatus clears the composed status for a fresh start after a prior run stopped.
func (t *impl) resetStatus() {
	t.status.mu.Lock()
	defer t.status.mu.Unlock()
	t.status.running = false
	t.status.terminal = nil
	t.status.runtimeErr = nil
	t.status.conflicts = nil
	t.status.reported = nil
	t.status.lastRendered = nil
}

func (t *impl) setRunning(ctx context.Context, running bool) {
	t.updateStatus(ctx, func(s *statusState) { s.running = running })
}

func (t *impl) setConflicts(ctx context.Context, conflicts []controlConflict) {
	t.updateStatus(ctx, func(s *statusState) { s.conflicts = conflicts })
}

// setTerminal records the final stop/fail status; the first write wins so a shutdown
// cannot mask the failure that triggered it. Callers must hold mu.
func (s *statusState) setTerminal(variant status.Variant, message string) {
	if s.terminal == nil {
		s.terminal = &statusContribution{variant: variant, message: message}
	}
}

func (t *impl) setTerminal(ctx context.Context, variant status.Variant, message string) {
	t.updateStatus(ctx, func(s *statusState) { s.setTerminal(variant, message) })
}

func (t *impl) reporter() taskreporter.Reporter {
	return func(ctx context.Context, variant status.Variant, message string) {
		t.updateStatus(ctx, func(s *statusState) {
			s.reported = &statusContribution{
				variant: variant,
				message: fmt.Sprintf("[%s] %s", t.task.Name, message),
			}
		})
	}
}

func (t *impl) setRuntimeError(ctx context.Context, nodeKey string, err error) {
	nodeType := nodeKey
	if n, ok := t.prog.Program.Nodes.Find(nodeKey); ok {
		nodeType = n.Type
	}
	t.updateStatus(ctx, func(s *statusState) {
		s.runtimeErr = &statusContribution{
			variant:     status.VariantWarning,
			message:     fmt.Sprintf("Runtime error in %s", nodeType),
			description: err.Error(),
		}
	})
}

// clearRuntimeError drops a runtime error once a tick runs clean, so a recovered error
// stops masking a later control conflict.
func (t *impl) clearRuntimeError(ctx context.Context) {
	t.updateStatus(ctx, func(s *statusState) { s.runtimeErr = nil })
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
	// task is the owning task, used to reconcile per-tick runtime-error status.
	task      *impl
	startTime telem.TimeStamp
	scheduler *scheduler.Scheduler
	writeKeys channel.Keys
	subject   control.Subject
	state     state
}

func (d *dataRuntime) next(
	ctx context.Context,
	res framer.StreamerResponse,
	reason node.RunReason,
) error {
	d.task.tickErrored = false
	d.state.channel.Ingest(res.Frame.ToStorage())
	d.scheduler.Next(ctx, telem.Since(d.startTime), reason)
	if !d.task.tickErrored {
		d.task.clearRuntimeError(ctx)
	}
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
			Command: framer.WriterCommandWrite,
		}
		return signal.SendUnderContext(ctx, d.Out.Inlet(), req)
	}
	return nil
}

func (d *dataRuntime) flushAuthorityChanges(ctx context.Context) error {
	changes := d.state.authority.Flush()
	if len(changes) == 0 {
		return nil
	}
	cfg := framer.WriterConfig{}
	for _, change := range changes {
		authority := control.Authority(change.Authority)
		if change.Channel != nil {
			cfg.Keys = append(cfg.Keys, channel.Key(*change.Channel))
			cfg.Authorities = append(cfg.Authorities, authority)
		} else {
			cfg.Keys = append(cfg.Keys, d.writeKeys...)
			for range d.writeKeys {
				cfg.Authorities = append(cfg.Authorities, authority)
			}
		}
	}
	req := framer.WriterRequest{Command: framer.WriterCommandSetAuthority, Config: cfg}
	return signal.SendUnderContext(ctx, d.Out.Inlet(), req)
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
