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

	"github.com/tetratelabs/wazero"
)

// WazeroHostRuntime implements stl.HostRuntime using wazero's
// HostModuleBuilder API. Each unique WASM module name gets its own builder.
type WazeroHostRuntime struct {
	ctx      context.Context
	rt       wazero.Runtime
	builders map[string]wazero.HostModuleBuilder
}

// NewWazeroHostRuntime creates a new WazeroHostRuntime for the given wazero runtime.
func NewWazeroHostRuntime(ctx context.Context, rt wazero.Runtime) *WazeroHostRuntime {
	return &WazeroHostRuntime{
		ctx:      ctx,
		rt:       rt,
		builders: make(map[string]wazero.HostModuleBuilder),
	}
}

func (w *WazeroHostRuntime) Export(wasmModule, name string, impl any) error {
	builder, ok := w.builders[wasmModule]
	if !ok {
		builder = w.rt.NewHostModuleBuilder(wasmModule)
		w.builders[wasmModule] = builder
	}
	builder.NewFunctionBuilder().WithFunc(impl).Export(name)
	return nil
}

// Instantiate finalizes all host module builders, making the host functions
// available for WASM module instantiation.
func (w *WazeroHostRuntime) Instantiate() error {
	for _, builder := range w.builders {
		if _, err := builder.Instantiate(w.ctx); err != nil {
			return err
		}
	}
	return nil
}
