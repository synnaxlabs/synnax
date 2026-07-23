// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/arc/ir/versions/v1"
)

var _ = Describe("Member", func() {
	Describe("Key", func() {
		It("Should return the node key for a leaf member", func() {
			m := v1.Member{NodeKey: new(string("n1"))}
			Expect(m.Key()).To(Equal("n1"))
		})

		It("Should return the nested scope's key for a scope member", func() {
			m := v1.Member{Scope: new(v1.Scope{Key: "nested"})}
			Expect(m.Key()).To(Equal("nested"))
		})

		It("Should return the empty string for an unset member", func() {
			Expect(v1.Member{}.Key()).To(Equal(""))
		})
	})

	Describe("String", func() {
		It("Should render a leaf member as its node key", func() {
			m := v1.Member{NodeKey: new(string("n1"))}
			Expect(m.String()).To(Equal("n1\n"))
		})

		It("Should render a scope member as the nested scope's tree", func() {
			m := v1.Member{Scope: new(v1.Scope{
				Key:      "nested",
				Mode:     v1.ScopeModeSequential,
				Liveness: v1.LivenessGated,
			})}
			Expect(m.String()).To(HavePrefix("nested ["))
		})

		It("Should render an unset member as a placeholder", func() {
			Expect(v1.Member{}.String()).To(Equal("(empty member)\n"))
		})
	})
})
