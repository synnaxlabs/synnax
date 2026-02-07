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
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

const (
	intervalSymbolName  = "interval"
	waitSymbolName      = "wait"
	nowSymbolName       = "now"
	periodConfigParam   = "period"
	durationConfigParam = "duration"
)

// MinTolerance is the minimum tolerance for timing comparisons,
// handling OS scheduling jitter even when BaseInterval is very small.
const MinTolerance = 5 * telem.Millisecond

// unsetBaseInterval is the sentinel value indicating BaseInterval hasn't been set yet.
const unsetBaseInterval = telem.TimeSpanMax

var SymbolResolver = symbol.MapResolver{
	intervalSymbolName: {
		Name: intervalSymbolName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
			Config:  types.Params{{Name: periodConfigParam, Type: types.TimeSpan()}},
		}),
	},
	waitSymbolName: {
		Name: waitSymbolName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
			Config:  types.Params{{Name: durationConfigParam, Type: types.TimeSpan()}},
		}),
	},
	nowSymbolName: {
		Name: nowSymbolName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.TimeStamp()}},
		}),
	},
}

type Module struct {
	// BaseInterval is the GCD of all timer periods, used for scheduler timing.
	BaseInterval telem.TimeSpan
}

var _ stl.Module = (*Module)(nil)

func NewModule() *Module {
	return &Module{BaseInterval: unsetBaseInterval}
}

var compilerModResolver = &symbol.ModuleResolver{
	Name: "time",
	Members: symbol.MapResolver{
		"now": {
			Name: "now",
			Type: types.Function(types.FunctionProperties{
				Outputs: types.Params{{Name: "ts", Type: types.I64()}},
			}),
		},
	},
}

func (m *Module) Resolve(ctx context.Context, name string) (symbol.Symbol, error) {
	if sym, err := SymbolResolver.Resolve(ctx, name); err == nil {
		return sym, nil
	}
	return compilerModResolver.Resolve(ctx, name)
}

func (m *Module) Search(ctx context.Context, term string) ([]symbol.Symbol, error) {
	syms1, _ := SymbolResolver.Search(ctx, term)
	syms2, _ := compilerModResolver.Search(ctx, term)
	return append(syms1, syms2...), nil
}

func (m *Module) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	switch cfg.Node.Type {
	case intervalSymbolName:
		periodParam, ok := cfg.Node.Config.Get(periodConfigParam)
		if !ok {
			return nil, query.ErrNotFound
		}
		period, err := parseTime(periodParam.Value, periodParam.Name)
		if err != nil {
			return nil, err
		}
		m.updateBaseInterval(period)
		return &Interval{
			Node:      cfg.State,
			period:    period,
			lastFired: -period,
		}, nil

	case waitSymbolName:
		durationParam, ok := cfg.Node.Config.Get(durationConfigParam)
		if !ok {
			return nil, query.ErrNotFound
		}
		duration, err := parseTime(durationParam.Value, durationParam.Name)
		if err != nil {
			return nil, err
		}
		m.updateBaseInterval(duration)
		return &Wait{
			Node:      cfg.State,
			duration:  duration,
			startTime: -1,
			fired:     false,
		}, nil

	default:
		return nil, query.ErrNotFound
	}
}

func (m *Module) BindTo(_ context.Context, rt stl.HostRuntime) error {
	stl.MustExport(rt, "time", "now", func(_ context.Context) uint64 {
		return uint64(telem.Now())
	})
	return nil
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

func (m *Module) updateBaseInterval(span telem.TimeSpan) {
	if m.BaseInterval == unsetBaseInterval {
		m.BaseInterval = span
	} else {
		m.BaseInterval = telem.TimeSpan(gcd(int64(m.BaseInterval), int64(span)))
	}
}

func gcd(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func parseTime(v any, name string) (telem.TimeSpan, error) {
	span, ok := v.(telem.TimeSpan)
	if !ok {
		return 0, errors.Wrapf(
			validate.ErrValidation,
			"configuration parameter %s has invalid type, expected type telem.TimeSpan, received %s",
			name,
			reflect.TypeOf(v).Name(),
		)
	}
	return span, nil
}

// Interval is a node that fires repeatedly at a specified period.
type Interval struct {
	*state.Node
	period    telem.TimeSpan
	lastFired telem.TimeSpan
}

func (i *Interval) Init(_ node.Context) {}

func (i *Interval) Next(ctx node.Context) {
	if ctx.Reason != node.ReasonTimerTick ||
		ctx.Elapsed-i.lastFired < i.period-ctx.Tolerance {
		return
	}
	i.lastFired = ctx.Elapsed
	ctx.MarkChanged(ir.DefaultOutputParam)
	output := i.Output(0)
	outputTime := i.OutputTime(0)
	output.Resize(1)
	outputTime.Resize(1)
	telem.SetValueAt[uint8](*output, 0, uint8(1))
	telem.SetValueAt[telem.TimeStamp](*outputTime, 0, telem.TimeStamp(ctx.Elapsed))
}

// Wait is a one-shot timer that fires once after a specified duration.
type Wait struct {
	*state.Node
	duration  telem.TimeSpan
	startTime telem.TimeSpan
	fired     bool
}

func (w *Wait) Init(_ node.Context) {}

func (w *Wait) Next(ctx node.Context) {
	if ctx.Reason != node.ReasonTimerTick || w.fired {
		return
	}
	if w.startTime < 0 {
		w.startTime = ctx.Elapsed
	}
	if ctx.Elapsed-w.startTime < w.duration-ctx.Tolerance {
		return
	}
	w.fired = true
	output := w.Output(0)
	outputTime := w.OutputTime(0)
	output.Resize(1)
	outputTime.Resize(1)
	telem.SetValueAt[uint8](*output, 0, uint8(1))
	telem.SetValueAt[telem.TimeStamp](*outputTime, 0, telem.TimeStamp(ctx.Elapsed))
	ctx.MarkChanged(ir.DefaultOutputParam)
}

func (w *Wait) Reset() {
	w.Node.Reset()
	w.startTime = -1
	w.fired = false
}
