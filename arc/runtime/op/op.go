// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package op

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem/op"
)

type binaryOperator struct {
	state   *state.State
	inputs  struct{ lhs, rhs ir.Edge }
	output  ir.Handle
	compare op.Binary
}

func (n *binaryOperator) Init(context.Context, func(string)) {}

func (n *binaryOperator) Next(_ context.Context, markChanged func(output string)) {
	seriesA := n.state.Outputs[n.inputs.lhs.Source]
	seriesB := n.state.Outputs[n.inputs.rhs.Source]
	aLength := seriesA.Len()
	bLength := seriesB.Len()
	if aLength == 0 || bLength == 0 {
		return
	}
	outputSeries := n.state.Outputs[n.output]
	n.compare(seriesA, seriesB, &outputSeries)
	n.state.Outputs[n.output] = outputSeries
	markChanged(ir.DefaultOutputParam)
}

type operatorFactory struct{}

func (o operatorFactory) Create(cfg node.Config) (node.Node, error) {
	opCat, ok := comparisons[cfg.Node.Type]
	if !ok {
		return nil, query.NotFound
	}
	lhsEdge := cfg.Module.GetEdgeByTargetHandle(ir.Handle{Node: cfg.Node.Key, Param: lhsParam})
	rhsEdge := cfg.Module.GetEdgeByTargetHandle(ir.Handle{Node: cfg.Node.Key, Param: rhsParam})
	outputHandle := ir.Handle{Node: cfg.Node.Key, Param: ir.DefaultOutputParam}
	seriesA := cfg.State.Outputs[lhsEdge.Source]
	comp := opCat[seriesA.DataType]
	n := &binaryOperator{state: cfg.State, output: outputHandle, compare: comp}
	n.inputs.lhs = lhsEdge
	n.inputs.rhs = rhsEdge
	return n, nil
}

func NewFactory() node.Factory { return operatorFactory{} }
