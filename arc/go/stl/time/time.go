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

// MinTolerance is the minimum tolerance for timing comparisons,
// handling OS scheduling jitter even when BaseInterval is very small.
const MinTolerance = 5 * telem.Millisecond

// unsetBaseInterval is the sentinel value indicating BaseInterval hasn't been set yet.
const unsetBaseInterval = telem.TimeSpanMax

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
	// BaseInterval is the GCD of known timer periods, declared and literal
	// reassignments. Its only use is deriving the timing tolerance.
	BaseInterval telem.TimeSpan
	// clock provides monotonically increasing timestamps, avoiding
	// duplicate values on platforms with coarse clock resolution.
	clock telem.MonoClock
}

// NewHost registers the time module's `now` WASM host binding with rt and
// returns a Host handle that acts as the node factory for interval / wait.
func NewHost(ctx context.Context, rt wazero.Runtime) (*Host, error) {
	h := &Host{BaseInterval: unsetBaseInterval}
	if rt == nil {
		return h, nil
	}
	builder := rt.NewHostModuleBuilder(name)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context) uint64 {
			return uint64(h.clock.Now())
		}).Export("now")
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *Host) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	switch cfg.Node.Type {
	case intervalSymbolName:
		periodParam, ok := cfg.Node.Inputs.Get(periodInputParam)
		if !ok {
			return nil, query.ErrNotFound
		}
		period, err := parseTime(periodParam.Value, periodParam.Name)
		if err != nil {
			return nil, err
		}
		if err = validateStaticSpan(period, periodParam); err != nil {
			return nil, err
		}
		h.updateBaseInterval(period)
		h.foldReassignedSpans(cfg, periodParam)
		return &Interval{
			State:     cfg.State,
			lastFired: -period,
			clock:     &h.clock,
		}, nil

	case waitSymbolName:
		durationParam, ok := cfg.Node.Inputs.Get(durationInputParam)
		if !ok {
			return nil, query.ErrNotFound
		}
		duration, err := parseTime(durationParam.Value, durationParam.Name)
		if err != nil {
			return nil, err
		}
		if err = validateStaticSpan(duration, durationParam); err != nil {
			return nil, err
		}
		h.updateBaseInterval(duration)
		h.foldReassignedSpans(cfg, durationParam)
		return &Wait{
			State:     cfg.State,
			startTime: -1,
			fired:     false,
			clock:     &h.clock,
		}, nil

	case nowSymbolName:
		return &Now{State: cfg.State, clock: &h.clock}, nil

	default:
		return nil, query.ErrNotFound
	}
}

// CalculateTolerance returns the timing tolerance for the given base interval.
func CalculateTolerance(baseInterval telem.TimeSpan) telem.TimeSpan {
	if baseInterval == unsetBaseInterval {
		return MinTolerance
	}
	halfInterval := baseInterval / 2
	if halfInterval < MinTolerance {
		return MinTolerance
	}
	return halfInterval
}

// foldReassignedSpans folds the literal reassignment values of a var-bound
// timer param into BaseInterval, so tolerance tracks the fastest known period.
func (h *Host) foldReassignedSpans(cfg node.Config, p types.Param) {
	if p.Type.Kind != types.KindVarRef {
		return
	}
	for _, e := range cfg.Program.Edges {
		if e.Target.Node != p.Type.Name {
			continue
		}
		src, ok := cfg.Program.Nodes.Find(e.Source.Node)
		if !ok || src.Type != "constant" {
			continue
		}
		v, ok := src.Inputs.Get("value")
		if !ok || v.Value == nil {
			continue
		}
		if span, err := parseTime(v.Value, v.Name); err == nil {
			h.updateBaseInterval(span)
		}
	}
}

func (h *Host) updateBaseInterval(span telem.TimeSpan) {
	// A non-positive span is not a real timer period. Folding it in would
	// poison the GCD and drive the loop cadence off a parked timer.
	if span <= 0 {
		return
	}
	if h.BaseInterval == unsetBaseInterval {
		h.BaseInterval = span
	} else {
		h.BaseInterval = telem.TimeSpan(gcd(int64(h.BaseInterval), int64(span)))
	}
}

func gcd(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// validateStaticSpan rejects a non-positive span stamped at compile time.
// Var-bound params are exempt: the runtime guard covers their live values.
func validateStaticSpan(span telem.TimeSpan, p types.Param) error {
	if p.Type.Kind == types.KindVarRef || span > 0 {
		return nil
	}
	return validate.PathedError(
		errors.Wrapf(validate.ErrValidation, "must be positive, got %s", span),
		p.Name,
	)
}

func parseTime(v any, name string) (telem.TimeSpan, error) {
	span, ok := v.(telem.TimeSpan)
	if !ok {
		return 0, validate.PathedError(
			errors.Wrapf(
				validate.ErrInvalidType,
				"expected type telem.TimeSpan, received %s",
				reflect.TypeOf(v).Name(),
			),
			name,
		)
	}
	return span, nil
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
	lastFired telem.TimeSpan
	clock     *telem.MonoClock
	guard     spanGuard
}

func (i *Interval) Init(_ node.Context) {}

func (i *Interval) Next(ctx node.Context) {
	period := liveSpan(i.State, periodInputParam)
	// A non-positive period would keep the deadline permanently in the past,
	// spinning the scheduler loop. Park without a deadline instead; a later
	// reassignment to a positive value resumes the timer.
	if !i.guard.usable(ctx, period, "interval period") {
		return
	}
	if ctx.Reason != node.ReasonTimerTick {
		ctx.MarkSelfChanged()
		ctx.SetDeadline(i.lastFired + period)
		return
	}
	if ctx.Elapsed-i.lastFired < period-ctx.Tolerance {
		ctx.MarkSelfChanged()
		ctx.SetDeadline(i.lastFired + period)
		return
	}
	i.lastFired = ctx.Elapsed
	ctx.MarkSelfChanged()
	ctx.SetDeadline(i.lastFired + period)
	ctx.MarkChanged(0)
	output := i.Output(0)
	outputTime := i.OutputTime(0)
	output.Resize(1)
	outputTime.Resize(1)
	output.SetValueAt[uint8](0, uint8(1))
	outputTime.SetValueAt[telem.TimeStamp](0, i.clock.Now())
}

// Reset resets the interval so it fires immediately on the next timer tick.
func (i *Interval) Reset() {
	i.State.Reset()
	i.lastFired = -liveSpan(i.State, periodInputParam)
	i.guard.reset()
}

// Wait is a one-shot timer that fires once after a specified duration.
type Wait struct {
	*node.State
	startTime telem.TimeSpan
	fired     bool
	clock     *telem.MonoClock
	guard     spanGuard
}

func (w *Wait) Init(_ node.Context) {}

func (w *Wait) Next(ctx node.Context) {
	if w.fired {
		return
	}
	duration := liveSpan(w.State, durationInputParam)
	// A non-positive duration is a configuration error, not an instant fire:
	// park instead. Timing stays anchored to startTime, so recovery re-checks
	// the live duration against the original activation.
	if !w.guard.usable(ctx, duration, "wait duration") {
		return
	}
	if w.startTime < 0 {
		w.startTime = ctx.Elapsed
	}
	ctx.SetDeadline(w.startTime + duration)
	if ctx.Reason != node.ReasonTimerTick {
		ctx.MarkSelfChanged()
		return
	}
	if ctx.Elapsed-w.startTime < duration-ctx.Tolerance {
		ctx.MarkSelfChanged()
		return
	}
	w.fired = true
	output := w.Output(0)
	outputTime := w.OutputTime(0)
	output.Resize(1)
	outputTime.Resize(1)
	output.SetValueAt[uint8](0, uint8(1))
	outputTime.SetValueAt[telem.TimeStamp](0, w.clock.Now())
	ctx.MarkChanged(0)
}

func (w *Wait) Reset() {
	w.State.Reset()
	w.startTime = -1
	w.fired = false
	w.guard.reset()
}

// Now outputs the current wall-clock timestamp when triggered.
type Now struct {
	*node.State
	clock *telem.MonoClock
}

func (n *Now) Init(_ node.Context) {}

func (n *Now) Next(ctx node.Context) {
	ts := n.clock.Now()
	output := n.Output(0)
	outputTime := n.OutputTime(0)
	output.Resize(1)
	outputTime.Resize(1)
	output.SetValueAt[telem.TimeStamp](0, ts)
	outputTime.SetValueAt[telem.TimeStamp](0, ts)
	ctx.MarkChanged(0)
}

func (n *Now) Reset() { n.State.Reset() }
