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
