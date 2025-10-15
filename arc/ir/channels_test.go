// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/set"
)

var _ = Describe("Channels", func() {
	Describe("NewChannels", func() {
		It("Should create empty Channels with initialized maps", func() {
			ch := ir.NewChannels()
			Expect(ch.Read).ToNot(BeNil())
			Expect(ch.Write).ToNot(BeNil())
			Expect(ch.Read).To(HaveLen(0))
			Expect(ch.Write).To(HaveLen(0))
		})
	})

	Describe("OverrideChannels", func() {
		It("Should preserve non-nil maps", func() {
			original := ir.Channels{
				Read:  set.Set[uint32]{1: {}, 2: {}},
				Write: set.Set[uint32]{3: {}},
			}
			result := ir.OverrideChannels(original)
			Expect(result.Read).To(HaveLen(2))
			Expect(result.Write).To(HaveLen(1))
			_, ok1 := result.Read[1]
			_, ok2 := result.Read[2]
			_, ok3 := result.Write[3]
			Expect(ok1).To(BeTrue())
			Expect(ok2).To(BeTrue())
			Expect(ok3).To(BeTrue())
		})

		It("Should create empty maps for nil inputs", func() {
			nilChannels := ir.Channels{}
			result := ir.OverrideChannels(nilChannels)
			Expect(result.Read).ToNot(BeNil())
			Expect(result.Write).ToNot(BeNil())
			Expect(result.Read).To(HaveLen(0))
			Expect(result.Write).To(HaveLen(0))
		})
	})
})