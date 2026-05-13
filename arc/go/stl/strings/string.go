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
	"strconv"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

var SymbolResolver = &symbol.ModuleResolver{
	Name: "string",
	Members: symbol.MapResolver{
		"from_literal": {
			Name:     "from_literal",
			Kind:     symbol.KindFunction,
			Exec:     symbol.ExecWASM,
			Internal: true,
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "ptr", Type: types.I32()}, {Name: "len", Type: types.I32()}},
				Outputs: types.Params{{Name: "handle", Type: types.I32()}},
			}),
		},
		"concat": {
			Name:     "concat",
			Kind:     symbol.KindFunction,
			Exec:     symbol.ExecWASM,
			Internal: true,
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "a", Type: types.String()}, {Name: "b", Type: types.String()}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
			}),
		},
		"equal": {
			Name:     "equal",
			Kind:     symbol.KindFunction,
			Exec:     symbol.ExecWASM,
			Internal: true,
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "a", Type: types.String()}, {Name: "b", Type: types.String()}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I32()}},
			}),
		},
		"len": {
			Name:     "len",
			Kind:     symbol.KindFunction,
			Exec:     symbol.ExecWASM,
			Internal: true,
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "handle", Type: types.String()}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
			}),
		},
		"from_i32": {
			Name:     "from_i32",
			Kind:     symbol.KindFunction,
			Exec:     symbol.ExecWASM,
			Internal: true,
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "value", Type: types.I32()}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
			}),
		},
		"from_u32": {
			Name:     "from_u32",
			Kind:     symbol.KindFunction,
			Exec:     symbol.ExecWASM,
			Internal: true,
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "value", Type: types.U32()}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
			}),
		},
		"from_i64": {
			Name:     "from_i64",
			Kind:     symbol.KindFunction,
			Exec:     symbol.ExecWASM,
			Internal: true,
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "value", Type: types.I64()}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
			}),
		},
		"from_u64": {
			Name:     "from_u64",
			Kind:     symbol.KindFunction,
			Exec:     symbol.ExecWASM,
			Internal: true,
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "value", Type: types.U64()}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
			}),
		},
		"from_f32": {
			Name:     "from_f32",
			Kind:     symbol.KindFunction,
			Exec:     symbol.ExecWASM,
			Internal: true,
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "value", Type: types.F32()}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
			}),
		},
		"from_f64": {
			Name:     "from_f64",
			Kind:     symbol.KindFunction,
			Exec:     symbol.ExecWASM,
			Internal: true,
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "value", Type: types.F64()}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
			}),
		},
	},
}

type Module struct {
	strings *ProgramState
	memory  api.Memory
}

func (m *Module) SetMemory(memory api.Memory) { m.memory = memory }

func NewModule(
	ctx context.Context,
	s *ProgramState,
	rat wazero.Runtime,
	memory api.Memory,
) (*Module, error) {
	m := &Module{strings: s, memory: memory}
	if rat == nil {
		return m, nil
	}
	builder := rat.NewHostModuleBuilder("string")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, ptr uint32, length uint32) uint32 {
			if m.memory == nil {
				return 0
			}
			data, ok := m.memory.Read(ptr, length)
			if !ok {
				return 0
			}
			return s.Create(string(data))
		}).Export("from_literal")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, h1 uint32, h2 uint32) uint32 {
			s1, ok1 := s.Get(h1)
			s2, ok2 := s.Get(h2)
			if !ok1 || !ok2 {
				return 0
			}
			return s.Create(s1 + s2)
		}).Export("concat")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, h1 uint32, h2 uint32) uint32 {
			s1, ok1 := s.Get(h1)
			s2, ok2 := s.Get(h2)
			if ok1 && ok2 && s1 == s2 {
				return 1
			}
			return 0
		}).Export("equal")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle uint32) uint64 {
			if str, ok := s.Get(handle); ok {
				return uint64(len(str))
			}
			return 0
		}).Export("len")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, v int32) uint32 {
			return s.Create(strconv.FormatInt(int64(v), 10))
		}).Export("from_i32")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, v uint32) uint32 {
			return s.Create(strconv.FormatUint(uint64(v), 10))
		}).Export("from_u32")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, v int64) uint32 {
			return s.Create(strconv.FormatInt(v, 10))
		}).Export("from_i64")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, v uint64) uint32 {
			return s.Create(strconv.FormatUint(v, 10))
		}).Export("from_u64")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, v float32) uint32 {
			return s.Create(strconv.FormatFloat(float64(v), 'g', -1, 32))
		}).Export("from_f32")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, v float64) uint32 {
			return s.Create(strconv.FormatFloat(v, 'g', -1, 64))
		}).Export("from_f64")
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return m, nil
}
