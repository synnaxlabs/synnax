// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/telem/op"
)

type binary struct {
	*state.Node
	op op.Binary
}

func (n *binary) Next(ctx node.Context) {
	if !n.RefreshInputs() {
		return
	}
	lhs, rhs := n.Input(0), n.Input(1)
	n.op(lhs, rhs, n.Output(0))
	*n.OutputTime(0) = n.InputTime(0)
	// Propagate alignment and time range from inputs to output
	alignment := lhs.Alignment + rhs.Alignment
	timeRange := telem.TimeRange{Start: lhs.TimeRange.Start, End: lhs.TimeRange.End}
	if !rhs.TimeRange.Start.IsZero() && (timeRange.Start.IsZero() || rhs.TimeRange.Start < timeRange.Start) {
		timeRange.Start = rhs.TimeRange.Start
	}
	if rhs.TimeRange.End > timeRange.End {
		timeRange.End = rhs.TimeRange.End
	}
	n.Output(0).Alignment = alignment
	n.Output(0).TimeRange = timeRange
	n.OutputTime(0).Alignment = alignment
	n.OutputTime(0).TimeRange = timeRange
	ctx.MarkChanged(ir.DefaultOutputParam)
}

type unary struct {
	*state.Node
	op op.Unary
}

var _ node.Node = (*unary)(nil)

func (n *unary) Next(ctx node.Context) {
	if !n.RefreshInputs() {
		return
	}
	input := n.Input(0)
	n.op(input, n.Output(0))
	*n.OutputTime(0) = n.InputTime(0)
	// Propagate alignment and time range from input to output
	n.Output(0).Alignment = input.Alignment
	n.Output(0).TimeRange = input.TimeRange
	n.OutputTime(0).Alignment = input.Alignment
	n.OutputTime(0).TimeRange = input.TimeRange
	ctx.MarkChanged(ir.DefaultOutputParam)
}

type operatorFactory struct{}

func (o operatorFactory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	cat, ok := typedOps[cfg.Node.Type]
	if ok {
		return &binary{Node: cfg.State, op: cat[cfg.State.Input(0).DataType]}, nil
	}
	opFn, ok := logicalOps[cfg.Node.Type]
	if ok {
		return &binary{Node: cfg.State, op: opFn}, nil
	}
	unCat, ok := typedUnaryOps[cfg.Node.Type]
	if ok {
		return &unary{Node: cfg.State, op: unCat[cfg.State.Input(0).DataType]}, nil
	}
	unOpFn, ok := unaryOps[cfg.Node.Type]
	if ok {
		return &unary{Node: cfg.State, op: unOpFn}, nil
	}
	return nil, query.NotFound
}

func NewFactory() node.Factory { return operatorFactory{} }
