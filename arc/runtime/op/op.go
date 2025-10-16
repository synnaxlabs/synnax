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

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/align"
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
	aligner *align.Aligner
}

func (n *binaryOperator) Init(context.Context, func(string)) {}

func (n *binaryOperator) Next(_ context.Context, markChanged func(output string)) {
	dataA := n.state.Outputs[n.inputs.lhs.Source]
	lo.Must0(n.aligner.Add(n.inputs.lhs.Target.Param, dataA.Data, dataA.Time))
	dataB := n.state.Outputs[n.inputs.rhs.Source]
	lo.Must0(n.aligner.Add(n.inputs.rhs.Target.Param, dataB.Data, dataA.Time))
	ops, ok := n.aligner.Next()
	if !ok {
		return
	}
	outputSeries := n.state.Outputs[n.output]
	a := ops.Inputs[n.inputs.lhs.Target.Param]
	b := ops.Inputs[n.inputs.rhs.Target.Param]
	n.compare(a.Data, b.Data, &outputSeries.Data)
	if a.Data.Len() >= b.Data.Len() {
		outputSeries.Time = a.Time
	} else {
		outputSeries.Time = b.Time
	}
	n.state.Outputs[n.output] = outputSeries
	markChanged(ir.DefaultOutputParam)
}

type operatorFactory struct{}

func (o operatorFactory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	opCat, ok := comparisons[cfg.Node.Type]
	if !ok {
		return nil, query.NotFound
	}
	lhsEdge := cfg.Module.Edges.GetByTarget(ir.Handle{Node: cfg.Node.Key, Param: ir.LHSInputParam})
	rhsEdge := cfg.Module.Edges.GetByTarget(ir.Handle{Node: cfg.Node.Key, Param: ir.RHSInputParam})
	outputHandle := ir.Handle{Node: cfg.Node.Key, Param: ir.DefaultOutputParam}
	seriesA := cfg.State.Outputs[lhsEdge.Source]
	comp := opCat[seriesA.Data.DataType]
	n := &binaryOperator{state: cfg.State, output: outputHandle, compare: comp}
	n.aligner = align.NewAligner([]string{ir.LHSInputParam, ir.RHSInputParam})
	n.inputs.lhs = lhsEdge
	n.inputs.rhs = rhsEdge
	return n, nil
}

func NewFactory() node.Factory { return operatorFactory{} }
