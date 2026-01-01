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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Bool", func() {
	Describe("Basic Parsing", func() {
		Specify("boolean value", func() {
			var dest bool
			Expect(zyn.Bool().Parse(true, &dest)).To(Succeed())
			Expect(dest).To(BeTrue())
		})

		Specify("string 'true'", func() {
			var dest bool
			Expect(zyn.Bool().Parse("true", &dest)).To(Succeed())
			Expect(dest).To(BeTrue())
		})

		Specify("string '1'", func() {
			var dest bool
			Expect(zyn.Bool().Parse("1", &dest)).To(Succeed())
			Expect(dest).To(BeTrue())
		})

		Specify("string 'false'", func() {
			var dest bool
			Expect(zyn.Bool().Parse("false", &dest)).To(Succeed())
			Expect(dest).To(BeFalse())
		})

		Specify("string '0'", func() {
			var dest bool
			Expect(zyn.Bool().Parse("0", &dest)).To(Succeed())
			Expect(dest).To(BeFalse())
		})

		Specify("integer 1", func() {
			var dest bool
			Expect(zyn.Bool().Parse(1, &dest)).To(Succeed())
			Expect(dest).To(BeTrue())
		})

		Specify("integer 0", func() {
			var dest bool
			Expect(zyn.Bool().Parse(0, &dest)).To(Succeed())
			Expect(dest).To(BeFalse())
		})

		Specify("float 1.0", func() {
			var dest bool
			Expect(zyn.Bool().Parse(1.0, &dest)).To(Succeed())
			Expect(dest).To(BeTrue())
		})

		Specify("float 0.0", func() {
			var dest bool
			Expect(zyn.Bool().Parse(0.0, &dest)).To(Succeed())
			Expect(dest).To(BeFalse())
		})
	})

	Describe("Validate", func() {
		It("Should return nil if the value is a valid boolean", func() {
			Expect(zyn.Bool().Validate(true)).To(Succeed())
		})
		It("Should return nil if the value is not a valid boolean", func() {
			Expect(zyn.Bool().Validate(struct{}{})).To(HaveOccurred())
		})
	})

	Describe("Invalid Inputs", func() {
		Specify("invalid string", func() {
			var dest bool
			Expect(zyn.Bool().Parse("invalid", &dest)).To(MatchError(ContainSubstring("invalid boolean string 'invalid': must be 'true', 'false', '1', or '0'")))
		})

		Specify("nil pointer", func() {
			var dest *bool
			Expect(zyn.Bool().Parse(true, dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})

		Specify("non-pointer destination", func() {
			var dest bool
			Expect(zyn.Bool().Parse(true, dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})

		Specify("nil interface", func() {
			var dest any
			Expect(zyn.Bool().Parse(true, dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})

		Specify("invalid type", func() {
			var dest bool
			Expect(zyn.Bool().Parse(struct{}{}, &dest)).To(MatchError(ContainSubstring("expected boolean, string, number, or nil")))
		})

		Specify("string destination", func() {
			var dest string
			Expect(zyn.Bool().Parse(true, &dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})

		Specify("numeric destination", func() {
			var dest int
			Expect(zyn.Bool().Parse(true, &dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})

		Specify("float destination", func() {
			var dest float64
			Expect(zyn.Bool().Parse(true, &dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})

		Specify("channel destination", func() {
			var dest chan bool
			Expect(zyn.Bool().Parse(true, &dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})

		Specify("slice destination", func() {
			var dest []bool
			Expect(zyn.Bool().Parse(true, &dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})

		Specify("map destination", func() {
			var dest map[string]bool
			Expect(zyn.Bool().Parse(true, &dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})

		Specify("struct destination", func() {
			var dest struct{ Flag bool }
			Expect(zyn.Bool().Parse(true, &dest)).To(HaveOccurredAs(zyn.InvalidDestinationTypeError))
		})
	})

	Describe("Optional Fields", func() {
		Specify("optional field with nil value", func() {
			var dest *bool
			Expect(zyn.Bool().Optional().Parse(nil, &dest)).To(Succeed())
			Expect(dest).To(BeNil())
		})

		Specify("required field with nil value", func() {
			var dest bool
			Expect(zyn.Bool().Parse(nil, &dest)).To(HaveOccurredAs(validate.RequiredError))
		})

		Specify("optional field with value", func() {
			var dest *bool
			Expect(zyn.Bool().Optional().Parse(true, &dest)).To(Succeed())
			Expect(*dest).To(BeTrue())
		})

		Specify("optional field with custom type", func() {
			type MyBool bool
			var dest *MyBool
			Expect(zyn.Bool().Optional().Parse(true, &dest)).To(Succeed())
			Expect(*dest).To(Equal(MyBool(true)))
		})
	})

	Describe("Dump", func() {
		Specify("boolean value", func() {
			result, err := zyn.Bool().Dump(true)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(true))
		})

		Specify("string 'true'", func() {
			result, err := zyn.Bool().Dump("true")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(true))
		})

		Specify("string '1'", func() {
			result, err := zyn.Bool().Dump("1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(true))
		})

		Specify("string 'false'", func() {
			result, err := zyn.Bool().Dump("false")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(false))
		})

		Specify("string '0'", func() {
			result, err := zyn.Bool().Dump("0")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(false))
		})

		Specify("integer 1", func() {
			result, err := zyn.Bool().Dump(1)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(true))
		})

		Specify("integer 0", func() {
			result, err := zyn.Bool().Dump(0)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(false))
		})

		Specify("float 1.0", func() {
			result, err := zyn.Bool().Dump(1.0)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(true))
		})

		Specify("float 0.0", func() {
			result, err := zyn.Bool().Dump(0.0)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(false))
		})

		Specify("invalid string", func() {
			_, err := zyn.Bool().Dump("invalid")
			Expect(err).To(MatchError(ContainSubstring("invalid boolean string 'invalid': must be 'true', 'false', '1', or '0'")))
		})

		Specify("nil value", func() {
			_, err := zyn.Bool().Dump(nil)
			Expect(err).To(HaveOccurredAs(validate.RequiredError))
		})

		Specify("nil pointer", func() {
			var b *bool
			_, err := zyn.Bool().Dump(b)
			Expect(err).To(HaveOccurredAs(validate.RequiredError))
		})

		Specify("invalid type", func() {
			_, err := zyn.Bool().Dump(struct{}{})
			Expect(err).To(MatchError(ContainSubstring("expected boolean, string, number, or nil")))
		})

		Specify("optional nil value", func() {
			result, err := zyn.Bool().Optional().Dump(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})

		Specify("optional nil pointer", func() {
			var b *bool
			result, err := zyn.Bool().Optional().Dump(b)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})

		Specify("custom type", func() {
			type MyBool bool
			result, err := zyn.Bool().Dump(MyBool(true))
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(true))
		})
	})
})
