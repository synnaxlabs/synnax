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
	v0 "github.com/synnaxlabs/x/telem/types/v0"
)

var _ = Describe("Sample", func() {
	Describe("MarshalFixedSamples", func() {
		It("Should encode samples as contiguous little-endian bytes", func() {
			Expect(v0.MarshalFixedSamples[int32](1, 2)).To(Equal([]byte{1, 0, 0, 0, 2, 0, 0, 0}))
		})
		It("Should return an empty buffer for no samples", func() {
			Expect(v0.MarshalFixedSamples[int64]()).To(BeEmpty())
		})
	})

	Describe("UnmarshalFixedSamples", func() {
		It("Should decode a fixed-density buffer into typed samples", func() {
			Expect(v0.UnmarshalFixedSamples[int32]([]byte{1, 0, 0, 0, 2, 0, 0, 0})).
				To(Equal([]int32{1, 2}))
		})
		It("Should decode an empty buffer to no samples", func() {
			Expect(v0.UnmarshalFixedSamples[int64](nil)).To(BeEmpty())
		})
	})

	Describe("MarshalVariableSamples", func() {
		It("Should length-prefix each sample with its uint32 LE byte length", func() {
			Expect(v0.MarshalVariableSamples("ab", "cde")).
				To(Equal([]byte{2, 0, 0, 0, 'a', 'b', 3, 0, 0, 0, 'c', 'd', 'e'}))
		})
		It("Should encode empty samples as bare length prefixes", func() {
			Expect(v0.MarshalVariableSamples("", "")).To(Equal([]byte{0, 0, 0, 0, 0, 0, 0, 0}))
		})
	})

	Describe("UnmarshalVariableSamples", func() {
		It("Should decode length-prefixed samples", func() {
			Expect(v0.UnmarshalVariableSamples[string](
				[]byte{2, 0, 0, 0, 'a', 'b', 3, 0, 0, 0, 'c', 'd', 'e'},
			)).To(Equal([]string{"ab", "cde"}))
		})
		It("Should stop decoding when a length prefix overruns the buffer", func() {
			Expect(v0.UnmarshalVariableSamples[string]([]byte{0xff, 0, 0, 0, 'a'})).To(BeEmpty())
		})
	})
})
