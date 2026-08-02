// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constant

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var symbolName = "constant"

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes: the `constant` builtin installed at root scope. The symbol
// is Internal — it is emitted by graph-mode lowering of literal flow nodes,
// not called directly from user source.
func NewSymbols() []*symbol.Symbol {
	constraint := types.NumericConstraint()
	typeVar := types.Variable("T", &constraint)
	return []*symbol.Symbol{
		{
			Name:     symbolName,
			Kind:     symbol.KindFunction,
			Exec:     symbol.ExecFlow,
			Internal: true,
			Type: types.Function(types.FunctionProperties{
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: typeVar}},
				Inputs:  types.Params{{Name: "value", Type: typeVar}},
			}),
			Trigger: symbol.TriggerOnly,
		},
	}
}

// Host is the runtime host-side support for the constant builtin: a node
// factory only. No WASM bindings, no per-program state.
type Host struct{}

// NewHost constructs a constant Host.
func NewHost() *Host { return &Host{} }

func (*Host) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName {
		return nil, query.ErrNotFound
	}
	return &constant{
		State:       cfg.State,
		value:       cfg.Node.Inputs[0].Value,
		isEntryNode: cfg.Node.IsEntryNode(cfg.Program.Edges),
	}, nil
}

type constant struct {
	*node.State
	clock       telem.MonoClock
	value       any
	isEntryNode bool
	initialized bool
}

var _ node.Node = (*constant)(nil)

func (c *constant) Next(ctx node.Context) {
	if c.isEntryNode {
		if c.initialized {
			return
		}
		c.initialized = true
	}
	d := c.Output(0)
	// A var-bound value input emits the referenced variable's latest value;
	// otherwise the configured value is emitted.
	if s := c.RefInput(0); c.RefSourced(0) && s.Len() > 0 {
		if s.DataType.IsVariable() {
			*d = telem.NewSeriesFromAny(string(s.At(-1)), s.DataType)
		} else {
			*d = telem.Series{
				DataType: s.DataType,
				Data:     append([]byte(nil), s.At(-1)...),
			}
		}
	} else {
		*d = telem.NewSeriesFromAny(c.value, d.DataType)
	}
	t := c.OutputTime(0)
	*t = telem.NewSeriesV[telem.TimeStamp](c.clock.Now())
	ctx.MarkChanged(0)
}

func (c *constant) Reset() { c.initialized = false }
