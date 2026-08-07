// Copyright 2026 Synnax Labs, Inc.
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

var _ = Describe("Member", func() {
	Describe("NodeMember", func() {
		It("Should build a member referencing the node key", func() {
			m := ir.NodeMember("n1")
			Expect(m.NodeKey).To(HaveValue(Equal("n1")))
			Expect(m.Scope).To(BeNil())
		})
	})

	Describe("ScopeMember", func() {
		It("Should build a member wrapping the nested scope", func() {
			s := ir.Scope{
				Key:      "sub",
				Mode:     ir.ScopeModeParallel,
				Liveness: ir.LivenessAlways,
			}
			m := ir.ScopeMember(s)
			Expect(m.Scope).To(HaveValue(Equal(s)))
			Expect(m.NodeKey).To(BeNil())
		})
	})
})
