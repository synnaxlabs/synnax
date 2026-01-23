// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package time provides time-based nodes for the Arc runtime.
// Interval fires repeatedly at a specified period.
// Wait fires once after a specified duration and can be reset when a stage is entered.
package time

import (
	"context"
	"math"
	"reflect"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
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
	periodConfigParam   = "period"
	durationConfigParam = "duration"
)

var (
	intervalSymbol = symbol.Symbol{
		Name: intervalSymbolName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
			Config:  types.Params{{Name: periodConfigParam, Type: types.TimeSpan()}},
		}),
	}
	waitSymbol = symbol.Symbol{
		Name: waitSymbolName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
			Config:  types.Params{{Name: durationConfigParam, Type: types.TimeSpan()}},
		}),
	}
	// SymbolResolver provides the interval and wait symbols for the Arc analyzer.
	SymbolResolver = symbol.MapResolver{
		intervalSymbolName: intervalSymbol,
		waitSymbolName:     waitSymbol,
	}
)

// Interval is a node that fires repeatedly at a specified period.
// Each time the period elapses, it outputs u8(1) and marks the output as changed.
type Interval struct {
	*state.Node
	period    telem.TimeSpan
	lastFired telem.TimeSpan
}

// Init performs one-time initialization (no-op for Interval).
func (i *Interval) Init(_ node.Context) {}

// Next checks if the period has elapsed and fires if so.
func (i *Interval) Next(ctx node.Context) {
	if ctx.Elapsed-i.lastFired < i.period {
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
// Unlike Interval, Wait only fires once and can be reset when a stage is entered.
type Wait struct {
	*state.Node
	duration  telem.TimeSpan
	startTime telem.TimeSpan
	fired     bool
}

// Init performs one-time initialization (no-op for Wait).
func (w *Wait) Init(_ node.Context) {}

// Next checks if the duration has elapsed and fires if so (only once).
func (w *Wait) Next(ctx node.Context) {
	// One-shot: if already fired, do nothing
	if w.fired {
		return
	}

	if w.startTime < 0 {
		w.startTime = ctx.Elapsed
	}

	// Check if duration has elapsed
	if ctx.Elapsed-w.startTime < w.duration {
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

// Reset resets the timer so it can fire again.
// Called by the scheduler when a stage containing this node is entered.
// Overrides the embedded state.Node.Reset() to also reset timer-specific state.
func (w *Wait) Reset() {
	w.Node.Reset() // Reset one-shot edge tracking
	w.startTime = -1
	w.fired = false
}

// Factory creates Interval and Wait nodes.
type Factory struct {
	// TimingBase is the GCD of all timer periods, used for scheduler loop timing.
	TimingBase telem.TimeSpan
}

// NewFactory creates a new time Factory.
func NewFactory() *Factory {
	return &Factory{TimingBase: telem.TimeSpan(math.MaxInt64)}
}

// Create constructs an Interval or Wait node from the given configuration.
// Returns query.NotFound if the node type is not "interval" or "wait".
func (f *Factory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
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
		f.updateTimingBase(period)
		return &Interval{
			Node:      cfg.State,
			period:    period,
			lastFired: -period, // Ensures first tick fires immediately
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
		f.updateTimingBase(duration)
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

// updateTimingBase updates the timing base to be the GCD of all timer periods.
func (f *Factory) updateTimingBase(span telem.TimeSpan) {
	if f.TimingBase == telem.TimeSpan(math.MaxInt64) {
		f.TimingBase = span
	} else {
		f.TimingBase = telem.TimeSpan(gcd(int64(f.TimingBase), int64(span)))
	}
}

// gcd computes the greatest common divisor of two integers.
func gcd(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func parseTime(v any, name string) (telem.TimeSpan, error){
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
