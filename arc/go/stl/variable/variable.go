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

// Create dispatches on shape: a value-carrying first input makes a register;
// an edge-fed one an exprRead deref.
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
// derivation index. Writes are last-wins; the unedged f0 holds the initial value.
type register struct {
	*node.State
	clock    telem.MonoClock
	stateful bool
}

var _ node.Node = (*register)(nil)

// Reset restores a `:=` variable's initial value on scope entry. A `$=`
// persists. The value is emitted immediately, superseding any pending feeder.
func (v *register) Reset() {
	if v.stateful {
		return
	}
	v.AbsorbInputs()
	v.Output(0).CopyFrom(v.Input(0))
	telem.SetSeriesV(v.OutputTime(0), v.clock.Now())
}

func (v *register) Next(ctx node.Context) {
	data, ok := v.LastChanged()
	if !ok {
		return
	}
	// Feeders reuse their output buffers in place; the register value must not alias them.
	v.Output(0).CopyFrom(data)
	telem.SetSeriesV(v.OutputTime(0), v.clock.Now())
	ctx.MarkChanged(0)
}

// exprRead derefs its variable's dispatcher: values pending at a re-point
// predate it and are absorbed, so only later inputs fire.
type exprRead struct {
	*node.State
	clock  telem.MonoClock
	selIdx int
}

var _ node.Node = (*exprRead)(nil)

// Reset absorbs pending inputs, initial sel included, so only post-entry values fire.
func (v *exprRead) Reset() { v.AbsorbInputs() }

// Next re-points on sel first: the dispatcher never emits on a sel-only change,
// so a value paired with a fresh sel predates the re-point.
func (v *exprRead) Next(ctx node.Context) {
	repointed := false
	if v.selIdx >= 0 {
		if _, ok := v.ConsumeInput(v.selIdx); ok {
			repointed = true
		}
	}
	data, ok := v.ConsumeInput(0)
	if !ok || repointed {
		return
	}
	v.Output(0).CopyFrom(data)
	telem.SetSeriesV(v.OutputTime(0), v.clock.Now())
	ctx.MarkChanged(0)
}
