// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package stage provides the entry node for Arc runtime stage transitions.
// entry nodes listen for activation signals (u8 value of 1) and trigger
// stage transitions via the node context's ActivateStage callback.
package stage

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

const symName = "stage_entry"

var (
	sym = symbol.Symbol{
		Name: symName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: ir.DefaultInputParam, Type: types.U8()}},
		}),
	}
	// SymbolResolver provides the stage_entry symbol for the Arc analyzer.
	SymbolResolver = symbol.MapResolver{symName: sym}
)

// entry is a node that triggers stage transitions when it receives
// an activation signal (input value of u8(1)).
type entry struct {
	*state.Node
	nodeKey string
}

var _ node.Node = (*entry)(nil)

func (s *entry) Next(ctx node.Context) {
	if !s.RefreshInputs() {
		return
	}

	input := s.Input(0)
	if input.Len() == 0 {
		return
	}

	// Activation signal is a u8 with value 1
	if telem.ValueAt[uint8](input, 0) == 1 {
		ctx.ActivateStage(s.nodeKey)
	}
}

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symName {
		return nil, query.NotFound
	}
	return &entry{Node: cfg.State, nodeKey: cfg.Node.Key}, nil
}
