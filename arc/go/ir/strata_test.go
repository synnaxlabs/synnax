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
)

var _ = Describe("Strata", func() {
	var strata ir.Strata

	BeforeEach(func() {
		strata = ir.Strata{
			{"node1", "node2"},
			{"node3"},
			{"node4", "node5", "node6"},
		}
	})

	Describe("Get", func() {
		It("Should return correct stratum level for existing nodes", func() {
			Expect(strata.Get("node1")).To(Equal(0))
			Expect(strata.Get("node2")).To(Equal(0))
			Expect(strata.Get("node3")).To(Equal(1))
			Expect(strata.Get("node4")).To(Equal(2))
			Expect(strata.Get("node5")).To(Equal(2))
			Expect(strata.Get("node6")).To(Equal(2))
		})

		It("Should return -1 for non-existent nodes", func() {
			Expect(strata.Get("nonexistent")).To(Equal(-1))
		})
	})

	Describe("Has", func() {
		It("Should return true for existing nodes", func() {
			Expect(strata.Has("node1")).To(BeTrue())
			Expect(strata.Has("node3")).To(BeTrue())
			Expect(strata.Has("node6")).To(BeTrue())
		})

		It("Should return false for non-existent nodes", func() {
			Expect(strata.Has("nonexistent")).To(BeFalse())
		})
	})

	Describe("NodeCount", func() {
		It("Should return total number of nodes across all strata", func() {
			Expect(strata.NodeCount()).To(Equal(6))
		})

		It("Should return 0 for empty strata", func() {
			empty := ir.Strata{}
			Expect(empty.NodeCount()).To(Equal(0))
		})

		It("Should handle single stratum", func() {
			single := ir.Strata{{"node1", "node2", "node3"}}
			Expect(single.NodeCount()).To(Equal(3))
		})
	})
})
