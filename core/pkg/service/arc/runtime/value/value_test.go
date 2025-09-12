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
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Value", func() {
	var addr address.Address

	BeforeEach(func() {
		addr = address.Rand()
	})

	Describe("Put and Get operations", func() {
		It("Should store and retrieve uint64", func() {
			v := value.Value{Address: addr, Type: ir.U64{}}
			v = v.PutUint64(12345)
			Expect(v.GetUint64()).To(Equal(uint64(12345)))
		})

		It("Should store and retrieve uint32", func() {
			v := value.Value{Address: addr, Type: ir.U32{}}
			v = v.PutUint32(12345)
			Expect(v.GetUint32()).To(Equal(uint32(12345)))
		})

		It("Should store and retrieve int64", func() {
			v := value.Value{Address: addr, Type: ir.I64{}}
			v = v.PutInt64(-12345)
			Expect(v.GetInt64()).To(Equal(int64(-12345)))
		})

		It("Should store and retrieve float64", func() {
			v := value.Value{Address: addr, Type: ir.F64{}}
			v = v.PutFloat64(123.45)
			Expect(v.GetFloat64()).To(Equal(123.45))
		})

		It("Should store and retrieve float32", func() {
			v := value.Value{Address: addr, Type: ir.F32{}}
			v = v.PutFloat32(123.45)
			Expect(v.GetFloat32()).To(Equal(float32(123.45)))
		})
	})

	Describe("Put interface{}", func() {
		It("Should handle uint64", func() {
			v := value.Value{Address: addr}
			v = v.Put(uint64(100))
			Expect(v.GetUint64()).To(Equal(uint64(100)))
		})

		It("Should handle int64", func() {
			v := value.Value{Address: addr}
			v = v.Put(int64(-100))
			Expect(v.GetInt64()).To(Equal(int64(-100)))
		})

		It("Should handle float64", func() {
			v := value.Value{Address: addr}
			v = v.Put(float64(100.5))
			Expect(v.GetFloat64()).To(Equal(100.5))
		})

		It("Should handle bool true as 1", func() {
			v := value.Value{Address: addr}
			v = v.Put(true)
			Expect(v.GetUint8()).To(Equal(uint8(1)))
		})

		It("Should handle bool false as 0", func() {
			v := value.Value{Address: addr}
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
				v := value.Value{Address: addr, Type: tt.typ, Value: tt.putValue}
				result := v.Get()
				Expect(result).To(Equal(tt.expected))
			}
		})
	})

	Describe("Comparison operations", func() {
		Context("Integer comparisons", func() {
			It("Should compare integers correctly", func() {
				v1 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(20)
				v3 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(10)

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
				v1 := value.Value{Address: addr, Type: ir.F64{}}.PutFloat64(10.5)
				v2 := value.Value{Address: addr, Type: ir.F64{}}.PutFloat64(20.5)
				v3 := value.Value{Address: addr, Type: ir.F64{}}.PutFloat64(10.5)

				Expect(v1.Eq(v3)).To(BeTrue())
				Expect(v1.Eq(v2)).To(BeFalse())
				Expect(v1.Lt(v2)).To(BeTrue())
				Expect(v2.Gt(v1)).To(BeTrue())
			})
		})

		Context("Mixed type comparisons", func() {
			It("Should coerce types correctly", func() {
				vInt := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(10)
				vFloat := value.Value{Address: addr, Type: ir.F32{}}.PutFloat32(10.0)
				vUint := value.Value{Address: addr, Type: ir.U32{}}.PutUint32(10)

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
				v1 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(10)
				v2 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(3)

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
				v1 := value.Value{Address: addr, Type: ir.F64{}}.PutFloat64(10.5)
				v2 := value.Value{Address: addr, Type: ir.F64{}}.PutFloat64(2.5)

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
				vInt := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(10)
				vFloat := value.Value{Address: addr, Type: ir.F32{}}.PutFloat32(2.5)

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
				v1 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(10)
				v0 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(0)

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
				{telem.DataType("unknown"), ir.Number{}},
			}

			for _, tt := range tests {
				result := value.DataTypeToIRType(tt.dt)
				Expect(result).To(Equal(tt.expected))
			}
		})
	})

	Describe("FromSeries", func() {
		var param string

		BeforeEach(func() {
			param = "test_param"
		})

		It("Should convert uint64 series", func() {
			data := []uint64{10, 20, 30, 40, 50}
			series := telem.NewSeries(data)

			values := value.FromSeries(series, addr, param)

			Expect(values).To(HaveLen(5))
			for i, v := range values {
				Expect(v.Address).To(Equal(addr))
				Expect(v.Param).To(Equal(param))
				Expect(v.Type).To(Equal(ir.U64{}))
				Expect(v.GetUint64()).To(Equal(data[i]))
			}
		})

		It("Should convert float32 series", func() {
			data := []float32{1.5, 2.5, 3.5, 4.5}
			series := telem.NewSeries(data)

			values := value.FromSeries(series, addr, param)

			Expect(values).To(HaveLen(4))
			for i, v := range values {
				Expect(v.Address).To(Equal(addr))
				Expect(v.Param).To(Equal(param))
				Expect(v.Type).To(Equal(ir.F32{}))
				Expect(v.GetFloat32()).To(Equal(data[i]))
			}
		})

		It("Should convert int32 series", func() {
			data := []int32{-10, -20, 30, 40}
			series := telem.NewSeries(data)

			values := value.FromSeries(series, addr, param)

			Expect(values).To(HaveLen(4))
			for i, v := range values {
				Expect(v.Address).To(Equal(addr))
				Expect(v.Param).To(Equal(param))
				Expect(v.Type).To(Equal(ir.I32{}))
				Expect(v.GetInt32()).To(Equal(data[i]))
			}
		})
	})

	Describe("Complex operations", func() {
		It("Should chain operations correctly", func() {
			v1 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(10)
			v2 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(3)
			v3 := value.Value{Address: addr, Type: ir.I32{}}.PutInt32(2)

			// (10 + 3) * 2 = 26
			result := v1.Add(v2).Mul(v3)
			Expect(result.GetInt32()).To(Equal(int32(26)))

			// (10 - 3) / 2 = 3
			result = v1.Sub(v2).Div(v3)
			Expect(result.GetInt32()).To(Equal(int32(3)))
		})

		It("Should preserve type in operations", func() {
			vU8 := value.Value{Address: addr, Type: ir.U8{}}.PutUint8(250)
			vU32 := value.Value{Address: addr, Type: ir.U32{}}.PutUint32(10)

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
