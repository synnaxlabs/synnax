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
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var (
	symbolName = "constant"
	symbol     = ir.Symbol{
		Name: symbolName,
		Kind: ir.KindStage,
		Type: ir.Stage{
			Config: ir.NamedTypes{
				Keys:   []string{"value"},
				Values: []ir.Type{ir.NewTypeVariable("T", ir.NumericConstraint{})},
			},
			Outputs: ir.NamedTypes{
				Keys:   []string{ir.DefaultOutputParam},
				Values: []ir.Type{ir.NewTypeVariable("T", ir.NumericConstraint{})},
			},
		},
	}
	SymbolResolver = ir.MapResolver{symbolName: symbol}
)

type constant struct {
	output ir.Handle
	state  *state.State
	value  any
}

func (c constant) Init(_ context.Context, onOutputChange func(output string)) {
	outputType := c.state.Outputs[c.output].DataType
	c.state.Outputs[c.output] = telem.NewSeriesFromAny(c.value, outputType)
	onOutputChange(ir.DefaultOutputParam)
}

func (c constant) Next(context.Context, func(output string)) {}

type constantFactory struct{}

func (c *constantFactory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName {
		return nil, query.NotFound
	}
	return constant{
		output: ir.Handle{Node: cfg.Node.Key, Param: ir.DefaultOutputParam},
		state:  cfg.State,
		value:  cfg.Node.Config["value"],
	}, nil
}

func NewFactory() node.Factory { return &constantFactory{} }
