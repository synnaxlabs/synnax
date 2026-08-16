// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolve_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/compiler/resolve"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// mathModuleWithAbs returns a sealed math module with an abs(T) -> T entry
// whose Type carries a numeric type variable. Used by EmitCall tests that
// exercise polymorphic suffix derivation.
func mathModuleWithAbs() *symbol.Symbol {
	numConstraint := types.NumericConstraint()
	mod := &symbol.Symbol{Name: "math", Kind: symbol.KindModule}
	mod.AddChild(&symbol.Symbol{
		Name: "abs",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{
				{Name: "x", Type: types.Variable("T", &numConstraint)},
			},
			Outputs: types.Params{
				{Name: "result", Type: types.Variable("T", &numConstraint)},
			},
		}),
	})
	return mod
}

// monoMathAbs returns a math module with a monomorphic f64 abs entry.
func monoMathAbs() *symbol.Symbol {
	mod := &symbol.Symbol{Name: "math", Kind: symbol.KindModule}
	mod.AddChild(&symbol.Symbol{
		Name: "abs",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: "x", Type: types.F64()}},
			Outputs: types.Params{{Name: "result", Type: types.F64()}},
		}),
	})
	return mod
}

var _ = Describe("Resolver", func() {
	Describe("Finalize", func() {
		It("Should assign an import index for a host-import reference", func() {
			r := resolve.NewResolver()
			w := wasm.NewWriter()
			wID := r.TrackWriter(w)
			target := monoMathAbs().FindChild("abs")
			r.EmitCall(w, wID, target, target.Type)

			m := wasm.NewModule()
			patches := r.Finalize(m)
			Expect(patches).To(HaveLen(1))
			Expect(m.ImportCount()).To(Equal(uint32(1)))
			Expect(m.ImportNames()).To(ConsistOf("abs"))
		})

		It("Should map local refs to importCount + bodyIndex", func() {
			r := resolve.NewResolver()
			w := wasm.NewWriter()
			wID := r.TrackWriter(w)
			absTarget := monoMathAbs().FindChild("abs")
			r.EmitCall(w, wID, absTarget, absTarget.Type)
			myFunc := &symbol.Symbol{Name: "myFunc", Kind: symbol.KindFunction}
			r.EmitCall(w, wID, myFunc, types.Function(types.FunctionProperties{}))
			r.RegisterLocal("myFunc", 0)

			m := wasm.NewModule()
			patches := r.Finalize(m)
			Expect(m.ImportCount()).To(Equal(uint32(1)))
			values := []uint32{}
			for _, v := range patches {
				values = append(values, v)
			}
			Expect(values).To(ConsistOf(uint32(0), uint32(1)))
		})

		It("Should deduplicate imports with identical WASM coordinates", func() {
			r := resolve.NewResolver()
			w := wasm.NewWriter()
			wID := r.TrackWriter(w)
			target := monoMathAbs().FindChild("abs")
			r.EmitCall(w, wID, target, target.Type)
			r.EmitCall(w, wID, target, target.Type)

			m := wasm.NewModule()
			patches := r.Finalize(m)
			Expect(patches).To(HaveLen(2))
			var first, second uint32
			i := 0
			for _, v := range patches {
				if i == 0 {
					first = v
				} else {
					second = v
				}
				i++
			}
			Expect(first).To(Equal(second))
			Expect(m.ImportCount()).To(Equal(uint32(1)))
		})

		It("Should append a type suffix for polymorphic targets", func() {
			r := resolve.NewResolver()
			w := wasm.NewWriter()
			wID := r.TrackWriter(w)
			target := mathModuleWithAbs().FindChild("abs")
			concrete := types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "x", Type: types.F64()}},
				Outputs: types.Params{{Name: "result", Type: types.F64()}},
			})
			r.EmitCall(w, wID, target, concrete)

			m := wasm.NewModule()
			r.Finalize(m)
			Expect(m.ImportNames()).To(ConsistOf("abs_f64"))
		})
	})
})

var _ = Describe("DeriveTypeSuffix", func() {
	It("Should return empty for non-polymorphic types", func() {
		original := types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: "x", Type: types.F64()}},
		})
		concrete := types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: "x", Type: types.F64()}},
		})
		Expect(resolve.DeriveTypeSuffix(original, concrete)).To(Equal(""))
	})

	It("Should return the correct suffix for polymorphic types", func() {
		numConstraint := types.NumericConstraint()
		original := types.Function(types.FunctionProperties{
			Inputs: types.Params{
				{Name: "x", Type: types.Variable("T", &numConstraint)},
			},
		})
		concrete := types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: "x", Type: types.I32()}},
		})
		Expect(resolve.DeriveTypeSuffix(original, concrete)).To(Equal("i32"))
	})

	It("Should return suffixes for all Arc numeric types", func() {
		numConstraint := types.NumericConstraint()
		original := types.Function(types.FunctionProperties{
			Inputs: types.Params{
				{Name: "x", Type: types.Variable("T", &numConstraint)},
			},
		})
		cases := []struct {
			concreteType types.Type
			expected     string
		}{
			{types.U8(), "u8"},
			{types.U16(), "u16"},
			{types.U32(), "u32"},
			{types.U64(), "u64"},
			{types.I8(), "i8"},
			{types.I16(), "i16"},
			{types.I32(), "i32"},
			{types.I64(), "i64"},
			{types.F32(), "f32"},
			{types.F64(), "f64"},
		}
		for _, tc := range cases {
			concrete := types.Function(types.FunctionProperties{
				Inputs: types.Params{{Name: "x", Type: tc.concreteType}},
			})
			Expect(resolve.DeriveTypeSuffix(original, concrete)).To(Equal(tc.expected))
		}
	})
})

var _ = Describe("DeriveWASMFuncType", func() {
	It("Should convert Arc types to WASM value types", func() {
		arcType := types.Function(types.FunctionProperties{
			Inputs: types.Params{
				{Name: "a", Type: types.I32()},
				{Name: "b", Type: types.F64()},
			},
			Outputs: types.Params{{Name: "result", Type: types.I64()}},
		})
		ft := resolve.DeriveWASMFuncType(arcType)
		Expect(ft.Params).To(Equal([]wasm.ValueType{wasm.I32, wasm.F64}))
		Expect(ft.Results).To(Equal([]wasm.ValueType{wasm.I64}))
	})

	It("Should return empty for non-function types", func() {
		ft := resolve.DeriveWASMFuncType(types.I32())
		Expect(ft.Params).To(BeNil())
		Expect(ft.Results).To(BeNil())
	})

	It("Should handle functions with no outputs", func() {
		arcType := types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: "x", Type: types.F32()}},
		})
		ft := resolve.DeriveWASMFuncType(arcType)
		Expect(ft.Params).To(Equal([]wasm.ValueType{wasm.F32}))
		Expect(ft.Results).To(BeNil())
	})

	It("Should handle functions with no inputs", func() {
		arcType := types.Function(types.FunctionProperties{
			Outputs: types.Params{{Name: "result", Type: types.F64()}},
		})
		ft := resolve.DeriveWASMFuncType(arcType)
		Expect(ft.Params).To(BeNil())
		Expect(ft.Results).To(Equal([]wasm.ValueType{wasm.F64}))
	})
})
