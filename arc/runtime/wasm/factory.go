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

	compilerbindings "github.com/synnaxlabs/arc/compiler/bindings"
	"github.com/synnaxlabs/arc/module"
	node2 "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	runtimebindings "github.com/synnaxlabs/arc/runtime/wasm/bindings"
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
		ir:    cfg.Node,
		state: cfg.State,
		wasm: WrapFunction(
			wasmFn,
			w.wasm.Memory(),
			irFn.Outputs,
			cfg.Module.OutputMemoryBases[cfg.Node.Type],
		),
		inputs:  make([]uint64, len(irFn.Inputs.Keys)),
		offsets: make([]int, len(irFn.Outputs.Keys)),
	}
	return n, nil
}

type FactoryConfig struct {
	Module module.Module
	State  *state.State
}

func NewFactory(ctx context.Context, cfg FactoryConfig) (node2.Factory, error) {
	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
	bindings := compilerbindings.NewBindings()
	arcRuntime := runtimebindings.NewRuntime(cfg.State, nil)
	runtimebindings.BindRuntime(arcRuntime, bindings)
	if err := bindings.Bind(ctx, runtime); err != nil {
		return nil, err
	}
	wasmModule, err := runtime.Instantiate(ctx, cfg.Module.WASM)
	if err != nil {
		return nil, err
	}
	arcRuntime.SetMemory(wasmModule.Memory())
	return &factory{wasm: wasmModule}, nil
}
