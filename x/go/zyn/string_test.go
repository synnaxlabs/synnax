// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package zyn_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("String", func() {
	Describe("Basic Parsing", func() {
		Specify("string value", func() {
			var dest string
			Expect(zyn.String().Parse("hello", &dest)).To(Succeed())
			Expect(dest).To(Equal("hello"))
		})

		Specify("empty string", func() {
			var dest string
			Expect(zyn.String().Parse("", &dest)).To(Succeed())
			Expect(dest).To(Equal(""))
		})

		Specify("non-string type", func() {
			var dest string
			Expect(zyn.String().Parse(42, &dest)).To(Succeed())
			Expect(dest).To(Equal("42"))
		})
	})

	Describe("Validate", func() {
		It("Should return nil if the value is a valid string", func() {
			Expect(zyn.String().Validate("hello")).To(Succeed())
		})
		It("Should return an error if the value is not a valid string", func() {
			Expect(zyn.String().Validate(struct{}{})).To(HaveOccurred())
		})
	})

	Describe("Invalid Inputs", func() {
		Specify("Non-string destination", func() {
			var dest chan struct{}
			Expect(zyn.String().Parse("hello", &dest)).To(HaveOccurredAs(zyn.ErrInvalidDestinationType))
		})

		Specify("Numeric destination", func() {
			var dest int
			Expect(zyn.String().Parse("hello", &dest)).To(HaveOccurredAs(zyn.ErrInvalidDestinationType))
		})

		Specify("Float destination", func() {
			var dest float64
			Expect(zyn.String().Parse("hello", &dest)).To(HaveOccurredAs(zyn.ErrInvalidDestinationType))
		})

		Specify("Bool destination", func() {
			var dest bool
			Expect(zyn.String().Parse("hello", &dest)).To(HaveOccurredAs(zyn.ErrInvalidDestinationType))
		})

		Specify("Slice destination", func() {
			var dest []string
			Expect(zyn.String().Parse("hello", &dest)).To(HaveOccurredAs(zyn.ErrInvalidDestinationType))
		})

		Specify("Map destination", func() {
			var dest map[string]any
			Expect(zyn.String().Parse("hello", &dest)).To(HaveOccurredAs(zyn.ErrInvalidDestinationType))
		})

		Specify("Struct destination", func() {
			var dest struct{ Name string }
			Expect(zyn.String().Parse("hello", &dest)).To(HaveOccurredAs(zyn.ErrInvalidDestinationType))
		})

		Specify("nil pointer", func() {
			var dest *string
			Expect(zyn.String().Parse("hello", dest)).To(HaveOccurredAs(zyn.ErrInvalidDestinationType))
		})

		Specify("non-pointer destination", func() {
			var dest string
			Expect(zyn.String().Parse("hello", dest)).To(HaveOccurredAs(zyn.ErrInvalidDestinationType))
		})

		Specify("nil interface", func() {
			var dest any
			Expect(zyn.String().Parse("hello", dest)).To(HaveOccurredAs(zyn.ErrInvalidDestinationType))
		})

		Specify("invalid type", func() {
			var dest string
			Expect(zyn.String().Parse(struct{}{}, &dest)).To(MatchError(ContainSubstring("expected string or convertible to string")))
		})
	})

	Describe("Optional Fields", func() {
		Specify("optional field with nil value", func() {
			var dest *string
			Expect(zyn.String().Optional().Parse(nil, &dest)).To(Succeed())
			Expect(dest).To(BeNil())
		})

		Specify("required field with nil value", func() {
			var dest string
			Expect(zyn.String().Parse(nil, &dest)).To(HaveOccurredAs(validate.ErrRequired))
		})

		Specify("optional field with value", func() {
			var dest *string
			Expect(zyn.String().Optional().Parse("hello", &dest)).To(Succeed())
			Expect(*dest).To(Equal("hello"))
		})

		Specify("optional field with custom type", func() {
			type MyString string
			var dest *MyString
			Expect(zyn.String().Optional().Parse("hello", &dest)).To(Succeed())
			Expect(*dest).To(Equal(MyString("hello")))
		})
	})

	Describe("Dump", func() {
		Describe("Basic DataTypes", func() {
			Specify("string value", func() {
				result, err := zyn.String().Dump("hello")
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("hello"))
			})

			Specify("empty string", func() {
				result, err := zyn.String().Dump("")
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(""))
			})

			Specify("numeric value", func() {
				result, err := zyn.String().Dump(42)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("42"))
			})

			Specify("float value", func() {
				result, err := zyn.String().Dump(42.5)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("42.5"))
			})

			Specify("boolean value", func() {
				result, err := zyn.String().Dump(true)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("true"))
			})
		})

		Describe("Custom DataTypes", func() {
			type MyString string
			type MyInt int
			type MyFloat float64
			type MyBool bool

			Specify("custom string type", func() {
				result, err := zyn.String().Dump(MyString("hello"))
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("hello"))
			})

			Specify("custom int type", func() {
				result, err := zyn.String().Dump(MyInt(42))
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("42"))
			})

			Specify("custom float type", func() {
				result, err := zyn.String().Dump(MyFloat(42.5))
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("42.5"))
			})

			Specify("custom bool type", func() {
				result, err := zyn.String().Dump(MyBool(true))
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("true"))
			})
		})

		Describe("Invalid Inputs", func() {
			Specify("nil value", func() {
				_, err := zyn.String().Dump(nil)
				Expect(err).To(HaveOccurredAs(validate.ErrRequired))
			})

			Specify("nil pointer", func() {
				var s *string
				_, err := zyn.String().Dump(s)
				Expect(err).To(HaveOccurredAs(validate.ErrRequired))
			})

			Specify("optional nil value", func() {
				result, err := zyn.String().Optional().Dump(nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeNil())
			})

			Specify("optional nil pointer", func() {
				var s *string
				result, err := zyn.String().Optional().Dump(s)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeNil())
			})

			Specify("complex type", func() {
				type Complex struct{ x int }
				_, err := zyn.String().Dump(Complex{42})
				Expect(err).To(MatchError(ContainSubstring("expected string or convertible to string")))
			})
		})
	})

	Describe("UUID", func() {
		Describe("Parse", func() {
			Specify("valid string UUID", func() {
				var dest string
				Expect(zyn.String().UUID().Parse("123e4567-e89b-12d3-a456-426614174000", &dest)).To(Succeed())
				Expect(dest).To(Equal("123e4567-e89b-12d3-a456-426614174000"))
			})

			Specify("valid UUID type", func() {
				u := uuid.New()
				var dest string
				Expect(zyn.String().UUID().Parse(u, &dest)).To(Succeed())
				Expect(dest).To(Equal(u.String()))
			})

			Specify("invalid string UUID", func() {
				var dest string
				Expect(zyn.String().UUID().Parse("not-a-uuid", &dest)).To(MatchError(ContainSubstring("must be a valid UUID string")))
			})

			Specify("invalid type", func() {
				var dest string
				Expect(zyn.String().UUID().Parse(42, &dest)).To(MatchError(ContainSubstring("expected UUID or string")))
			})

			Specify("nil value", func() {
				var dest string
				Expect(zyn.String().UUID().Parse(nil, &dest)).To(HaveOccurredAs(validate.ErrRequired))
			})

			Specify("optional nil value", func() {
				var dest *string
				Expect(zyn.String().UUID().Optional().Parse(nil, &dest)).To(Succeed())
				Expect(dest).To(BeNil())
			})

			Specify("custom UUID type", func() {
				type MyUUID uuid.UUID
				u := uuid.New()
				var dest string
				Expect(zyn.String().UUID().Parse(MyUUID(u), &dest)).To(MatchError(ContainSubstring("expected UUID or string")))
			})

			Specify("numeric destination", func() {
				var dest int
				Expect(zyn.String().UUID().Parse("123e4567-e89b-12d3-a456-426614174000", &dest)).To(HaveOccurredAs(zyn.ErrInvalidDestinationType))
			})

			Specify("bool destination", func() {
				var dest bool
				Expect(zyn.String().UUID().Parse("123e4567-e89b-12d3-a456-426614174000", &dest)).To(HaveOccurredAs(zyn.ErrInvalidDestinationType))
			})

			Specify("slice destination", func() {
				var dest []string
				Expect(zyn.String().UUID().Parse("123e4567-e89b-12d3-a456-426614174000", &dest)).To(HaveOccurredAs(zyn.ErrInvalidDestinationType))
			})

			Specify("struct destination", func() {
				var dest struct{ ID string }
				Expect(zyn.String().UUID().Parse("123e4567-e89b-12d3-a456-426614174000", &dest)).To(HaveOccurredAs(zyn.ErrInvalidDestinationType))
			})
		})

		Describe("Dump", func() {
			Specify("valid string UUID", func() {
				result, err := zyn.String().UUID().Dump("123e4567-e89b-12d3-a456-426614174000")
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal("123e4567-e89b-12d3-a456-426614174000"))
			})

			Specify("valid UUID type", func() {
				u := uuid.New()
				result, err := zyn.String().UUID().Dump(u)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(u.String()))
			})

			Specify("invalid string UUID", func() {
				_, err := zyn.String().UUID().Dump("not-a-uuid")
				Expect(err).To(MatchError(ContainSubstring("must be a valid UUID string")))
			})

			Specify("invalid type", func() {
				_, err := zyn.String().UUID().Dump(42)
				Expect(err).To(MatchError(ContainSubstring("expected UUID or string")))
			})

			Specify("nil value", func() {
				_, err := zyn.String().UUID().Dump(nil)
				Expect(err).To(HaveOccurredAs(validate.ErrRequired))
			})

			Specify("optional nil value", func() {
				result, err := zyn.String().UUID().Optional().Dump(nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeNil())
			})

			Specify("custom UUID type", func() {
				type MyUUID uuid.UUID
				u := uuid.New()
				_, err := zyn.String().UUID().Dump(MyUUID(u))
				Expect(err).To(MatchError(ContainSubstring("expected UUID or string")))
			})
		})

		Describe("Validate", Focus, func() {
			Specify("valid string UUID", func() {
				Expect(zyn.String().UUID().Validate("123e4567-e89b-12d3-a456-426614174000")).To(Succeed())
			})
			Specify("invalid string UUID", func() {
				Expect(zyn.String().UUID().Validate("not-a-uuid")).To(HaveOccurred())
			})
		})
	})
})
