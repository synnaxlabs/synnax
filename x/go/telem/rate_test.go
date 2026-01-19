// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Rate", func() {
	Describe("Rate", func() {
		Describe("Period", func() {
			It("Should return the correct period for the data rate", func() {
				Expect(telem.Rate(1).Period()).To(Equal(telem.Second))
			})
		})
		Describe("SampleCount", func() {
			It("Should return the number of samples that fit in the span", func() {
				Expect(telem.Rate(10).SampleCount(telem.Second)).To(Equal(10))
			})
		})
	})
})
