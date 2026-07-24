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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

var _ = Describe("FindCompletion", func() {
	var items []protocol.CompletionItem

	BeforeEach(func() {
		items = []protocol.CompletionItem{
			{Label: "sensor", Detail: protocol.NewOptional("chan f32"), Kind: protocol.CompletionItemKindVariable},
			{Label: "pressure", Detail: protocol.NewOptional("chan f64"), Kind: protocol.CompletionItemKindVariable},
			{Label: "len", Detail: protocol.NewOptional("func"), Kind: protocol.CompletionItemKindFunction},
		}
	})

	It("should find an existing completion item by label", func() {
		item := MustBeOk(testutil.FindCompletion(items, "sensor"))
		Expect(item.Label).To(Equal("sensor"))
		Expect(testutil.ItemDetail(item)).To(Equal("chan f32"))
	})

	It("should return false for a non-existent label", func() {
		_, found := testutil.FindCompletion(items, "nonexistent")
		Expect(found).To(BeFalse())
	})

	It("should find the correct item when multiple items exist", func() {
		item := MustBeOk(testutil.FindCompletion(items, "len"))
		Expect(item.Kind).To(Equal(protocol.CompletionItemKindFunction))
	})

	It("should return false for an empty label", func() {
		_, found := testutil.FindCompletion(items, "")
		Expect(found).To(BeFalse())
	})

	It("should return false when items slice is empty", func() {
		_, found := testutil.FindCompletion([]protocol.CompletionItem{}, "sensor")
		Expect(found).To(BeFalse())
	})

	It("should match the exact label and not a prefix", func() {
		_, found := testutil.FindCompletion(items, "sens")
		Expect(found).To(BeFalse())
	})
})

var _ = Describe("HasCompletion", func() {
	var items []protocol.CompletionItem

	BeforeEach(func() {
		items = []protocol.CompletionItem{
			{Label: "sensor", Detail: protocol.NewOptional("chan f32")},
			{Label: "pressure", Detail: protocol.NewOptional("chan f64")},
			{Label: "now", Detail: protocol.NewOptional("func")},
		}
	})

	It("should return true for an existing label", func() {
		Expect(testutil.HasCompletion(items, "sensor")).To(BeTrue())
	})

	It("should return false for a non-existent label", func() {
		Expect(testutil.HasCompletion(items, "temperature")).To(BeFalse())
	})

	It("should return false for an empty items slice", func() {
		Expect(testutil.HasCompletion([]protocol.CompletionItem{}, "sensor")).To(BeFalse())
	})

	It("should return true for the last item in the slice", func() {
		Expect(testutil.HasCompletion(items, "now")).To(BeTrue())
	})

	It("should not match partial labels", func() {
		Expect(testutil.HasCompletion(items, "press")).To(BeFalse())
	})

	It("should be case-sensitive", func() {
		Expect(testutil.HasCompletion(items, "Sensor")).To(BeFalse())
	})
})

var _ = Describe("HoverContents", func() {
	It("should return the empty string for a nil hover", func() {
		Expect(testutil.HoverContents(nil)).To(Equal(""))
	})

	It("should extract the value from markup content", func() {
		hover := &protocol.Hover{Contents: &protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "**sensor** chan f32",
		}}
		Expect(testutil.HoverContents(hover)).To(Equal("**sensor** chan f32"))
	})

	It("should extract a plain string", func() {
		hover := &protocol.Hover{Contents: protocol.String("sensor chan f32")}
		Expect(testutil.HoverContents(hover)).To(Equal("sensor chan f32"))
	})

	It("should return the empty string when contents are unset", func() {
		Expect(testutil.HoverContents(&protocol.Hover{})).To(Equal(""))
	})
})

var _ = Describe("ItemDetail", func() {
	It("should return the detail when set", func() {
		item := protocol.CompletionItem{Detail: protocol.NewOptional("chan f32")}
		Expect(testutil.ItemDetail(item)).To(Equal("chan f32"))
	})

	It("should return the empty string when unset", func() {
		Expect(testutil.ItemDetail(protocol.CompletionItem{})).To(Equal(""))
	})
})

var _ = Describe("ItemInsertText", func() {
	It("should return the insert text when set", func() {
		item := protocol.CompletionItem{InsertText: protocol.NewOptional("sensor")}
		Expect(testutil.ItemInsertText(item)).To(Equal("sensor"))
	})

	It("should return the empty string when unset", func() {
		Expect(testutil.ItemInsertText(protocol.CompletionItem{})).To(Equal(""))
	})
})

var _ = Describe("ItemTextEdit", func() {
	It("should return a plain text edit", func() {
		edit := &protocol.TextEdit{
			Range:   protocol.Range{End: protocol.Position{Character: 3}},
			NewText: "sensor",
		}
		item := protocol.CompletionItem{TextEdit: edit}
		Expect(testutil.ItemTextEdit(item)).To(BeIdenticalTo(edit))
	})

	It("should return nil when unset", func() {
		Expect(testutil.ItemTextEdit(protocol.CompletionItem{})).To(BeNil())
	})

	It("should return nil for an insert-replace edit", func() {
		item := protocol.CompletionItem{
			TextEdit: &protocol.InsertReplaceEdit{NewText: "sensor"},
		}
		Expect(testutil.ItemTextEdit(item)).To(BeNil())
	})
})

var _ = Describe("DiagnosticCode", func() {
	It("should return a string code", func() {
		d := protocol.Diagnostic{Code: protocol.String("ARC001")}
		Expect(testutil.DiagnosticCode(d)).To(Equal("ARC001"))
	})

	It("should return the empty string when unset", func() {
		Expect(testutil.DiagnosticCode(protocol.Diagnostic{})).To(Equal(""))
	})

	It("should return the empty string for a numeric code", func() {
		d := protocol.Diagnostic{Code: protocol.Integer(42)}
		Expect(testutil.DiagnosticCode(d)).To(Equal(""))
	})
})

var _ = Describe("DiagnosticMessage", func() {
	It("should return a plain string message", func() {
		d := protocol.Diagnostic{Message: protocol.String("undefined symbol")}
		Expect(testutil.DiagnosticMessage(d)).To(Equal("undefined symbol"))
	})

	It("should extract the value from a markup message", func() {
		d := protocol.Diagnostic{Message: &protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "undefined **symbol**",
		}}
		Expect(testutil.DiagnosticMessage(d)).To(Equal("undefined **symbol**"))
	})

	It("should return the empty string when unset", func() {
		Expect(testutil.DiagnosticMessage(protocol.Diagnostic{})).To(Equal(""))
	})
})

// stubServer answers the document and language requests the helpers drive with
// canned results.
type stubServer struct {
	protocol.UnimplementedServer
	lastOpened  *protocol.DidOpenTextDocumentParams
	lastChanged *protocol.DidChangeTextDocumentParams
	hover       *protocol.Hover
	definition  protocol.DefinitionResult
	completion  protocol.CompletionResult
	tokens      *protocol.SemanticTokens
}

var _ protocol.Server = (*stubServer)(nil)

func (s *stubServer) DidOpen(
	_ context.Context,
	params *protocol.DidOpenTextDocumentParams,
) error {
	s.lastOpened = params
	return nil
}

func (s *stubServer) DidChange(
	_ context.Context,
	params *protocol.DidChangeTextDocumentParams,
) error {
	s.lastChanged = params
	return nil
}

func (s *stubServer) Hover(
	context.Context,
	*protocol.HoverParams,
) (*protocol.Hover, error) {
	return s.hover, nil
}

func (s *stubServer) Definition(
	context.Context,
	*protocol.DefinitionParams,
) (protocol.DefinitionResult, error) {
	return s.definition, nil
}

func (s *stubServer) Completion(
	context.Context,
	*protocol.CompletionParams,
) (protocol.CompletionResult, error) {
	return s.completion, nil
}

func (s *stubServer) SemanticTokensFull(
	context.Context,
	*protocol.SemanticTokensParams,
) (*protocol.SemanticTokens, error) {
	return s.tokens, nil
}

var _ = Describe("Server request helpers", func() {
	var (
		server *stubServer
		docURI = uri.URI("file:///test.arc")
	)

	BeforeEach(func() { server = &stubServer{} })

	Describe("OpenDocument", func() {
		It("Should send a DidOpen with the document's identity and content", func(ctx SpecContext) {
			testutil.OpenDocument(server, ctx, docURI, "func main() {}", "arc")
			doc := server.lastOpened.TextDocument
			Expect(doc.URI).To(Equal(docURI))
			Expect(doc.LanguageID).To(Equal(protocol.LanguageKind("arc")))
			Expect(doc.Version).To(Equal(int32(1)))
			Expect(doc.Text).To(Equal("func main() {}"))
		})
	})

	Describe("ChangeDocument", func() {
		It("Should send a whole-document change with the given version", func(ctx SpecContext) {
			testutil.ChangeDocument(server, ctx, docURI, "updated", 7)
			Expect(server.lastChanged.TextDocument.Version).To(Equal(int32(7)))
			Expect(server.lastChanged.ContentChanges).To(HaveLen(1))
			whole, ok := server.lastChanged.ContentChanges[0].(*protocol.TextDocumentContentChangeWholeDocument)
			Expect(ok).To(BeTrue())
			Expect(whole.Text).To(Equal("updated"))
		})
	})

	Describe("Hover", func() {
		It("Should return the server's hover result", func(ctx SpecContext) {
			server.hover = &protocol.Hover{Contents: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "docs",
			}}
			hover := testutil.Hover(server, ctx, docURI, 0, 0)
			Expect(testutil.HoverContents(hover)).To(Equal("docs"))
		})
	})

	Describe("Definition", func() {
		It("Should unwrap a location slice", func(ctx SpecContext) {
			server.definition = protocol.LocationSlice{{URI: docURI}}
			locations := testutil.Definition(server, ctx, docURI, 0, 0)
			Expect(locations).To(HaveLen(1))
			Expect(locations[0].URI).To(Equal(docURI))
		})

		It("Should unwrap a single location", func(ctx SpecContext) {
			server.definition = &protocol.Location{URI: docURI}
			locations := testutil.Definition(server, ctx, docURI, 0, 0)
			Expect(locations).To(HaveLen(1))
			Expect(locations[0].URI).To(Equal(docURI))
		})

		It("Should return nil for a nil result", func(ctx SpecContext) {
			Expect(testutil.Definition(server, ctx, docURI, 0, 0)).To(BeNil())
		})
	})

	Describe("Completion", func() {
		It("Should return a completion list unchanged", func(ctx SpecContext) {
			server.completion = &protocol.CompletionList{
				Items: []protocol.CompletionItem{{Label: "sensor"}},
			}
			list := testutil.Completion(server, ctx, docURI, 0, 0)
			Expect(list.Items).To(HaveLen(1))
		})

		It("Should wrap a bare item slice into a list", func(ctx SpecContext) {
			server.completion = protocol.CompletionItemSlice{{Label: "sensor"}}
			list := testutil.Completion(server, ctx, docURI, 0, 0)
			Expect(list.Items).To(HaveLen(1))
			Expect(list.Items[0].Label).To(Equal("sensor"))
		})

		It("Should return nil for a nil result", func(ctx SpecContext) {
			Expect(testutil.Completion(server, ctx, docURI, 0, 0)).To(BeNil())
		})
	})

	Describe("SemanticTokens", func() {
		It("Should return the server's token data", func(ctx SpecContext) {
			server.tokens = &protocol.SemanticTokens{Data: []uint32{0, 1, 2, 3, 0}}
			tokens := testutil.SemanticTokens(server, ctx, docURI)
			Expect(tokens.Data).To(Equal([]uint32{0, 1, 2, 3, 0}))
		})
	})
})
