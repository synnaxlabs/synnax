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
	state *state.Node
	op    op.Binary
}

func (n *binaryOperator) Init(context.Context, func(string)) {}

func (n *binaryOperator) Next(_ context.Context, markChanged func(output string)) {
	if !n.state.RefreshInputs() {
		return
	}
	n.op(n.state.Input(0), n.state.Input(1), n.state.Output(0))
	*n.state.OutputTime(0) = n.state.InputTime(0)
	markChanged(ir.DefaultOutputParam)
}

type operatorFactory struct{}

func (o operatorFactory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	cat, ok := ops[cfg.Node.Type]
	if !ok {
		return nil, query.NotFound
	}
	return &binaryOperator{
		state: cfg.State,
		op:    cat[cfg.State.Input(0).DataType],
	}, nil
}

func NewFactory() node.Factory { return operatorFactory{} }
