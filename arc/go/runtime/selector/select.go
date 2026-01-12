// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/telem"
)

var (
	trueParamName  = "true"
	falseParamName = "false"
	symbolName     = "select"
	symbolSelect   = symbol.Symbol{
		Name: symbolName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{
				{Name: ir.DefaultOutputParam, Type: types.U8()},
			},
			Outputs: types.Params{
				{Name: "true", Type: types.U8()},
				{Name: "false", Type: types.U8()},
			},
		}),
	}
	SymbolResolver = symbol.MapResolver{symbolName: symbolSelect}
)

type selectNode struct{ *state.Node }

func (s *selectNode) Next(ctx node.Context) {
	if !s.RefreshInputs() {
		return
	}
	data := s.Input(0)
	time := s.InputTime(0)
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
	trueData := s.Output(0)
	trueTime := s.OutputTime(0)
	falseData := s.Output(1)
	falseTime := s.OutputTime(1)
	trueData.Resize(trueCount)
	trueTime.Resize(trueCount)
	falseData.Resize(falseCount)
	falseTime.Resize(falseCount)
	// Propagate alignment and time range from input to both outputs
	trueData.Alignment = data.Alignment
	trueData.TimeRange = data.TimeRange
	trueTime.Alignment = data.Alignment
	trueTime.TimeRange = data.TimeRange
	falseData.Alignment = data.Alignment
	falseData.TimeRange = data.TimeRange
	falseTime.Alignment = data.Alignment
	falseTime.TimeRange = data.TimeRange
	var trueIdx, falseIdx = 0, 0
	for i := range data.Data {
		if data.Data[i] == 1 {
			trueData.Data[trueIdx] = 1
			telem.CopyValue(*trueTime, time, trueIdx, i)
			trueIdx++
		} else {
			falseData.Data[falseIdx] = 0
			telem.CopyValue(*falseTime, time, falseIdx, i)
			falseIdx++
		}
	}
	if trueData.Len() > 0 {
		ctx.MarkChanged(trueParamName)
	}
	if falseData.Len() > 0 {
		ctx.MarkChanged(falseParamName)
	}
}

type selectFactory struct{}

func (s *selectFactory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName {
		return nil, query.NotFound
	}
	return &selectNode{Node: cfg.State}, nil
}

func NewFactory() node.Factory {
	return &selectFactory{}
}
