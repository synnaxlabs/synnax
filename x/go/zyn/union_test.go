package zyn_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Union", func() {
	Describe("Basic Parsing", func() {
		Specify("string or int union", func() {
			var dest string
			Expect(zyn.Union(zyn.String(), zyn.Int()).Parse("hello", &dest)).To(Succeed())
			Expect(dest).To(Equal("hello"))

			var destInt int
			Expect(zyn.Union(zyn.String(), zyn.Int()).Parse(42, &destInt)).To(Succeed())
			Expect(destInt).To(Equal(42))
		})

		Specify("multiple number types", func() {
			var dest int
			Expect(zyn.Union(zyn.Float64(), zyn.Int()).Parse(42, &dest)).To(Succeed())
			Expect(dest).To(Equal(42))

			var res float64
			Expect(zyn.Union(zyn.Float64(), zyn.Int()).Parse(42.5, &res)).To(Succeed())
			Expect(res).To(Equal(42.5))
		})

		Specify("Parsing into any", func() {
			var dest any
			Expect(zyn.Union(zyn.Float64(), zyn.Int()).Parse(42, &dest)).To(Succeed())
			Expect(dest).To(Equal(42))
		})

		Specify("Should respect exact matches", func() {
			var dest any
			Expect(zyn.Union(zyn.Float64(), zyn.Int()).Parse(float64(12), &dest)).To(Succeed())
			Expect(dest).To(Equal(float64(12)))
		})
	})

	Describe("Type Validation", func() {
		Specify("invalid value", func() {
			var dest string
			Expect(zyn.Union(zyn.String(), zyn.Int()).Parse(true, &dest)).To(Succeed())
			Expect(dest).To(Equal("true"))
		})

		Specify("type conversion", func() {
			var dest int
			Expect(zyn.Union(zyn.Int().Coerce(), zyn.Float64()).Parse(int64(42), &dest)).To(Succeed())
			Expect(dest).To(Equal(42))
		})
	})

	Describe("Optional Fields", func() {
		Specify("optional field with nil value", func() {
			var dest *string
			Expect(zyn.Union(zyn.String(), zyn.Int()).Optional().Parse(nil, &dest)).To(Succeed())
			Expect(dest).To(BeNil())
		})

		Specify("required field with nil value", func() {
			var dest string
			Expect(zyn.Union(zyn.String(), zyn.Int()).Parse(nil, &dest)).To(MatchError(ContainSubstring("required")))
		})

		Specify("optional field with value", func() {
			var dest string
			Expect(zyn.Union(zyn.String(), zyn.Int()).Optional().Parse("hello", &dest)).To(Succeed())
			Expect(dest).To(Equal("hello"))
		})
	})

	Describe("Dump", func() {
		Specify("valid string value", func() {
			result, err := zyn.Union(zyn.String(), zyn.Int()).Dump("hello")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("hello"))
		})

		Specify("valid int value", func() {
			result, err := zyn.Union(zyn.String(), zyn.Int()).Dump(42)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(42))
		})

		Specify("invalid value", func() {
			_, err := zyn.Union(zyn.String(), zyn.Int()).Dump(struct{}{})
			Expect(err).To(MatchError(ContainSubstring("expected int but received struct {}")))
		})

		Specify("nil value", func() {
			_, err := zyn.Union(zyn.String(), zyn.Int()).Dump(nil)
			Expect(err).To(MatchError(ContainSubstring("required")))
		})

		Specify("optional nil value", func() {
			result, err := zyn.Union(zyn.String(), zyn.Int()).Optional().Dump(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("Complex Unions", func() {
		Specify("multiple types with custom type", func() {
			type MyString string
			var dest MyString
			Expect(zyn.Union(zyn.String(), zyn.Enum(MyString("a"), MyString("b"))).Parse(MyString("a"), &dest)).To(Succeed())
			Expect(dest).To(Equal(MyString("a")))
		})

		Specify("nested unions", func() {
			var dest string
			Expect(zyn.Union(
				zyn.String(),
				zyn.Union(zyn.Int(), zyn.Float64()),
			).Parse(42, &dest)).To(Succeed())
			Expect(dest).To(Equal("42"))
		})
	})
})
