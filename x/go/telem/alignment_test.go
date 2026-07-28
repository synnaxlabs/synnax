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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("NewAlignment", func() {
	It("Should construct the alignment from the given domain and sample indexes", func() {
		align := telem.NewAlignment(2, 1)
		Expect(align.SampleIndex()).To(Equal(uint32(1)))
		Expect(align.DomainIndex()).To(Equal(uint32(2)))
	})
	It("Should construct a zero alignment", func() {
		Expect(uint64(telem.NewAlignment(0, 0))).To(Equal(uint64(0)))
	})
})

var _ = Describe("Alignment", func() {
	Describe("DomainIndex", func() {
		It("Should return the domain index held in the upper 32 bits", func() {
			Expect(telem.NewAlignment(2, 1).DomainIndex()).To(Equal(uint32(2)))
		})
	})

	Describe("SampleIndex", func() {
		It("Should return the sample index held in the lower 32 bits", func() {
			Expect(telem.NewAlignment(2, 1).SampleIndex()).To(Equal(uint32(1)))
		})
	})

	Describe("MarshalJSON", func() {
		It("Should marshal the alignment as a JSON string", func() {
			a := telem.NewAlignment(2, 1)
			marshalled := MustSucceed(a.MarshalJSON())
			Expect(string(marshalled)).To(Equal(fmt.Sprintf(`"%v"`, uint64(a))))
		})
	})

	Describe("UnmarshalJSON", func() {
		It("Should unmarshal the alignment from a JSON string", func() {
			a := telem.NewAlignment(2, 1)
			marshalled := MustSucceed(a.MarshalJSON())
			var unmarshalled telem.Alignment
			Expect(unmarshalled.UnmarshalJSON(marshalled)).To(Succeed())
			Expect(unmarshalled).To(Equal(a))
		})

		It("Should unmarshal the alignment from a number", func() {
			var unmarshalled telem.Alignment
			Expect(unmarshalled.UnmarshalJSON([]byte("123"))).To(Succeed())
			Expect(unmarshalled).To(Equal(telem.Alignment(123)))
		})

		It("Should return an error and leave the alignment untouched on invalid input", func() {
			a := telem.NewAlignment(2, 1)
			Expect(a.UnmarshalJSON([]byte(`"not-a-number"`))).To(
				MatchError(ContainSubstring("invalid syntax")),
			)
			Expect(a).To(Equal(telem.NewAlignment(2, 1)))
		})
	})

	Describe("AddSamples", func() {
		It("Should add to the alignment sample index", func() {
			Expect(telem.NewAlignment(2, 1).AddSamples(3).SampleIndex()).To(Equal(uint32(4)))
		})
	})

	Describe("String", func() {
		It("Should return the string representation of the alignment", func() {
			Expect(telem.NewAlignment(5, 7).String()).To(Equal("5-7"))
		})
	})
})
