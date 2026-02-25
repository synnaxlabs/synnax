// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/testutil"
)

var _ = Describe("MockHostRuntime", func() {
	var rt *testutil.MockHostRuntime

	BeforeEach(func() {
		rt = testutil.NewMockHostRuntime()
	})

	Describe("Export", func() {
		It("Should register a function and allow retrieval via Get", func() {
			fn := func(x int) int { return x * 2 }
			Expect(rt.Export("math", "double", fn)).To(Succeed())
			retrieved := testutil.Get[func(int) int](rt, "math", "double")
			Expect(retrieved(5)).To(Equal(10))
		})

		It("Should return an error on duplicate registration", func() {
			Expect(rt.Export("math", "add", func() {})).To(Succeed())
			Expect(rt.Export("math", "add", func() {})).To(MatchError(
				ContainSubstring("duplicate export: math.add"),
			))
		})

		It("Should allow the same name in different modules", func() {
			Expect(rt.Export("math", "pow", func() int { return 1 })).To(Succeed())
			Expect(rt.Export("string", "pow", func() string { return "s" })).To(Succeed())
			mathPow := testutil.Get[func() int](rt, "math", "pow")
			stringPow := testutil.Get[func() string](rt, "string", "pow")
			Expect(mathPow()).To(Equal(1))
			Expect(stringPow()).To(Equal("s"))
		})
	})

	Describe("Get", func() {
		It("Should panic when the module does not exist", func() {
			Expect(func() {
				testutil.Get[func()](rt, "nonexistent", "fn")
			}).To(PanicWith(ContainSubstring("module \"nonexistent\" not found")))
		})

		It("Should panic when the function does not exist in the module", func() {
			Expect(rt.Export("math", "add", func() {})).To(Succeed())
			Expect(func() {
				testutil.Get[func()](rt, "math", "missing")
			}).To(PanicWith(ContainSubstring("function math.missing not found")))
		})

		It("Should panic when the type assertion fails", func() {
			Expect(rt.Export("math", "add", func(a, b int) int { return a + b })).To(Succeed())
			Expect(func() {
				testutil.Get[func(string) string](rt, "math", "add")
			}).To(PanicWith(ContainSubstring("math.add has type")))
		})
	})
})
