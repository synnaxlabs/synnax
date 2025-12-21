// Copyright 2025 Synnax Labs, Inc.
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

type constant struct {
	*state.Node
	value       any
	initialized bool
}

var _ node.Node = (*constant)(nil)

func (c *constant) Next(ctx node.Context) {
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

func (c *constant) Reset() {
	c.initialized = false
}

type constantFactory struct{}

func (c *constantFactory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symName {
		return nil, query.NotFound
	}
	return &constant{Node: cfg.State, value: cfg.Node.Config[0].Value}, nil
}

func NewFactory() node.Factory { return &constantFactory{} }
