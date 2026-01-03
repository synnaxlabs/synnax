// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/validate"
)

type testZeroable struct {
	value int
}

func (t testZeroable) IsZero() bool { return t.value == 0 }

var _ = Describe("Validate", func() {
	var v *validate.Validator

	BeforeEach(func() {
		v = validate.New("test")
	})

	Describe("Ternary", func() {
		It("Should accumulate the error if the condition is met", func() {
			v.Ternary("field", true, "error message")
			Expect(v.Error()).To(HaveOccurred())
		})

		It("Should not accumulate the error if the condition is not met", func() {
			v.Ternary("field", false, "error message")
			Expect(v.Error()).NotTo(HaveOccurred())
		})

		It("Should return true if an error was accumulated", func() {
			result := v.Ternary("field", true, "error message")
			Expect(result).To(BeTrue())
		})
	})

	Describe("Ternaryf", func() {
		It("Should format the error message correctly", func() {
			v.Ternaryf("field", true, "error %d", 42)
			Expect(v.Error().Error()).To(ContainSubstring("error 42"))
		})

		It("Should include the field name in the error", func() {
			v.Ternaryf("myField", true, "invalid value")
			Expect(v.Error().Error()).To(ContainSubstring("my_field"))
		})
	})

	Describe("Validation Helpers", func() {
		Describe("NotNil", func() {
			It("Should validate non-nil values", func() {
				value := "not nil"
				Expect(validate.NotNil(v, "field", &value)).To(BeFalse())
				Expect(v.Error()).NotTo(HaveOccurred())
			})
			var p *any
			var f func()
			var m map[any]any
			var s []any
			var c chan any
			var i any
			DescribeTable("Should catch nil values", func(value any) {
				Expect(validate.NotNil(v, "field", value)).To(BeTrue())
				Expect(v.Error()).To(MatchError(ContainSubstring("must be non-nil")))
			},
				Entry("pointers", p),
				Entry("functions", f),
				Entry("maps", m),
				Entry("slices", s),
				Entry("channels", c),
				Entry("interfaces", i),
			)
		})

		Describe("Numeric Validations", func() {
			Describe("Positive", func() {
				It("Should validate positive numbers", func() {
					Expect(validate.Positive(v, "field", 42)).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
				})

				It("Should catch non-positive numbers", func() {
					Expect(validate.Positive(v, "field", 0)).To(BeTrue())
					Expect(v.Error()).To(HaveOccurred())
				})
			})
		})

		Describe("Collection Validations", func() {
			Describe("NotEmptySlice", func() {
				It("Should validate non-empty slices", func() {
					slice := []int{1, 2, 3}
					Expect(validate.NotEmptySlice(v, "field", slice)).To(BeFalse())
					Expect(v.Error()).ToNot(HaveOccurred())
				})

				It("Should catch empty slices", func() {
					slice := []int{}
					Expect(validate.NotEmptySlice(v, "field", slice)).To(BeTrue())
					Expect(v.Error()).To(HaveOccurred())
				})
			})

			Describe("NotEmptyString", func() {
				It("Should validate non-empty strings", func() {
					Expect(validate.NotEmptyString(v, "field", "hello")).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
				})

				It("Should catch empty strings", func() {
					Expect(validate.NotEmptyString(v, "field", "")).To(BeTrue())
					Expect(v.Error()).To(HaveOccurred())
				})
			})
		})

		Describe("Zeroable", func() {

			It("Should validate non-zero zeroables", func() {
				z := testZeroable{value: 42}
				Expect(validate.NonZeroable(v, "field", z)).To(BeFalse())
				Expect(v.Error()).NotTo(HaveOccurred())
			})

			It("Should catch zero zeroables", func() {
				z := testZeroable{value: 0}
				Expect(validate.NonZeroable(v, "field", z)).To(BeTrue())
				Expect(v.Error()).To(HaveOccurred())
			})
		})
	})
})
