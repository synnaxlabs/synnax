// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package time

import (
	"context"
	"reflect"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/lsp/doc"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/tetratelabs/wazero"
)

const (
	intervalSymbolName = "interval"
	waitSymbolName     = "wait"
	nowSymbolName      = "now"
	periodInputParam   = "period"
	durationInputParam = "duration"
	name               = "time"
)

var (
	intervalDoc = doc.New(
		doc.Paragraph("Fires repeatedly at a specified period."),
		doc.Divider(),
		doc.Code("arc", "time.interval{period=1s} -> tick"),
	)
	waitDoc = doc.New(
		doc.Paragraph("Fires once after a specified duration."),
		doc.Divider(),
		doc.Code("arc", "time.wait{duration=500ms} -> done"),
	)
	nowDoc = doc.New(
		doc.Paragraph("Returns the current timestamp."),
		doc.Divider(),
		doc.Code("arc", "t := time.now()"),
	)
	moduleDoc = doc.New(
		doc.Paragraph(
			"Time-related primitives: reading the current timestamp, firing periodic intervals, and waiting fixed durations.",
		),
	)
)

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes: the time module plus the deprecated bare aliases (interval,
// wait, now) whose Deprecated fields point at the canonical members.
func NewSymbols() []*symbol.Symbol {
	interval := &symbol.Symbol{
		Name: intervalSymbolName,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecFlow,
		Type: types.Function(types.FunctionProperties{
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
			Inputs:  types.Params{{Name: periodInputParam, Type: types.TimeSpan()}},
		}),
		Trigger:          symbol.TriggerOnly,
		Doc:              intervalDoc,
		AnalyzeArguments: rejectNonPositiveSpan(periodInputParam),
	}
	wait := &symbol.Symbol{
		Name: waitSymbolName,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecFlow,
		Type: types.Function(types.FunctionProperties{
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
			Inputs:  types.Params{{Name: durationInputParam, Type: types.TimeSpan()}},
		}),
		Trigger:          symbol.TriggerOnly,
		Doc:              waitDoc,
		AnalyzeArguments: rejectNonPositiveSpan(durationInputParam),
	}
	now := &symbol.Symbol{
		Name: nowSymbolName,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecBoth,
		Type: types.Function(types.FunctionProperties{
			Outputs: types.Params{
				{Name: ir.DefaultOutputParam, Type: types.TimeStamp()},
			},
		}),
		Trigger: symbol.TriggerOnly,
		Doc:     nowDoc,
	}
	mod := &symbol.Symbol{Name: name, Kind: symbol.KindModule, Doc: moduleDoc}
	mod.AddChild(interval, wait, now)
	intervalBare := *interval
	intervalBare.Deprecated = interval
	waitBare := *wait
	waitBare.Deprecated = wait
	nowBare := *now
	nowBare.Deprecated = now
	return []*symbol.Symbol{mod, &intervalBare, &waitBare, &nowBare}
}

// rejectNonPositiveSpan returns an AnalyzeArguments hook that rejects a
// literal non-positive span bound to the named param. Non-literal arguments
// pass through; the runtime guard covers their live values.
func rejectNonPositiveSpan(param string) symbol.ArgumentsHook {
	return func(diags *diagnostics.Diagnostics, args []symbol.Argument) {
		for _, arg := range args {
			if arg.Name != param && (arg.Name != "" || arg.Index != 0) {
				continue
			}
			if arg.Expr == nil || !parser.IsLiteral(arg.Expr) {
				continue
			}
			lit := parser.GetLiteral(arg.Expr)
			if lit == nil || lit.NumericLiteral() == nil {
				continue
			}
			// Type mismatches are the analyzer's to report, not the hook's.
			parsed, err := literal.Parse(lit, types.TimeSpan())
			if err != nil {
				continue
			}
			var span telem.TimeSpan
			switch v := parsed.Value.(type) {
			case telem.TimeSpan:
				span = v
			case int64:
				span = telem.TimeSpan(v)
			default:
				continue
			}
			if parser.IsNegatedLiteral(arg.Expr) {
				span = -span
			}
			if span <= 0 {
				diags.Add(diagnostics.Errorf(
					arg.AST, "%s must be positive, got %s", param, span,
				))
			}
		}
	}
}

// Host is the runtime host-side support for the time module: it registers
// the `now` WASM host function and acts as the node factory for interval
// and wait.
type Host struct {
	// now is the current cycle's stamp, set by the runtime loop before each pass.
	// The `now` WASM binding is called from guest code, which has no node Context
	// to read, so the value is pushed here instead.
	now telem.TimeStamp
}

// SetNow binds the cycle stamp the `now` host function returns for the coming pass.
// The runtime loop calls it before every Scheduler.Next.
func (h *Host) SetNow(now telem.TimeStamp) { h.now = now }

// NewHost registers the time module's `now` WASM host binding with rt and
// returns a Host handle that acts as the node factory for interval / wait.
func NewHost(ctx context.Context, rt wazero.Runtime) (*Host, error) {
	h := &Host{}
	if rt == nil {
		return h, nil
	}
	builder := rt.NewHostModuleBuilder(name)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) uint64 {
			return uint64(h.now)
		}).Export("now")
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *Host) Create(cfg node.Config) (node.Node, error) {
	switch cfg.Node.Type {
	case intervalSymbolName:
		if err := validateSpan(cfg.Node.Inputs, periodInputParam); err != nil {
			return nil, err
		}
		return &Interval{State: cfg.State}, nil

	case waitSymbolName:
		if err := validateSpan(cfg.Node.Inputs, durationInputParam); err != nil {
			return nil, err
		}
		return &Wait{State: cfg.State, startTime: -1}, nil

	case nowSymbolName:
		return &Now{State: cfg.State}, nil

	default:
		return nil, query.ErrNotFound
	}
}

// validateSpan returns query.ErrNotFound when the named span input is missing, and a
// validation error when its value is not a telem.TimeSpan or, unless var-bound, is not
// positive. The runtime guard covers the live values of var-bound inputs.
func validateSpan(inputs types.Params, name string) error {
	p, ok := inputs.Get(name)
	if !ok {
		return query.ErrNotFound
	}
	span, ok := p.Value.(telem.TimeSpan)
	if !ok {
		return validate.PathedError(
			errors.Wrapf(
				validate.ErrInvalidType,
				"expected type telem.TimeSpan, received %s",
				reflect.TypeOf(p.Value).Name(),
			),
			p.Name,
		)
	}
	if p.Type.Kind == types.KindVarRef || span > 0 {
		return nil
	}
	return validate.PathedError(
		errors.Wrapf(validate.ErrValidation, "must be positive, got %s", span),
		p.Name,
	)
}

// liveSpan returns the named input's current span: the referenced variable's
// latest value when var-bound, else the value stamped at compile time.
func liveSpan(s *node.State, name string) telem.TimeSpan {
	return telem.TimeSpan(s.NumericInput[int64](name))
}

// spanGuard guards a live timer span against non-positive values. It reports
// the first offense only, so a parked node does not re-report on every pass.
type spanGuard struct{ reported bool }

// usable reports whether span can drive a deadline. A non-positive span
// reports a validation error naming label and returns false.
func (g *spanGuard) usable(ctx node.Context, span telem.TimeSpan, label string) bool {
	if span > 0 {
		g.reported = false
		return true
	}
	if !g.reported {
		ctx.ReportError(errors.Wrapf(
			validate.ErrValidation, "%s must be positive, got %s", label, span,
		))
		g.reported = true
	}
	return false
}

func (g *spanGuard) reset() { g.reported = false }

// Interval is a node that fires repeatedly at a specified period.
type Interval struct {
	*node.State
	// lastFired is the elapsed time of the last scheduled fire. It is valid only when
	// started is true.
	lastFired telem.TimeSpan
	started   bool
	guard     spanGuard
}

func (i *Interval) Init(_ node.Context) {}

func (i *Interval) Next(ctx node.Context) {
	period := liveSpan(i.State, periodInputParam)
	// A non-positive period would keep the deadline permanently in the past,
	// spinning the scheduler loop. Park without a deadline until the node runs
	// again, such as when its stage is entered again.
	if !i.guard.usable(ctx, period, "interval period") {
		return
	}
	// The first usable run puts the first fire at now, so the interval fires on its
	// first timer tick.
	if !i.started {
		i.lastFired = ctx.Elapsed - period
		i.started = true
	}
	// A timer never fires before its deadline. An early wake re-arms it.
	if ctx.Reason != node.ReasonTimerTick || ctx.Elapsed-i.lastFired < period {
		ctx.MarkSelfChanged()
		ctx.SetDeadline(i.lastFired + period)
		return
	}
	// Fires count from the schedule, so a late fire does not delay the fires after
	// it. A fire more than one period late skips the fires it missed.
	behind := ctx.Elapsed - i.lastFired
	i.lastFired += behind - behind%period
	ctx.MarkSelfChanged()
	ctx.SetDeadline(i.lastFired + period)
	output := i.Output(0)
	outputTime := i.OutputTime(0)
	output.Resize(1)
	outputTime.Resize(1)
	output.SetValueAt(0, uint8(1))
	outputTime.SetValueAt(0, ctx.Now)
	i.Emit(ctx, 0)
}

// Reset resets the interval so it fires immediately on the next timer tick.
func (i *Interval) Reset(ctx node.Context) {
	i.State.Reset(ctx)
	i.started = false
	i.guard.reset()
}

// Wait is a one-shot timer that fires once after a specified duration.
type Wait struct {
	*node.State
	startTime telem.TimeSpan
	fired     bool
	guard     spanGuard
}

func (w *Wait) Init(_ node.Context) {}

func (w *Wait) Next(ctx node.Context) {
	if w.fired {
		return
	}
	duration := liveSpan(w.State, durationInputParam)
	// A non-positive duration is a configuration error, not an instant fire.
	// Park without a deadline until the node runs again, such as when its stage
	// is entered again.
	if !w.guard.usable(ctx, duration, "wait duration") {
		return
	}
	if w.startTime < 0 {
		w.startTime = ctx.Elapsed
	}
	// A timer never fires before its deadline. An early wake re-arms it.
	if ctx.Reason != node.ReasonTimerTick ||
		ctx.Elapsed-w.startTime < duration {
		ctx.MarkSelfChanged()
		ctx.SetDeadline(w.startTime + duration)
		return
	}
	w.fired = true
	output := w.Output(0)
	outputTime := w.OutputTime(0)
	output.Resize(1)
	outputTime.Resize(1)
	output.SetValueAt(0, uint8(1))
	outputTime.SetValueAt(0, ctx.Now)
	w.Emit(ctx, 0)
}

func (w *Wait) Reset(ctx node.Context) {
	w.State.Reset(ctx)
	w.startTime = -1
	w.fired = false
	w.guard.reset()
}

// Now outputs the current wall-clock timestamp when triggered.
type Now struct {
	*node.State
}

func (n *Now) Init(_ node.Context) {}

func (n *Now) Next(ctx node.Context) {
	ts := ctx.Now
	output := n.Output(0)
	outputTime := n.OutputTime(0)
	output.Resize(1)
	outputTime.Resize(1)
	output.SetValueAt(0, ts)
	outputTime.SetValueAt(0, ts)
	n.Emit(ctx, 0)
}

func (n *Now) Reset(ctx node.Context) { n.State.Reset(ctx) }
