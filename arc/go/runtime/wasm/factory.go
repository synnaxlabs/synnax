// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm

import (
	"context"
	"strings"

	node2 "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/query"
	"github.com/tetratelabs/wazero/api"
)

type factory struct {
	wasm api.Module
}

func (w *factory) Create(_ context.Context, cfg node2.Config) (node2.Node, error) {
	irFn, ok := cfg.Module.Functions.Find(cfg.Node.Type)
	if !ok {
		return nil, query.ErrNotFound
	}
	wasmFn := w.wasm.ExportedFunction(cfg.Node.Type)
	// Count incoming edges to this node
	incomingEdges := 0
	for _, edge := range cfg.Module.Edges {
		if edge.Target.Node == cfg.Node.Key {
			incomingEdges++
		}
	}
	// Entry nodes have no incoming edges and are not expression nodes.
	// They should only execute once per stage entry.
	isEntryNode := !strings.HasPrefix(cfg.Node.Key, "expression_") && incomingEdges == 0
	n := &nodeImpl{
		Node: cfg.State,
		ir:   cfg.Node,
		wasm: WrapFunction(
			wasmFn,
			w.wasm.Memory(),
			irFn.Outputs,
			cfg.Module.OutputMemoryBases[cfg.Node.Type],
		),
		inputs:      make([]uint64, len(irFn.Inputs)),
		offsets:     make([]int, len(irFn.Outputs)),
		isEntryNode: isEntryNode,
	}
	return n, nil
}

type FactoryConfig struct {
	Module *Module
	State  *state.State
}

func NewFactory(mod *Module) (node2.Factory, error) {
	return &factory{wasm: mod.wasmModule}, nil
}
