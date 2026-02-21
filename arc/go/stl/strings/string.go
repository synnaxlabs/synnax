// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package strings

import (
	"context"

	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/tetratelabs/wazero/api"
)

var symResolver = &symbol.ModuleResolver{
	Name: "string",
	Members: symbol.MapResolver{
		"from_literal": {
			Name: "from_literal",
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "ptr", Type: types.I32()}, {Name: "len", Type: types.I32()}},
				Outputs: types.Params{{Name: "handle", Type: types.I32()}},
			}),
		},
		"concat": {
			Name: "concat",
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "a", Type: types.I32()}, {Name: "b", Type: types.I32()}},
				Outputs: types.Params{{Name: "result", Type: types.I32()}},
			}),
		},
		"equal": {
			Name: "equal",
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "a", Type: types.I32()}, {Name: "b", Type: types.I32()}},
				Outputs: types.Params{{Name: "result", Type: types.I32()}},
			}),
		},
		"len": {
			Name: "len",
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "handle", Type: types.I32()}},
				Outputs: types.Params{{Name: "length", Type: types.I64()}},
			}),
		},
	},
}

type Module struct {
	strings *state.StringHandleStore
	memory  api.Memory
}

func NewModule(s *state.StringHandleStore) *Module { return &Module{strings: s} }

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
	s := m.strings
	stl.MustExport(rt, "string", "from_literal",
		func(_ context.Context, ptr uint32, length uint32) uint32 {
			if m.memory == nil {
				return 0
			}
			data, ok := m.memory.Read(ptr, length)
			if !ok {
				return 0
			}
			return s.Create(string(data))
		})
	stl.MustExport(rt, "string", "concat",
		func(_ context.Context, h1 uint32, h2 uint32) uint32 {
			s1, ok1 := s.Get(h1)
			s2, ok2 := s.Get(h2)
			if !ok1 || !ok2 {
				return 0
			}
			return s.Create(s1 + s2)
		})
	stl.MustExport(rt, "string", "equal",
		func(_ context.Context, h1 uint32, h2 uint32) uint32 {
			s1, ok1 := s.Get(h1)
			s2, ok2 := s.Get(h2)
			if ok1 && ok2 && s1 == s2 {
				return 1
			}
			return 0
		})
	stl.MustExport(rt, "string", "len",
		func(_ context.Context, handle uint32) uint64 {
			if str, ok := s.Get(handle); ok {
				return uint64(len(str))
			}
			return 0
		})
	return nil
}

var _ stl.Module = (*Module)(nil)
