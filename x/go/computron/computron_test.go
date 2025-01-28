package computron_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/computron"
	. "github.com/synnaxlabs/x/testutil"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("Computron", func() {
	Describe("Basic Expressions", func() {
		It("Should multiply two numbers", func() {
			c := MustSucceed(computron.Open("return 2 * 3"))
			v := MustSucceed(c.Run())
			Expect(v.(lua.LNumber)).To(Equal(lua.LNumber(6)))
		})
		It("Should multiply two numeric variables", func() {
			c := MustSucceed(computron.Open("return a * b"))
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
			v := MustSucceed(c.Run())
			Expect(v.(lua.LString)).To(Equal(lua.LString("hello world")))
		})

		It("Should handle nil values", func() {
			c := MustSucceed(computron.Open("return nil"))
			v := MustSucceed(c.Run())
			Expect(v).To(Equal(lua.LNil))
		})
	})

	Describe("Error Handling", func() {
		It("Should handle runtime errors", func() {
			c := MustSucceed(computron.Open("return a + b"))
			_, err := c.Run()
			Expect(err).To(HaveOccurred())
		})

		It("Should handle undefined variables", func() {
			c := MustSucceed(computron.Open("return undefined_variable + 1"))
			_, err := c.Run()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Multiple Returns", func() {
		It("Should handle scripts with multiple return values", func() {
			// Note: Current implementation only returns first value
			c := MustSucceed(computron.Open("return 1, 2, 3"))
			v := MustSucceed(c.Run())
			Expect(v.(lua.LNumber)).To(Equal(lua.LNumber(1)))
		})
	})
})
