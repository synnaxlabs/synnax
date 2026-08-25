// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/arc/ir/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Functions", func() {
	var (
		fn1, fn2, fn3 v1.Function
		fns           v1.Functions
	)

	BeforeEach(func() {
		fn1 = v1.Function{Key: "add"}
		fn2 = v1.Function{Key: "multiply"}
		fn3 = v1.Function{Key: "divide"}
		fns = v1.Functions{fn1, fn2, fn3}
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

	Describe("Empty Collection", func() {
		It("Should handle Find on empty collection", func() {
			empty := v1.Functions{}
			_, found := empty.Find("anything")
			Expect(found).To(BeFalse())
		})
	})
})
