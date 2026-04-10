// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package math

import (
	"context"
	"math"

	"github.com/tetratelabs/wazero"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	xmath "github.com/synnaxlabs/x/math"
)

var numConstraint = types.NumericConstraint()

var SymbolResolver = &symbol.ModuleResolver{
	Name: "math",
	Members: symbol.MapResolver{
		"pow": {
			Name: "pow",
			Kind: symbol.KindFunction,
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "base", Type: types.Variable("T", &numConstraint)}, {Name: "exp", Type: types.Variable("T", &numConstraint)}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.Variable("T", &numConstraint)}},
			}),
		},
	},
}

type Module struct{}

func NewModule(
	ctx context.Context,
	rt wazero.Runtime,
) (*Module, error) {
	m := &Module{}
	if rt == nil {
		return m, nil
	}
	builder := rt.NewHostModuleBuilder("math")
	// i32-compatible types: WASM uses uint32, convert internally
	builder = bindI32Pow[uint8](builder, "u8")
	builder = bindI32Pow[uint16](builder, "u16")
	builder = bindI32Pow[uint32](builder, "u32")
	builder = bindI32Pow[int8](builder, "i8")
	builder = bindI32Pow[int16](builder, "i16")
	builder = bindI32Pow[int32](builder, "i32")
	// i64-compatible types
	builder = bindI64Pow[uint64](builder, "u64")
	builder = bindI64Pow[int64](builder, "i64")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, base float32, exp float32) float32 {
			return float32(math.Pow(float64(base), float64(exp)))
		}).Export("pow_f32")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, base float64, exp float64) float64 {
			return math.Pow(base, exp)
		}).Export("pow_f64")
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return m, nil
}

type i32Powable interface {
	uint8 | uint16 | uint32 | int8 | int16 | int32
}

type i64Powable interface {
	uint64 | int64
}

// bindI32Pow binds an integer power function for a WASM i32-compatible type.
// The exponent arrives as uint32 from WASM, so negative Arc exponents appear as
// large positive values (e.g. -1 becomes 4294967295). On 64-bit platforms,
// int(uint32(x)) is always non-negative, making the 0^(-n) panic in IntPow
// unreachable through this interface.
func bindI32Pow[T i32Powable](builder wazero.HostModuleBuilder, suffix string) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, base uint32, exp uint32) uint32 {
			return uint32(xmath.IntPow(T(base), int(exp)))
		}).Export("pow_" + suffix)
}

// bindI64Pow binds an integer power function for a WASM i64-compatible type.
// Same unsigned exponent representation as bindI32Pow.
func bindI64Pow[T i64Powable](builder wazero.HostModuleBuilder, suffix string) wazero.HostModuleBuilder {
	return builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, base uint64, exp uint64) uint64 {
			return uint64(xmath.IntPow(T(base), int(exp)))
		}).Export("pow_" + suffix)
}
