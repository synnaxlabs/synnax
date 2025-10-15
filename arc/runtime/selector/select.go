// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package selector

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
)

var (
	trueParamName  = "true"
	falseParamName = "false"
	symbolName     = "select"
	symbolSelect   = symbol.Symbol{
		Name: symbolName,
		Kind: symbol.KindFunction,
		Type: ir.Stage{
			Params: types.Params{
				Keys:   []string{ir.DefaultOutputParam},
				Values: []types.Type{types.U8()},
			},
			Outputs: types.Params{
				Keys:   []string{"true", "false"},
				Values: []types.Type{types.U8(), types.U8()},
			},
		},
	}
	SymbolResolver = symbol.MapResolver{symbolName: symbolSelect}
)

type selectNode struct {
	state   *state.State
	input   ir.Edge
	outputs struct{ true, false ir.Handle }
}

func (s *selectNode) Init(context.Context, func(string)) {}

func (s *selectNode) Next(ctx context.Context, onOutput func(string)) {
	inputSeries := s.state.Outputs[s.Inputs.Source]
	if inputSeries.Data.Len() == 0 {
		return
	}

	// Count true and false values
	var trueCount int64 = 0
	for _, v := range inputSeries.Data.Data {
		if v == 1 {
			trueCount++
		}
	}
	falseCount := inputSeries.Data.Len() - trueCount

	// Allocate output series
	trueOutputSeries := s.state.Outputs[s.outputs.true]
	trueOutputSeries.Data.Resize(trueCount)
	trueOutputSeries.Time.Resize(trueCount)

	falseOutputSeries := s.state.Outputs[s.outputs.false]
	falseOutputSeries.Data.Resize(falseCount)
	falseOutputSeries.Time.Resize(falseCount)

	var trueIdx, falseIdx int64 = 0, 0
	for i, v := range inputSeries.Data.Data {
		timeOffset := int64(i) * 8
		if v == 1 {
			trueOutputSeries.Data.Data[trueIdx] = 1
			copy(trueOutputSeries.Time.Data[trueIdx*8:(trueIdx+1)*8], inputSeries.Time.Data[timeOffset:timeOffset+8])
			trueIdx++
		} else {
			falseOutputSeries.Data.Data[falseIdx] = 0
			copy(falseOutputSeries.Time.Data[falseIdx*8:(falseIdx+1)*8], inputSeries.Time.Data[timeOffset:timeOffset+8])
			falseIdx++
		}
	}

	s.state.Outputs[s.outputs.true] = trueOutputSeries
	if trueCount > 0 {
		onOutput(trueParamName)
	}

	s.state.Outputs[s.outputs.false] = falseOutputSeries
	if falseCount > 0 {
		onOutput(falseParamName)
	}
}

type selectFactory struct {
}

func (s *selectFactory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName {
		return nil, query.NotFound
	}
	inputEdge := cfg.Module.GetEdgeByTargetHandle(ir.Handle{
		Node:  cfg.Node.Key,
		Param: ir.DefaultOutputParam,
	})
	trueHandle := ir.Handle{Node: cfg.Node.Key, Param: trueParamName}
	falseHandle := ir.Handle{Node: cfg.Node.Key, Param: falseParamName}
	n := &selectNode{state: cfg.State, input: inputEdge}
	n.outputs.true = trueHandle
	n.outputs.false = falseHandle
	return n, nil
}

func NewFactory() node.Factory {
	return &selectFactory{}
}
