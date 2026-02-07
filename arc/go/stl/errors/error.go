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

	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/tetratelabs/wazero/api"
)

var symResolver = &symbol.ModuleResolver{
	Name: "error",
	Members: symbol.MapResolver{
		"panic": {
			Name: "panic",
			Type: types.Function(types.FunctionProperties{
				Inputs: types.Params{{Name: "ptr", Type: types.I32()}, {Name: "len", Type: types.I32()}},
			}),
		},
	},
}

type Module struct {
	memory api.Memory
}

func NewModule() *Module { return &Module{} }

func (m *Module) SetMemory(mem api.Memory) { m.memory = mem }

func (m *Module) Resolve(ctx context.Context, name string) (symbol.Symbol, error) {
	return symResolver.Resolve(ctx, name)
}

func (m *Module) Search(ctx context.Context, term string) ([]symbol.Symbol, error) {
	return symResolver.Search(ctx, term)
}

func (m *Module) Create(_ context.Context, _ node.Config) (node.Node, error) {
	return nil, query.ErrNotFound
}

func (m *Module) BindTo(_ context.Context, rt stl.HostRuntime) error {
	stl.MustExport(rt, "error", "panic",
		func(_ context.Context, ptr uint32, length uint32) {
			if m.memory == nil {
				panic("arc panic (memory not set)")
			}
			msg, ok := m.memory.Read(ptr, length)
			if !ok {
				panic("arc panic (message unreadable)")
			}
			panic("arc panic: " + string(msg))
		})
	return nil
}

var _ stl.Module = (*Module)(nil)
