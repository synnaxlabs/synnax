// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v0"
)

var _ = Describe("Variant", func() {
	Describe("IsValid", func() {
		DescribeTable("Should report whether the variant is defined",
			func(v v0.Variant, valid bool) {
				Expect(v.IsValid()).To(Equal(valid))
			},
			Entry("success", v0.VariantSuccess, true),
			Entry("info", v0.VariantInfo, true),
			Entry("warning", v0.VariantWarning, true),
			Entry("error", v0.VariantError, true),
			Entry("loading", v0.VariantLoading, true),
			Entry("disabled", v0.VariantDisabled, true),
			Entry("unknown", v0.Variant("bogus"), false),
			Entry("empty", v0.Variant(""), false),
		)
	})
})

var _ = Describe("Status", func() {
	Describe("GorpKey", func() {
		It("Should return the status's key", func() {
			Expect(v0.Status[any]{Key: "s1"}.GorpKey()).To(Equal("s1"))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v0.Status[any]{}.SetOptions()).To(BeNil())
		})
	})
})
