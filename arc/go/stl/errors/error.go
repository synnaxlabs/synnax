// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors

import (
	"context"

	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const name = "error"

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes: the error module containing panic. Both module and member
// are Internal — panic is emitted by lowering passes (e.g., out-of-bounds
// checks), not called from user source.
func NewSymbols() []*symbol.Symbol {
	mod := &symbol.Symbol{Name: name, Kind: symbol.KindModule, Internal: true}
	mod.AddChild(&symbol.Symbol{
		Name:     "panic",
		Kind:     symbol.KindFunction,
		Exec:     symbol.ExecWASM,
		Internal: true,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: "ptr", Type: types.I32()}, {Name: "len", Type: types.I32()}},
		}),
	})
	return []*symbol.Symbol{mod}
}

// Host is the runtime host-side support for the error module: it registers
// the panic host function and holds a reference to the WASM guest's
// memory so panic can read its message argument.
type Host struct {
	memory api.Memory
}

// NewHost registers the error module's WASM host binding (panic) with rt.
// memory may be nil at construction time; call SetMemory once the WASM
// guest has been instantiated.
func NewHost(ctx context.Context, rt wazero.Runtime, memory api.Memory) (*Host, error) {
	h := &Host{memory: memory}
	if rt == nil {
		return h, nil
	}
	builder := rt.NewHostModuleBuilder(name)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, ptr uint32, length uint32) {
			if h.memory == nil {
				panic("arc panic (memory not set)")
			}
			msg, ok := h.memory.Read(ptr, length)
			if !ok {
				panic("arc panic (message unreadable)")
			}
			panic("arc panic: " + string(msg))
		}).Export("panic")
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return h, nil
}

// SetMemory updates the WASM guest memory reference used by the panic
// host function. Call this after the guest is instantiated.
func (h *Host) SetMemory(mem api.Memory) { h.memory = mem }
