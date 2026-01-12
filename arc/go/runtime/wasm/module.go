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
	wasmRuntime wazero.Runtime
	wasmModule  api.Module
	arcRuntime  *runtimebindings.Runtime
}

func (m *Module) Close() error {
	// Wazero runtime context is only used for logging and value lookup, which we
	// don't use. Creating the context here means we can maintain the io.Closer interface
	// for the module, which means a simpler shutdown callstack.
	ctx := context.TODO()
	c := errors.NewCatcher(errors.WithAggregation())
	c.Exec(func() error { return m.wasmModule.Close(ctx) })
	c.Exec(func() error { return m.wasmRuntime.Close(ctx) })
	return c.Error()
}

// OnCycleEnd implements scheduler.CycleCallback to clear temporary series handles.
// This should be called at the end of each scheduler cycle to prevent memory leaks.
func (m *Module) OnCycleEnd() {
	m.arcRuntime.ClearTemporarySeries()
}

type ModuleConfig struct {
	State  *state.State
	Module module.Module
}

func OpenModule(ctx context.Context, cfg ModuleConfig) (*Module, error) {
	wasmRuntime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
	bindings := compilerbindings.NewBindings()
	arcRuntime := runtimebindings.NewRuntime(cfg.State, nil)
	runtimebindings.BindRuntime(arcRuntime, bindings)
	if err := bindings.Bind(ctx, wasmRuntime); err != nil {
		return nil, err
	}
	wasmModule, err := wasmRuntime.Instantiate(ctx, cfg.Module.WASM)
	if err != nil {
		return nil, err
	}
	arcRuntime.SetMemory(wasmModule.Memory())
	return &Module{
		wasmModule:  wasmModule,
		wasmRuntime: wasmRuntime,
		arcRuntime:  arcRuntime,
	}, nil
}
