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
			series := telem.AllocSeries(telem.Uint32T, 1)
			computron.SetLValueOnSeries(v, series, 0)
		})
	})

	Describe("LValueFromSeries", func() {
		It("Should return the correct lua value from the series", func() {
			series := telem.NewSeriesV[float32](1.0)
			v := computron.LValueFromSeries(series, 0)
			Expect(v).To(Equal(lua.LNumber(1.0)))
		})
	})
})
