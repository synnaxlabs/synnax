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
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Union", func() {
	Describe("Basic Parsing", func() {
		Specify("matches first variant (string)", func() {
			var dest any
			Expect(zyn.Union(zyn.String(), zyn.Number()).Parse("hello", &dest)).To(Succeed())
			Expect(dest).To(Equal("hello"))
		})
		Specify("matches second variant (number)", func() {
			var dest any
			Expect(zyn.Union(zyn.String(), zyn.Number()).Parse(42.0, &dest)).To(Succeed())
			Expect(dest).To(Equal(42.0))
		})
		Specify("matches bool variant", func() {
			var dest any
			Expect(zyn.Union(zyn.String(), zyn.Bool()).Parse(true, &dest)).To(Succeed())
			Expect(dest).To(Equal(true))
		})
		Specify("first match wins", func() {
			var dest any
			Expect(zyn.Union(zyn.String(), zyn.String()).Parse("hello", &dest)).To(Succeed())
			Expect(dest).To(Equal("hello"))
		})
		Specify("concrete destination: first variant", func() {
			var dest string
			Expect(zyn.Union(zyn.String(), zyn.Number()).Parse("hello", &dest)).To(Succeed())
			Expect(dest).To(Equal("hello"))
		})
		Specify("concrete destination: second variant", func() {
			var dest float64
			Expect(zyn.Union(zyn.String(), zyn.Number()).Parse(42.0, &dest)).To(Succeed())
			Expect(dest).To(Equal(42.0))
		})
		Specify("concrete destination: no match", func() {
			var dest bool
			Expect(zyn.Union(zyn.String(), zyn.Number()).Parse("hello", &dest)).
				To(MatchError(ContainSubstring("did not match any of 2 union variants")))
		})
	})
	Describe("Validate", func() {
		It("Should succeed when data matches a variant", func() {
			Expect(zyn.Union(zyn.String(), zyn.Number()).Validate("hello")).To(Succeed())
		})
		It("Should fail when data matches no variant", func() {
			Expect(zyn.Union(zyn.Uint32(), zyn.Bool()).Validate([]any{1, 2})).
				To(MatchError(ContainSubstring("did not match any of 2 union variants")))
		})
		It("Should succeed when optional union receives nil", func() {
			Expect(zyn.Union(zyn.String(), zyn.Number()).Optional().Validate(nil)).To(Succeed())
		})
		It("Should fail when required union receives nil", func() {
			Expect(zyn.Union(zyn.String(), zyn.Number()).Validate(nil)).
				To(MatchError(validate.ErrRequired))
		})
	})
	Describe("No Match", func() {
		Specify("returns error when no variant matches", func() {
			var dest any
			Expect(zyn.Union(zyn.Uint32(), zyn.Bool()).Parse([]any{1, 2}, &dest)).
				To(MatchError(ContainSubstring("did not match any of 2 union variants")))
		})
	})
	Describe("Invalid Inputs", func() {
		Specify("non-pointer destination", func() {
			var dest string
			Expect(zyn.Union(zyn.String(), zyn.Number()).Parse("hello", dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
		Specify("nil pointer destination", func() {
			var dest *string
			Expect(zyn.Union(zyn.String(), zyn.Number()).Parse("hello", dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
	})
	Describe("Optional Fields", func() {
		Specify("optional with nil value", func() {
			var dest any
			Expect(zyn.Union(zyn.String(), zyn.Number()).Optional().Parse(nil, &dest)).To(Succeed())
		})
		Specify("required with nil value", func() {
			var dest any
			Expect(zyn.Union(zyn.String(), zyn.Number()).Parse(nil, &dest)).
				To(MatchError(validate.ErrRequired))
		})
	})
	Describe("Dump", func() {
		Specify("matches first variant", func() {
			Expect(zyn.Union(zyn.String(), zyn.Number()).Dump("hello")).
				To(Equal("hello"))
		})
		Specify("matches second variant", func() {
			Expect(zyn.Union(zyn.Enum("a", "b"), zyn.Number()).Dump(42)).
				To(Equal(int64(42)))
		})
		Specify("no match", func() {
			Expect(zyn.Union(zyn.Uint32(), zyn.Bool()).Dump([]string{"a"})).Error().
				To(MatchError(ContainSubstring("did not match any of 2 union variants")))
		})
		Specify("nil required", func() {
			Expect(zyn.Union(zyn.String(), zyn.Number()).Dump(nil)).Error().
				To(MatchError(validate.ErrRequired))
		})
		Specify("nil pointer", func() {
			var s *string
			Expect(zyn.Union(zyn.String(), zyn.Number()).Dump(s)).Error().
				To(MatchError(validate.ErrRequired))
		})
		Specify("optional nil value", func() {
			Expect(zyn.Union(zyn.String(), zyn.Number()).Optional().Dump(nil)).
				To(BeNil())
		})
		Specify("optional nil pointer", func() {
			var s *string
			Expect(zyn.Union(zyn.String(), zyn.Number()).Optional().Dump(s)).
				To(BeNil())
		})
		Specify("non-nil pointer is dereferenced", func() {
			s := "hello"
			Expect(zyn.Union(zyn.String(), zyn.Number()).Dump(&s)).To(Equal("hello"))
		})
	})
	Describe("Shape", func() {
		It("Should return a UnionShape", func() {
			shape := zyn.Union(zyn.String(), zyn.Number()).Shape()
			Expect(shape.DataType()).To(Equal(zyn.UnionT))
			Expect(shape.Optional()).To(BeFalse())
			Expect(shape.Fields()).To(BeNil())
			unionShape, ok := shape.(zyn.UnionShape)
			Expect(ok).To(BeTrue())
			Expect(unionShape.Variants()).To(HaveLen(2))
			Expect(unionShape.Variants()[0].DataType()).To(Equal(zyn.StringT))
			Expect(unionShape.Variants()[1].DataType()).To(Equal(zyn.NumberT))
		})
		It("Should reflect optional", func() {
			Expect(zyn.Union(zyn.String(), zyn.Number()).Optional().Shape().Optional()).To(BeTrue())
		})
	})
	Describe("Constructor", func() {
		It("Should panic with fewer than 2 schemas", func() {
			Expect(func() { zyn.Union(zyn.String()) }).To(Panic())
		})
		It("Should panic with zero schemas", func() {
			Expect(func() { zyn.Union() }).To(Panic())
		})
	})
})
