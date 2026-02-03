// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bindings_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/compiler/bindings"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

// Production-grade ImportIndex created via SetupImports
var idx *bindings.ImportIndex

func init() {
	m := wasm.NewModule()
	idx = bindings.SetupImports(m)
}

var _ = Describe("ImportIndex Helpers", func() {
	Describe("GetChannelRead", func() {
		// Note: In actual usage, callers pass the unwrapped element type,
		// not the channel type itself (see variable.go:400, identifier.go:94)
		It("Should return import index for all numeric types", func() {
			for _, typ := range types.Numerics {
				MustSucceed(idx.GetChannelRead(typ))
			}
		})

		It("Should return import index for string type", func() {
			MustSucceed(idx.GetChannelRead(types.String()))
		})

		It("Should return error for series type", func() {
			// Series types don't have channel read functions
			Expect(idx.GetChannelRead(types.Series(types.I64()))).Error().To(
				MatchError(ContainSubstring("no channel read function")),
			)
		})
	})

	Describe("GetChannelWrite", func() {
		// Note: In actual usage, callers pass the unwrapped element type,
		// not the channel type itself (see variable.go:400)
		It("Should return import index for all numeric types", func() {
			for _, typ := range types.Numerics {
				MustSucceed(idx.GetChannelWrite(typ))
			}
		})

		It("Should return import index for string type", func() {
			MustSucceed(idx.GetChannelWrite(types.String()))
		})

		It("Should return error for series type", func() {
			Expect(idx.GetChannelWrite(types.Series(types.I64()))).Error().To(
				MatchError(ContainSubstring("no channel write function")),
			)
		})
	})

	Describe("GetSeriesCreateEmpty", func() {
		It("Should return import index for all numeric types", func() {
			for _, typ := range types.Numerics {
				MustSucceed(idx.GetSeriesCreateEmpty(typ))
			}
		})

		It("Should return error for string type", func() {
			Expect(idx.GetSeriesCreateEmpty(types.String())).Error().To(
				MatchError(ContainSubstring("no series create function")),
			)
		})
	})

	Describe("GetSeriesIndex", func() {
		It("Should return import index for all numeric types", func() {
			for _, typ := range types.Numerics {
				MustSucceed(idx.GetSeriesIndex(typ))
			}
		})

		It("Should return error for string type", func() {
			Expect(idx.GetSeriesIndex(types.String())).Error().To(
				MatchError(ContainSubstring("no series index function")),
			)
		})
	})

	Describe("GetStateLoad", func() {
		It("Should return import index for all numeric types", func() {
			for _, typ := range types.Numerics {
				MustSucceed(idx.GetStateLoad(typ))
			}
		})

		It("Should return import index for string type", func() {
			MustSucceed(idx.GetStateLoad(types.String()))
		})
	})

	Describe("GetStateStore", func() {
		It("Should return import index for all numeric types", func() {
			for _, typ := range types.Numerics {
				MustSucceed(idx.GetStateStore(typ))
			}
		})

		It("Should return import index for string type", func() {
			MustSucceed(idx.GetStateStore(types.String()))
		})
	})

	Describe("GetSeriesArithmetic", func() {
		arithmeticOps := []string{"+", "-", "*", "/", "%"}

		Context("Scalar operations (isScalar = true)", func() {
			It("Should return import index for all numeric types and operators", func() {
				for _, typ := range types.Numerics {
					for _, op := range arithmeticOps {
						MustSucceed(idx.GetSeriesArithmetic(op, typ, true))
					}
				}
			})

			It("Should return error for string type", func() {
				Expect(idx.GetSeriesArithmetic("+", types.String(), true)).Error().To(
					MatchError(ContainSubstring("no series + function")),
				)
			})

			It("Should return error for unknown operator", func() {
				Expect(idx.GetSeriesArithmetic("^", types.I64(), true)).Error().To(
					MatchError(ContainSubstring("unknown arithmetic operator")),
				)
			})
		})

		Context("Series-to-series operations (isScalar = false)", func() {
			It("Should return import index for all numeric types and operators", func() {
				for _, typ := range types.Numerics {
					for _, op := range arithmeticOps {
						MustSucceed(idx.GetSeriesArithmetic(op, typ, false))
					}
				}
			})

			It("Should return error for string type", func() {
				Expect(idx.GetSeriesArithmetic("+", types.String(), false)).Error().To(
					MatchError(ContainSubstring("no series + function")),
				)
			})
		})
	})

	Describe("GetSeriesComparison", func() {
		comparisonOps := []string{">", "<", ">=", "<=", "==", "!="}

		It("Should return import index for all numeric types and operators", func() {
			for _, typ := range types.Numerics {
				for _, op := range comparisonOps {
					MustSucceed(idx.GetSeriesComparison(op, typ))
				}
			}
		})

		It("Should return error for unknown operator", func() {
			Expect(idx.GetSeriesComparison("===", types.I64())).Error().To(
				MatchError(ContainSubstring("unknown comparison operator")),
			)
		})

		It("Should return error for string type", func() {
			Expect(idx.GetSeriesComparison(">", types.String())).Error().To(
				MatchError(ContainSubstring("no series comparison > function")),
			)
		})
	})

	Describe("GetSeriesSetElement", func() {
		It("Should return import index for all numeric types", func() {
			for _, typ := range types.Numerics {
				MustSucceed(idx.GetSeriesSetElement(typ))
			}
		})

		It("Should return error for string type", func() {
			Expect(idx.GetSeriesSetElement(types.String())).Error().To(
				MatchError(ContainSubstring("no series set element function")),
			)
		})
	})

	Describe("GetSeriesReverseArithmetic", func() {
		reverseOps := []string{"+", "-", "*", "/", "%"}

		It("Should return import index for all numeric types and operators", func() {
			for _, typ := range types.Numerics {
				for _, op := range reverseOps {
					MustSucceed(idx.GetSeriesReverseArithmetic(op, typ))
				}
			}
		})

		It("Should return error for unsupported operator", func() {
			Expect(idx.GetSeriesReverseArithmetic("^", types.I64())).Error().To(
				MatchError(ContainSubstring("reverse arithmetic not supported for: ^")),
			)
		})

		It("Should return error for string type", func() {
			Expect(idx.GetSeriesReverseArithmetic("-", types.String())).Error().To(
				MatchError(ContainSubstring("no series reverse - function")),
			)
		})
	})

	Describe("GetSeriesScalarComparison", func() {
		comparisonOps := []string{">", "<", ">=", "<=", "==", "!="}

		It("Should return import index for all numeric types and operators", func() {
			for _, typ := range types.Numerics {
				for _, op := range comparisonOps {
					MustSucceed(idx.GetSeriesScalarComparison(op, typ))
				}
			}
		})

		It("Should return error for unknown operator", func() {
			Expect(idx.GetSeriesScalarComparison("===", types.I64())).Error().To(
				MatchError(ContainSubstring("unknown comparison operator")),
			)
		})

		It("Should return error for string type", func() {
			Expect(idx.GetSeriesScalarComparison(">", types.String())).Error().To(
				MatchError(ContainSubstring("no series scalar comparison > function")),
			)
		})
	})

	Describe("GetStateLoadSeries", func() {
		It("Should return import index for all numeric types", func() {
			for _, typ := range types.Numerics {
				MustSucceed(idx.GetStateLoadSeries(typ))
			}
		})

		It("Should return error for string type", func() {
			Expect(idx.GetStateLoadSeries(types.String())).Error().To(
				MatchError(ContainSubstring("no series state load function")),
			)
		})
	})

	Describe("GetStateStoreSeries", func() {
		It("Should return import index for all numeric types", func() {
			for _, typ := range types.Numerics {
				MustSucceed(idx.GetStateStoreSeries(typ))
			}
		})

		It("Should return error for string type", func() {
			Expect(idx.GetStateStoreSeries(types.String())).Error().To(
				MatchError(ContainSubstring("no series state store function")),
			)
		})
	})

	Describe("GetSeriesNegate", func() {
		It("Should return import index for signed integer types", func() {
			for _, typ := range types.SignedIntegers {
				MustSucceed(idx.GetSeriesNegate(typ))
			}
		})

		It("Should return import index for float types", func() {
			for _, typ := range types.Floats {
				MustSucceed(idx.GetSeriesNegate(typ))
			}
		})

		It("Should return error for unsigned integer types", func() {
			for _, typ := range types.UnsignedIntegers {
				Expect(idx.GetSeriesNegate(typ)).Error().To(
					MatchError(ContainSubstring("no series negate function")),
				)
			}
		})

		It("Should return error for string type", func() {
			Expect(idx.GetSeriesNegate(types.String())).Error().To(
				MatchError(ContainSubstring("no series negate function")),
			)
		})
	})

	Describe("Index uniqueness", func() {
		It("Should return different indices for different operations", func() {
			addIdx := MustSucceed(idx.GetSeriesArithmetic("+", types.I64(), true))
			subIdx := MustSucceed(idx.GetSeriesArithmetic("-", types.I64(), true))
			Expect(addIdx).NotTo(Equal(subIdx))
		})

		It("Should return different indices for scalar vs series operations", func() {
			scalarIdx := MustSucceed(idx.GetSeriesArithmetic("+", types.I64(), true))
			seriesIdx := MustSucceed(idx.GetSeriesArithmetic("+", types.I64(), false))
			Expect(scalarIdx).NotTo(Equal(seriesIdx))
		})

		It("Should return different indices for different types", func() {
			i64Idx := MustSucceed(idx.GetSeriesArithmetic("+", types.I64(), true))
			f64Idx := MustSucceed(idx.GetSeriesArithmetic("+", types.F64(), true))
			Expect(i64Idx).NotTo(Equal(f64Idx))
		})
	})
})
