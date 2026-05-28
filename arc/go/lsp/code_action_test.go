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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/x/lsp/protocol"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("CodeAction", func() {
	var (
		server *lsp.Server
		uri    protocol.DocumentURI
		client *MockClient
	)

	BeforeEach(func() {
		server, uri, client = SetupTestServerWithClient()
	})

	codeActionParams := func(diags []protocol.Diagnostic) *protocol.CodeActionParams {
		return &protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Range:        protocol.Range{},
			Context:      protocol.CodeActionContext{Diagnostics: diags},
		}
	}

	Describe("Unused import quick fix", func() {
		It("should delete the whole statement when it imports a single module", func(ctx SpecContext) {
			content := "import time\n\nfunc test() i64 { return 0 }\n"
			OpenArcDocument(server, ctx, uri, content)

			diags := client.Diagnostics()
			Expect(diags).To(HaveLen(1))
			Expect(diags[0].Code).To(Equal(string(codes.UnusedImport)))

			actions := MustSucceed(server.CodeAction(ctx, codeActionParams(diags)))
			Expect(actions).To(HaveLen(1))

			action := actions[0]
			Expect(action.Title).To(Equal("Remove unused import"))
			Expect(action.Kind).To(Equal(protocol.QuickFix))
			Expect(action.IsPreferred).To(BeTrue())
			Expect(action.Diagnostics).To(ConsistOf(diags[0]))

			Expect(action.Edit).ToNot(BeNil())
			edits := action.Edit.Changes[uri]
			Expect(edits).To(HaveLen(1))
			Expect(edits[0].NewText).To(BeEmpty())
			Expect(edits[0].Range.Start).To(Equal(protocol.Position{Line: 0, Character: 0}))
			Expect(edits[0].Range.End).To(Equal(protocol.Position{Line: 1, Character: 0}))
		})

		It("should delete just the item from a parenthesized multi-import block", func(ctx SpecContext) {
			content := "import (time control)\n\nfunc test() i64 { return 0 }\n"
			OpenArcDocument(server, ctx, uri, content)

			diags := client.Diagnostics()
			Expect(diags).To(HaveLen(2))
			for _, d := range diags {
				Expect(d.Code).To(Equal(string(codes.UnusedImport)))
			}

			actions := MustSucceed(server.CodeAction(ctx, codeActionParams(diags)))
			Expect(actions).To(HaveLen(2))

			byStart := map[uint32]protocol.TextEdit{}
			for _, a := range actions {
				Expect(a.Edit).ToNot(BeNil())
				edits := a.Edit.Changes[uri]
				Expect(edits).To(HaveLen(1))
				byStart[edits[0].Range.Start.Character] = edits[0]
			}

			timeEdit, ok := byStart[8]
			Expect(ok).To(BeTrue())
			Expect(timeEdit.Range.End).To(Equal(protocol.Position{Line: 0, Character: 13}))

			controlEdit, ok := byStart[12]
			Expect(ok).To(BeTrue())
			Expect(controlEdit.Range.End).To(Equal(protocol.Position{Line: 0, Character: 20}))
		})

		It("should delete a middle item from a 3-item parenthesized block", func(ctx SpecContext) {
			content := "import (time control math)\n\nfunc test() i64 { return 0 }\n"
			OpenArcDocument(server, ctx, uri, content)

			diags := client.Diagnostics()
			Expect(diags).To(HaveLen(3))
			for _, d := range diags {
				Expect(d.Code).To(Equal(string(codes.UnusedImport)))
			}

			actions := MustSucceed(server.CodeAction(ctx, codeActionParams(diags)))
			Expect(actions).To(HaveLen(3))

			byStart := map[uint32]protocol.TextEdit{}
			for _, a := range actions {
				Expect(a.Edit).ToNot(BeNil())
				edits := a.Edit.Changes[uri]
				Expect(edits).To(HaveLen(1))
				byStart[edits[0].Range.Start.Character] = edits[0]
			}

			// First item eats forward: start of "time" -> start of "control".
			timeEdit, ok := byStart[8]
			Expect(ok).To(BeTrue())
			Expect(timeEdit.Range.End).To(Equal(protocol.Position{Line: 0, Character: 13}))

			// Middle item eats forward: start of "control" -> start of "math".
			controlEdit, ok := byStart[13]
			Expect(ok).To(BeTrue())
			Expect(controlEdit.Range.End).To(Equal(protocol.Position{Line: 0, Character: 21}))

			// Last item eats backward: end of "control" -> end of "math".
			mathEdit, ok := byStart[20]
			Expect(ok).To(BeTrue())
			Expect(mathEdit.Range.End).To(Equal(protocol.Position{Line: 0, Character: 25}))
		})

		It("should return no actions when context has no diagnostics", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "import time\n\nfunc test() i64 { return 0 }\n")

			actions := MustSucceed(server.CodeAction(ctx, codeActionParams(nil)))
			Expect(actions).To(BeEmpty())
		})

		It("should ignore diagnostics for unrelated codes", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "func test() i64 { return missing }\n")

			diags := client.Diagnostics()
			Expect(diags).ToNot(BeEmpty())

			actions := MustSucceed(server.CodeAction(ctx, codeActionParams(diags)))
			Expect(actions).To(BeEmpty())
		})

		It("should return nil when the document is not open", func(ctx SpecContext) {
			actions := MustSucceed(server.CodeAction(ctx, &protocol.CodeActionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///missing.arc"},
				Context:      protocol.CodeActionContext{},
			}))
			Expect(actions).To(BeEmpty())
		})
	})

	Describe("Deprecated symbol quick fix", func() {
		filterByCode := func(diags []protocol.Diagnostic, code string) []protocol.Diagnostic {
			out := make([]protocol.Diagnostic, 0, len(diags))
			for _, d := range diags {
				if c, _ := d.Code.(string); c == code {
					out = append(out, d)
				}
			}
			return out
		}

		It("should replace a bare deprecated call and add the missing import", func(ctx SpecContext) {
			content := "func test() i64 {\n    return now()\n}\n"
			OpenArcDocument(server, ctx, uri, content)

			diags := filterByCode(client.Diagnostics(), string(codes.DeprecatedSymbol))
			Expect(diags).ToNot(BeEmpty())

			actions := MustSucceed(server.CodeAction(ctx, codeActionParams(diags)))
			Expect(actions).To(HaveLen(1))

			action := actions[0]
			Expect(action.Title).To(Equal("Replace 'now' with 'time.now'"))
			Expect(action.Kind).To(Equal(protocol.QuickFix))
			Expect(action.IsPreferred).To(BeTrue())
			Expect(action.Edit).ToNot(BeNil())

			edits := action.Edit.Changes[uri]
			Expect(edits).To(HaveLen(2))

			importEdit := edits[0]
			Expect(importEdit.NewText).To(ContainSubstring("import time"))
			Expect(importEdit.Range.Start).To(Equal(protocol.Position{Line: 0, Character: 0}))

			renameEdit := edits[1]
			Expect(renameEdit.NewText).To(Equal("time.now"))
			Expect(renameEdit.Range.Start).To(Equal(protocol.Position{Line: 1, Character: 11}))
			Expect(renameEdit.Range.End).To(Equal(protocol.Position{Line: 1, Character: 14}))
		})

		It("should skip the import edit when the module is already imported", func(ctx SpecContext) {
			content := "import time\n\nfunc test() i64 {\n    return now()\n}\n"
			OpenArcDocument(server, ctx, uri, content)

			diags := filterByCode(client.Diagnostics(), string(codes.DeprecatedSymbol))
			Expect(diags).ToNot(BeEmpty())

			actions := MustSucceed(server.CodeAction(ctx, codeActionParams(diags)))
			Expect(actions).To(HaveLen(1))

			edits := actions[0].Edit.Changes[uri]
			Expect(edits).To(HaveLen(1))
			Expect(edits[0].NewText).To(Equal("time.now"))
		})

		It("should reuse an existing alias when the module is imported under one", func(ctx SpecContext) {
			content := "import time as t\n\nfunc test() i64 {\n    return now()\n}\n"
			OpenArcDocument(server, ctx, uri, content)

			diags := filterByCode(client.Diagnostics(), string(codes.DeprecatedSymbol))
			Expect(diags).ToNot(BeEmpty())

			actions := MustSucceed(server.CodeAction(ctx, codeActionParams(diags)))
			Expect(actions).To(HaveLen(1))
			Expect(actions[0].Title).To(Equal("Replace 'now' with 't.now'"))

			edits := actions[0].Edit.Changes[uri]
			Expect(edits).To(HaveLen(1))
			Expect(edits[0].NewText).To(Equal("t.now"))
		})

		It("should not produce an action when the deprecated identifier is part of a qualified path", func(ctx SpecContext) {
			content := "import time\n\nfunc test() i64 {\n    return time.now()\n}\n"
			OpenArcDocument(server, ctx, uri, content)

			deprecated := filterByCode(client.Diagnostics(), string(codes.DeprecatedSymbol))
			Expect(deprecated).To(BeEmpty())
		})
	})

	Describe("Missing import quick fix", func() {
		findActionByTitle := func(actions []protocol.CodeAction, title string) *protocol.CodeAction {
			for i := range actions {
				if actions[i].Title == title {
					return &actions[i]
				}
			}
			return nil
		}

		It("should add an import when a qualified call references an unimported ambient module", func(ctx SpecContext) {
			content := "sequence main {\n    math.avg{}\n}\n"
			OpenArcDocument(server, ctx, uri, content)

			diags := client.Diagnostics()
			Expect(diags).ToNot(BeEmpty())

			actions := MustSucceed(server.CodeAction(ctx, codeActionParams(diags)))
			action := findActionByTitle(actions, "Add import 'math'")
			Expect(action).ToNot(BeNil())
			Expect(action.Kind).To(Equal(protocol.QuickFix))
			Expect(action.IsPreferred).To(BeTrue())

			edits := action.Edit.Changes[uri]
			Expect(edits).To(HaveLen(1))
			Expect(edits[0].NewText).To(ContainSubstring("import math"))
			Expect(edits[0].Range.Start).To(Equal(protocol.Position{Line: 0, Character: 0}))
		})

		It("should not offer the fix when the module is already imported", func(ctx SpecContext) {
			content := "import math\n\nsequence main {\n    math.avg{}\n}\n"
			OpenArcDocument(server, ctx, uri, content)

			diags := client.Diagnostics()
			actions := MustSucceed(server.CodeAction(ctx, codeActionParams(diags)))
			Expect(findActionByTitle(actions, "Add import 'math'")).To(BeNil())
		})

		It("should not offer the fix for an undefined symbol that is not an ambient module", func(ctx SpecContext) {
			content := "func test() i64 {\n    return notARealModule\n}\n"
			OpenArcDocument(server, ctx, uri, content)

			diags := client.Diagnostics()
			Expect(diags).ToNot(BeEmpty())

			actions := MustSucceed(server.CodeAction(ctx, codeActionParams(diags)))
			for _, a := range actions {
				Expect(a.Title).ToNot(HavePrefix("Add import"))
			}
		})

		It("should return no action when context has no diagnostics", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "sequence main {\n    math.avg{}\n}\n")

			actions := MustSucceed(server.CodeAction(ctx, codeActionParams(nil)))
			Expect(actions).To(BeEmpty())
		})
	})
})
