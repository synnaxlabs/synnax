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

	compilerbindings "github.com/synnaxlabs/arc/compiler/bindings"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/runtime/state"
	runtimebindings "github.com/synnaxlabs/arc/runtime/wasm/bindings"
	"github.com/synnaxlabs/x/errors"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type Module struct {
	runtime wazero.Runtime
	module  api.Module
}

func (m *Module) Close() error {
	// Wazero runtime context is only used for logging and value lookup, which we
	// don't use. Creating the context here means we can maintain the io.Closer interface
	// for the module, which means a simpler shutdown callstack.
	ctx := context.TODO()
	c := errors.NewCatcher(errors.WithAggregation())
	c.Exec(func() error { return m.module.Close(ctx) })
	c.Exec(func() error { return m.runtime.Close(ctx) })
	return c.Error()
}

type ModuleConfig struct {
	Module module.Module
	State  *state.State
}

func OpenModule(ctx context.Context, cfg ModuleConfig) (*Module, error) {
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
	if err != nil {
		return nil, err
	}
	return &Module{module: wasmModule, runtime: runtime}, nil
}
