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
	v0 "github.com/synnaxlabs/x/telem/versions/v0"
)

var _ = Describe("Density", func() {
	Describe("SampleCount", func() {
		It("Should return the number of samples within the number of bytes", func() {
			Expect(v0.Bit64.SampleCount(16)).To(Equal(int64(2)))
		})
		It("Should panic if the density if unknown", func() {
			Expect(func() {
				v0.UnknownDensity.SampleCount(16)
			}).To(PanicWith("attempted to call SampleCount() on undefined density"))
		})
	})

	Describe("Size", func() {
		It("Should return the number of bytes occupied by the given sample count", func() {
			Expect(v0.Bit64.Size(2)).To(Equal(v0.Size(16)))
		})
		It("Should panic if the density if unknown", func() {
			Expect(func() {
				v0.UnknownDensity.Size(16)
			}).To(PanicWith("attempted to call Size() on undefined density"))
		})
	})

})
