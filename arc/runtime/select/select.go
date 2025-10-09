// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package std

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/query"
)

var (
	trueParamName  = "true"
	falseParamName = "false"
	symbolName     = "select"
	symbolSelect   = ir.Symbol{
		Name: symbolName,
		Kind: ir.KindStage,
		Type: ir.Stage{
			Params: ir.NamedTypes{
				Keys:   []string{ir.DefaultInputParam},
				Values: []ir.Type{ir.U8{}},
			},
			Outputs: ir.NamedTypes{
				Keys:   []string{"true", "false"},
				Values: []ir.Type{ir.U8{}, ir.U8{}},
			},
		},
	}
	SymbolResolver = ir.MapResolver{symbolName: symbolSelect}
)

type selectNode struct {
	state   *state.State
	input   ir.Edge
	outputs struct{ true, false ir.Handle }
}

func (s *selectNode) Init(context.Context, func(string)) {}

func (s *selectNode) Next(ctx context.Context, onOutput func(string)) {
	var trueCount int64 = 0
	inputSeries := s.state.Outputs[s.input.Source]
	for _, v := range inputSeries.Data {
		if v == 1 {
			trueCount++
		}
	}
	falseCount := inputSeries.Len() - trueCount

	trueOutputSeries := s.state.Outputs[s.outputs.true]
	trueOutputSeries.Resize(trueCount)
	for i := range trueOutputSeries.Data {
		trueOutputSeries.Data[i] = 1
	}
	s.state.Outputs[s.outputs.true] = trueOutputSeries
	if trueCount > 0 {
		onOutput(trueParamName)
	}

	falseOutputSeries := s.state.Outputs[s.outputs.false]
	falseOutputSeries.Resize(falseCount)
	for i := range falseOutputSeries.Data {
		falseOutputSeries.Data[i] = 0
	}
	s.state.Outputs[s.outputs.false] = falseOutputSeries
	if falseCount > 0 {
		onOutput(falseParamName)
	}
}

type selectFactory struct {
}

func (s *selectFactory) Create(cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName {
		return nil, query.NotFound
	}
	inputEdge := cfg.Module.GetEdgeByTargetHandle(ir.Handle{
		Node:  cfg.Node.Key,
		Param: ir.DefaultInputParam,
	})
	trueHandle := ir.Handle{Node: cfg.Node.Key, Param: trueParamName}
	falseHandle := ir.Handle{Node: cfg.Node.Key, Param: falseParamName}
	n := &selectNode{
		state: cfg.State,
		input: inputEdge,
	}
	n.outputs.true = trueHandle
	n.outputs.false = falseHandle
	return n, nil
}

func NewFactory() node.Factory {
	return &selectFactory{}
}
