package gorp_test

import (
	"github.com/arya-analytics/x/gorp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Compound", func() {
	Describe("Next", func() {
		It("Should return the next Retrieve Clause", func() {
			c := &gorp.Compound[int, entry]{}
			r := c.Next()
			Expect(r).To(Equal(c.Clauses[0]))
		})
	})
	Describe("Current", func() {
		It("Should return the current Retrieve Clause", func() {
			c := &gorp.Compound[int, entry]{}
			nr := c.Next()
			r := c.Current()
			Expect(r).To(Equal(nr))
		})
	})

})
