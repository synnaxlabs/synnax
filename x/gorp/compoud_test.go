package gorp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("CompoundRetrieve", func() {
	Describe("Next", func() {
		It("Should return the next RetrieveP Clause", func() {
			c := &gorp.CompoundRetrieve[int, entry]{}
			r := c.Next()
			Expect(r).To(Equal(c.Clauses[0]))
		})
	})
	Describe("Current", func() {
		It("Should return the current RetrieveP Clause", func() {
			c := &gorp.CompoundRetrieve[int, entry]{}
			nr := c.Next()
			r := c.Current()
			Expect(r).To(Equal(nr))
		})
	})

})
