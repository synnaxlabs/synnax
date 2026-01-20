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
	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/arc/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

var _ = Describe("Formatting", func() {
	var (
		ctx    context.Context
		server *lsp.Server
		uri    protocol.DocumentURI
	)

	BeforeEach(func() {
		ctx = context.Background()
		server, uri = testutil.SetupTestServer()
	})

	Describe("Full Document Formatting", func() {
		It("should format a simple function", func() {
			content := "func add(x i32,y i32)i32{return x+y}"
			testutil.OpenDocument(server, ctx, uri, content)

			edits := MustSucceed(server.Formatting(ctx, &protocol.DocumentFormattingParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Options:      protocol.FormattingOptions{},
			}))

			Expect(edits).ToNot(BeNil())
			Expect(len(edits)).To(Equal(1))
			Expect(edits[0].NewText).To(ContainSubstring("func add(x i32, y i32) i32"))
			Expect(edits[0].NewText).To(ContainSubstring("return x + y"))
		})

		It("should return nil for already-formatted code", func() {
			content := "func add(x i32, y i32) i32 {\n    return x + y\n}\n"
			testutil.OpenDocument(server, ctx, uri, content)

			edits := MustSucceed(server.Formatting(ctx, &protocol.DocumentFormattingParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Options:      protocol.FormattingOptions{},
			}))

			Expect(edits).To(BeNil())
		})

		It("should return nil for closed document", func() {
			edits := MustSucceed(server.Formatting(ctx, &protocol.DocumentFormattingParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.arc"},
				Options:      protocol.FormattingOptions{},
			}))

			Expect(edits).To(BeNil())
		})

		It("should format binary operators with spaces", func() {
			content := "x:=a+b*c"
			testutil.OpenDocument(server, ctx, uri, content)

			edits := MustSucceed(server.Formatting(ctx, &protocol.DocumentFormattingParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Options:      protocol.FormattingOptions{},
			}))

			Expect(edits).ToNot(BeNil())
			Expect(edits[0].NewText).To(ContainSubstring("x := a + b * c"))
		})

		It("should respect tab size option", func() {
			content := "func test(){x:=1}"
			testutil.OpenDocument(server, ctx, uri, content)

			edits := MustSucceed(server.Formatting(ctx, &protocol.DocumentFormattingParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Options: protocol.FormattingOptions{
					TabSize: 2,
				},
			}))

			Expect(edits).ToNot(BeNil())
			Expect(edits[0].NewText).To(ContainSubstring("  x := 1"))
		})

		It("should preserve unit literals without space", func() {
			content := "delay:=100ms"
			testutil.OpenDocument(server, ctx, uri, content)

			edits := MustSucceed(server.Formatting(ctx, &protocol.DocumentFormattingParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Options:      protocol.FormattingOptions{},
			}))

			Expect(edits).ToNot(BeNil())
			Expect(edits[0].NewText).To(ContainSubstring("100ms"))
			Expect(edits[0].NewText).ToNot(ContainSubstring("100 ms"))
		})
	})

	Describe("Range Formatting", func() {
		It("should format a specific range", func() {
			content := "x:=1\ny:=2\nz:=3"
			testutil.OpenDocument(server, ctx, uri, content)

			edits := MustSucceed(server.RangeFormatting(ctx, &protocol.DocumentRangeFormattingParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 4},
				},
				Options: protocol.FormattingOptions{},
			}))

			Expect(edits).ToNot(BeNil())
		})

		It("should return nil for closed document", func() {
			edits := MustSucceed(server.RangeFormatting(ctx, &protocol.DocumentRangeFormattingParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.arc"},
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 10},
				},
				Options: protocol.FormattingOptions{},
			}))

			Expect(edits).To(BeNil())
		})
	})
})
