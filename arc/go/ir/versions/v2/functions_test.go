// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/arc/ir/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Functions", func() {
	var (
		fn1, fn2, fn3 v2.Function
		fns           v2.Functions
	)

	BeforeEach(func() {
		fn1 = v2.Function{Key: "add"}
		fn2 = v2.Function{Key: "multiply"}
		fn3 = v2.Function{Key: "divide"}
		fns = v2.Functions{fn1, fn2, fn3}
	})

	Describe("Find", func() {
		It("Should find existing function by key", func() {
			fn := MustBeOk(fns.Find("multiply"))
			Expect(fn.Key).To(Equal("multiply"))
		})

		It("Should return false for non-existent key", func() {
			_, found := fns.Find("nonexistent")
			Expect(found).To(BeFalse())
		})
	})

	Describe("Get", func() {
		It("Should get existing function by key", func() {
			fn := fns.Get("add")
			Expect(fn.Key).To(Equal("add"))
		})

		It("Should panic for non-existent key", func() {
			Expect(func() {
				_ = fns.Get("nonexistent")
			}).To(Panic())
		})
	})

	Describe("Empty Collection", func() {
		It("Should handle Find on empty collection", func() {
			empty := v2.Functions{}
			_, found := empty.Find("anything")
			Expect(found).To(BeFalse())
		})

		It("Should panic on Get with empty collection", func() {
			empty := v2.Functions{}
			Expect(func() {
				_ = empty.Get("anything")
			}).To(Panic())
		})
	})
})
