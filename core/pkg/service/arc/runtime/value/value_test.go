// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package value_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/value"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Value", func() {
	Describe("Put and Get operations", func() {
		It("Should store and retrieve uint64", func() {
			v := value.Value{Type: ir.U64{}}
			v = v.PutUint64(12345)
			Expect(v.GetUint64()).To(Equal(uint64(12345)))
		})

		It("Should store and retrieve uint32", func() {
			v := value.Value{Type: ir.U32{}}
			v = v.PutUint32(12345)
			Expect(v.GetUint32()).To(Equal(uint32(12345)))
		})

		It("Should store and retrieve int64", func() {
			v := value.Value{Type: ir.I64{}}
			v = v.PutInt64(-12345)
			Expect(v.GetInt64()).To(Equal(int64(-12345)))
		})

		It("Should store and retrieve float64", func() {
			v := value.Value{Type: ir.F64{}}
			v = v.PutFloat64(123.45)
			Expect(v.GetFloat64()).To(Equal(123.45))
		})

		It("Should store and retrieve float32", func() {
			v := value.Value{Type: ir.F32{}}
			v = v.PutFloat32(123.45)
			Expect(v.GetFloat32()).To(Equal(float32(123.45)))
		})
	})

	Describe("Put interface{}", func() {
		It("Should handle uint64", func() {
			v := value.Value{}
			v = v.Put(uint64(100))
			Expect(v.GetUint64()).To(Equal(uint64(100)))
		})

		It("Should handle int64", func() {
			v := value.Value{}
			v = v.Put(int64(-100))
			Expect(v.GetInt64()).To(Equal(int64(-100)))
		})

		It("Should handle float64", func() {
			v := value.Value{}
			v = v.Put(float64(100.5))
			Expect(v.GetFloat64()).To(Equal(100.5))
		})

		It("Should handle bool true as 1", func() {
			v := value.Value{}
			v = v.Put(true)
			Expect(v.GetUint8()).To(Equal(uint8(1)))
		})

		It("Should handle bool false as 0", func() {
			v := value.Value{}
			v = v.Put(false)
			Expect(v.GetUint8()).To(Equal(uint8(0)))
		})
	})

	Describe("Get interface{}", func() {
		It("Should return correct type based on IR type", func() {
			tests := []struct {
				typ      ir.Type
				putValue uint64
				expected interface{}
			}{
				{ir.U64{}, 100, uint64(100)},
				{ir.U32{}, 100, uint32(100)},
				{ir.U16{}, 100, uint16(100)},
				{ir.U8{}, 100, uint8(100)},
				{ir.I64{}, ^uint64(99), int64(-100)},
				{ir.I32{}, ^uint64(99), int32(-100)},
				{ir.I16{}, ^uint64(99), int16(-100)},
				{ir.I8{}, ^uint64(99), int8(-100)},
			}

			for _, tt := range tests {
				v := value.Value{Type: tt.typ, Value: tt.putValue}
				result := v.Get()
				Expect(result).To(Equal(tt.expected))
			}
		})
	})

	Describe("Comparison operations", func() {
		Context("Integer comparisons", func() {
			It("Should compare integers correctly", func() {
				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(20)
				v3 := value.Value{Type: ir.I32{}}.PutInt32(10)

				Expect(v1.Eq(v3)).To(BeTrue())
				Expect(v1.Eq(v2)).To(BeFalse())
				Expect(v1.Lt(v2)).To(BeTrue())
				Expect(v1.Le(v2)).To(BeTrue())
				Expect(v1.Le(v3)).To(BeTrue())
				Expect(v2.Gt(v1)).To(BeTrue())
				Expect(v2.Ge(v1)).To(BeTrue())
				Expect(v1.Ge(v3)).To(BeTrue())
			})
		})

		Context("Float comparisons", func() {
			It("Should compare floats correctly", func() {
				v1 := value.Value{Type: ir.F64{}}.PutFloat64(10.5)
				v2 := value.Value{Type: ir.F64{}}.PutFloat64(20.5)
				v3 := value.Value{Type: ir.F64{}}.PutFloat64(10.5)

				Expect(v1.Eq(v3)).To(BeTrue())
				Expect(v1.Eq(v2)).To(BeFalse())
				Expect(v1.Lt(v2)).To(BeTrue())
				Expect(v2.Gt(v1)).To(BeTrue())
			})
		})

		Context("Mixed type comparisons", func() {
			It("Should coerce types correctly", func() {
				vInt := value.Value{Type: ir.I32{}}.PutInt32(10)
				vFloat := value.Value{Type: ir.F32{}}.PutFloat32(10.0)
				vUint := value.Value{Type: ir.U32{}}.PutUint32(10)

				// When comparing, right value is coerced to left value's type
				Expect(vInt.Eq(vFloat)).To(BeTrue())
				Expect(vInt.Eq(vUint)).To(BeTrue())
				Expect(vFloat.Eq(vInt)).To(BeTrue())
			})
		})
	})

	Describe("Arithmetic operations", func() {
		Context("Integer arithmetic", func() {
			It("Should perform integer arithmetic correctly", func() {
				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Type: ir.I32{}}.PutInt32(3)

				result := v1.Add(v2)
				Expect(result.GetInt32()).To(Equal(int32(13)))
				Expect(result.Type).To(Equal(ir.I32{}))

				result = v1.Sub(v2)
				Expect(result.GetInt32()).To(Equal(int32(7)))

				result = v1.Mul(v2)
				Expect(result.GetInt32()).To(Equal(int32(30)))

				result = v1.Div(v2)
				Expect(result.GetInt32()).To(Equal(int32(3)))

				result = v1.Mod(v2)
				Expect(result.GetInt32()).To(Equal(int32(1)))
			})
		})

		Context("Float arithmetic", func() {
			It("Should perform float arithmetic correctly", func() {
				v1 := value.Value{Type: ir.F64{}}.PutFloat64(10.5)
				v2 := value.Value{Type: ir.F64{}}.PutFloat64(2.5)

				result := v1.Add(v2)
				Expect(result.GetFloat64()).To(Equal(13.0))
				Expect(result.Type).To(Equal(ir.F64{}))

				result = v1.Sub(v2)
				Expect(result.GetFloat64()).To(Equal(8.0))

				result = v1.Mul(v2)
				Expect(result.GetFloat64()).To(Equal(26.25))

				result = v1.Div(v2)
				Expect(result.GetFloat64()).To(Equal(4.2))

				result = v1.Mod(v2)
				Expect(result.GetFloat64()).To(Equal(0.5))
			})
		})

		Context("Mixed type arithmetic", func() {
			It("Should coerce types to left operand", func() {
				vInt := value.Value{Type: ir.I32{}}.PutInt32(10)
				vFloat := value.Value{Type: ir.F32{}}.PutFloat32(2.5)

				// Result type follows left operand
				result := vInt.Add(vFloat)
				Expect(result.GetInt32()).To(Equal(int32(12))) // 10 + 2 (float truncated)
				Expect(result.Type).To(Equal(ir.I32{}))

				result = vFloat.Add(vInt)
				Expect(result.GetFloat32()).To(Equal(float32(12.5))) // 2.5 + 10
				Expect(result.Type).To(Equal(ir.F32{}))
			})
		})

		Context("Division by zero", func() {
			It("Should handle division by zero", func() {
				v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
				v0 := value.Value{Type: ir.I32{}}.PutInt32(0)

				result := v1.Div(v0)
				Expect(result.GetInt32()).To(Equal(int32(0)))

				result = v1.Mod(v0)
				Expect(result.GetInt32()).To(Equal(int32(0)))
			})
		})
	})

	Describe("DataTypeToIRType", func() {
		It("Should convert telem DataTypes to IR types correctly", func() {
			tests := []struct {
				dt       telem.DataType
				expected ir.Type
			}{
				{telem.Uint64T, ir.U64{}},
				{telem.Uint32T, ir.U32{}},
				{telem.Uint16T, ir.U16{}},
				{telem.Uint8T, ir.U8{}},
				{telem.Int64T, ir.I64{}},
				{telem.Int32T, ir.I32{}},
				{telem.Int16T, ir.I16{}},
				{telem.Int8T, ir.I8{}},
				{telem.Float64T, ir.F64{}},
				{telem.Float32T, ir.F32{}},
				{telem.StringT, ir.String{}},
				{telem.TimeStampT, ir.TimeStamp{}},
				{telem.DataType("unknown"), nil},
			}

			for _, tt := range tests {
				result := value.DataTypeToIRType(tt.dt)
				if tt.expected == nil {
					Expect(result).To(BeNil())
				} else {
					Expect(result).To(Equal(tt.expected))
				}
			}
		})
	})

	Describe("FromSeries", func() {
		It("Should convert uint64 series", func() {
			data := []uint64{10, 20, 30, 40, 50}
			series := telem.NewSeries(data)

			values := value.FromSeries(series)

			Expect(values).To(HaveLen(5))
			for i, v := range values {
				Expect(v.Type).To(Equal(ir.U64{}))
				Expect(v.GetUint64()).To(Equal(data[i]))
			}
		})

		It("Should convert float32 series", func() {
			data := []float32{1.5, 2.5, 3.5, 4.5}
			series := telem.NewSeries(data)

			values := value.FromSeries(series)

			Expect(values).To(HaveLen(4))
			for i, v := range values {
				Expect(v.Type).To(Equal(ir.F32{}))
				Expect(v.GetFloat32()).To(Equal(data[i]))
			}
		})

		It("Should convert int32 series", func() {
			data := []int32{-10, -20, 30, 40}
			series := telem.NewSeries(data)

			values := value.FromSeries(series)

			Expect(values).To(HaveLen(4))
			for i, v := range values {
				Expect(v.Type).To(Equal(ir.I32{}))
				Expect(v.GetInt32()).To(Equal(data[i]))
			}
		})
	})

	Describe("ToSeries", func() {
		It("Should handle empty slice", func() {
			values := []value.Value{}
			series := value.ToSeries(values, telem.Uint64T)
			Expect(series.Len()).To(Equal(int64(0)))
		})

		It("Should convert uint64 values to series", func() {
			values := []value.Value{
				value.Value{Type: ir.U64{}}.PutUint64(10),
				value.Value{Type: ir.U64{}}.PutUint64(20),
				value.Value{Type: ir.U64{}}.PutUint64(30),
			}
			series := value.ToSeries(values, telem.Uint64T)

			Expect(series.Len()).To(Equal(int64(3)))
			Expect(series.DataType).To(Equal(telem.Uint64T))
			Expect(telem.ValueAt[uint64](series, 0)).To(Equal(uint64(10)))
			Expect(telem.ValueAt[uint64](series, 1)).To(Equal(uint64(20)))
			Expect(telem.ValueAt[uint64](series, 2)).To(Equal(uint64(30)))
		})

		It("Should convert uint32 values to series", func() {
			values := []value.Value{
				value.Value{Type: ir.U32{}}.PutUint32(100),
				value.Value{Type: ir.U32{}}.PutUint32(200),
			}
			series := value.ToSeries(values, telem.Uint32T)

			Expect(series.Len()).To(Equal(int64(2)))
			Expect(series.DataType).To(Equal(telem.Uint32T))
			Expect(telem.ValueAt[uint32](series, 0)).To(Equal(uint32(100)))
			Expect(telem.ValueAt[uint32](series, 1)).To(Equal(uint32(200)))
		})

		It("Should convert uint16 values to series", func() {
			values := []value.Value{
				value.Value{Type: ir.U16{}}.PutUint16(50),
				value.Value{Type: ir.U16{}}.PutUint16(60),
			}
			series := value.ToSeries(values, telem.Uint16T)

			Expect(series.Len()).To(Equal(int64(2)))
			Expect(series.DataType).To(Equal(telem.Uint16T))
			Expect(telem.ValueAt[uint16](series, 0)).To(Equal(uint16(50)))
			Expect(telem.ValueAt[uint16](series, 1)).To(Equal(uint16(60)))
		})

		It("Should convert uint8 values to series", func() {
			values := []value.Value{
				value.Value{Type: ir.U8{}}.PutUint8(5),
				value.Value{Type: ir.U8{}}.PutUint8(10),
			}
			series := value.ToSeries(values, telem.Uint8T)

			Expect(series.Len()).To(Equal(int64(2)))
			Expect(series.DataType).To(Equal(telem.Uint8T))
			Expect(telem.ValueAt[uint8](series, 0)).To(Equal(uint8(5)))
			Expect(telem.ValueAt[uint8](series, 1)).To(Equal(uint8(10)))
		})

		It("Should convert int64 values to series", func() {
			values := []value.Value{
				value.Value{Type: ir.I64{}}.PutInt64(-100),
				value.Value{Type: ir.I64{}}.PutInt64(200),
				value.Value{Type: ir.I64{}}.PutInt64(-300),
			}
			series := value.ToSeries(values, telem.Int64T)

			Expect(series.Len()).To(Equal(int64(3)))
			Expect(series.DataType).To(Equal(telem.Int64T))
			Expect(telem.ValueAt[int64](series, 0)).To(Equal(int64(-100)))
			Expect(telem.ValueAt[int64](series, 1)).To(Equal(int64(200)))
			Expect(telem.ValueAt[int64](series, 2)).To(Equal(int64(-300)))
		})

		It("Should convert int32 values to series", func() {
			values := []value.Value{
				value.Value{Type: ir.I32{}}.PutInt32(-50),
				value.Value{Type: ir.I32{}}.PutInt32(50),
			}
			series := value.ToSeries(values, telem.Int32T)

			Expect(series.Len()).To(Equal(int64(2)))
			Expect(series.DataType).To(Equal(telem.Int32T))
			Expect(telem.ValueAt[int32](series, 0)).To(Equal(int32(-50)))
			Expect(telem.ValueAt[int32](series, 1)).To(Equal(int32(50)))
		})

		It("Should convert int16 values to series", func() {
			values := []value.Value{
				value.Value{Type: ir.I16{}}.PutInt16(-25),
				value.Value{Type: ir.I16{}}.PutInt16(25),
			}
			series := value.ToSeries(values, telem.Int16T)

			Expect(series.Len()).To(Equal(int64(2)))
			Expect(series.DataType).To(Equal(telem.Int16T))
			Expect(telem.ValueAt[int16](series, 0)).To(Equal(int16(-25)))
			Expect(telem.ValueAt[int16](series, 1)).To(Equal(int16(25)))
		})

		It("Should convert int8 values to series", func() {
			values := []value.Value{
				value.Value{Type: ir.I8{}}.PutInt8(-5),
				value.Value{Type: ir.I8{}}.PutInt8(10),
			}
			series := value.ToSeries(values, telem.Int8T)

			Expect(series.Len()).To(Equal(int64(2)))
			Expect(series.DataType).To(Equal(telem.Int8T))
			Expect(telem.ValueAt[int8](series, 0)).To(Equal(int8(-5)))
			Expect(telem.ValueAt[int8](series, 1)).To(Equal(int8(10)))
		})

		It("Should convert float64 values to series", func() {
			values := []value.Value{
				value.Value{Type: ir.F64{}}.PutFloat64(1.5),
				value.Value{Type: ir.F64{}}.PutFloat64(2.75),
				value.Value{Type: ir.F64{}}.PutFloat64(-3.25),
			}
			series := value.ToSeries(values, telem.Float64T)

			Expect(series.Len()).To(Equal(int64(3)))
			Expect(series.DataType).To(Equal(telem.Float64T))
			Expect(telem.ValueAt[float64](series, 0)).To(Equal(1.5))
			Expect(telem.ValueAt[float64](series, 1)).To(Equal(2.75))
			Expect(telem.ValueAt[float64](series, 2)).To(Equal(-3.25))
		})

		It("Should convert float32 values to series", func() {
			values := []value.Value{
				value.Value{Type: ir.F32{}}.PutFloat32(1.5),
				value.Value{Type: ir.F32{}}.PutFloat32(2.5),
			}
			series := value.ToSeries(values, telem.Float32T)

			Expect(series.Len()).To(Equal(int64(2)))
			Expect(series.DataType).To(Equal(telem.Float32T))
			Expect(telem.ValueAt[float32](series, 0)).To(Equal(float32(1.5)))
			Expect(telem.ValueAt[float32](series, 1)).To(Equal(float32(2.5)))
		})

		It("Should cast values to target data type", func() {
			// Float64 values cast to Int32
			values := []value.Value{
				value.Value{Type: ir.F64{}}.PutFloat64(10.7),
				value.Value{Type: ir.F64{}}.PutFloat64(20.3),
				value.Value{Type: ir.F64{}}.PutFloat64(-30.9),
			}
			series := value.ToSeries(values, telem.Int32T)

			Expect(series.Len()).To(Equal(int64(3)))
			Expect(series.DataType).To(Equal(telem.Int32T))
			Expect(telem.ValueAt[int32](series, 0)).To(Equal(int32(10)))
			Expect(telem.ValueAt[int32](series, 1)).To(Equal(int32(20)))
			Expect(telem.ValueAt[int32](series, 2)).To(Equal(int32(-30)))
		})

		It("Should cast int values to float", func() {
			// Int32 values cast to Float64
			values := []value.Value{
				value.Value{Type: ir.I32{}}.PutInt32(10),
				value.Value{Type: ir.I32{}}.PutInt32(-20),
				value.Value{Type: ir.I32{}}.PutInt32(30),
			}
			series := value.ToSeries(values, telem.Float64T)

			Expect(series.Len()).To(Equal(int64(3)))
			Expect(series.DataType).To(Equal(telem.Float64T))
			Expect(telem.ValueAt[float64](series, 0)).To(Equal(float64(10)))
			Expect(telem.ValueAt[float64](series, 1)).To(Equal(float64(-20)))
			Expect(telem.ValueAt[float64](series, 2)).To(Equal(float64(30)))
		})

		It("Should cast uint64 to uint8 with overflow", func() {
			// Uint64 values cast to Uint8
			values := []value.Value{
				value.Value{Type: ir.U64{}}.PutUint64(10),
				value.Value{Type: ir.U64{}}.PutUint64(256), // overflows to 0
				value.Value{Type: ir.U64{}}.PutUint64(257), // overflows to 1
			}
			series := value.ToSeries(values, telem.Uint8T)

			Expect(series.Len()).To(Equal(int64(3)))
			Expect(series.DataType).To(Equal(telem.Uint8T))
			Expect(telem.ValueAt[uint8](series, 0)).To(Equal(uint8(10)))
			Expect(telem.ValueAt[uint8](series, 1)).To(Equal(uint8(0)))
			Expect(telem.ValueAt[uint8](series, 2)).To(Equal(uint8(1)))
		})
	})

	Describe("Round-trip conversion", func() {
		It("Should preserve uint64 data through FromSeries -> ToSeries", func() {
			original := []uint64{10, 20, 30, 40, 50}
			series := telem.NewSeries(original)
			values := value.FromSeries(series)
			result := value.ToSeries(values, telem.Uint64T)

			Expect(result.Len()).To(Equal(int64(len(original))))
			Expect(result.DataType).To(Equal(telem.Uint64T))
			for i, expected := range original {
				Expect(telem.ValueAt[uint64](result, i)).To(Equal(expected))
			}
		})

		It("Should preserve int32 data through FromSeries -> ToSeries", func() {
			original := []int32{-10, 20, -30, 40}
			series := telem.NewSeries(original)
			values := value.FromSeries(series)
			result := value.ToSeries(values, telem.Int32T)

			Expect(result.Len()).To(Equal(int64(len(original))))
			Expect(result.DataType).To(Equal(telem.Int32T))
			for i, expected := range original {
				Expect(telem.ValueAt[int32](result, i)).To(Equal(expected))
			}
		})

		It("Should preserve float64 data through FromSeries -> ToSeries", func() {
			original := []float64{1.5, 2.75, -3.25, 4.125}
			series := telem.NewSeries(original)
			values := value.FromSeries(series)
			result := value.ToSeries(values, telem.Float64T)

			Expect(result.Len()).To(Equal(int64(len(original))))
			Expect(result.DataType).To(Equal(telem.Float64T))
			for i, expected := range original {
				Expect(telem.ValueAt[float64](result, i)).To(Equal(expected))
			}
		})

		It("Should preserve float32 data through FromSeries -> ToSeries", func() {
			original := []float32{1.5, 2.5, 3.5}
			series := telem.NewSeries(original)
			values := value.FromSeries(series)
			result := value.ToSeries(values, telem.Float32T)

			Expect(result.Len()).To(Equal(int64(len(original))))
			Expect(result.DataType).To(Equal(telem.Float32T))
			for i, expected := range original {
				Expect(telem.ValueAt[float32](result, i)).To(Equal(expected))
			}
		})
	})

	Describe("Complex operations", func() {
		It("Should chain operations correctly", func() {
			v1 := value.Value{Type: ir.I32{}}.PutInt32(10)
			v2 := value.Value{Type: ir.I32{}}.PutInt32(3)
			v3 := value.Value{Type: ir.I32{}}.PutInt32(2)

			// (10 + 3) * 2 = 26
			result := v1.Add(v2).Mul(v3)
			Expect(result.GetInt32()).To(Equal(int32(26)))

			// (10 - 3) / 2 = 3
			result = v1.Sub(v2).Div(v3)
			Expect(result.GetInt32()).To(Equal(int32(3)))
		})

		It("Should preserve type in operations", func() {
			vU8 := value.Value{Type: ir.U8{}}.PutUint8(250)
			vU32 := value.Value{Type: ir.U32{}}.PutUint32(10)

			// Result should be U8 type (left operand)
			result := vU8.Add(vU32)
			Expect(result.Type).To(Equal(ir.U8{}))
			Expect(result.GetUint8()).To(Equal(uint8(4))) // 250 + 10 = 260, overflows to 4

			// Result should be U32 type (left operand)
			result = vU32.Add(vU8)
			Expect(result.Type).To(Equal(ir.U32{}))
			Expect(result.GetUint32()).To(Equal(uint32(260)))
		})
	})
})
