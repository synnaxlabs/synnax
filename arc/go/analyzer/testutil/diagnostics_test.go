// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/testutil"
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

var _ = Describe("Warnings", func() {
	It("Should return only the warnings, in order", func() {
		var d diagnostics.Diagnostics
		d.Add(diagnostics.Warningf(nil, "first"))
		d.Add(diagnostics.Errorf(nil, "error"))
		d.Add(diagnostics.Warningf(nil, "second"))
		Expect(testutil.Warnings(&d)).To(HaveExactElements(
			HaveField("Message", "first"),
			HaveField("Message", "second"),
		))
	})

	It("Should return nothing when no warning was added", func() {
		var d diagnostics.Diagnostics
		d.Add(diagnostics.Errorf(nil, "error"))
		Expect(testutil.Warnings(&d)).To(BeEmpty())
	})
})

var _ = Describe("ApplyFix", func() {
	It("Should make each edit of the fix", func() {
		fix := diagnostics.Fix{Edits: []protocol.TextEdit{
			{
				Range: protocol.Range{
					Start: protocol.Position{Character: 0},
					End:   protocol.Position{Character: 1},
				},
				NewText: "first",
			},
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 0},
					End:   protocol.Position{Line: 1, Character: 1},
				},
				NewText: "second",
			},
		}}
		Expect(testutil.ApplyFix("a\nb\n", fix)).To(Equal("first\nsecond\n"))
	})

	It("Should return the source when the fix has no edit", func() {
		Expect(testutil.ApplyFix("a\n", diagnostics.Fix{})).To(Equal("a\n"))
	})
})
