// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Types", func() {
	Describe("Unwrap", func() {
		Describe("Channel types", func() {
			It("should unwrap channel of i32 to i32", func() {
				chanType := types.Chan(types.I32())
				Expect(chanType.Unwrap()).To(Equal(types.I32()))
			})

			It("should unwrap channel of f64 to f64", func() {
				chanType := types.Chan(types.F64())
				Expect(chanType.Unwrap()).To(Equal(types.F64()))
			})

			It("should unwrap channel of u8 to u8", func() {
				chanType := types.Chan(types.U8())
				Expect(chanType.Unwrap()).To(Equal(types.U8()))
			})

			It("should handle nested channels (chan of chan)", func() {
				innerChan := types.Chan(types.I32())
				outerChan := types.Chan(innerChan)
				Expect(outerChan.Unwrap()).To(Equal(innerChan))
				Expect(outerChan.Unwrap().Unwrap()).To(Equal(types.I32()))
			})
		})

		Describe("Series types", func() {
			It("should unwrap series of i32 to i32", func() {
				seriesType := types.Series(types.I32())
				Expect(seriesType.Unwrap()).To(Equal(types.I32()))
			})

			It("should unwrap series of f32 to f32", func() {
				seriesType := types.Series(types.F32())
				Expect(seriesType.Unwrap()).To(Equal(types.F32()))
			})

			It("should unwrap series of timestamp to timestamp", func() {
				seriesType := types.Series(types.TimeStamp())
				Expect(seriesType.Unwrap()).To(Equal(types.TimeStamp()))
			})

			It("should handle nested series (series of series)", func() {
				innerSeries := types.Series(types.F64())
				outerSeries := types.Series(innerSeries)
				Expect(outerSeries.Unwrap()).To(Equal(innerSeries))
				Expect(outerSeries.Unwrap().Unwrap()).To(Equal(types.F64()))
			})
		})

		DescribeTable("Primitive types should return unchanged",
			func(t types.Type) {
				Expect(t.Unwrap()).To(Equal(t))
			},
			Entry("i32", types.I32()),
			Entry("f64", types.F64()),
			Entry("u8", types.U8()),
			Entry("timestamp", types.TimeStamp()),
			Entry("timespan", types.TimeSpan()),
		)

		Describe("Type variables", func() {
			It("should return type variable unchanged", func() {
				tv := types.Variable("T", nil)
				Expect(tv.Unwrap()).To(Equal(tv))
			})

			It("should return constrained type variable unchanged", func() {
				constraint := types.NumericConstraint()
				tv := types.Variable("N", &constraint)
				Expect(tv.Unwrap()).To(Equal(tv))
			})
		})

		Describe("Function types", func() {
			It("should return function type unchanged", func() {
				props := types.FunctionProperties{
					Inputs:  types.Params{{Name: "x", Type: types.I32()}},
					Outputs: types.Params{{Name: "result", Type: types.I32()}},
				}
				fnType := types.Function(props)
				Expect(fnType.Unwrap()).To(Equal(fnType))
			})
		})

		Describe("Mixed channel and series", func() {
			It("should unwrap channel of series correctly", func() {
				seriesType := types.Series(types.F32())
				chanType := types.Chan(seriesType)
				Expect(chanType.Unwrap()).To(Equal(seriesType))
				Expect(chanType.Unwrap().Unwrap()).To(Equal(types.F32()))
			})

			It("should unwrap series of channel correctly", func() {
				chanType := types.Chan(types.I64())
				seriesType := types.Series(chanType)
				Expect(seriesType.Unwrap()).To(Equal(chanType))
				Expect(seriesType.Unwrap().Unwrap()).To(Equal(types.I64()))
			})
		})

		Describe("Edge cases", func() {
			It("should handle invalid/zero type", func() {
				var t types.Type
				Expect(func() { t.Unwrap() }).NotTo(Panic())
				Expect(t.Unwrap()).To(Equal(t))
			})

			It("should handle channel with nil Elem", func() {
				chanType := types.Type{Kind: types.KindChan, Elem: nil}
				Expect(chanType.Unwrap()).To(Equal(chanType))
			})

			It("should handle series with nil Elem", func() {
				seriesType := types.Type{Kind: types.KindSeries, Elem: nil}
				Expect(seriesType.Unwrap()).To(Equal(seriesType))
			})
		})

		Describe("Idempotence", func() {
			It("should be idempotent for primitives", func() {
				t := types.I32()
				Expect(t.Unwrap().Unwrap()).To(Equal(t.Unwrap()))
			})

			It("should be idempotent after unwrapping once", func() {
				chanType := types.Chan(types.F64())
				unwrapped := chanType.Unwrap()
				Expect(unwrapped.Unwrap()).To(Equal(unwrapped))
			})

			It("should fully unwrap nested types with repeated calls", func() {
				nested := types.Chan(types.Series(types.I32()))
				firstUnwrap := nested.Unwrap()
				Expect(firstUnwrap.Kind).To(Equal(types.KindSeries))
				secondUnwrap := firstUnwrap.Unwrap()
				Expect(secondUnwrap).To(Equal(types.I32()))
				thirdUnwrap := secondUnwrap.Unwrap()
				Expect(thirdUnwrap).To(Equal(types.I32()))
			})
		})
	})

	Describe("Type predicates", func() {
		Describe("IsNumeric", func() {
			DescribeTable("Should return true for numeric types",
				func(t types.Type) {
					Expect(t.IsNumeric()).To(BeTrue())
				},
				Entry("U8", types.U8()),
				Entry("U16", types.U16()),
				Entry("U32", types.U32()),
				Entry("U64", types.U64()),
				Entry("I8", types.I8()),
				Entry("I16", types.I16()),
				Entry("I32", types.I32()),
				Entry("I64", types.I64()),
				Entry("F32", types.F32()),
				Entry("F64", types.F64()),
			)

			DescribeTable("Should return false for non-numeric types",
				func(t types.Type) {
					Expect(t.IsNumeric()).To(BeFalse())
				},
				Entry("String", types.String()),
				// Note: TimeStamp() and TimeSpan() are now i64 with units, so they ARE numeric
			)

			It("Should check value type for channels", func() {
				Expect(types.Chan(types.F64()).IsNumeric()).To(BeTrue())
				Expect(types.Chan(types.String()).IsNumeric()).To(BeFalse())
			})

			It("Should check value type for series", func() {
				Expect(types.Series(types.F64()).IsNumeric()).To(BeTrue())
				Expect(types.Series(types.String()).IsNumeric()).To(BeFalse())
			})

			It("Should handle type variables with numeric constraint", func() {
				constraint := types.NumericConstraint()
				tv := types.Variable("N", &constraint)
				Expect(tv.IsNumeric()).To(BeTrue())
			})

			It("Should return false for unconstrained type variables", func() {
				tv := types.Variable("T", nil)
				Expect(tv.IsNumeric()).To(BeFalse())
			})

			It("Should return true for type variable with integer constraint", func() {
				constraint := types.IntegerConstraint()
				tv := types.Variable("I", &constraint)
				Expect(tv.IsNumeric()).To(BeTrue())
			})

			It("Should return true for type variable with float constraint", func() {
				constraint := types.FloatConstraint()
				tv := types.Variable("F", &constraint)
				Expect(tv.IsNumeric()).To(BeTrue())
			})

			It("Should return true for type variable with concrete numeric type constraint", func() {
				constraint := types.I32()
				tv := types.Variable("N", &constraint)
				Expect(tv.IsNumeric()).To(BeTrue())
			})
		})

		Describe("IsInteger", func() {
			DescribeTable("Should return true for integer types",
				func(t types.Type) {
					Expect(t.IsInteger()).To(BeTrue())
				},
				Entry("U8", types.U8()),
				Entry("U16", types.U16()),
				Entry("U32", types.U32()),
				Entry("U64", types.U64()),
				Entry("I8", types.I8()),
				Entry("I16", types.I16()),
				Entry("I32", types.I32()),
				Entry("I64", types.I64()),
			)

			DescribeTable("Should return false for non-integer types",
				func(t types.Type) {
					Expect(t.IsInteger()).To(BeFalse())
				},
				Entry("F32", types.F32()),
				Entry("F64", types.F64()),
				Entry("String", types.String()),
			)
		})

		Describe("IsSignedInteger", func() {
			DescribeTable("Should return true for signed integers",
				func(t types.Type) {
					Expect(t.IsSignedInteger()).To(BeTrue())
				},
				Entry("I8", types.I8()),
				Entry("I16", types.I16()),
				Entry("I32", types.I32()),
				Entry("I64", types.I64()),
			)

			DescribeTable("Should return false for non-signed integers",
				func(t types.Type) {
					Expect(t.IsSignedInteger()).To(BeFalse())
				},
				Entry("U8", types.U8()),
				Entry("U32", types.U32()),
			)
		})

		Describe("IsUnsignedInteger", func() {
			DescribeTable("Should return true for unsigned integers",
				func(t types.Type) {
					Expect(t.IsUnsignedInteger()).To(BeTrue())
				},
				Entry("U8", types.U8()),
				Entry("U16", types.U16()),
				Entry("U32", types.U32()),
				Entry("U64", types.U64()),
			)

			DescribeTable("Should return false for non-unsigned integers",
				func(t types.Type) {
					Expect(t.IsUnsignedInteger()).To(BeFalse())
				},
				Entry("I8", types.I8()),
				Entry("I32", types.I32()),
			)
		})

		Describe("IsFloat", func() {
			DescribeTable("Should return true for float types",
				func(t types.Type) {
					Expect(t.IsFloat()).To(BeTrue())
				},
				Entry("F32", types.F32()),
				Entry("F64", types.F64()),
			)

			DescribeTable("Should return false for non-float types",
				func(t types.Type) {
					Expect(t.IsFloat()).To(BeFalse())
				},
				Entry("I32", types.I32()),
				Entry("String", types.String()),
			)
		})

		Describe("IntegerMaxValue", func() {
			DescribeTable("Should return correct max value for integer types",
				func(t types.Type, expected int64) {
					Expect(t.IntegerMaxValue()).To(Equal(expected))
				},
				Entry("I8", types.I8(), int64(math.MaxInt8)),
				Entry("I16", types.I16(), int64(math.MaxInt16)),
				Entry("I32", types.I32(), int64(math.MaxInt32)),
				Entry("I64", types.I64(), int64(math.MaxInt64)),
				Entry("U8", types.U8(), int64(math.MaxUint8)),
				Entry("U16", types.U16(), int64(math.MaxUint16)),
				Entry("U32", types.U32(), int64(math.MaxUint32)),
				Entry("U64", types.U64(), int64(math.MaxInt64)), // Uses MaxInt64 for comparison safety
			)

			DescribeTable("Should panic for non-integer types",
				func(t types.Type) {
					Expect(func() { t.IntegerMaxValue() }).To(Panic())
				},
				Entry("F32", types.F32()),
				Entry("F64", types.F64()),
				Entry("String", types.String()),
			)
		})

		Describe("IntegerMinValue", func() {
			DescribeTable("Should return correct min value for signed integer types",
				func(t types.Type, expected int64) {
					Expect(t.IntegerMinValue()).To(Equal(expected))
				},
				Entry("I8", types.I8(), int64(math.MinInt8)),
				Entry("I16", types.I16(), int64(math.MinInt16)),
				Entry("I32", types.I32(), int64(math.MinInt32)),
				Entry("I64", types.I64(), int64(math.MinInt64)),
			)

			DescribeTable("Should return 0 for unsigned integer types",
				func(t types.Type) {
					Expect(t.IntegerMinValue()).To(Equal(int64(0)))
				},
				Entry("U8", types.U8()),
				Entry("U16", types.U16()),
				Entry("U32", types.U32()),
				Entry("U64", types.U64()),
			)

			DescribeTable("Should panic for non-integer types",
				func(t types.Type) {
					Expect(func() { t.IntegerMinValue() }).To(Panic())
				},
				Entry("F32", types.F32()),
				Entry("F64", types.F64()),
				Entry("String", types.String()),
			)
		})

		Describe("Is64Bit", func() {
			DescribeTable("Should return true for 64-bit types",
				func(t types.Type) {
					Expect(t.Is64Bit()).To(BeTrue())
				},
				Entry("I64", types.I64()),
				Entry("U64", types.U64()),
				Entry("F64", types.F64()),
				Entry("TimeStamp", types.TimeStamp()),
				Entry("TimeSpan", types.TimeSpan()),
			)

			DescribeTable("Should return false for non-64-bit types",
				func(t types.Type) {
					Expect(t.Is64Bit()).To(BeFalse())
				},
				Entry("I32", types.I32()),
				Entry("F32", types.F32()),
			)
		})

		Describe("IsBool", func() {
			It("Should return true for U8", func() {
				Expect(types.U8().IsBool()).To(BeTrue())
			})

			It("Should return false for other types", func() {
				Expect(types.I32().IsBool()).To(BeFalse())
				Expect(types.String().IsBool()).To(BeFalse())
			})

			It("Should check value type for channels", func() {
				Expect(types.Chan(types.U8()).IsBool()).To(BeTrue())
				Expect(types.Chan(types.I32()).IsBool()).To(BeFalse())
			})

			It("Should check value type for series", func() {
				Expect(types.Series(types.U8()).IsBool()).To(BeTrue())
				Expect(types.Series(types.I32()).IsBool()).To(BeFalse())
			})
		})

		Describe("IsValid", func() {
			It("Should return true for valid types", func() {
				t := types.I32()
				Expect(t.IsValid()).To(BeTrue())
			})

			It("Should return false for invalid types", func() {
				var t types.Type
				Expect(t.IsValid()).To(BeFalse())
			})
		})
	})

	Describe("Equal", func() {
		It("Should return true for identical primitive types", func() {
			Expect(types.Equal(types.I32(), types.I32())).To(BeTrue())
			Expect(types.Equal(types.F64(), types.F64())).To(BeTrue())
		})

		It("Should return false for different primitive types", func() {
			Expect(types.Equal(types.I32(), types.I64())).To(BeFalse())
			Expect(types.Equal(types.F32(), types.F64())).To(BeFalse())
		})

		It("Should compare chan types recursively", func() {
			Expect(types.Equal(types.Chan(types.I32()), types.Chan(types.I32()))).To(BeTrue())
			Expect(types.Equal(types.Chan(types.I32()), types.Chan(types.I64()))).To(BeFalse())
		})

		It("Should handle chan types with nil Elem", func() {
			chan1 := types.Type{Kind: types.KindChan, Elem: nil}
			chan2 := types.Type{Kind: types.KindChan, Elem: nil}
			Expect(types.Equal(chan1, chan2)).To(BeTrue())

			chan3 := types.Chan(types.I32())
			Expect(types.Equal(chan1, chan3)).To(BeFalse())
			Expect(types.Equal(chan3, chan1)).To(BeFalse())
		})

		It("Should compare series types recursively", func() {
			Expect(types.Equal(types.Series(types.F64()), types.Series(types.F64()))).To(BeTrue())
			Expect(types.Equal(types.Series(types.F32()), types.Series(types.F64()))).To(BeFalse())
		})

		It("Should handle series types with nil Elem", func() {
			series1 := types.Type{Kind: types.KindSeries, Elem: nil}
			series2 := types.Type{Kind: types.KindSeries, Elem: nil}
			Expect(types.Equal(series1, series2)).To(BeTrue())

			series3 := types.Series(types.F64())
			Expect(types.Equal(series1, series3)).To(BeFalse())
		})

		It("Should compare type variables by name and constraint", func() {
			tv1 := types.Variable("T", nil)
			tv2 := types.Variable("T", nil)
			tv3 := types.Variable("U", nil)
			Expect(types.Equal(tv1, tv2)).To(BeTrue())
			Expect(types.Equal(tv1, tv3)).To(BeFalse())
		})

		It("Should handle type variables with different constraints", func() {
			numConstraint := types.NumericConstraint()
			intConstraint := types.IntegerConstraint()
			tv1 := types.Variable("T", &numConstraint)
			tv2 := types.Variable("T", &intConstraint)
			Expect(types.Equal(tv1, tv2)).To(BeFalse())
		})

		It("Should handle type variables with nil vs non-nil constraint", func() {
			constraint := types.NumericConstraint()
			tv1 := types.Variable("T", nil)
			tv2 := types.Variable("T", &constraint)
			Expect(types.Equal(tv1, tv2)).To(BeFalse())
			Expect(types.Equal(tv2, tv1)).To(BeFalse())
		})

		It("Should compare function types", func() {
			props1 := types.FunctionProperties{
				Inputs:  types.Params{{Name: "x", Type: types.I32()}},
				Outputs: types.Params{{Name: "y", Type: types.I32()}},
			}
			props2 := types.FunctionProperties{
				Inputs:  types.Params{{Name: "x", Type: types.I32()}},
				Outputs: types.Params{{Name: "y", Type: types.I32()}},
			}
			Expect(types.Equal(types.Function(props1), types.Function(props2))).To(BeTrue())
		})

		It("Should return false for function types with different inputs", func() {
			props1 := types.FunctionProperties{
				Inputs: types.Params{{Name: "x", Type: types.I32()}},
			}
			props2 := types.FunctionProperties{
				Inputs: types.Params{{Name: "y", Type: types.I32()}},
			}
			Expect(types.Equal(types.Function(props1), types.Function(props2))).To(BeFalse())
		})

		It("Should return false for function types with different input types", func() {
			props1 := types.FunctionProperties{
				Inputs: types.Params{{Name: "x", Type: types.I32()}},
			}
			props2 := types.FunctionProperties{
				Inputs: types.Params{{Name: "x", Type: types.F64()}},
			}
			Expect(types.Equal(types.Function(props1), types.Function(props2))).To(BeFalse())
		})

		It("Should return false for function types with different input counts", func() {
			props1 := types.FunctionProperties{
				Inputs: types.Params{
					{Name: "x", Type: types.I32()},
					{Name: "y", Type: types.I32()},
				},
			}
			props2 := types.FunctionProperties{
				Inputs: types.Params{{Name: "x", Type: types.I32()}},
			}
			Expect(types.Equal(types.Function(props1), types.Function(props2))).To(BeFalse())
		})

		It("Should return false for function types with different outputs", func() {
			props1 := types.FunctionProperties{
				Outputs: types.Params{{Name: "result", Type: types.I32()}},
			}
			props2 := types.FunctionProperties{
				Outputs: types.Params{{Name: "result", Type: types.F64()}},
			}
			Expect(types.Equal(types.Function(props1), types.Function(props2))).To(BeFalse())
		})

		It("Should return false for function types with different config", func() {
			props1 := types.FunctionProperties{
				Config: types.Params{{Name: "option", Type: types.I32()}},
			}
			props2 := types.FunctionProperties{
				Config: types.Params{{Name: "option", Type: types.F64()}},
			}
			Expect(types.Equal(types.Function(props1), types.Function(props2))).To(BeFalse())
		})
	})

	Describe("Function constructor", func() {
		It("Should create function with nil inputs/outputs/config", func() {
			var props types.FunctionProperties
			fn := types.Function(props)
			Expect(fn.Kind).To(Equal(types.KindFunction))
			Expect(fn.Inputs).To(BeNil())
			Expect(fn.Outputs).To(BeNil())
			Expect(fn.Config).To(BeNil())
		})

		It("Should preserve provided inputs/outputs/config", func() {
			props := types.FunctionProperties{
				Inputs: types.Params{{Name: "x", Type: types.I32()}},
			}
			fn := types.Function(props)
			Expect(len(fn.Inputs)).To(Equal(1))
		})
	})

	Describe("String", func() {
		DescribeTable("Should return correct strings for primitives",
			func(t types.Type, expected string) {
				Expect(t.String()).To(Equal(expected))
			},
			Entry("I8", types.I8(), "i8"),
			Entry("I16", types.I16(), "i16"),
			Entry("I32", types.I32(), "i32"),
			Entry("I64", types.I64(), "i64"),
			Entry("U8", types.U8(), "u8"),
			Entry("U16", types.U16(), "u16"),
			Entry("U32", types.U32(), "u32"),
			Entry("U64", types.U64(), "u64"),
			Entry("F32", types.F32(), "f32"),
			Entry("F64", types.F64(), "f64"),
			Entry("String", types.String(), "str"),
			Entry("TimeStamp", types.TimeStamp(), "i64 ns"),
			Entry("TimeSpan", types.TimeSpan(), "i64 ns"),
		)

		DescribeTable("Should return correct strings for compound types",
			func(t types.Type, expected string) {
				Expect(t.String()).To(Equal(expected))
			},
			Entry("chan i32", types.Chan(types.I32()), "chan i32"),
			Entry("chan f64", types.Chan(types.F64()), "chan f64"),
			Entry("series i32", types.Series(types.I32()), "series i32"),
			Entry("series f64", types.Series(types.F64()), "series f64"),
			Entry("chan with nil Elem", types.Type{Kind: types.KindChan, Elem: nil}, "chan <invalid>"),
			Entry("series with nil Elem", types.Type{Kind: types.KindSeries, Elem: nil}, "series <invalid>"),
		)

		DescribeTable("Should return correct strings for type variables and constraints",
			func(t types.Type, expected string) {
				Expect(t.String()).To(Equal(expected))
			},
			Entry("unconstrained", types.Variable("T", nil), "T"),
			Entry("numeric constraint", func() types.Type {
				c := types.NumericConstraint()
				return types.Variable("N", &c)
			}(), "N:numeric"),
			Entry("integer constraint", func() types.Type {
				c := types.IntegerConstraint()
				return types.Variable("I", &c)
			}(), "I:integer"),
			Entry("float constraint", func() types.Type {
				c := types.FloatConstraint()
				return types.Variable("F", &c)
			}(), "F:float"),
			Entry("numeric constraint kind", types.NumericConstraint(), "numeric"),
			Entry("integer constraint kind", types.IntegerConstraint(), "integer"),
			Entry("float constraint kind", types.FloatConstraint(), "float"),
		)

		It("Should return 'function' for function types", func() {
			fnType := types.Function(types.FunctionProperties{})
			Expect(fnType.String()).To(Equal("function"))
		})

		It("Should return 'invalid' for invalid types", func() {
			var invalidType types.Type
			Expect(invalidType.String()).To(Equal("invalid"))
		})
	})

	Describe("Telem Conversions", func() {
		DescribeTable("FromTelem should convert telem types to arc types",
			func(telemType telem.DataType, expected types.Type) {
				Expect(types.FromTelem(telemType)).To(Equal(expected))
			},
			Entry("Uint8T", telem.Uint8T, types.U8()),
			Entry("Uint16T", telem.Uint16T, types.U16()),
			Entry("Uint32T", telem.Uint32T, types.U32()),
			Entry("Uint64T", telem.Uint64T, types.U64()),
			Entry("Int8T", telem.Int8T, types.I8()),
			Entry("Int16T", telem.Int16T, types.I16()),
			Entry("Int32T", telem.Int32T, types.I32()),
			Entry("Int64T", telem.Int64T, types.I64()),
			Entry("Float32T", telem.Float32T, types.F32()),
			Entry("Float64T", telem.Float64T, types.F64()),
			Entry("StringT", telem.StringT, types.String()),
			Entry("JSONT", telem.JSONT, types.String()),
			Entry("UUIDT", telem.UUIDT, types.String()),
			Entry("TimeStampT", telem.TimeStampT, types.TimeStamp()),
		)

		It("Should return invalid type for unknown telem type", func() {
			result := types.FromTelem(telem.UnknownT)
			Expect(result.Kind).To(Equal(types.KindInvalid))
		})

		DescribeTable("ToTelem should convert arc types to telem types",
			func(arcType types.Type, expected telem.DataType) {
				Expect(types.ToTelem(arcType)).To(Equal(expected))
			},
			Entry("U8", types.U8(), telem.Uint8T),
			Entry("U16", types.U16(), telem.Uint16T),
			Entry("U32", types.U32(), telem.Uint32T),
			Entry("U64", types.U64(), telem.Uint64T),
			Entry("I8", types.I8(), telem.Int8T),
			Entry("I16", types.I16(), telem.Int16T),
			Entry("I32", types.I32(), telem.Int32T),
			Entry("I64", types.I64(), telem.Int64T),
			Entry("F32", types.F32(), telem.Float32T),
			Entry("F64", types.F64(), telem.Float64T),
			Entry("String", types.String(), telem.StringT),
			Entry("TimeStamp", types.TimeStamp(), telem.TimeStampT),
			Entry("TimeSpan", types.TimeSpan(), telem.TimeStampT),
		)

		It("Should return UnknownT for types that don't map to telem", func() {
			chanType := types.Chan(types.I32())
			Expect(types.ToTelem(chanType)).To(Equal(telem.UnknownT))

			fnType := types.Function(types.FunctionProperties{})
			Expect(types.ToTelem(fnType)).To(Equal(telem.UnknownT))
		})
	})

	Describe("Density", func() {
		DescribeTable("Should return correct byte size for fixed-size primitives",
			func(t types.Type, expectedDensity int) {
				Expect(t.Density()).To(Equal(expectedDensity))
			},
			Entry("U8 -> 1 byte", types.U8(), 1),
			Entry("I8 -> 1 byte", types.I8(), 1),
			Entry("U16 -> 2 bytes", types.U16(), 2),
			Entry("I16 -> 2 bytes", types.I16(), 2),
			Entry("U32 -> 4 bytes", types.U32(), 4),
			Entry("I32 -> 4 bytes", types.I32(), 4),
			Entry("F32 -> 4 bytes", types.F32(), 4),
			Entry("U64 -> 8 bytes", types.U64(), 8),
			Entry("I64 -> 8 bytes", types.I64(), 8),
			Entry("F64 -> 8 bytes", types.F64(), 8),
			Entry("TimeStamp -> 8 bytes", types.TimeStamp(), 8),
			Entry("TimeSpan -> 8 bytes", types.TimeSpan(), 8),
		)

		DescribeTable("Should panic for non-fixed-size types",
			func(t types.Type) {
				Expect(func() { t.Density() }).To(Panic())
			},
			Entry("String", types.String()),
			Entry("Chan", types.Chan(types.I32())),
			Entry("Series", types.Series(types.F64())),
			Entry("Variable", types.Variable("T", nil)),
			Entry("NumericConstraint", types.NumericConstraint()),
			Entry("IntegerConstraint", types.IntegerConstraint()),
			Entry("FloatConstraint", types.FloatConstraint()),
			Entry("Function", types.Function(types.FunctionProperties{})),
			Entry("Invalid", types.Type{Kind: types.KindInvalid}),
		)
	})

	Describe("Params", func() {
		var params types.Params
		BeforeEach(func() {
			params = types.Params{
				{Name: "x", Type: types.I32(), Value: 42},
				{Name: "y", Type: types.F64(), Value: 3.14},
				{Name: "flag", Type: types.U8(), Value: uint8(1)},
			}
		})
		Describe("Get", func() {
			It("Should return parameter when found", func() {
				param, ok := params.Get("x")
				Expect(ok).To(BeTrue())
				Expect(param.Name).To(Equal("x"))
				Expect(param.Type).To(Equal(types.I32()))
				Expect(param.Value).To(Equal(42))
			})
			It("Should return false when parameter not found", func() {
				param, ok := params.Get("nonexistent")
				Expect(ok).To(BeFalse())
				Expect(param).To(Equal(types.Param{}))
			})
			It("Should find last parameter", func() {
				param, ok := params.Get("flag")
				Expect(ok).To(BeTrue())
				Expect(param.Name).To(Equal("flag"))
			})
			It("Should work with empty params", func() {
				empty := types.Params{}
				param, ok := empty.Get("x")
				Expect(ok).To(BeFalse())
				Expect(param).To(Equal(types.Param{}))
			})
		})
		Describe("GetIndex", func() {
			It("Should return correct index when found", func() {
				Expect(params.GetIndex("x")).To(Equal(0))
				Expect(params.GetIndex("y")).To(Equal(1))
				Expect(params.GetIndex("flag")).To(Equal(2))
			})
			It("Should return -1 when not found", func() {
				Expect(params.GetIndex("nonexistent")).To(Equal(-1))
			})
			It("Should return -1 for empty params", func() {
				empty := types.Params{}
				Expect(empty.GetIndex("x")).To(Equal(-1))
			})
		})
		Describe("Has", func() {
			It("Should return true for existing parameters", func() {
				Expect(params.Has("x")).To(BeTrue())
				Expect(params.Has("y")).To(BeTrue())
				Expect(params.Has("flag")).To(BeTrue())
			})
			It("Should return false for non-existing parameters", func() {
				Expect(params.Has("nonexistent")).To(BeFalse())
			})
			It("Should return false for empty params", func() {
				empty := types.Params{}
				Expect(empty.Has("x")).To(BeFalse())
			})
		})
		Describe("ValueMap", func() {
			It("Should return map of parameter names to values", func() {
				valueMap := params.ValueMap()
				Expect(valueMap).To(HaveLen(3))
				Expect(valueMap["x"]).To(Equal(42))
				Expect(valueMap["y"]).To(Equal(3.14))
				Expect(valueMap["flag"]).To(Equal(uint8(1)))
			})
			It("Should return empty map for empty params", func() {
				empty := types.Params{}
				valueMap := empty.ValueMap()
				Expect(valueMap).To(HaveLen(0))
			})
			It("Should handle nil values", func() {
				paramsWithNil := types.Params{
					{Name: "a", Type: types.I32(), Value: nil},
					{Name: "b", Type: types.F64(), Value: 1.5},
				}
				valueMap := paramsWithNil.ValueMap()
				Expect(valueMap).To(HaveLen(2))
				Expect(valueMap["a"]).To(BeNil())
				Expect(valueMap["b"]).To(Equal(1.5))
			})
		})
	})

	Describe("Dimensions", func() {
		Describe("Mul", func() {
			It("Should add exponents (m * m = m^2)", func() {
				result := types.DimLength.Mul(types.DimLength)
				Expect(result.Length).To(Equal(int8(2)))
			})

			It("Should produce velocity (m * s^-1)", func() {
				result := types.DimLength.Mul(types.DimFrequency)
				Expect(result).To(Equal(types.DimVelocity))
			})

			It("Should handle dimensionless", func() {
				result := types.DimNone.Mul(types.DimLength)
				Expect(result).To(Equal(types.DimLength))
			})
		})

		Describe("Div", func() {
			It("Should subtract exponents (m / s = velocity)", func() {
				result := types.DimLength.Div(types.DimTime)
				Expect(result).To(Equal(types.DimVelocity))
			})

			It("Should cancel dimensions (m / m = dimensionless)", func() {
				result := types.DimLength.Div(types.DimLength)
				Expect(result.IsZero()).To(BeTrue())
			})

			It("Should produce frequency (1 / s)", func() {
				result := types.DimNone.Div(types.DimTime)
				Expect(result).To(Equal(types.DimFrequency))
			})
		})

		Describe("IsZero", func() {
			It("Should return true for dimensionless", func() {
				Expect(types.DimNone.IsZero()).To(BeTrue())
			})

			It("Should return false for dimensioned", func() {
				Expect(types.DimLength.IsZero()).To(BeFalse())
				Expect(types.DimPressure.IsZero()).To(BeFalse())
			})
		})

		Describe("String", func() {
			It("Should format velocity", func() {
				s := types.DimVelocity.String()
				Expect(s).To(ContainSubstring("length^1"))
				Expect(s).To(ContainSubstring("time^-1"))
			})

			It("Should return dimensionless for zero", func() {
				Expect(types.DimNone.String()).To(Equal("dimensionless"))
			})
		})
	})

	Describe("Unit", func() {
		It("Should compare equal units", func() {
			u1 := types.Unit{Dimensions: types.DimLength, Scale: 1000, Name: "km"}
			u2 := types.Unit{Dimensions: types.DimLength, Scale: 1000, Name: "km"}
			Expect(u1.Equal(u2)).To(BeTrue())
		})

		It("Should detect different scales", func() {
			u1 := types.Unit{Dimensions: types.DimLength, Scale: 1000, Name: "km"}
			u2 := types.Unit{Dimensions: types.DimLength, Scale: 1, Name: "m"}
			Expect(u1.Equal(u2)).To(BeFalse())
		})

		It("Should detect different dimensions", func() {
			u1 := types.Unit{Dimensions: types.DimLength, Scale: 1, Name: "m"}
			u2 := types.Unit{Dimensions: types.DimTime, Scale: 1, Name: "s"}
			Expect(u1.Equal(u2)).To(BeFalse())
		})
	})
})
