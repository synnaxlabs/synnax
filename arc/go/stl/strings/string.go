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

// Name is the module name.
const Name = "strings"

func formatHostFunc(t types.Type) *symbol.Symbol {
	return symbol.InternalHostFunc(
		"format_"+t.String(),
		types.Params{
			{Name: "value", Type: t},
			{Name: "spec_ptr", Type: types.I32()},
			{Name: "spec_len", Type: types.I32()},
		},
		types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
	)
}

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes. Strings contributes only its module (no bare globals).
func NewSymbols() []*symbol.Symbol {
	mod := &symbol.Symbol{Name: Name, Kind: symbol.KindModule, Internal: true}
	mod.AddChild(
		symbol.InternalHostFunc(
			"from_literal",
			types.Params{{Name: "ptr", Type: types.I32()}, {Name: "len", Type: types.I32()}},
			types.Params{{Name: "handle", Type: types.I32()}},
		),
		symbol.InternalHostFunc(
			"concat",
			types.Params{{Name: "a", Type: types.String()}, {Name: "b", Type: types.String()}},
			types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
		),
		symbol.InternalHostFunc(
			"equal",
			types.Params{{Name: "a", Type: types.String()}, {Name: "b", Type: types.String()}},
			types.Params{{Name: ir.DefaultOutputParam, Type: types.I32()}},
		),
		symbol.InternalHostFunc(
			"len",
			types.Params{{Name: "handle", Type: types.String()}},
			types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
		),
		symbol.InternalHostFunc(
			"from_i32",
			types.Params{{Name: "value", Type: types.I32()}},
			types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
		),
		symbol.InternalHostFunc(
			"from_u32",
			types.Params{{Name: "value", Type: types.U32()}},
			types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
		),
		symbol.InternalHostFunc(
			"from_i64",
			types.Params{{Name: "value", Type: types.I64()}},
			types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
		),
		symbol.InternalHostFunc(
			"from_u64",
			types.Params{{Name: "value", Type: types.U64()}},
			types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
		),
		symbol.InternalHostFunc(
			"from_f32",
			types.Params{{Name: "value", Type: types.F32()}},
			types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
		),
		symbol.InternalHostFunc(
			"from_f64",
			types.Params{{Name: "value", Type: types.F64()}},
			types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
		),
		formatHostFunc(types.I32()),
		formatHostFunc(types.U32()),
		formatHostFunc(types.I64()),
		formatHostFunc(types.U64()),
		formatHostFunc(types.F32()),
		formatHostFunc(types.F64()),
		formatHostFunc(types.String()),
	)
	return []*symbol.Symbol{mod}
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
	h *Host,
	name string,
	coerce func(T) any,
) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, v T, ptr, length uint32) uint32 {
			return h.strings.Create(formatWithSpec(h.memory, ptr, length, coerce(v)))
		}).Export(name)
}

func formatWithSpec(memory api.Memory, ptr, length uint32, value any) string {
	spec, ok := memory.Read(ptr, length)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%"+string(spec), value)
}

// Host is the runtime host-side support for the string module: it registers
// the WASM host bindings that allocate and manipulate string handles
// against a strings ProgramState. memory is the WASM guest's linear memory,
// set after the guest is instantiated via SetMemory.
type Host struct {
	strings *ProgramState
	memory  api.Memory
}

// SetMemory updates the WASM guest memory reference used by host functions
// that read string bytes (e.g., from_literal). Call after guest
// instantiation.
func (h *Host) SetMemory(memory api.Memory) { h.memory = memory }

// NewHost registers the string module's WASM host bindings with rt. The
// bindings allocate and read string handles against ps. memory may be nil
// at construction time; call SetMemory once the WASM guest is instantiated.
func NewHost(
	ctx context.Context,
	rt wazero.Runtime,
	ps *ProgramState,
	memory api.Memory,
) (*Host, error) {
	h := &Host{strings: ps, memory: memory}
	s := ps
	if rt == nil {
		return h, nil
	}
	builder := rt.NewHostModuleBuilder(Name)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, ptr, length uint32) uint32 {
			if h.memory == nil {
				return 0
			}
			data, ok := h.memory.Read(ptr, length)
			if !ok {
				return 0
			}
			return s.Create(string(data))
		}).Export("from_literal")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, h1, h2 uint32) uint32 {
			s1, ok1 := s.Get(h1)
			s2, ok2 := s.Get(h2)
			if !ok1 || !ok2 {
				return 0
			}
			return s.Create(s1 + s2)
		}).Export("concat")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, h1, h2 uint32) uint32 {
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
	builder = registerFormat(builder, h, "format_i32", func(v int32) any { return int64(v) })
	builder = registerFormat(builder, h, "format_u32", func(v uint32) any { return uint64(v) })
	builder = registerFormat(builder, h, "format_i64", func(v int64) any { return v })
	builder = registerFormat(builder, h, "format_u64", func(v uint64) any { return v })
	builder = registerFormat(builder, h, "format_f32", func(v float32) any { return float64(v) })
	builder = registerFormat(builder, h, "format_f64", func(v float64) any { return v })
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, handle, ptr, length uint32) uint32 {
			str, ok := s.Get(handle)
			if !ok {
				return 0
			}
			return s.Create(formatWithSpec(h.memory, ptr, length, str))
		}).Export("format_str")
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return h, nil
}
