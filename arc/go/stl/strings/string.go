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

const name = "string"

var module = symbol.NewModule(
	name,
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
)

// Symbols are the symbols this package contributes to a program's ambient
// prelude. Strings contributes only its module (no bare globals).
var Symbols = []*symbol.Symbol{module}

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
	builder := rt.NewHostModuleBuilder(name)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, ptr uint32, length uint32) uint32 {
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
	return h, nil
}
