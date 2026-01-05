// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package version_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/version"
)

var _ = Describe("Counter", func() {
	Describe("Increment", func() {
		It("should increment the counter", func() {
			c := version.Counter(0)
			c = c.Increment()
			Expect(c).To(Equal(version.Counter(1)))
		})
	})

	Describe("NewerThan", func() {
		It("should correctly compare counters", func() {
			c1 := version.Counter(5)
			c2 := version.Counter(3)
			Expect(c1.NewerThan(c2)).To(BeTrue())
			Expect(c2.NewerThan(c1)).To(BeFalse())
		})
	})

	Describe("OlderThan", func() {
		It("should correctly compare counters", func() {
			c1 := version.Counter(3)
			c2 := version.Counter(5)
			Expect(c1.OlderThan(c2)).To(BeTrue())
			Expect(c2.OlderThan(c1)).To(BeFalse())
		})
	})

	Describe("EqualTo", func() {
		It("should correctly compare counters", func() {
			c1 := version.Counter(5)
			c2 := version.Counter(5)
			c3 := version.Counter(3)
			Expect(c1.EqualTo(c2)).To(BeTrue())
			Expect(c1.EqualTo(c3)).To(BeFalse())
		})
	})
})
