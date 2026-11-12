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
	"math"
	"runtime"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
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

	Describe("Exec", func() {
		It("Should accumulate no error for a passing check", func() {
			v.Exec(func() error { return nil })
			Expect(v.Error()).To(Succeed())
		})

		It("Should return a sole failure unchanged", func() {
			err := errors.New("failure")
			v.Exec(func() error { return err })
			v.Exec(func() error { return nil })
			Expect(v.Error()).To(BeIdenticalTo(err))
		})

		It("Should flatten joined failures", func() {
			v.Exec(func() error {
				return errors.Join(errors.New("first"), errors.New("second"))
			})
			v.Exec(func() error { return errors.New("third") })
			Expect(errors.UnwrapMulti(v.Error())).To(HaveLen(3))
			Expect(v.Error()).To(MatchError("first\nsecond\nthird"))
		})

		It("Should allocate in linear proportion to the number of failures", func() {
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			for range 20000 {
				v.Exec(func() error { return validate.ErrRequired })
			}
			runtime.ReadMemStats(&after)
			Expect(after.TotalAlloc - before.TotalAlloc).To(
				BeNumerically("<", 256<<20),
			)
			Expect(errors.UnwrapMulti(v.Error())).To(HaveLen(20000))
		})
	})

	Describe("Ternary", func() {
		It("Should accumulate the error if the condition is met", func() {
			v.Ternary("field", true, "error message")
			Expect(v.Error()).To(MatchError(ContainSubstring("field: error message")))
		})

		It("Should not accumulate the error if the condition is not met", func() {
			v.Ternary("field", false, "error message")
			Expect(v.Error()).NotTo(HaveOccurred())
		})

		It("Should keep one error when later checks pass", func() {
			v.Ternary("field", true, "error message")
			for range 64 {
				v.Ternary("other", false, "other message")
			}
			Expect(errors.UnwrapMulti(v.Error())).To(BeNil())
			Expect(v.Error()).To(MatchError("field: error message"))
		})

		It("Should keep every failure flat when later checks pass", func() {
			v.Ternary("first", true, "first message")
			v.Ternary("second", true, "second message")
			for range 64 {
				v.Ternary("other", false, "other message")
			}
			Expect(errors.UnwrapMulti(v.Error())).To(HaveLen(2))
			Expect(v.Error()).To(MatchError(ContainSubstring("second: second message")))
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

	Describe("Failing checks", func() {
		It("Should keep the Ternary message and path", func() {
			v.Ternary("parent.childField", true, "100% wrong")
			Expect(v.Error()).To(MatchError("parent.child_field: 100% wrong"))
			var pathErr validate.PathError
			Expect(errors.As(v.Error(), &pathErr)).To(BeTrue())
			Expect(pathErr.Path).To(Equal([]string{"parent", "child_field"}))
		})

		It("Should keep the Ternaryf message and path", func() {
			v.Ternaryf("parent.childField", true, "invalid value: %v", 12)
			Expect(v.Error()).To(MatchError("parent.child_field: invalid value: 12"))
			var pathErr validate.PathError
			Expect(errors.As(v.Error(), &pathErr)).To(BeTrue())
			Expect(pathErr.Path).To(Equal([]string{"parent", "child_field"}))
		})

		It("Should mark the Ternary error as a validation error", func() {
			v.Ternary("field", true, "wrong")
			Expect(v.Error()).To(MatchError(validate.ErrValidation))
			Expect(errors.CheapIs(v.Error(), validate.ErrValidation)).To(BeTrue())
		})

		It("Should mark the Ternaryf error as a validation error", func() {
			v.Ternaryf("field", true, "wrong: %d", 1)
			Expect(v.Error()).To(MatchError(validate.ErrValidation))
			Expect(errors.CheapIs(v.Error(), validate.ErrValidation)).To(BeTrue())
		})

		DescribeTable("Should decode a failure with its exact text", func(
			ctx SpecContext,
			err error,
		) {
			decoded := errors.Decode(ctx, errors.Encode(ctx, err, false))
			Expect(decoded).To(MatchError(validate.ErrValidation))
			Expect(decoded.Error()).To(Equal(err.Error()))
		},
			Entry("a Ternary failure", func() error {
				v := validate.New("test")
				v.Ternary("field", true, "100% wrong")
				return v.Error()
			}()),
			Entry("a wrapped validation error", validate.ErrRequired),
		)

		It("Should keep an earlier failure through a passing check", func() {
			v.Ternary("first", true, "failed")
			Expect(v.Ternary("second", false, "passed")).To(BeTrue())
			Expect(v.Ternaryf("third", false, "passed: %d", 1)).To(BeTrue())
			Expect(v.Error()).To(MatchError("first: failed"))
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
					Expect(
						v.Error(),
					).To(MatchError(ContainSubstring("field: must be positive")))
				})
			})

			Describe("Filtering", func() {
				It("Should validate numbers greater than threshold", func() {
					Expect(v.GreaterThan("field", 10, 5)).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
				})

				It("Should catch numbers less than or equal to threshold", func() {
					Expect(v.GreaterThan("field", 5, 5)).To(BeTrue())
					Expect(
						v.Error(),
					).To(MatchError(ContainSubstring("field: must be greater than 5")))
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
					Expect(v.Error()).To(MatchError(
						ContainSubstring("field: must be in bounds [1, 10)"),
					))
				})

				It("Should reject values below lower bound", func() {
					Expect(v.InBounds("field", 0, 1, 10)).To(BeTrue())
					Expect(v.Error()).To(MatchError(
						ContainSubstring("field: must be in bounds [1, 10)"),
					))
				})

				It("Should reject values above upper bound", func() {
					Expect(v.InBounds("field", 11, 1, 10)).To(BeTrue())
					Expect(v.Error()).To(MatchError(
						ContainSubstring("field: must be in bounds [1, 10)"),
					))
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

		Describe("Non-finite values", func() {
			positive := func(value float64) bool { return v.Positive("field", value) }
			inBounds := func(value float64) bool {
				return v.InBounds("field", value, math.Inf(-1), math.Inf(1))
			}
			greaterThan := func(value float64) bool {
				return v.GreaterThan("field", value, math.Inf(-1))
			}
			greaterThanEq := func(value float64) bool {
				return v.GreaterThanEq("field", value, math.Inf(-1))
			}
			lessThanEq := func(value float64) bool {
				return v.LessThanEq("field", value, math.Inf(1))
			}
			DescribeTable("Should reject non-finite values in every numeric comparison",
				func(check func(value float64) bool, value float64) {
					Expect(check(value)).To(BeTrue())
					Expect(v.Error()).To(MatchError(ContainSubstring("field: must")))
				},
				Entry("Positive with NaN", positive, math.NaN()),
				Entry("Positive with +Inf", positive, math.Inf(1)),
				Entry("Positive with -Inf", positive, math.Inf(-1)),
				Entry("InBounds with NaN", inBounds, math.NaN()),
				Entry("InBounds with +Inf", inBounds, math.Inf(1)),
				Entry("InBounds with -Inf", inBounds, math.Inf(-1)),
				Entry("GreaterThan with NaN", greaterThan, math.NaN()),
				Entry("GreaterThan with +Inf", greaterThan, math.Inf(1)),
				Entry("GreaterThan with -Inf", greaterThan, math.Inf(-1)),
				Entry("GreaterThanEq with NaN", greaterThanEq, math.NaN()),
				Entry("GreaterThanEq with +Inf", greaterThanEq, math.Inf(1)),
				Entry("GreaterThanEq with -Inf", greaterThanEq, math.Inf(-1)),
				Entry("LessThanEq with NaN", lessThanEq, math.NaN()),
				Entry("LessThanEq with +Inf", lessThanEq, math.Inf(1)),
				Entry("LessThanEq with -Inf", lessThanEq, math.Inf(-1)),
			)

			It("Should accept the integer extremes", func() {
				Expect(v.GreaterThanEq("field", int64(math.MaxInt64), 0)).To(BeFalse())
				Expect(v.LessThanEq("field", uint64(math.MaxUint64), math.MaxUint64)).
					To(BeFalse())
				Expect(v.Error()).ToNot(HaveOccurred())
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
					Expect(
						v.Error(),
					).To(MatchError(ContainSubstring("field: must be non-empty")))
				})
			})

			Describe("NotEmptyString", func() {
				It("Should validate non-empty strings", func() {
					Expect(v.NotEmptyString("field", "hello")).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
				})

				It("Should catch empty strings", func() {
					Expect(v.NotEmptyString("field", "")).To(BeTrue())
					Expect(
						v.Error(),
					).To(MatchError(ContainSubstring("field: required")))
				})
			})

			Describe("MinLength", func() {
				DescribeTable("Should count Unicode code points", func(value string) {
					Expect(v.MinLength("field", value, 4)).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
				},
					Entry("ASCII letters", "abcd"),
					Entry("accented letters", "éééé"),
					Entry("emoji", "😀😀😀😀"),
				)

				DescribeTable("Should catch short strings", func(value string) {
					Expect(v.MinLength("field", value, 4)).To(BeTrue())
					Expect(v.Error()).To(MatchError(ContainSubstring(
						"field: must be at least 4 characters long",
					)))
				},
					Entry("ASCII letters", "abc"),
					Entry("accented letters", "ééé"),
					Entry("emoji", "😀😀😀"),
				)
			})

			Describe("MaxLength", func() {
				DescribeTable("Should count Unicode code points", func(value string) {
					Expect(v.MaxLength("field", value, 4)).To(BeFalse())
					Expect(v.Error()).NotTo(HaveOccurred())
				},
					Entry("ASCII letters", "abcd"),
					Entry("accented letters", "éééé"),
					Entry("emoji", "😀😀😀😀"),
				)

				DescribeTable("Should catch long strings", func(value string) {
					Expect(v.MaxLength("field", value, 4)).To(BeTrue())
					Expect(v.Error()).To(MatchError(ContainSubstring(
						"field: must be at most 4 characters long",
					)))
				},
					Entry("ASCII letters", "abcde"),
					Entry("accented letters", "ééééé"),
					Entry("emoji", "😀😀😀😀😀"),
					Entry("a family emoji, which is five code points", "👨‍👩‍👧"),
				)
			})
		})

		Describe("NonZero", func() {
			DescribeTable("Should accept a non-zero value", func(check func() bool) {
				Expect(check()).To(BeFalse())
				Expect(v.Error()).NotTo(HaveOccurred())
			},
				Entry("number", func() bool { return v.NonZero("field", 42) }),
				Entry("string", func() bool { return v.NonZero("field", "a") }),
				Entry("UUID", func() bool { return v.NonZero("field", uuid.New()) }),
			)

			DescribeTable("Should catch a zero value", func(check func() bool) {
				Expect(check()).To(BeTrue())
				Expect(
					v.Error(),
				).To(MatchError(ContainSubstring("field: must be non-zero")))
			},
				Entry("number", func() bool { return v.NonZero("field", 0) }),
				Entry("string", func() bool { return v.NonZero("field", "") }),
				Entry("UUID", func() bool { return v.NonZero("field", uuid.Nil()) }),
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
				Expect(
					v.Error(),
				).To(MatchError(ContainSubstring("field: must be non-zero")))
			})
		})
	})
})
