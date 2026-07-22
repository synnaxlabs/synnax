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

var _ = Describe("Channels", func() {
	Describe("Copy", func() {
		It("Should deep copy the read and write maps", func() {
			original := v0.Channels{
				Read:  map[uint32]string{1: "sensor"},
				Write: map[uint32]string{2: "actuator"},
			}
			copied := original.Copy()
			copied.Read[1] = "changed"
			copied.Write[2] = "changed"
			Expect(original.Read[1]).To(Equal("sensor"))
			Expect(original.Write[2]).To(Equal("actuator"))
		})
		It("Should replace nil maps with empty maps", func() {
			copied := v0.Channels{}.Copy()
			Expect(copied.Read).ToNot(BeNil())
			Expect(copied.Read).To(BeEmpty())
			Expect(copied.Write).ToNot(BeNil())
			Expect(copied.Write).To(BeEmpty())
		})
	})
})
