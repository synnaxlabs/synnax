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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Resolver", func() {
	Describe("Resolve", func() {
		It("Should return incrementing handles", func() {
			r := resolve.NewResolver(symbol.MapResolver{})
			h0 := MustSucceed(r.Resolve("foo", types.Function(types.FunctionProperties{})))
			h1 := MustSucceed(r.Resolve("bar", types.Function(types.FunctionProperties{})))
			h2 := MustSucceed(r.Resolve("baz", types.Function(types.FunctionProperties{})))
			Expect(h0).To(Equal(uint32(0)))
			Expect(h1).To(Equal(uint32(1)))
			Expect(h2).To(Equal(uint32(2)))
		})
	})

	Describe("Finalize", func() {
		It("Should assign import indices for uncompiled refs", func() {
			symbols := symbol.MapResolver{
				"math.abs": symbol.Symbol{
					Name: "math.abs",
					Type: types.Function(types.FunctionProperties{
						Inputs:  types.Params{{Name: "x", Type: types.F64()}},
						Outputs: types.Params{{Name: "result", Type: types.F64()}},
					}),
				},
			}
			r := resolve.NewResolver(symbols)
			h0 := MustSucceed(r.Resolve("math.abs", types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "x", Type: types.F64()}},
				Outputs: types.Params{{Name: "result", Type: types.F64()}},
			})))

			m := wasm.NewModule()
			patches := MustSucceed(r.Finalize(m))
			Expect(patches[h0]).To(Equal(uint32(0)))
			Expect(m.ImportCount()).To(Equal(uint32(1)))
		})

		It("Should assign importCount + bodyIndex for local refs", func() {
			symbols := symbol.MapResolver{
				"test.print": symbol.Symbol{
					Name: "test.print",
					Type: types.Function(types.FunctionProperties{
						Inputs: types.Params{{Name: "x", Type: types.I32()}},
					}),
				},
			}
			r := resolve.NewResolver(symbols)
			hImport := MustSucceed(r.Resolve("test.print", types.Function(types.FunctionProperties{
				Inputs: types.Params{{Name: "x", Type: types.I32()}},
			})))
			hLocal := MustSucceed(r.Resolve("myFunc", types.Function(types.FunctionProperties{})))
			r.RegisterLocal("myFunc", 0)

			m := wasm.NewModule()
			patches := MustSucceed(r.Finalize(m))
			Expect(patches[hImport]).To(Equal(uint32(0)))
			Expect(patches[hLocal]).To(Equal(uint32(1)))
		})

		It("Should deduplicate imports with same WASM coordinates", func() {
			symbols := symbol.MapResolver{
				"math.abs": symbol.Symbol{
					Name: "math.abs",
					Type: types.Function(types.FunctionProperties{
						Inputs:  types.Params{{Name: "x", Type: types.F64()}},
						Outputs: types.Params{{Name: "result", Type: types.F64()}},
					}),
				},
			}
			r := resolve.NewResolver(symbols)
			ft := types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "x", Type: types.F64()}},
				Outputs: types.Params{{Name: "result", Type: types.F64()}},
			})
			h0 := MustSucceed(r.Resolve("math.abs", ft))
			h1 := MustSucceed(r.Resolve("math.abs", ft))

			m := wasm.NewModule()
			patches := MustSucceed(r.Finalize(m))
			Expect(patches[h0]).To(Equal(patches[h1]))
			Expect(m.ImportCount()).To(Equal(uint32(1)))
		})

		It("Should correctly map handles to real indices with mixed refs", func() {
			symbols := symbol.MapResolver{
				"test.log": symbol.Symbol{
					Name: "test.log",
					Type: types.Function(types.FunctionProperties{
						Inputs: types.Params{{Name: "x", Type: types.I32()}},
					}),
				},
				"test.print": symbol.Symbol{
					Name: "test.print",
					Type: types.Function(types.FunctionProperties{
						Inputs: types.Params{{Name: "x", Type: types.I32()}},
					}),
				},
			}
			r := resolve.NewResolver(symbols)
			hLog := MustSucceed(r.Resolve("test.log", types.Function(types.FunctionProperties{
				Inputs: types.Params{{Name: "x", Type: types.I32()}},
			})))
			hLocal0 := MustSucceed(r.Resolve("localA", types.Function(types.FunctionProperties{})))
			hPrint := MustSucceed(r.Resolve("test.print", types.Function(types.FunctionProperties{
				Inputs: types.Params{{Name: "x", Type: types.I32()}},
			})))
			hLocal1 := MustSucceed(r.Resolve("localB", types.Function(types.FunctionProperties{})))

			r.RegisterLocal("localA", 0)
			r.RegisterLocal("localB", 1)

			m := wasm.NewModule()
			patches := MustSucceed(r.Finalize(m))
			Expect(m.ImportCount()).To(Equal(uint32(2)))
			Expect(patches[hLog]).To(Equal(uint32(0)))
			Expect(patches[hPrint]).To(Equal(uint32(1)))
			Expect(patches[hLocal0]).To(Equal(uint32(2)))
			Expect(patches[hLocal1]).To(Equal(uint32(3)))
		})
	})
})

var _ = Describe("DeriveWASMCoordinates", func() {
	It("Should produce per-module names for qualified symbols", func() {
		symbols := symbol.MapResolver{
			"math.abs": symbol.Symbol{
				Name: "math.abs",
				Type: types.Function(types.FunctionProperties{
					Inputs:  types.Params{{Name: "x", Type: types.F64()}},
					Outputs: types.Params{{Name: "result", Type: types.F64()}},
				}),
			},
		}
		r := resolve.NewResolver(symbols)
		h := MustSucceed(r.Resolve("math.abs", types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: "x", Type: types.F64()}},
			Outputs: types.Params{{Name: "result", Type: types.F64()}},
		})))
		_ = h

		m := wasm.NewModule()
		MustSucceed(r.Finalize(m))
		names := m.ImportNames()
		Expect(names).To(HaveLen(1))
		Expect(names[0]).To(Equal("abs"))
	})

	It("Should append type suffix for polymorphic symbols", func() {
		numConstraint := types.NumericConstraint()
		symbols := symbol.MapResolver{
			"math.abs": symbol.Symbol{
				Name: "math.abs",
				Type: types.Function(types.FunctionProperties{
					Inputs:  types.Params{{Name: "x", Type: types.Variable("T", &numConstraint)}},
					Outputs: types.Params{{Name: "result", Type: types.Variable("T", &numConstraint)}},
				}),
			},
		}
		r := resolve.NewResolver(symbols)
		MustSucceed(r.Resolve("math.abs", types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: "x", Type: types.F64()}},
			Outputs: types.Params{{Name: "result", Type: types.F64()}},
		})))

		m := wasm.NewModule()
		MustSucceed(r.Finalize(m))
		names := m.ImportNames()
		Expect(names).To(HaveLen(1))
		Expect(names[0]).To(Equal("abs_f64"))
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
			Inputs: types.Params{{Name: "x", Type: types.Variable("T", &numConstraint)}},
		})
		concrete := types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: "x", Type: types.I32()}},
		})
		Expect(resolve.DeriveTypeSuffix(original, concrete)).To(Equal("i32"))
	})

	It("Should return suffixes for all Arc numeric types", func() {
		numConstraint := types.NumericConstraint()
		original := types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: "x", Type: types.Variable("T", &numConstraint)}},
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
			Inputs:  types.Params{{Name: "a", Type: types.I32()}, {Name: "b", Type: types.F64()}},
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
