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
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	telemOp "github.com/synnaxlabs/x/telem/op"
)

type Module struct{}

var _ stl.Module = (*Module)(nil)

func NewModule() *Module { return &Module{} }

func (m *Module) Resolve(ctx context.Context, name string) (symbol.Symbol, error) {
	return SymbolResolver.Resolve(ctx, name)
}

func (m *Module) Search(ctx context.Context, term string) ([]symbol.Symbol, error) {
	return SymbolResolver.Search(ctx, term)
}

func (m *Module) Create(_ context.Context, cfg node.Config) (node.Node, error) {
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
	return nil, query.ErrNotFound
}

func (m *Module) BindTo(_ context.Context, _ stl.HostRuntime) error {
	return nil
}

type binary struct {
	*state.Node
	op telemOp.Binary
}

func (n *binary) Next(ctx node.Context) {
	if !n.RefreshInputs() {
		return
	}
	lhs, rhs := n.Input(0), n.Input(1)
	n.op(lhs, rhs, n.Output(0))
	*n.OutputTime(0) = n.InputTime(0)
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
	op telemOp.Unary
}

var _ node.Node = (*unary)(nil)

func (n *unary) Next(ctx node.Context) {
	if !n.RefreshInputs() {
		return
	}
	input := n.Input(0)
	n.op(input, n.Output(0))
	*n.OutputTime(0) = n.InputTime(0)
	n.Output(0).Alignment = input.Alignment
	n.Output(0).TimeRange = input.TimeRange
	n.OutputTime(0).Alignment = input.Alignment
	n.OutputTime(0).TimeRange = input.TimeRange
	ctx.MarkChanged(ir.DefaultOutputParam)
}
