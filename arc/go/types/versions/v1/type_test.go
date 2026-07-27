// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Type", func() {
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

	Describe("UnwrapChan", func() {
		DescribeTable("should unwrap channel types to their element type",
			func(t types.Type, expected types.Type) {
				Expect(t.UnwrapChan()).To(Equal(expected))
			},
			Entry("chan i32 -> i32", types.Chan(types.I32()), types.I32()),
			Entry("chan f64 -> f64", types.Chan(types.F64()), types.F64()),
			Entry("chan u8 -> u8", types.Chan(types.U8()), types.U8()),
			Entry("chan timestamp -> timestamp", types.Chan(types.TimeStamp()), types.TimeStamp()),
			Entry("chan series i32 -> series i32", types.Chan(types.Series(types.I32())), types.Series(types.I32())),
		)

		DescribeTable("should leave series types unchanged",
			func(t types.Type) {
				Expect(t.UnwrapChan()).To(Equal(t))
			},
			Entry("series i32", types.Series(types.I32())),
			Entry("series f64", types.Series(types.F64())),
			Entry("series timestamp", types.Series(types.TimeStamp())),
		)

		DescribeTable("should leave primitive types unchanged",
			func(t types.Type) {
				Expect(t.UnwrapChan()).To(Equal(t))
			},
			Entry("i32", types.I32()),
			Entry("f64", types.F64()),
			Entry("u8", types.U8()),
			Entry("timestamp", types.TimeStamp()),
			Entry("timespan", types.TimeSpan()),
			Entry("string", types.String()),
		)

		DescribeTable("should leave other types unchanged",
			func(t types.Type) {
				Expect(t.UnwrapChan()).To(Equal(t))
			},
			Entry("type variable", types.Variable("T", nil)),
			Entry("constrained type variable", func() types.Type {
				c := types.NumericConstraint()
				return types.Variable("N", &c)
			}()),
			Entry("function", types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "x", Type: types.I32()}},
				Outputs: types.Params{{Name: "result", Type: types.I32()}},
			})),
			Entry("sequence", types.Sequence()),
			Entry("stage", types.Stage()),
		)

		DescribeTable("should handle edge cases",
			func(t types.Type, expected types.Type) {
				Expect(t.UnwrapChan()).To(Equal(expected))
			},
			Entry("invalid type", types.Type{}, types.Type{}),
			Entry("chan with nil Elem", types.Type{Kind: types.KindChan, Elem: nil}, types.Type{Kind: types.KindChan, Elem: nil}),
			Entry("nested chan", types.Chan(types.Chan(types.I32())), types.Chan(types.I32())),
		)
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

		Describe("IsSigned", func() {
			DescribeTable("Should return true for signed types",
				func(t types.Type) {
					Expect(t.IsSigned()).To(BeTrue())
				},
				Entry("I8", types.I8()),
				Entry("I16", types.I16()),
				Entry("I32", types.I32()),
				Entry("I64", types.I64()),
				Entry("F32", types.F32()),
				Entry("F64", types.F64()),
			)

			DescribeTable("Should return false for unsigned types",
				func(t types.Type) {
					Expect(t.IsSigned()).To(BeFalse())
				},
				Entry("U8", types.U8()),
				Entry("U16", types.U16()),
				Entry("U32", types.U32()),
				Entry("U64", types.U64()),
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
			Entry("Sequence", types.Sequence(), "sequence"),
			Entry("Stage", types.Stage(), "stage"),
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
			Entry("unconstrained", types.Variable("T", nil), "unknown"),
			Entry("numeric constraint", func() types.Type {
				c := types.NumericConstraint()
				return types.Variable("N", &c)
			}(), "numeric"),
			Entry("integer constraint", func() types.Type {
				c := types.IntegerConstraint()
				return types.Variable("I", &c)
			}(), "integer"),
			Entry("float constraint", func() types.Type {
				c := types.FloatConstraint()
				return types.Variable("F", &c)
			}(), "float"),
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

	Describe("ToTelem", func() {
		DescribeTable("ToTelem should convert arc types to telem types",
			func(arcType types.Type, expected telem.DataType) {
				Expect(arcType.ToTelem()).To(Equal(expected))
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
			Expect(chanType.ToTelem()).To(Equal(telem.UnknownT))

			fnType := types.Function(types.FunctionProperties{})
			Expect(fnType.ToTelem()).To(Equal(telem.UnknownT))
		})

		// The value type mirrors what literal.Parse emits per kind, so a missing
		// cast case in NewSeriesFromAny (the TimeSpan regression) is caught here.
		DescribeTable("ToTelem output must seed a series via NewSeriesFromAny",
			func(arcType types.Type, value any) {
				dt := arcType.ToTelem()
				s := telem.NewSeriesFromAny(value, dt)
				Expect(s.DataType).To(Equal(dt))
				Expect(s.Len()).To(Equal(int64(1)))
			},
			Entry("U8", types.U8(), uint8(1)),
			Entry("U16", types.U16(), uint16(1)),
			Entry("U32", types.U32(), uint32(1)),
			Entry("U64", types.U64(), uint64(1)),
			Entry("I8", types.I8(), int8(1)),
			Entry("I16", types.I16(), int16(1)),
			Entry("I32", types.I32(), int32(1)),
			Entry("I64", types.I64(), int64(1)),
			Entry("F32", types.F32(), float32(1)),
			Entry("F64", types.F64(), float64(1)),
			Entry("String", types.String(), "x"),
			Entry("TimeStamp", types.TimeStamp(), telem.TimeSpan(1)),
			Entry("TimeSpan", types.TimeSpan(), telem.TimeSpan(1)),
		)
	})
})
