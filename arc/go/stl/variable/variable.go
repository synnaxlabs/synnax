// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package variable

import (
	"bytes"
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

const (
	symbolName         = "variable"
	statefulSymbolName = "stateful_variable"
)

// NewSymbols returns the Internal variable builtins emitted by := / $= lowering.
func NewSymbols() []*symbol.Symbol {
	mk := func(name string) *symbol.Symbol {
		typeVar := types.Variable("T", nil)
		return &symbol.Symbol{
			Name:     name,
			Kind:     symbol.KindFunction,
			Exec:     symbol.ExecFlow,
			Internal: true,
			Type: types.Function(types.FunctionProperties{
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: typeVar}},
				Inputs:  types.Params{{Name: "value", Type: typeVar}},
			}),
			Trigger: symbol.TriggerOnly,
		}
	}
	return []*symbol.Symbol{mk(symbolName), mk(statefulSymbolName)}
}

// Host is the runtime factory for the variable builtin.
type Host struct{}

// NewHost constructs a variable Host.
func NewHost() *Host { return &Host{} }

// Create dispatches on shape: a seeded f0 makes a register; an edge-fed f0 an
// exprRead deref that demuxes derivations.
func (h *Host) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName && cfg.Node.Type != statefulSymbolName {
		return nil, query.ErrNotFound
	}
	if len(cfg.Node.Inputs) > 0 && cfg.Node.Inputs[0].Value == nil {
		selIdx := -1
		if idx, err := cfg.State.ResolveInput("sel"); err == nil {
			selIdx = idx
		}
		return &exprRead{State: cfg.State, selIdx: selIdx}, nil
	}
	return &register{
		State:    cfg.State,
		stateful: cfg.Node.Type == statefulSymbolName,
	}, nil
}

// register holds what its variable is mapped to: a value, a channel key, or a
// derivation index. Writes are last-wins; the unedged f0 carries the seed.
type register struct {
	*node.State
	clock    telem.MonoClock
	stateful bool
}

var _ node.Node = (*register)(nil)

// Reset re-seeds a `:=` variable on scope entry. A `$=` variable persists.
// The seed is emitted immediately, superseding any pending feeder value.
func (v *register) Reset() {
	if v.stateful {
		return
	}
	v.AbsorbInputs()
	*v.Output(0) = v.Input(0).DeepCopy()
	*v.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](v.clock.Now())
}

func (v *register) Next(ctx node.Context) {
	data, ok := v.LastChanged()
	if !ok {
		return
	}
	// Feeders reuse their output buffers in place; the register value must not alias them.
	*v.Output(0) = data.DeepCopy()
	*v.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](v.clock.Now())
	ctx.MarkChanged(0)
}

// exprRead derefs its variable's register: sel carries the active derivation's
// index, and only that feeder's value changes emit.
type exprRead struct {
	*node.State
	clock  telem.MonoClock
	selIdx int
	active int
}

var _ node.Node = (*exprRead)(nil)

// Next re-points on sel first: a re-point is not itself a value, and inactive
// feeders drain silently.
func (v *exprRead) Next(ctx node.Context) {
	if s, ok := v.ConsumeInput(v.selIdx); ok && s.Len() > 0 {
		v.active = int(telem.ValueAt[uint32](s, -1))
	}
	for i := range v.InputCount() {
		if i == v.selIdx {
			continue
		}
		data, ok := v.ConsumeInput(i)
		if !ok || i != v.active {
			continue
		}
		// A derivation only changes when its value does; recomputes are not events.
		if data.DataType == v.Output(0).DataType &&
			bytes.Equal(data.Data, v.Output(0).Data) {
			continue
		}
		*v.Output(0) = data.DeepCopy()
		*v.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](v.clock.Now())
		ctx.MarkChanged(0)
	}
}
