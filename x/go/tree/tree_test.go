// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package tree_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/tree"
)

var _ = Describe("Tree", func() {
	Describe("Prefix", func() {
		It("Should return the terminal branch glyph for the last item", func() {
			Expect(tree.Prefix(true)).To(Equal("└── "))
		})

		It("Should return the continuing branch glyph for a middle item", func() {
			Expect(tree.Prefix(false)).To(Equal("├── "))
		})
	})

	Describe("Indent", func() {
		It("Should return blank indent under the last item", func() {
			Expect(tree.Indent(true)).To(Equal("    "))
		})

		It("Should return a rail indent under a middle item", func() {
			Expect(tree.Indent(false)).To(Equal("│   "))
		})
	})
})
