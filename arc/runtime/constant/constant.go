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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/maps"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var (
	symbolName = "constant"
	symbol     = ir.Symbol{
		Name: symbolName,
		Kind: ir.KindStage,
		Type: ir.Stage{
			Config: maps.Ordered[string, ir.Type]{
				Keys:   []string{"value"},
				Values: []ir.Type{ir.NewTypeVariable("T", ir.NumericConstraint{})},
			},
			Outputs: maps.Ordered[string, ir.Type]{
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

func (c constant) Init(_ context.Context, changed func(output string)) {
	changed(ir.DefaultOutputParam)

	// Get the resolved output type from the state
	outputType := c.state.Outputs[c.output].DataType

	// Convert input value to float64 as intermediate representation
	var floatVal float64
	switch v := c.value.(type) {
	case int:
		floatVal = float64(v)
	case int64:
		floatVal = float64(v)
	case int32:
		floatVal = float64(v)
	case int16:
		floatVal = float64(v)
	case int8:
		floatVal = float64(v)
	case uint64:
		floatVal = float64(v)
	case uint32:
		floatVal = float64(v)
	case uint16:
		floatVal = float64(v)
	case uint8:
		floatVal = float64(v)
	case float64:
		floatVal = v
	case float32:
		floatVal = float64(v)
	}

	// Cast to the appropriate output type and create series
	var outputSeries telem.Series
	switch outputType {
	case telem.Int64T:
		outputSeries = telem.NewSeriesV[int64](int64(floatVal))
	case telem.Int32T:
		outputSeries = telem.NewSeriesV[int32](int32(floatVal))
	case telem.Int16T:
		outputSeries = telem.NewSeriesV[int16](int16(floatVal))
	case telem.Int8T:
		outputSeries = telem.NewSeriesV[int8](int8(floatVal))
	case telem.Uint64T:
		outputSeries = telem.NewSeriesV[uint64](uint64(floatVal))
	case telem.Uint32T:
		outputSeries = telem.NewSeriesV[uint32](uint32(floatVal))
	case telem.Uint16T:
		outputSeries = telem.NewSeriesV[uint16](uint16(floatVal))
	case telem.Uint8T:
		outputSeries = telem.NewSeriesV[uint8](uint8(floatVal))
	case telem.Float64T:
		outputSeries = telem.NewSeriesV[float64](floatVal)
	case telem.Float32T:
		outputSeries = telem.NewSeriesV[float32](float32(floatVal))
	}

	c.state.Outputs[c.output] = outputSeries
}

func (c constant) Next(context.Context, func(output string)) {}

type constantFactory struct{}

func (c *constantFactory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName {
		return nil, query.NotFound
	}
	value, ok := cfg.Node.Config["value"]
	if !ok {
		return nil, errors.Wrap(query.InvalidParameters, "constant node requires 'value' config parameter")
	}
	outputHandle := ir.Handle{Node: cfg.Node.Key, Param: ir.DefaultOutputParam}
	return constant{
		output: outputHandle,
		state:  cfg.State,
		value:  value,
	}, nil
}

func NewFactory() node.Factory { return &constantFactory{} }
