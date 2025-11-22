// Copyright 2025 Synnax Labs, Inc.
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

	Describe("Func", func() {
		It("Should execute the validation function and accumulate error", func() {
			executed := false
			v.Func(func() bool {
				executed = true
				return true
			}, "error message")
			Expect(executed).To(BeTrue())
			Expect(v.Error()).To(HaveOccurred())
		})

		It("Should not accumulate error if function returns false", func() {
			v.Func(func() bool {
				return false
			}, "error message")
			Expect(v.Error()).NotTo(HaveOccurred())
		})

		It("Should short circuit if previous error exists", func() {
			v.Ternary("field1", true, "first error")
			executed := false
			v.Func(func() bool {
				executed = true
				return true
			}, "second error")
			Expect(executed).To(BeFalse())
		})
	})

	Describe("Validation Helpers", func() {
		Describe("NotNil", func() {
			It("Should validate non-nil values", func() {
				value := "not nil"
				Expect(validate.NotNil(v, "field", &value)).To(BeFalse())
				Expect(v.Error()).NotTo(HaveOccurred())
			})

			It("Should catch pointers", func() {
				var value *string
				Expect(validate.NotNil(v, "field", value)).To(BeTrue())
				Expect(v.Error()).To(HaveOccurred())
			})
			It("Should catch functions", func() {
				var fn func() bool
				Expect(validate.NotNil(v, "field", fn)).To(BeTrue())
				Expect(v.Error()).To(MatchError(ContainSubstring("must be non-nil")))
			})
			It("should catch maps", func() {
				var m map[string]string
				Expect(validate.NotNil(v, "field", m)).To(BeTrue())
				Expect(v.Error()).To(MatchError(ContainSubstring("must be non-nil")))
			})
			It("should catch slices", func() {
				var s []string
				Expect(validate.NotNil(v, "field", s)).To(BeTrue())
				Expect(v.Error()).To(MatchError(ContainSubstring("must be non-nil")))
			})
			It("should catch channels", func() {
				var c chan string
				Expect(validate.NotNil(v, "field", c)).To(BeTrue())
				Expect(v.Error()).To(MatchError(ContainSubstring("must be non-nil")))
			})
			It("should catch interfaces", func() {
				var i any
				Expect(validate.NotNil(v, "field", i)).To(BeTrue())
				Expect(v.Error()).To(MatchError(ContainSubstring("must be non-nil")))
			})
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

			Describe("Filtering", func() {
				It("Should validate numbers greater than threshold", func() {
					Expect(validate.GreaterThan(v, "field", 10, 5)).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
				})

				It("Should catch numbers less than or equal to threshold", func() {
					Expect(validate.GreaterThan(v, "field", 5, 5)).To(BeTrue())
					Expect(v.Error()).To(HaveOccurred())
				})
			})

			Describe("LessThan", func() {
				It("Should validate numbers less than threshold", func() {
					Expect(validate.LessThan(v, "field", 5, 10)).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
				})

				It("Should catch numbers greater than or equal to threshold", func() {
					Expect(validate.LessThan(v, "field", 10, 10)).To(BeTrue())
					Expect(v.Error()).To(HaveOccurred())
				})
			})
		})

		Describe("Collection Validations", func() {
			Describe("NotEmptySlice", func() {
				It("Should validate non-empty slices", func() {
					slice := []int{1, 2, 3}
					Expect(validate.NotEmptySlice(v, "field", slice)).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
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
