// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unsafe_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/unsafe"
)

var _ = Describe("Unsafe", func() {
	Describe("ReinterpretSlice", func() {
		type myCustomUint32 uint32
		It("should convert a slice of one type to a slice of another type", func() {
			in := []uint32{1, 2, 3}
			out := unsafe.ReinterpretSlice[uint32, myCustomUint32](in)
			Expect(out).To(Equal([]myCustomUint32{1, 2, 3}))
		})
	})
	Describe("ReinterpretMap", func() {
		type myCustomUint32 uint32
		type myCustomUint64 uint64
		It("should convert a map of one type to a map of another type", func() {
			in := map[uint32]uint64{1: 1, 2: 2, 3: 3}
			out := unsafe.ReinterpretMap[uint32, uint64, myCustomUint32, myCustomUint64](in)
			Expect(out).To(Equal(map[myCustomUint32]myCustomUint64{1: 1, 2: 2, 3: 3}))
		})
	})

	Describe("CastSlice", func() {
		Context("Larger to Smaller Types", func() {
			It("should convert float64 (8 bytes) to uint8 (1 byte)", func() {
				// 1 float64 = 8 bytes = 8 uint8s
				in := []float64{1.0, 2.0}
				out := unsafe.CastSlice[float64, uint8](in)
				Expect(len(out)).To(Equal(16)) // 2 * 8 = 16 bytes
			})

			It("should convert int64 (8 bytes) to int32 (4 bytes)", func() {
				// 2 int64s = 16 bytes = 4 int32s
				in := []int64{1, 2}
				out := unsafe.CastSlice[int64, int32](in)
				Expect(len(out)).To(Equal(4)) // 2 * 2 = 4 int32s
			})

			It("should convert int64 (8 bytes) to int16 (2 bytes)", func() {
				// 1 int64 = 8 bytes = 4 int16s
				in := []int64{0x0004000300020001}
				out := unsafe.CastSlice[int64, int16](in)
				Expect(len(out)).To(Equal(4)) // 1 * 4 = 4 int16s
				// Verify byte-level preservation (little-endian)
				Expect(out[0]).To(Equal(int16(0x0001)))
				Expect(out[1]).To(Equal(int16(0x0002)))
				Expect(out[2]).To(Equal(int16(0x0003)))
				Expect(out[3]).To(Equal(int16(0x0004)))
			})

			It("should convert uint32 (4 bytes) to uint8 (1 byte)", func() {
				in := []uint32{0x04030201, 0x08070605}
				out := unsafe.CastSlice[uint32, uint8](in)
				Expect(len(out)).To(Equal(8)) // 2 * 4 = 8 bytes
				// Verify byte-level preservation (little-endian)
				Expect(out[0]).To(Equal(uint8(0x01)))
				Expect(out[1]).To(Equal(uint8(0x02)))
				Expect(out[2]).To(Equal(uint8(0x03)))
				Expect(out[3]).To(Equal(uint8(0x04)))
				Expect(out[4]).To(Equal(uint8(0x05)))
				Expect(out[5]).To(Equal(uint8(0x06)))
				Expect(out[6]).To(Equal(uint8(0x07)))
				Expect(out[7]).To(Equal(uint8(0x08)))
			})
		})

		Context("Smaller to Larger Types", func() {
			It("should convert uint8 (1 byte) to float64 (8 bytes)", func() {
				// 8 uint8s = 8 bytes = 1 float64
				in := []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 240, 63}
				out := unsafe.CastSlice[uint8, float64](in)
				Expect(len(out)).To(Equal(2)) // 16 bytes / 8 = 2 float64s
			})

			It("should convert int16 (2 bytes) to int64 (8 bytes)", func() {
				// 4 int16s = 8 bytes = 1 int64
				in := []int16{1, 2, 3, 4, 5, 6, 7, 8}
				out := unsafe.CastSlice[int16, int64](in)
				Expect(len(out)).To(Equal(2)) // 16 bytes / 8 = 2 int64s
			})

			It("should convert uint8 (1 byte) to uint32 (4 bytes)", func() {
				// 4 uint8s = 4 bytes = 1 uint32
				in := []uint8{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
				out := unsafe.CastSlice[uint8, uint32](in)
				Expect(len(out)).To(Equal(2)) // 8 bytes / 4 = 2 uint32s
				// Verify byte-level preservation (little-endian)
				Expect(out[0]).To(Equal(uint32(0x04030201)))
				Expect(out[1]).To(Equal(uint32(0x08070605)))
			})
		})

		Context("Same Size Types", func() {
			It("should convert float32 (4 bytes) to uint32 (4 bytes)", func() {
				in := []float32{1.0, 2.0, 3.0}
				out := unsafe.CastSlice[float32, uint32](in)
				Expect(len(out)).To(Equal(3)) // Same number of elements
			})

			It("should convert int32 to uint32", func() {
				in := []int32{-1, 0, 1, 100}
				out := unsafe.CastSlice[int32, uint32](in)
				Expect(len(out)).To(Equal(4))
				Expect(out[0]).To(Equal(uint32(0xFFFFFFFF))) // -1 as unsigned
				Expect(out[1]).To(Equal(uint32(0)))
				Expect(out[2]).To(Equal(uint32(1)))
				Expect(out[3]).To(Equal(uint32(100)))
			})

			It("should convert float64 to uint64", func() {
				in := []float64{1.0}
				out := unsafe.CastSlice[float64, uint64](in)
				Expect(len(out)).To(Equal(1))
				// 1.0 as float64 in IEEE 754 format
				Expect(out[0]).To(Equal(uint64(0x3FF0000000000000)))
			})
		})

		Context("Edge Cases", func() {
			It("should handle empty slices", func() {
				in := []float64{}
				out := unsafe.CastSlice[float64, uint8](in)
				Expect(out).To(BeNil())
			})

			It("should handle single element conversions", func() {
				in := []uint64{0x0807060504030201}
				out := unsafe.CastSlice[uint64, uint8](in)
				Expect(len(out)).To(Equal(8))
				Expect(out[0]).To(Equal(uint8(0x01)))
				Expect(out[7]).To(Equal(uint8(0x08)))
			})

			It("should handle large slices efficiently", func() {
				in := make([]float64, 1000)
				for i := range in {
					in[i] = float64(i)
				}
				out := unsafe.CastSlice[float64, uint8](in)
				Expect(len(out)).To(Equal(8000)) // 1000 * 8 = 8000 bytes
			})
		})

		Describe("Panic Conditions", func() {
			It("Should panic on zero length types", func() {
				in := []float32{1.0, 2.0, 3.0}
				Expect(func() {
					unsafe.CastSlice[float32, struct{}](in)
				}).To(Panic())
			})

			It("Should panic when the stride lengths are incompatible", func() {
				in := []byte{1, 2, 3}
				Expect(func() {
					unsafe.CastSlice[byte, uint32](in)
				}).To(PanicWith("unsafe.CastSlice: incompatible element size 1 (uint8) with total byte length 3 and element with size 4 (uint32)"))
			})
		})

		Context("Bidirectional Conversions", func() {
			It("should preserve data through round-trip conversions", func() {
				// float64 -> uint8 -> float64
				original := []float64{1.5, 2.5, 3.5}
				bytes := unsafe.CastSlice[float64, uint8](original)
				roundtrip := unsafe.CastSlice[uint8, float64](bytes)
				Expect(roundtrip).To(Equal(original))
			})

			It("should preserve data through int32 -> uint8 -> int32", func() {
				original := []int32{-100, 0, 100, 999}
				bytes := unsafe.CastSlice[int32, uint8](original)
				roundtrip := unsafe.CastSlice[uint8, int32](bytes)
				Expect(roundtrip).To(Equal(original))
			})
		})
	})

	Describe("CastBytes", func() {
		It("Should cast bytes to a single element", func() {
			b := unsafe.CastBytes[uint32]([]byte{1, 2, 3, 4})
			Expect(b).To(Equal(uint32(67305985)))
		})

	})
})
