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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/lsp/doc"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// TrueOutputParam and FalseOutputParam name the two outputs of the select
// node. The order they appear in the symbol's output params determines the
// ordinal each output uses in MarkChanged / IsOutputTruthy calls — see
// TrueOutputIdx and FalseOutputIdx below.
const (
	TrueOutputParam  = "true"
	FalseOutputParam = "false"
)

// TrueOutputIdx and FalseOutputIdx are the ordinals the runtime
// implementation passes to MarkChanged and IsOutputTruthy. They mirror
// the declaration order of TrueOutputParam and FalseOutputParam in the
// symbol's Outputs and must stay in sync with it.
const (
	TrueOutputIdx  = 0
	FalseOutputIdx = 1
)

var (
	symbolName = "select"
	symbolDoc  = doc.New(
		doc.Paragraph("Routes input values to 'true' or 'false' outputs. Values equal to 1 are routed to the true output; all others to false."),
		doc.Divider(),
		doc.Code("arc", "flag -> select{} -> {\n    true: open_valve,\n    false: shut_valve\n}"),
	)
)

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes: the `select` builtin installed at root scope.
func NewSymbols() []*symbol.Symbol {
	return []*symbol.Symbol{
		{
			Name: symbolName,
			Kind: symbol.KindFunction,
			Exec: symbol.ExecFlow,
			Type: types.Function(types.FunctionProperties{
				Inputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.U8()},
				},
				Outputs: types.Params{
					{Name: TrueOutputParam, Type: types.U8()},
					{Name: FalseOutputParam, Type: types.U8()},
				},
			}),
			Trigger: symbol.TriggerInput(ir.DefaultOutputParam),
			Doc:     symbolDoc,
		},
	}
}

// Host is the runtime host-side support for `select`: a node factory only.
// No WASM bindings, no per-program state.
type Host struct{}

// NewHost constructs a selector Host.
func NewHost() *Host { return &Host{} }

func (h *Host) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName {
		return nil, query.ErrNotFound
	}
	inputIdx, err := cfg.State.ResolveInput(ir.DefaultOutputParam)
	if err != nil {
		return nil, err
	}
	return &selectNode{State: cfg.State, inputIdx: inputIdx}, nil
}

type selectNode struct {
	*node.State
	inputIdx int
}

func (s *selectNode) Next(ctx node.Context) {
	if !s.RefreshInputs() {
		return
	}
	data := s.Input(s.inputIdx)
	time := s.InputTime(s.inputIdx)
	if data.Len() == 0 {
		return
	}
	var trueCount int64
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
			falseData.Data[falseIdx] = 1
			telem.CopyValue(*falseTime, time, falseIdx, i)
			falseIdx++
		}
	}
	if trueData.Len() > 0 {
		ctx.MarkChanged(0)
	}
	if falseData.Len() > 0 {
		ctx.MarkChanged(1)
	}
}
