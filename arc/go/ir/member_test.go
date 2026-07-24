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
	Describe("Key", func() {
		It("Should return the node key for a node-backed member", func() {
			Expect(ir.NodeMember("n1").Key()).To(Equal("n1"))
		})

		It("Should return the scope key for a scope-backed member", func() {
			m := ir.ScopeMember(ir.Scope{
				Key:      "sub",
				Mode:     ir.ScopeModeParallel,
				Liveness: ir.LivenessAlways,
			})
			Expect(m.Key()).To(Equal("sub"))
		})

		It("Should return an empty key for a zero member", func() {
			var m ir.Member
			Expect(m.Key()).To(BeEmpty())
		})
	})

	Describe("String", func() {
		It("Should render the node key for a node-backed member", func() {
			Expect(ir.NodeMember("n1").String()).To(Equal("n1\n"))
		})

		It("Should render the nested scope for a scope-backed member", func() {
			m := ir.ScopeMember(ir.Scope{
				Key:      "sub",
				Mode:     ir.ScopeModeParallel,
				Liveness: ir.LivenessAlways,
			})
			Expect(m.String()).To(ContainSubstring("sub"))
		})

		It("Should render a placeholder for a zero member", func() {
			var m ir.Member
			Expect(m.String()).To(Equal("(empty member)\n"))
		})
	})
})
