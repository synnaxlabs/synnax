// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package stage provides the StageEntry node for Arc runtime stage transitions.
// StageEntry nodes listen for activation signals (u8 value of 1) and trigger
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

// StageEntry is a node that triggers stage transitions when it receives
// an activation signal (input value of u8(1)).
type StageEntry struct {
	*state.Node
	nodeKey string
}

// Init performs one-time initialization (no-op for StageEntry).
func (s *StageEntry) Init(_ node.Context) {}

// Next checks for activation signals and triggers stage transitions.
func (s *StageEntry) Next(ctx node.Context) {
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

// Factory creates StageEntry nodes for "stage_entry" type nodes in the IR.
type Factory struct{}

// NewFactory creates a new StageEntry factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Create constructs a StageEntry node from the given configuration.
// Returns query.NotFound if the node type is not "stage_entry".
func (f *Factory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symName {
		return nil, query.NotFound
	}
	return &StageEntry{
		Node:    cfg.State,
		nodeKey: cfg.Node.Key,
	}, nil
}
