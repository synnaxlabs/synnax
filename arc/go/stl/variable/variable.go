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

func (h *Host) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName && cfg.Node.Type != statefulSymbolName {
		return nil, query.ErrNotFound
	}
	return &variable{State: cfg.State, persistent: cfg.Node.Type == statefulSymbolName}, nil
}

// variable holds the value of its most-recently-fired feeder input,
// last-write-wins. The unedged f0 param carries the declaration seed.
type variable struct {
	*node.State
	clock      telem.MonoClock
	persistent bool
}

var _ node.Node = (*variable)(nil)

// Reset re-seeds a `:=` variable on scope entry. A `$=` variable persists.
func (v *variable) Reset() {
	if v.persistent {
		return
	}
	v.State.Reset()
}

func (v *variable) Next(ctx node.Context) {
	data, ok := v.LastChanged()
	if !ok {
		return
	}
	d := v.Output(0)
	*d = data
	*v.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](v.clock.Now())
	ctx.MarkChanged(0)
}
