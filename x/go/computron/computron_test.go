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
})
