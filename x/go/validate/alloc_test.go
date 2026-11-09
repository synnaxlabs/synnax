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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Allocation", func() {
	var (
		name  = "name"
		count = 100000
		ptr   = &count
		items = []int{1}
		zero  = testZeroable{value: 1}
	)

	DescribeTable("Should not allocate for a passing check",
		func(check func(v *validate.Validator) bool) {
			v := validate.New("test")
			Expect(testing.AllocsPerRun(100, func() {
				check(v)
			})).To(BeZero())
			Expect(v.Error()).ToNot(HaveOccurred())
		},
		Entry("Ternary", func(v *validate.Validator) bool {
			return v.Ternary("field", name == "", "message")
		}),
		Entry("Ternaryf", func(v *validate.Validator) bool {
			return v.Ternaryf("field", name == "", "message")
		}),
		Entry("NotNil", func(v *validate.Validator) bool {
			return v.NotNil("field", ptr)
		}),
		Entry("Positive", func(v *validate.Validator) bool {
			return v.Positive("field", count)
		}),
		Entry("NonZero", func(v *validate.Validator) bool {
			return v.NonZero("field", count)
		}),
		Entry("NonZeroable", func(v *validate.Validator) bool {
			return v.NonZeroable("field", zero)
		}),
		Entry("NotEmptySlice", func(v *validate.Validator) bool {
			return v.NotEmptySlice("field", items)
		}),
		Entry("NotEmptyString", func(v *validate.Validator) bool {
			return v.NotEmptyString("field", name)
		}),
	)

	It("Should only box the arguments of a passing Ternaryf", func() {
		v := validate.New("test")
		Expect(testing.AllocsPerRun(100, func() {
			v.Ternaryf("field", name == "", "invalid: %v", name)
		})).To(BeNumerically("<=", 1))
		Expect(v.Error()).ToNot(HaveOccurred())
	})
})
