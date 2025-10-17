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
		Type: types.Function(types.FunctionProperties{
			Inputs: &types.Params{
				Keys:   []string{ir.DefaultOutputParam},
				Values: []types.Type{types.U8()},
			},
			Outputs: &types.Params{
				Keys:   []string{"true", "false"},
				Values: []types.Type{types.U8(), types.U8()},
			},
		}),
	}
	SymbolResolver = symbol.MapResolver{symbolName: symbolSelect}
)

type selectNode struct {
	snode *state.Node
}

func (s *selectNode) Init(context.Context, func(string)) {}

func (s *selectNode) Next(_ context.Context, onOutput func(string)) {
	data := s.snode.InputData(0)
	time := s.snode.InputTime(0)
	if data.Len() == 0 {
		return
	}
	var trueCount int64 = 0
	for _, v := range data.Data {
		if v == 1 {
			trueCount++
		}
	}
	falseCount := data.Len() - trueCount
	trueData := s.snode.OutputData(0)
	trueTime := s.snode.OutputTime(0)
	falseData := s.snode.OutputData(1)
	falseTime := s.snode.OutputTime(1)
	trueData.Resize(trueCount)
	trueTime.Resize(trueCount)
	falseData.Resize(falseCount)
	falseTime.Resize(falseCount)
	var trueIdx, falseIdx int64 = 0, 0
	for i := range data.Data {
		timeOffset := int64(i) * 8
		if data.Data[i] == 1 {
			trueData.Data[trueIdx] = 1
			copy(trueTime.Data[trueIdx*8:(trueIdx+1)*8], time.Data[timeOffset:timeOffset+8])
			trueIdx++
		} else {
			falseData.Data[falseIdx] = 0
			copy(falseTime.Data[falseIdx*8:(falseIdx+1)*8], time.Data[timeOffset:timeOffset+8])
			falseIdx++
		}
	}
	if trueData.Len() > 0 {
		onOutput(trueParamName)
	}
	if falseData.Len() > 0 {
		onOutput(falseParamName)
	}
}

type selectFactory struct{}

func (s *selectFactory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName {
		return nil, query.NotFound
	}
	return &selectNode{snode: cfg.State}, nil
}

func NewFactory() node.Factory {
	return &selectFactory{}
}
