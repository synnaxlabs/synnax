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
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Literal", func() {
	Describe("Basic Parsing", func() {
		Specify("string literal", func() {
			var dest string
			Expect(zyn.Literal("a").Parse("a", &dest)).To(Succeed())
			Expect(dest).To(Equal("a"))
		})
		Specify("int literal", func() {
			var dest int
			Expect(zyn.Literal(1).Parse(1, &dest)).To(Succeed())
			Expect(dest).To(Equal(1))
		})
		Specify("float literal", func() {
			var dest float64
			Expect(zyn.Literal(1.0).Parse(1.0, &dest)).To(Succeed())
			Expect(dest).To(Equal(1.0))
		})
	})
	Describe("DataType Validation", func() {
		Specify("invalid value", func() {
			var dest string
			Expect(zyn.Literal("a").Parse("b", &dest)).
				To(MatchError(ContainSubstring("invalid enum value")))
		})
		Specify("invalid type", func() {
			var dest string
			Expect(zyn.Literal("a").Parse(1, &dest)).
				To(MatchError(ContainSubstring("invalid enum value")))
		})
		Specify("type conversion", func() {
			var dest int
			Expect(zyn.Literal(1).Parse(int64(1), &dest)).To(Succeed())
			Expect(dest).To(Equal(1))
		})
	})
	Describe("Optional Fields", func() {
		Specify("optional field with nil value", func() {
			var dest *string
			Expect(zyn.Literal("a").Optional().Parse(nil, &dest)).To(Succeed())
			Expect(dest).To(BeNil())
		})
		Specify("required field with nil value", func() {
			var dest string
			Expect(zyn.Literal("a").Parse(nil, &dest)).
				To(MatchError(ContainSubstring("required")))
		})
		Specify("optional field with value", func() {
			var dest *string
			Expect(zyn.Literal("a").Optional().Parse("a", &dest)).To(Succeed())
			Expect(*dest).To(Equal("a"))
		})
	})
	Describe("Dump", func() {
		Specify("valid value", func() {
			Expect(zyn.Literal("a").Dump("a")).To(Equal("a"))
		})
		Specify("invalid value", func() {
			Expect(zyn.Literal("a").Dump("b")).Error().
				To(MatchError(ContainSubstring("invalid enum value")))
		})
		Specify("nil value", func() {
			Expect(zyn.Literal("a").Dump(nil)).Error().
				To(MatchError(ContainSubstring("required")))
		})
		Specify("optional nil value", func() {
			Expect(zyn.Literal("a").Optional().Dump(nil)).To(BeNil())
		})
	})
	Describe("Custom DataTypes", func() {
		type MyEnum string
		Specify("custom type literal", func() {
			var dest MyEnum
			Expect(zyn.Literal(MyEnum("a")).Parse(MyEnum("a"), &dest)).To(Succeed())
			Expect(dest).To(Equal(MyEnum("a")))
		})
		Specify("custom type conversion", func() {
			var dest string
			Expect(zyn.Literal(MyEnum("a")).Parse(MyEnum("a"), &dest)).To(Succeed())
			Expect(dest).To(Equal("a"))
		})
	})
})
