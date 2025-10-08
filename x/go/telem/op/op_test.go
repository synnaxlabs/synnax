// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

package op_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/telem/op"
)

func TestOp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Op Suite")
}

var _ = Describe("Vectorized Operations", func() {
	Describe("Comparison Operations", func() {
		It("should perform GreaterThanF32", func() {
			a := telem.NewSeriesV[float32](1, 5, 3, 8, 2)
			b := telem.NewSeriesV[float32](2, 4, 3, 7, 3)
			output := telem.NewSeriesV[uint8](0, 0, 0, 0, 0)

			op.GreaterThanF32(a, b, output)

			expected := []uint8{0, 1, 0, 1, 0}
			Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
		})

		It("should perform LessThanF64", func() {
			a := telem.NewSeriesV[float64](1.5, 2.5, 3.5)
			b := telem.NewSeriesV[float64](2.0, 2.0, 4.0)
			output := telem.NewSeriesV[uint8](0, 0, 0)

			op.LessThanF64(a, b, output)

			expected := []uint8{1, 0, 1}
			Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
		})

		It("should perform EqualI32", func() {
			a := telem.NewSeriesV[int32](10, 20, 30, 40)
			b := telem.NewSeriesV[int32](10, 25, 30, 35)
			output := telem.NewSeriesV[uint8](0, 0, 0, 0)

			op.EqualI32(a, b, output)

			expected := []uint8{1, 0, 1, 0}
			Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
		})
	})

	Describe("Arithmetic Operations", func() {
		It("should perform AddF32", func() {
			a := telem.NewSeriesV[float32](1.0, 2.0, 3.0, 4.0)
			b := telem.NewSeriesV[float32](0.5, 1.5, 2.5, 3.5)
			output := telem.NewSeriesV[float32](0, 0, 0, 0)

			op.AddF32(a, b, output)

			expected := []float32{1.5, 3.5, 5.5, 7.5}
			Expect(telem.UnmarshalSlice[float32](output.Data, telem.Float32T)).To(Equal(expected))
		})

		It("should perform SubtractF64", func() {
			a := telem.NewSeriesV[float64](10.0, 20.0, 30.0)
			b := telem.NewSeriesV[float64](3.0, 5.0, 7.0)
			output := telem.NewSeriesV[float64](0, 0, 0)

			op.SubtractF64(a, b, output)

			expected := []float64{7.0, 15.0, 23.0}
			Expect(telem.UnmarshalSlice[float64](output.Data, telem.Float64T)).To(Equal(expected))
		})

		It("should perform MultiplyI32", func() {
			a := telem.NewSeriesV[int32](2, 3, 4, 5)
			b := telem.NewSeriesV[int32](3, 4, 5, 6)
			output := telem.NewSeriesV[int32](0, 0, 0, 0)

			op.MultiplyI32(a, b, output)

			expected := []int32{6, 12, 20, 30}
			Expect(telem.UnmarshalSlice[int32](output.Data, telem.Int32T)).To(Equal(expected))
		})

		It("should perform DivideI64", func() {
			a := telem.NewSeriesV[int64](20, 30, 40, 50)
			b := telem.NewSeriesV[int64](2, 3, 4, 5)
			output := telem.NewSeriesV[int64](0, 0, 0, 0)

			op.DivideI64(a, b, output)

			expected := []int64{10, 10, 10, 10}
			Expect(telem.UnmarshalSlice[int64](output.Data, telem.Int64T)).To(Equal(expected))
		})
	})

})