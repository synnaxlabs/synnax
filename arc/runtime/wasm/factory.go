// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/arc/module"
	node2 "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/x/query"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type factory struct {
	wasm api.Module
}

func (w *factory) Create(_ context.Context, cfg node2.Config) (node2.Node, error) {
	irFn, ok := cfg.Module.Functions.Find(cfg.Node.Type)
	if !ok {
		return nil, query.NotFound
	}
	wasmFn := w.wasm.ExportedFunction(cfg.Node.Type)
	n := &node{
		ir: cfg.Node,
		wasm: WrapFunction(
			wasmFn,
			w.wasm.Memory(),
			irFn.Outputs,
			cfg.Module.OutputMemoryBases[cfg.Node.Type],
		),
		state:  cfg.State,
		inputs: make([]uint64, len(irFn.Outputs.Keys)),
	}
	n.edges.input = cfg.Module.Edges.GetInputs(cfg.Node.Key)
	n.edges.output = cfg.Module.Edges.GetOutputs(cfg.Node.Key)
	return n, nil
}

type FactoryConfig struct {
	Module module.Module
}

func NewFactory(ctx context.Context, cfg FactoryConfig) (node2.Factory, error) {
	runtime := wazero.NewRuntime(ctx)
	wasmModule, err := runtime.Instantiate(ctx, cfg.Module.WASM)
	if err != nil {
		return nil, err
	}
	return &factory{wasm: wasmModule}, nil
}
