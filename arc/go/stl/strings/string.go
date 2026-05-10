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
	"fmt"
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
		"from_i32":      fromSym(types.I32()),
		"from_u32":      fromSym(types.U32()),
		"from_i64":      fromSym(types.I64()),
		"from_u64":      fromSym(types.U64()),
		"from_f32":      fromSym(types.F32()),
		"from_f64":      fromSym(types.F64()),
		"format_i32":    formatSym(types.I32()),
		"format_u32":    formatSym(types.U32()),
		"format_i64":    formatSym(types.I64()),
		"format_u64":    formatSym(types.U64()),
		"format_f32":    formatSym(types.F32()),
		"format_f64":    formatSym(types.F64()),
		"format_string": formatSym(types.String()),
	},
}

func registerFrom[T any](
	builder wazero.HostModuleBuilder,
	s *ProgramState,
	name string,
	conv func(T) string,
) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, v T) uint32 {
			return s.Create(conv(v))
		}).Export(name)
}

func registerFormat[T any](
	builder wazero.HostModuleBuilder,
	m *Module,
	name string,
	coerce func(T) any,
) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, v T, ptr, length uint32) uint32 {
			return m.strings.Create(formatWithSpec(m.memory, ptr, length, coerce(v)))
		}).Export(name)
}

func formatWithSpec(memory api.Memory, ptr, length uint32, value any) string {
	if memory == nil {
		return ""
	}
	spec, ok := memory.Read(ptr, length)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%"+string(spec), value)
}

func fromSym(value types.Type) symbol.Symbol {
	return symbol.Symbol{
		Name:     "from_" + value.String(),
		Kind:     symbol.KindFunction,
		Exec:     symbol.ExecWASM,
		Internal: true,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: "value", Type: value}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
		}),
	}
}

func formatSym(value types.Type) symbol.Symbol {
	return symbol.Symbol{
		Name:     "format_" + value.String(),
		Kind:     symbol.KindFunction,
		Exec:     symbol.ExecWASM,
		Internal: true,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{
				{Name: "value", Type: value},
				{Name: "spec_ptr", Type: types.I32()},
				{Name: "spec_len", Type: types.I32()},
			},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
		}),
	}
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
	builder = registerFrom(builder, s, "from_i32", func(v int32) string { return strconv.FormatInt(int64(v), 10) })
	builder = registerFrom(builder, s, "from_u32", func(v uint32) string { return strconv.FormatUint(uint64(v), 10) })
	builder = registerFrom(builder, s, "from_i64", func(v int64) string { return strconv.FormatInt(v, 10) })
	builder = registerFrom(builder, s, "from_u64", func(v uint64) string { return strconv.FormatUint(v, 10) })
	builder = registerFrom(builder, s, "from_f32", func(v float32) string { return strconv.FormatFloat(float64(v), 'g', -1, 32) })
	builder = registerFrom(builder, s, "from_f64", func(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) })
	builder = registerFormat(builder, m, "format_i32", func(v int32) any { return int64(v) })
	builder = registerFormat(builder, m, "format_u32", func(v uint32) any { return uint64(v) })
	builder = registerFormat(builder, m, "format_i64", func(v int64) any { return v })
	builder = registerFormat(builder, m, "format_u64", func(v uint64) any { return v })
	builder = registerFormat(builder, m, "format_f32", func(v float32) any { return float64(v) })
	builder = registerFormat(builder, m, "format_f64", func(v float64) any { return v })
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle, ptr, length uint32) uint32 {
			str, _ := s.Get(handle)
			return s.Create(formatWithSpec(m.memory, ptr, length, str))
		}).Export("format_string")
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return m, nil
}
