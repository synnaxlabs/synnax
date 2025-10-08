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
	"github.com/synnaxlabs/arc/runtime/node"
	telem2 "github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

type compare = func(a, b, output telem.Series)
type binaryOperator struct {
	ir      ir.Node
	state   *telem2.State
	inputs  struct{ a, b ir.Edge }
	output  ir.Edge
	compare compare
}

func (n *binaryOperator) Next(ctx context.Context, markChanged func(output string)) {
	seriesA := n.state.Outputs[n.inputs.a.Source]
	seriesB := n.state.Outputs[n.inputs.b.Source]
	aLength := seriesA.Len()
	bLength := seriesB.Len()
	if aLength == 0 || bLength == 0 {
		return
	}
	n.compare(seriesA, seriesB, n.state.Outputs[n.output.Source])
	markChanged(ir.DefaultOutput)
}

type operatorFactory struct{}

func (o operatorFactory) Create(cfg node.Config) (node.Node, error) {
	opCat, ok := comparisons[cfg.Node.Type]
	if !ok {
		return nil, query.NotFound
	}
	edgeA, _ := lo.Find(cfg.Module.Edges, func(item ir.Edge) bool {
		return item.Target.Node == cfg.Node.Key && item.Target.Param == "a"
	})
	edgeB, _ := lo.Find(cfg.Module.Edges, func(item ir.Edge) bool {
		return item.Target.Node == cfg.Node.Key && item.Target.Param == "b"
	})
	outputEdge, _ := lo.Find(cfg.Module.Edges, func(item ir.Edge) bool {
		return item.Source.Node == cfg.Node.Key && item.Source.Param == ir.DefaultOutput
	})
	seriesA := cfg.State.Outputs[edgeA.Source]
	comp := opCat[seriesA.DataType]
	op := &binaryOperator{}
	op.ir = cfg.Node
	op.inputs.a = edgeA
	op.inputs.b = edgeB
	op.output = outputEdge
	op.compare = comp
	op.state = cfg.State
	return op, nil
}

func NewFactory() node.Factory {
	return operatorFactory{}
}
