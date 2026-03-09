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
	rnode "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"

	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var (
	symName    = "constant"
	constraint = types.NumericConstraint()
	typeVar    = types.Variable("T", &constraint)
	sym        = symbol.Symbol{
		Name: symName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: typeVar}},
			Config:  types.Params{{Name: "value", Type: typeVar}},
		}),
	}
	SymbolResolver = symbol.MapResolver{symName: sym}
)

type Module struct{}

func NewModule() *Module { return &Module{} }

func (m *Module) Create(_ context.Context, cfg rnode.Config) (rnode.Node, error) {
	if cfg.Node.Type != symName {
		return nil, query.ErrNotFound
	}
	return &node{Node: cfg.State, value: cfg.Node.Config[0].Value}, nil
}

type node struct {
	*state.Node
	value       any
	initialized bool
}

var _ rnode.Node = (*node)(nil)

func (c *node) Next(ctx rnode.Context) {
	if c.initialized {
		return
	}
	c.initialized = true
	d := c.Output(0)
	*d = telem.NewSeriesFromAny(c.value, d.DataType)
	t := c.OutputTime(0)
	*t = telem.NewSeriesV[telem.TimeStamp](telem.Now())
	ctx.MarkChanged(ir.DefaultOutputParam)
}

func (c *node) Reset() { c.initialized = false }
