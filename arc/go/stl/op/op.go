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
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	telemOp "github.com/synnaxlabs/x/telem/op"
)

// Host is the runtime host-side support for operators: a node factory for
// arithmetic, comparison, logical, and unary ops. Operators have no WASM
// host bindings and no per-program state.
type Host struct{}

// NewHost constructs an op Host.
func NewHost() *Host { return &Host{} }

func resolveBinary(s *node.State) (lhs, rhs int, err error) {
	if lhs, err = s.ResolveInput(ir.LHSInputParam); err != nil {
		return lhs, rhs, err
	}
	rhs, err = s.ResolveInput(ir.RHSInputParam)
	return lhs, rhs, err
}

func (*Host) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cat, ok := typedOps[cfg.Node.Type]; ok {
		lhsIdx, rhsIdx, err := resolveBinary(cfg.State)
		if err != nil {
			return nil, err
		}
		op := cat[cfg.State.Input(lhsIdx).DataType]
		return &binary{State: cfg.State, lhsIdx: lhsIdx, rhsIdx: rhsIdx, op: op}, nil
	}
	if opFn, ok := logicalOps[cfg.Node.Type]; ok {
		lhsIdx, rhsIdx, err := resolveBinary(cfg.State)
		if err != nil {
			return nil, err
		}
		return &binary{State: cfg.State, lhsIdx: lhsIdx, rhsIdx: rhsIdx, op: opFn}, nil
	}
	if unOpFn, ok := unaryOps[cfg.Node.Type]; ok {
		inputIdx, err := cfg.State.ResolveInput(ir.DefaultInputParam)
		if err != nil {
			return nil, err
		}
		return &unary{State: cfg.State, inputIdx: inputIdx, op: unOpFn}, nil
	}
	return nil, query.ErrNotFound
}

type binary struct {
	*node.State
	op     telemOp.Binary
	lhsIdx int
	rhsIdx int
}

func (n *binary) Next(ctx node.Context) {
	if !n.RefreshInputs() {
		return
	}
	lhs, rhs := n.Input(n.lhsIdx), n.Input(n.rhsIdx)
	n.op(lhs, rhs, n.Output(0))
	*n.OutputTime(0) = n.InputTime(n.lhsIdx)
	alignment := lhs.Alignment + rhs.Alignment
	timeRange := telem.TimeRange{Start: lhs.TimeRange.Start, End: lhs.TimeRange.End}
	if !rhs.TimeRange.Start.IsZero() &&
		(timeRange.Start.IsZero() || rhs.TimeRange.Start < timeRange.Start) {
		timeRange.Start = rhs.TimeRange.Start
	}
	if rhs.TimeRange.End > timeRange.End {
		timeRange.End = rhs.TimeRange.End
	}
	n.Output(0).Alignment = alignment
	n.Output(0).TimeRange = timeRange
	n.OutputTime(0).Alignment = alignment
	n.OutputTime(0).TimeRange = timeRange
	ctx.MarkChanged(0)
}

type unary struct {
	*node.State
	op       telemOp.Unary
	inputIdx int
}

var _ node.Node = (*unary)(nil)

func (n *unary) Next(ctx node.Context) {
	if !n.RefreshInputs() {
		return
	}
	input := n.Input(n.inputIdx)
	n.op(input, n.Output(0))
	*n.OutputTime(0) = n.InputTime(n.inputIdx)
	n.Output(0).Alignment = input.Alignment
	n.Output(0).TimeRange = input.TimeRange
	n.OutputTime(0).Alignment = input.Alignment
	n.OutputTime(0).TimeRange = input.TimeRange
	ctx.MarkChanged(0)
}
