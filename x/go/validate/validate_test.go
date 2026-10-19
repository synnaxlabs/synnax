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
	"strings"

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
			Expect(v.Error()).To(MatchError(ContainSubstring("error 42")))
		})

		It("Should include the field name in the error", func() {
			v.Ternaryf("myField", true, "invalid value")
			Expect(v.Error()).To(MatchError(ContainSubstring("my_field")))
		})
	})

	Describe("Validation Helpers", func() {
		Describe("NotNil", func() {
			It("Should validate non-nil values", func() {
				value := "not nil"
				Expect(v.NotNil("field", &value)).To(BeFalse())
				Expect(v.Error()).NotTo(HaveOccurred())
			})
			var p *any
			var f func()
			var m map[any]any
			var s []any
			var c chan any
			var i any
			DescribeTable("Should catch nil values", func(value any) {
				Expect(v.NotNil("field", value)).To(BeTrue())
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
					Expect(v.Positive("field", 42)).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
				})

				It("Should catch non-positive numbers", func() {
					Expect(v.Positive("field", 0)).To(BeTrue())
					Expect(v.Error()).To(HaveOccurred())
				})
			})

			Describe("Filtering", func() {
				It("Should validate numbers greater than threshold", func() {
					Expect(v.GreaterThan("field", 10, 5)).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
				})

				It("Should catch numbers less than or equal to threshold", func() {
					Expect(v.GreaterThan("field", 5, 5)).To(BeTrue())
					Expect(v.Error()).To(HaveOccurred())
				})
			})

			Describe("InBounds", func() {
				It("Should validate values within bounds", func() {
					Expect(v.InBounds("field", 5, 1, 10)).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
				})

				It("Should accept values at lower bound (inclusive)", func() {
					Expect(v.InBounds("field", 1, 1, 10)).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
				})

				It("Should reject values at upper bound (exclusive)", func() {
					Expect(v.InBounds("field", 10, 1, 10)).To(BeTrue())
					Expect(v.Error()).To(HaveOccurred())
				})

				It("Should reject values below lower bound", func() {
					Expect(v.InBounds("field", 0, 1, 10)).To(BeTrue())
					Expect(v.Error()).To(HaveOccurred())
				})

				It("Should reject values above upper bound", func() {
					Expect(v.InBounds("field", 11, 1, 10)).To(BeTrue())
					Expect(v.Error()).To(HaveOccurred())
				})

				It("Should work with float values", func() {
					Expect(v.InBounds("field", 5.5, 1.0, 10.0)).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
				})

				It("Should include bounds in error message", func() {
					v.InBounds("field", 0, 1, 10)
					Expect(v.Error()).To(MatchError(ContainSubstring("[1, 10)")))
				})
			})
		})

		Describe("Collection Validations", func() {
			Describe("NotEmptySlice", func() {
				It("Should validate non-empty slices", func() {
					slice := []int{1, 2, 3}
					Expect(v.NotEmptySlice("field", slice)).To(BeFalse())
					Expect(v.Error()).ToNot(HaveOccurred())
				})

				It("Should catch empty slices", func() {
					slice := []int{}
					Expect(v.NotEmptySlice("field", slice)).To(BeTrue())
					Expect(v.Error()).To(HaveOccurred())
				})
			})

			Describe("NotEmptyString", func() {
				It("Should validate non-empty strings", func() {
					Expect(v.NotEmptyString("field", "hello")).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
				})

				It("Should catch empty strings", func() {
					Expect(v.NotEmptyString("field", "")).To(BeTrue())
					Expect(v.Error()).To(HaveOccurred())
				})
			})
		})

		Describe("IsIdentifier", func() {
			DescribeTable("Should accept identifiers",
				func(s string) { Expect(validate.IsIdentifier(s)).To(BeTrue()) },
				Entry("single lowercase letter", "a"),
				Entry("single uppercase letter", "Z"),
				Entry("single underscore", "_"),
				Entry("lowercase word", "tank"),
				Entry("uppercase word", "TANK"),
				Entry("mixed case", "TankPressure"),
				Entry("snake case", "tank_pressure"),
				Entry("screaming snake case", "TANK_PRESSURE"),
				Entry("underscore prefix", "_tank"),
				Entry("double underscore prefix", "__tank"),
				Entry("underscore suffix", "tank_"),
				Entry("only underscores", "___"),
				Entry("digit after a letter", "t1"),
				Entry("digit after an underscore", "_1"),
				Entry("digits after an underscore separator", "tank_1"),
				Entry("digits in the middle", "tank1pressure"),
				Entry("all digits after the first letter", "a0123456789"),
				Entry("every letter and digit", "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"),
				Entry("long name", strings.Repeat("a", 1000)),
			)

			DescribeTable("Should reject non-identifiers",
				func(s string) { Expect(validate.IsIdentifier(s)).To(BeFalse()) },
				Entry("empty", ""),
				Entry("single digit", "1"),
				Entry("leading digit", "1tank"),
				Entry("all digits", "123"),
				Entry("single space", " "),
				Entry("leading space", " tank"),
				Entry("trailing space", "tank "),
				Entry("inner space", "tank 1"),
				Entry("tab", "tank\t1"),
				Entry("newline", "tank\n1"),
				Entry("trailing newline", "tank\n"),
				Entry("carriage return", "tank\r"),
				Entry("dash", "tank-1"),
				Entry("leading dash", "-tank"),
				Entry("dot", "tank.1"),
				Entry("leading dot", ".tank"),
				Entry("slash", "tank/1"),
				Entry("backslash", "tank\\1"),
				Entry("colon", "tank:1"),
				Entry("comma", "tank,1"),
				Entry("semicolon", "tank;1"),
				Entry("plus", "tank+1"),
				Entry("asterisk", "tank*"),
				Entry("question mark", "tank?"),
				Entry("caret", "^tank"),
				Entry("dollar", "tank$"),
				Entry("pipe", "tank|1"),
				Entry("ampersand", "tank&1"),
				Entry("percent", "tank%"),
				Entry("hash", "#tank"),
				Entry("at sign", "@tank"),
				Entry("exclamation", "tank!"),
				Entry("tilde", "~tank"),
				Entry("backtick", "`tank`"),
				Entry("equals", "tank=1"),
				Entry("parentheses", "tank()"),
				Entry("brackets", "tank[1]"),
				Entry("braces", "tank{1}"),
				Entry("angle brackets", "<tank>"),
				Entry("single quote", "'tank'"),
				Entry("double quote", "\"tank\""),
				Entry("regex character class", "[a-z]+"),
				Entry("regex wildcard", "tank.*"),
				Entry("non-ASCII letter", "tänk"),
				Entry("leading non-ASCII letter", "étank"),
				Entry("non-ASCII digit", "tank١"),
				Entry("CJK", "タンク"),
				Entry("emoji", "tank🚀"),
				Entry("zero width space", "tank​1"),
				Entry("non-breaking space", "tank 1"),
				Entry("null byte", "tank\x001"),
				Entry("fullwidth underscore", "tank＿1"),
			)
		})

		Describe("Zeroable", func() {
			It("Should validate non-zero zeroables", func() {
				z := testZeroable{value: 42}
				Expect(v.NonZeroable("field", z)).To(BeFalse())
				Expect(v.Error()).NotTo(HaveOccurred())
			})

			It("Should catch zero zeroables", func() {
				z := testZeroable{value: 0}
				Expect(v.NonZeroable("field", z)).To(BeTrue())
				Expect(v.Error()).To(HaveOccurred())
			})
		})
	})
})
