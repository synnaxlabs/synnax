// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package stagemod provides the stage_entry STL module for Arc runtime stage
// transitions. Entry nodes listen for activation signals and trigger stage
// transitions via the node context's ActivateStage callback.
package stage

import (
	"context"

	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
)

const (
	EntryNodeName        = "stage_entry"
	EntryActivationParam = "activate"
)

// EntryNodeInputs defines the input parameters for stage entry nodes.
var EntryNodeInputs = types.Params{{
	Name:  EntryActivationParam,
	Type:  types.U8(),
	Value: uint8(0),
}}

var (
	entryNode = symbol.Symbol{
		Name: EntryNodeName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs: EntryNodeInputs,
		}),
	}
	// SymbolResolver provides the stage_entry symbol for the Arc analyzer.
	SymbolResolver = symbol.MapResolver{EntryNodeName: entryNode}
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
	if cfg.Node.Type != EntryNodeName {
		return nil, query.ErrNotFound
	}
	return &entry{Node: cfg.State}, nil
}

func (m *Module) BindTo(_ context.Context, _ stl.HostRuntime) error {
	return nil
}

type entry struct {
	*state.Node
}

var _ node.Node = (*entry)(nil)

func (s *entry) Next(ctx node.Context) {
	ctx.ActivateStage()
}
