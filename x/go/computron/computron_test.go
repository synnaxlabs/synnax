// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package computron_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/computron"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("Computron", func() {
	Describe("Basic Expressions", func() {
		It("Should multiply two numbers", func() {
			c := MustSucceed(computron.Open("return 2 * 3"))
			defer c.Close()
			v := MustSucceed(c.Run())
			Expect(v.(lua.LNumber)).To(Equal(lua.LNumber(6)))
		})
		It("Should multiply two numeric variables", func() {
			c := MustSucceed(computron.Open("return a * b"))
			defer c.Close()
			c.Set("a", lua.LNumber(2))
			c.Set("b", lua.LNumber(3))
			v := MustSucceed(c.Run())
			Expect(v.(lua.LNumber)).To(Equal(lua.LNumber(6)))
		})
	})

	Describe("Syntax errors", func() {
		It("Should return a nicely formatted error", func() {
			_, err := computron.Open("local a = 1 \n return 2 *")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Data Types", func() {
		It("Should handle string values", func() {
			c := MustSucceed(computron.Open("return 'hello' .. ' world'"))
			defer c.Close()
			v := MustSucceed(c.Run())
			Expect(v.(lua.LString)).To(Equal(lua.LString("hello world")))
		})

		It("Should handle nil values", func() {
			c := MustSucceed(computron.Open("return nil"))
			defer c.Close()
			v := MustSucceed(c.Run())
			Expect(v).To(Equal(lua.LNil))
		})
	})

	Describe("Error Handling", func() {
		It("Should handle runtime errors", func() {
			c := MustSucceed(computron.Open("return a + b"))
			defer c.Close()
			_, err := c.Run()
			Expect(err).To(HaveOccurred())
		})

		It("Should handle undefined variables", func() {
			c := MustSucceed(computron.Open("return undefined_variable + 1"))
			defer c.Close()
			_, err := c.Run()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Multiple Returns", func() {
		It("Should handle scripts with multiple return values", func() {
			// Note: Current implementation only returns first value
			c := MustSucceed(computron.Open("return 1, 2, 3"))
			defer c.Close()
			v := MustSucceed(c.Run())
			Expect(v.(lua.LNumber)).To(Equal(lua.LNumber(1)))
		})
	})

	Describe("SetLValueOnSeries", func() {
		It("Should set the provided lua value on the series", func() {
			c := MustSucceed(computron.Open("return 1"))
			defer c.Close()
			v := MustSucceed(c.Run())
			series := telem.MakeSeries(telem.Uint32T, 1)
			computron.SetLValueOnSeries(v, series, 0)
		})
	})

	Describe("LValueFromMultiSeriesAlignment", func() {
		It("Should return the correct lua value for int32 type", func() {
			series1 := telem.MakeSeries(telem.Int32T, 1)
			telem.SetValueAt(series1, 0, int32(42))
			series1.Alignment = telem.NewAlignment(0, 0)

			series2 := telem.MakeSeries(telem.Int32T, 1)
			telem.SetValueAt(series2, 0, int32(100))
			series2.Alignment = telem.NewAlignment(1, 0)

			multiSeries := telem.NewMultiSeries([]telem.Series{series1, series2})

			v := computron.LValueFromMultiSeriesAlignment(multiSeries, telem.NewAlignment(0, 0))
			Expect(v).To(Equal(lua.LNumber(42)))

			v = computron.LValueFromMultiSeriesAlignment(multiSeries, telem.NewAlignment(1, 0))
			Expect(v).To(Equal(lua.LNumber(100)))
		})

		It("Should return the correct lua value for float64 type", func() {
			series1 := telem.MakeSeries(telem.Float64T, 1)
			telem.SetValueAt(series1, 0, float64(3.14))
			series1.Alignment = telem.NewAlignment(0, 0)

			multiSeries := telem.NewMultiSeries([]telem.Series{series1})

			v := computron.LValueFromMultiSeriesAlignment(multiSeries, telem.NewAlignment(0, 0))
			Expect(v).To(Equal(lua.LNumber(3.14)))
		})

		It("Should return the correct lua value for uint8 type", func() {
			series1 := telem.MakeSeries(telem.Uint8T, 2)
			telem.SetValueAt(series1, 0, uint8(10))
			telem.SetValueAt(series1, 1, uint8(20))
			series1.Alignment = telem.NewAlignment(5, 10)

			multiSeries := telem.NewMultiSeries([]telem.Series{series1})

			v := computron.LValueFromMultiSeriesAlignment(multiSeries, telem.NewAlignment(5, 10))
			Expect(v).To(Equal(lua.LNumber(10)))

			v = computron.LValueFromMultiSeriesAlignment(multiSeries, telem.NewAlignment(5, 11))
			Expect(v).To(Equal(lua.LNumber(20)))
		})

		It("Should handle multiple series with different domain indices", func() {
			series1 := telem.MakeSeries(telem.Float32T, 1)
			telem.SetValueAt(series1, 0, float32(1.5))
			series1.Alignment = telem.NewAlignment(1, 0)

			series2 := telem.MakeSeries(telem.Float32T, 1)
			telem.SetValueAt(series2, 0, float32(2.5))
			series2.Alignment = telem.NewAlignment(2, 0)

			series3 := telem.MakeSeries(telem.Float32T, 1)
			telem.SetValueAt(series3, 0, float32(3.5))
			series3.Alignment = telem.NewAlignment(3, 0)

			multiSeries := telem.NewMultiSeries([]telem.Series{series1, series2, series3})

			v1 := computron.LValueFromMultiSeriesAlignment(multiSeries, telem.NewAlignment(1, 0))
			Expect(v1).To(Equal(lua.LNumber(1.5)))

			v2 := computron.LValueFromMultiSeriesAlignment(multiSeries, telem.NewAlignment(2, 0))
			Expect(v2).To(Equal(lua.LNumber(2.5)))

			v3 := computron.LValueFromMultiSeriesAlignment(multiSeries, telem.NewAlignment(3, 0))
			Expect(v3).To(Equal(lua.LNumber(3.5)))
		})
	})

	Describe("Variable Access", func() {
		It("Should access hyphenated variable names using get function", func() {
			c := MustSucceed(computron.Open("return get('my-variable')"))
			defer c.Close()
			c.Set("my-variable", lua.LNumber(42))
			v := MustSucceed(c.Run())
			Expect(v.(lua.LNumber)).To(Equal(lua.LNumber(42)))
		})
	})
})
