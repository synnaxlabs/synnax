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
	"github.com/synnaxlabs/x/telem/types/v0"
)

var _ = Describe("Sample codec", func() {
	Describe("Fixed", func() {
		It("Should round-trip a slice of integers", func() {
			data := []int64{1, 2, 3}
			Expect(v0.UnmarshalFixed[int64](v0.MarshalFixed(data))).To(Equal(data))
		})
		It("Should round-trip a slice of floats", func() {
			data := []float64{1.5, -2.25, 3}
			Expect(v0.UnmarshalFixed[float64](v0.MarshalFixed(data))).To(Equal(data))
		})
		It("Should round-trip timestamps", func() {
			data := []v0.TimeStamp{1, 2, 3}
			Expect(v0.UnmarshalFixed[v0.TimeStamp](v0.MarshalFixed(data))).To(Equal(data))
		})
		It("Should encode an empty slice to an empty buffer", func() {
			Expect(v0.MarshalFixed([]int32{})).To(BeEmpty())
		})
	})

	Describe("Variable", func() {
		It("Should round-trip strings, including empty ones", func() {
			data := []string{"hello", "", "world"}
			Expect(v0.UnmarshalVariable[string](v0.MarshalVariable(data))).To(Equal(data))
		})
		It("Should round-trip byte slices", func() {
			data := [][]byte{{1, 2, 3}, {}, {4}}
			Expect(v0.UnmarshalVariable[[]byte](v0.MarshalVariable(data))).To(Equal(data))
		})
		It("Should prefix each sample with a 4-byte length", func() {
			Expect(v0.MarshalVariable([]string{"ab", "cde"})).To(HaveLen(4 + 2 + 4 + 3))
		})
		It("Should stop decoding when a length prefix overruns the buffer", func() {
			Expect(v0.UnmarshalVariable[string]([]byte{0xff, 0x00, 0x00, 0x00, 'a'})).To(BeEmpty())
		})
	})
})
