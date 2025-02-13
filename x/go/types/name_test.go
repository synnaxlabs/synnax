package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/types"
)

type customNamed struct{}

func (c customNamed) CustomTypeName() string { return "CustomName" }

type regularType struct{}
type yEndingType struct{}
type boxType struct{}
type classType struct{}
type bushType struct{}
type catchType struct{}

type box struct{}
type class struct{}
type bus struct{}
type catch struct{}

var _ = Describe("Name", func() {
	Context("Name", func() {
		It("Should return the type name for a regular struct", func() {
			Expect(types.Name[regularType]()).To(Equal("regularType"))
		})

		It("Should return the custom name for a type implementing CustomTypeName", func() {
			Expect(types.Name[customNamed]()).To(Equal("CustomName"))
		})

		It("Should work with built-in types", func() {
			Expect(types.Name[string]()).To(Equal("string"))
			Expect(types.Name[int]()).To(Equal("int"))
		})
	})

	Context("PluralName", func() {
		It("Should convert y endings to ies", func() {
			Expect(types.PluralName[yEndingType]()).To(Equal("yEndingTypes"))
		})

		It("Should add es to words ending in s, x, z", func() {
			Expect(types.PluralName[box]()).To(Equal("boxes"))
			Expect(types.PluralName[class]()).To(Equal("classes"))
			Expect(types.PluralName[bus]()).To(Equal("buses"))
		})

		It("Should add es to words ending in ch, sh", func() {
			Expect(types.PluralName[catch]()).To(Equal("catches"))
		})

		It("Should add s to regular words", func() {
			Expect(types.PluralName[regularType]()).To(Equal("regularTypes"))
		})

		It("Should work with custom named types", func() {
			Expect(types.PluralName[customNamed]()).To(Equal("CustomNames"))
		})

		It("Should work with built-in types", func() {
			Expect(types.PluralName[string]()).To(Equal("strings"))
			Expect(types.PluralName[int]()).To(Equal("ints"))
		})
	})
})
