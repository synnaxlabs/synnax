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

var SymbolResolver = &symbol.ModuleResolver{
	Name: "error",
	Members: symbol.MapResolver{
		"panic": {
			Name: "panic",
			Kind: symbol.KindFunction,
			Exec: symbol.ExecWASM,
			Type: types.Function(types.FunctionProperties{
				Inputs: types.Params{{Name: "ptr", Type: types.I32()}, {Name: "len", Type: types.I32()}},
			}),
		},
	},
}

type Module struct {
	memory api.Memory
}

func NewModule(ctx context.Context, memory api.Memory, rat wazero.Runtime) (*Module, error) {
	mod := &Module{memory: memory}
	if rat == nil {
		return mod, nil
	}
	builder := rat.NewHostModuleBuilder("error")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, ptr uint32, length uint32) {
			if mod.memory == nil {
				panic("arc panic (memory not set)")
			}
			msg, ok := mod.memory.Read(ptr, length)
			if !ok {
				panic("arc panic (message unreadable)")
			}
			panic("arc panic: " + string(msg))
		}).Export("panic")
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return mod, nil
}

func (m *Module) SetMemory(mem api.Memory) { m.memory = mem }
