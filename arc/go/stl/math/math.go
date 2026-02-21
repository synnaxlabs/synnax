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

	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	xmath "github.com/synnaxlabs/x/math"
	"github.com/synnaxlabs/x/query"
)

var numConstraint = types.NumericConstraint()

var symResolver = &symbol.ModuleResolver{
	Name: "math",
	Members: symbol.MapResolver{
		"pow": {
			Name: "pow",
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "base", Type: types.Variable("T", &numConstraint)}, {Name: "exp", Type: types.Variable("T", &numConstraint)}},
				Outputs: types.Params{{Name: "result", Type: types.Variable("T", &numConstraint)}},
			}),
		},
	},
}

type Module struct{}

func NewModule() *Module { return &Module{} }

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
	// i32-compatible types: WASM uses uint32, convert internally
	bindI32Pow[uint8](rt, "u8")
	bindI32Pow[uint16](rt, "u16")
	bindI32Pow[uint32](rt, "u32")
	bindI32Pow[int8](rt, "i8")
	bindI32Pow[int16](rt, "i16")
	bindI32Pow[int32](rt, "i32")
	// i64-compatible types
	bindI64Pow[uint64](rt, "u64")
	bindI64Pow[int64](rt, "i64")
	stl.MustExport(rt, "math", "pow_f32",
		func(_ context.Context, base float32, exp float32) float32 {
			return float32(math.Pow(float64(base), float64(exp)))
		})
	stl.MustExport(rt, "math", "pow_f64",
		func(_ context.Context, base float64, exp float64) float64 {
			return math.Pow(base, exp)
		})
	return nil
}

type i32Powable interface {
	uint8 | uint16 | uint32 | int8 | int16 | int32
}

type i64Powable interface {
	uint64 | int64
}

func bindI32Pow[T i32Powable](rt stl.HostRuntime, suffix string) {
	stl.MustExport(rt, "math", "pow_"+suffix,
		func(_ context.Context, base uint32, exp uint32) uint32 {
			return uint32(xmath.IntPow(T(base), int(exp)))
		})
}

func bindI64Pow[T i64Powable](rt stl.HostRuntime, suffix string) {
	stl.MustExport(rt, "math", "pow_"+suffix,
		func(_ context.Context, base uint64, exp uint64) uint64 {
			return uint64(xmath.IntPow(T(base), int(exp)))
		})
}

var _ stl.Module = (*Module)(nil)
