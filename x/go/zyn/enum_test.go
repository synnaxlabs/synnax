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
			Expect(
				zyn.Enum("a", "b", "c").Validate("d"),
			).To(MatchError(ContainSubstring("invalid enum value d")))
		})
		It("Should fail when a required enum receives nil", func() {
			Expect(
				zyn.Enum("a", "b", "c").Validate(nil),
			).To(MatchError(validate.ErrRequired))
		})
	})
	Describe("DataType Validation", func() {
		Specify("invalid value", func() {
			var dest string
			Expect(zyn.Enum("a", "b", "c").Parse("d", &dest)).
				To(MatchError(ContainSubstring("invalid enum value")))
		})
		Specify("invalid type", func() {
			var dest string
			Expect(zyn.Enum("a", "b", "c").Parse(1, &dest)).
				To(MatchError(ContainSubstring("invalid enum value")))
		})
		Specify("type conversion", func() {
			var dest int
			Expect(zyn.Enum(1, 2, 3).Parse(int64(1), &dest)).To(Succeed())
			Expect(dest).To(Equal(1))
		})
		Specify("incompatible channel destination", func() {
			var dest chan string
			Expect(zyn.Enum("a", "b", "c").Parse("a", &dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
		Specify("incompatible slice destination", func() {
			var dest []string
			Expect(zyn.Enum("a", "b", "c").Parse("a", &dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
		Specify("incompatible map destination", func() {
			var dest map[string]any
			Expect(zyn.Enum("a", "b", "c").Parse("a", &dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
		Specify("incompatible struct destination", func() {
			var dest struct{ Value string }
			Expect(zyn.Enum("a", "b", "c").Parse("a", &dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
	})
	Describe("Required Fields", func() {
		Specify("required field with nil value", func() {
			var dest string
			Expect(zyn.Enum("a", "b", "c").Parse(nil, &dest)).
				To(MatchError(ContainSubstring("required")))
		})
	})
	Describe("Dump", func() {
		Specify("valid value", func() {
			Expect(zyn.Enum("a", "b", "c").Dump("a")).To(Equal("a"))
		})
		Specify("invalid value", func() {
			Expect(zyn.Enum("a", "b", "c").Dump("d")).Error().
				To(MatchError(ContainSubstring("invalid enum value")))
		})
		Specify("nil value", func() {
			Expect(zyn.Enum("a", "b", "c").Dump(nil)).Error().
				To(MatchError(ContainSubstring("required")))
		})
	})
	Describe("Enum", func() {
		It("should panic on empty", func() {
			Expect(func() { zyn.Enum[any]() }).To(Panic())
		})
	})
	Describe("Custom DataTypes", func() {
		type MyEnum string
		Specify("custom type enum", func() {
			var dest MyEnum
			Expect(zyn.Enum(MyEnum("a"), MyEnum("b")).Parse(MyEnum("a"), &dest)).
				To(Succeed())
			Expect(dest).To(Equal(MyEnum("a")))
		})
		Specify("custom type conversion", func() {
			var dest string
			Expect(zyn.Enum(MyEnum("a"), MyEnum("b")).Parse(MyEnum("a"), &dest)).
				To(Succeed())
			Expect(dest).To(Equal("a"))
		})
	})
})
