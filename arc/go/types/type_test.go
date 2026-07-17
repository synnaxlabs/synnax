// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Types", func() {
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
	})

	Describe("UnitsAssignable", func() {
		ns := &types.Unit{Dimensions: types.DimTime, Scale: 1, Name: "ns"}
		psi := &types.Unit{Name: "psi", Scale: 1}
		bar := &types.Unit{Name: "bar", Scale: 1}

		DescribeTable("unit compatibility",
			func(a, b *types.Unit, want bool) {
				Expect(types.UnitsAssignable(a, b)).To(Equal(want))
			},
			Entry("nil + nil", (*types.Unit)(nil), (*types.Unit)(nil), true),
			Entry("nil + ns (wildcard)", (*types.Unit)(nil), ns, true),
			Entry("ns + nil (wildcard)", ns, (*types.Unit)(nil), true),
			Entry("ns + ns (match)", ns, ns, true),
			Entry("psi + bar (mismatch)", psi, bar, false),
		)
	})

	Describe("Function constructor", func() {
		It("Should create function with nil inputs/outputs", func() {
			var props types.FunctionProperties
			fn := types.Function(props)
			Expect(fn.Kind).To(Equal(types.KindFunction))
			Expect(fn.Inputs).To(BeNil())
			Expect(fn.Outputs).To(BeNil())
		})

		It("Should preserve provided inputs/outputs", func() {
			props := types.FunctionProperties{
				Inputs: types.Params{{Name: "x", Type: types.I32()}},
			}
			fn := types.Function(props)
			Expect(fn.Inputs).To(HaveLen(1))
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

		// The value type mirrors what literal.Parse emits per kind, so a missing
		// cast case in NewSeriesFromAny (the TimeSpan regression) is caught here.
		DescribeTable("ToTelem output must seed a series via NewSeriesFromAny",
			func(arcType types.Type, value any) {
				dt := types.ToTelem(arcType)
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

	Describe("StructuralMatch", func() {
		DescribeTable("Should match types with same structure",
			func(t1, t2 types.Type) {
				Expect(types.StructuralMatch(t1, t2)).To(BeTrue())
			},
			Entry("scalar to scalar", types.I32(), types.F64()),
			Entry("series to series", types.Series(types.I32()), types.Series(types.F64())),
			Entry("channel to channel", types.Chan(types.I32()), types.Chan(types.F64())),
			Entry("string to int", types.String(), types.I32()),
			Entry("type variable to scalar", types.Variable("T", nil), types.I32()),
		)

		DescribeTable("Should not match types with different structure",
			func(t1, t2 types.Type) {
				Expect(types.StructuralMatch(t1, t2)).To(BeFalse())
			},
			Entry("scalar to series", types.I32(), types.Series(types.I32())),
			Entry("series to scalar", types.Series(types.I32()), types.I32()),
			Entry("scalar to channel", types.I32(), types.Chan(types.I32())),
			Entry("channel to scalar", types.Chan(types.I32()), types.I32()),
			Entry("series to channel", types.Series(types.I32()), types.Chan(types.I32())),
			Entry("channel to series", types.Chan(types.I32()), types.Series(types.I32())),
		)
	})

	Describe("ReadChan", func() {
		It("Should return the inner type for read channels", func() {
			readChan := types.ReadChan(types.I32())
			Expect(readChan.Unwrap()).To(Equal(types.I32()))
		})

		It("Should return true on IsRead", func() {
			readonlyChan := types.ReadChan(types.I32())
			Expect(readonlyChan.ChanDirection.IsRead()).To(BeTrue())
		})

		It("Should return false on IsWrite", func() {
			readonlyChan := types.ReadChan(types.I32())
			Expect(readonlyChan.ChanDirection.IsWrite()).To(BeFalse())
		})

		It("Should return true on IsSet", func() {
			readonlyChan := types.ReadChan(types.I32())
			Expect(readonlyChan.ChanDirection.IsSet()).To(BeTrue())
		})
	})
	Describe("WriteChan", func() {
		It("Should return the inner type for write channels", func() {
			writeChan := types.WriteChan(types.I32())
			Expect(writeChan.Unwrap()).To(Equal(types.I32()))
		})

		It("Should return false on IsRead", func() {
			writeChan := types.WriteChan(types.I32())
			Expect(writeChan.ChanDirection.IsRead()).To(BeFalse())
		})

		It("Should return true on IsWrite", func() {
			writeChan := types.WriteChan(types.I32())
			Expect(writeChan.ChanDirection.IsWrite()).To(BeTrue())
		})

		It("Should return true on IsSet", func() {
			writeChan := types.WriteChan(types.I32())
			Expect(writeChan.ChanDirection.IsSet()).To(BeTrue())
		})
	})
})
