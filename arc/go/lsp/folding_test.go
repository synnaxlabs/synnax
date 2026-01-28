// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

var _ = Describe("FoldingRange", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Functions", func() {
		It("Should return folding range for a simple function", func() {
			server, uri := SetupTestServer()
			OpenDocument(server, ctx, uri, "func test() {\n\tx := 1\n}")

			ranges := MustSucceed(server.FoldingRange(ctx, &protocol.FoldingRangeParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				},
			}))

			Expect(ranges).To(HaveLen(1))
			Expect(ranges[0].StartLine).To(Equal(uint32(0)))
			Expect(ranges[0].EndLine).To(Equal(uint32(2)))
			Expect(ranges[0].Kind).To(Equal(protocol.RegionFoldingRange))
		})

		It("Should return folding ranges for multiple functions", func() {
			server, uri := SetupTestServer()
			OpenDocument(server, ctx, uri, "func foo() {\n\tx := 1\n}\n\nfunc bar() {\n\ty := 2\n}")

			ranges := MustSucceed(server.FoldingRange(ctx, &protocol.FoldingRangeParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				},
			}))

			Expect(ranges).To(HaveLen(2))
			Expect(ranges[0].StartLine).To(Equal(uint32(0)))
			Expect(ranges[0].EndLine).To(Equal(uint32(2)))
			Expect(ranges[1].StartLine).To(Equal(uint32(4)))
			Expect(ranges[1].EndLine).To(Equal(uint32(6)))
		})

		It("Should not return folding range for single-line function", func() {
			server, uri := SetupTestServer()
			OpenDocument(server, ctx, uri, "func test() { x := 1 }")

			ranges := MustSucceed(server.FoldingRange(ctx, &protocol.FoldingRangeParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				},
			}))

			Expect(ranges).To(BeEmpty())
		})
	})

	Describe("Sequences", func() {
		It("Should return folding range for a sequence", func() {
			server, uri := SetupTestServer()
			OpenDocument(server, ctx, uri, "sequence main {\n\tstage first {\n\t}\n}")

			ranges := MustSucceed(server.FoldingRange(ctx, &protocol.FoldingRangeParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				},
			}))

			Expect(len(ranges)).To(BeNumerically(">=", 1))
			hasSequenceRange := false
			for _, r := range ranges {
				if r.StartLine == 0 && r.EndLine == 3 {
					hasSequenceRange = true
					break
				}
			}
			Expect(hasSequenceRange).To(BeTrue())
		})

		It("Should return folding ranges for nested stages", func() {
			server, uri := SetupTestServer()
			OpenDocument(server, ctx, uri, "sequence main {\n\tstage first {\n\t}\n\tstage second {\n\t}\n}")

			ranges := MustSucceed(server.FoldingRange(ctx, &protocol.FoldingRangeParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				},
			}))

			Expect(len(ranges)).To(BeNumerically(">=", 3))
		})
	})

	Describe("Edge Cases", func() {
		It("Should return empty ranges for empty document", func() {
			server, uri := SetupTestServer()
			OpenDocument(server, ctx, uri, "")

			ranges := MustSucceed(server.FoldingRange(ctx, &protocol.FoldingRangeParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				},
			}))

			Expect(ranges).To(BeEmpty())
		})

		It("Should return empty ranges for unknown document", func() {
			server, _ := SetupTestServer()

			ranges := MustSucceed(server.FoldingRange(ctx, &protocol.FoldingRangeParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: "file:///unknown.arc"},
				},
			}))

			Expect(ranges).To(BeEmpty())
		})

		It("Should handle block URIs", func() {
			server, _ := SetupTestServer()
			blockURI := protocol.DocumentURI("arc://block/test")
			OpenDocument(server, ctx, blockURI, "if x > 0 {\n\ty := 1\n}")

			ranges := MustSucceed(server.FoldingRange(ctx, &protocol.FoldingRangeParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: blockURI},
				},
			}))

			Expect(ranges).ToNot(BeNil())
		})
	})
})
