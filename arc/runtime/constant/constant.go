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
	symbol2 "github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var (
	symbolName = "constant"
	symbol     = symbol2.Symbol{
		Name: symbolName,
		Kind: symbol2.KindStage,
		Type: ir.Stage{
			Config: types.Params{
				Keys:   []string{"value"},
				Values: []types.Type{types.NewTypeVariable("T", types.NumericConstraint{})},
			},
			Outputs: types.Params{
				Keys:   []string{ir.DefaultOutputParam},
				Values: []types.Type{types.NewTypeVariable("T", types.NumericConstraint{})},
			},
		},
	}
	SymbolResolver = symbol2.MapResolver{symbolName: symbol}
)

type constant struct {
	output ir.Handle
	state  *state.State
	value  any
}

func (c constant) Init(_ context.Context, onOutputChange func(output string)) {
	outputState := c.state.Outputs[c.output]
	outputState.Data = telem.NewSeriesFromAny(c.value, outputState.Data.DataType)
	outputState.Time = telem.NewSeriesV[telem.TimeStamp](telem.Now())
	c.state.Outputs[c.output] = outputState
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
		value:  cfg.Node.ConfigValues["value"],
	}, nil
}

func NewFactory() node.Factory { return &constantFactory{} }
