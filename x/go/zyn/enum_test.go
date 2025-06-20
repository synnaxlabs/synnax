package zyn_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Enum", func() {
	Describe("Basic Parsing", func() {
		Specify("string enum", func() {
			var dest string
			Expect(zyn.Enum("a", "b", "c").Parse("a", &dest)).To(Succeed())
			Expect(dest).To(Equal("a"))
		})

		Specify("int enum", func() {
			var dest int
			Expect(zyn.Enum(1, 2, 3).Parse(1, &dest)).To(Succeed())
			Expect(dest).To(Equal(1))
		})

		Specify("float enum", func() {
			var dest float64
			Expect(zyn.Enum(1.0, 2.0, 3.0).Parse(1.0, &dest)).To(Succeed())
			Expect(dest).To(Equal(1.0))
		})
	})

	Describe("Validate", func() {
		It("Should return nil if the value is a valid enum", func() {
			Expect(zyn.Enum("a", "b", "c").Validate("a")).To(Succeed())
		})
		It("Should return nil if the value is not a valid enum", func() {
			Expect(zyn.Enum("a", "b", "c").Validate("d")).To(HaveOccurred())
		})
	})

	Describe("DataType Validation", func() {
		Specify("invalid value", func() {
			var dest string
			Expect(zyn.Enum("a", "b", "c").Parse("d", &dest)).To(MatchError(ContainSubstring("invalid enum value")))
		})

		Specify("invalid type", func() {
			var dest string
			Expect(zyn.Enum("a", "b", "c").Parse(1, &dest)).To(MatchError(ContainSubstring("invalid enum value")))
		})

		Specify("type conversion", func() {
			var dest int
			Expect(zyn.Enum[int](1, 2, 3).Parse(int64(1), &dest)).To(Succeed())
			Expect(dest).To(Equal(1))
		})
	})

	Describe("Optional Fields", func() {
		Specify("optional field with nil value", func() {
			var dest *string
			Expect(zyn.Enum("a", "b", "c").Optional().Parse(nil, &dest)).To(Succeed())
			Expect(dest).To(BeNil())
		})

		Specify("required field with nil value", func() {
			var dest string
			Expect(zyn.Enum("a", "b", "c").Parse(nil, &dest)).To(MatchError(ContainSubstring("required")))
		})

		Specify("optional field with value", func() {
			var dest *string
			Expect(zyn.Enum("a", "b", "c").Optional().Parse("a", &dest)).To(Succeed())
			Expect(*dest).To(Equal("a"))
		})
	})

	Describe("Dump", func() {
		Specify("valid value", func() {
			result, err := zyn.Enum("a", "b", "c").Dump("a")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("a"))
		})

		Specify("invalid value", func() {
			_, err := zyn.Enum("a", "b", "c").Dump("d")
			Expect(err).To(MatchError(ContainSubstring("invalid enum value")))
		})

		Specify("nil value", func() {
			_, err := zyn.Enum("a", "b", "c").Dump(nil)
			Expect(err).To(MatchError(ContainSubstring("required")))
		})

		Specify("optional nil value", func() {
			result, err := zyn.Enum("a", "b", "c").Optional().Dump(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("Values", func() {
		Specify("add values", func() {
			enum := zyn.Enum("a", "b").Values("c", "d")
			var dest string
			Expect(enum.Parse("c", &dest)).To(Succeed())
			Expect(dest).To(Equal("c"))
		})

		It("should panic on empty", func() {
			Expect(func() { zyn.Enum[any]() }).To(Panic())
		})
	})

	Describe("Custom Types", func() {
		type MyEnum string

		Specify("custom type enum", func() {
			var dest MyEnum
			Expect(zyn.Enum(MyEnum("a"), MyEnum("b")).Parse(MyEnum("a"), &dest)).To(Succeed())
			Expect(dest).To(Equal(MyEnum("a")))
		})

		Specify("custom type conversion", func() {
			var dest string
			Expect(zyn.Enum(MyEnum("a"), MyEnum("b")).Parse(MyEnum("a"), &dest)).To(Succeed())
			Expect(dest).To(Equal("a"))
		})
	})
})
