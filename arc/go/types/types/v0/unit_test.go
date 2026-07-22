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
	v0 "github.com/synnaxlabs/arc/types/types/v0"
)

var _ = Describe("Unit", func() {
	Describe("Equal", func() {
		It("Should compare equal units", func() {
			u1 := v0.Unit{Dimensions: dimLength, Scale: 1000, Name: "km"}
			u2 := v0.Unit{Dimensions: dimLength, Scale: 1000, Name: "km"}
			Expect(u1.Equal(u2)).To(BeTrue())
		})

		It("Should detect different scales", func() {
			u1 := v0.Unit{Dimensions: dimLength, Scale: 1000, Name: "km"}
			u2 := v0.Unit{Dimensions: dimLength, Scale: 1, Name: "m"}
			Expect(u1.Equal(u2)).To(BeFalse())
		})

		It("Should detect different dimensions", func() {
			u1 := v0.Unit{Dimensions: dimLength, Scale: 1, Name: "m"}
			u2 := v0.Unit{Dimensions: dimTime, Scale: 1, Name: "s"}
			Expect(u1.Equal(u2)).To(BeFalse())
		})
	})
})
