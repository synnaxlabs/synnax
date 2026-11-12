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
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	xlsp "github.com/synnaxlabs/x/lsp"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func asCodeActions(elems []protocol.CommandOrCodeAction) []protocol.CodeAction {
	GinkgoHelper()
	out := make([]protocol.CodeAction, len(elems))
	for i, e := range elems {
		a, ok := e.(*protocol.CodeAction)
		Expect(ok).To(BeTrue(), "expected a CodeAction, got %T", e)
		out[i] = *a
	}
	return out
}

var _ = Describe("CodeAction", func() {
	var (
		server *lsp.Server
		uri    uri.URI
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

	Describe("Hoist declaration quick fix", func() {
		// applyEdits applies edits from the last position backwards, so each
		// earlier offset stays valid.
		applyEdits := func(content string, edits []protocol.TextEdit) string {
			sorted := slices.Clone(edits)
			slices.SortFunc(sorted, func(a, b protocol.TextEdit) int {
				if a.Range.Start.Line != b.Range.Start.Line {
					return int(b.Range.Start.Line) - int(a.Range.Start.Line)
				}
				return int(b.Range.Start.Character) - int(a.Range.Start.Character)
			})
			for _, e := range sorted {
				content = xlsp.ApplyIncrementalChange(
					content,
					&protocol.TextDocumentContentChangePartial{
						Range: e.Range,
						Text:  e.NewText,
					},
				)
			}
			return content
		}

		It("should move every late declaration with one action", func(ctx SpecContext) {
			content := "import time\n\n" +
				"sequence main {\n" +
				"    base := 1\n" +
				"    stage a {\n" +
				"        true => next\n" +
				"    }\n" +
				"    t0 := time.now()\n" +
				"    stage b {\n" +
				"        time.wait(1s)\n" +
				"        n := 2\n" +
				"        true => next\n" +
				"    }\n" +
				"    count $= 0\n" +
				"    stage c {\n" +
				"        time.wait(1s)\n" +
				"    }\n" +
				"}\n\n" +
				"true => main\n"
			OpenArcDocument(server, ctx, uri, content)
			diags := client.Diagnostics()
			Expect(diags).To(HaveLen(3))
			for _, d := range diags {
				Expect(
					DiagnosticCode(d),
				).To(Equal(string(codes.DeclarationAfterStep.Code)))
			}
			actions := asCodeActions(
				MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
			)
			Expect(actions).To(HaveLen(1))
			action := actions[0]
			Expect(action.Title).To(Equal("Move declarations to the top"))
			Expect(action.Kind).To(HaveValue(Equal(protocol.CodeActionKindQuickFix)))
			Expect(action.Diagnostics).To(ConsistOf(diags))
			Expect(applyEdits(content, action.Edit.Changes[uri])).To(Equal(
				"import time\n\n" +
					"sequence main {\n" +
					"    base := 1\n" +
					"    t0 := time.now()\n" +
					"    count $= 0\n" +
					"    stage a {\n" +
					"        true => next\n" +
					"    }\n" +
					"    stage b {\n" +
					"        n := 2\n" +
					"        time.wait(1s)\n" +
					"        true => next\n" +
					"    }\n" +
					"    stage c {\n" +
					"        time.wait(1s)\n" +
					"    }\n" +
					"}\n\n" +
					"true => main\n",
			))
		})

		It("should move a trailing block comment too", func(ctx SpecContext) {
			content := "import time\n\n" +
				"sequence main {\n" +
				"    stage a {\n" +
				"        true => next\n" +
				"    }\n" +
				"    t0 := time.now() /* when\n" +
				"       the valve\n" +
				"       opened */\n" +
				"    stage b {\n" +
				"        time.wait(1s)\n" +
				"    }\n" +
				"}\n\n" +
				"true => main\n"
			OpenArcDocument(server, ctx, uri, content)
			diags := client.Diagnostics()
			Expect(diags).To(HaveLen(1))
			actions := asCodeActions(
				MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
			)
			Expect(actions).To(HaveLen(1))
			Expect(applyEdits(content, actions[0].Edit.Changes[uri])).To(Equal(
				"import time\n\n" +
					"sequence main {\n" +
					"    t0 := time.now() /* when\n" +
					"       the valve\n" +
					"       opened */\n" +
					"    stage a {\n" +
					"        true => next\n" +
					"    }\n" +
					"    stage b {\n" +
					"        time.wait(1s)\n" +
					"    }\n" +
					"}\n\n" +
					"true => main\n",
			))
		})
	})

	Describe("Numeric condition quick fix", func() {
		const (
			transition = "sequence main {\n    stage a { %s => next }\n    stage b {}\n}\n"
			ifStmt     = "func f(x i64) i64 {\n    if %s { return 1 }\n    return 0\n}\n"
			elseIfStmt = "func f(x i64) i64 {\n    if false { return 1 } " +
				"else if %s { return 2 }\n    return 0\n}\n"
		)

		BeforeEach(func() {
			server, uri, client = SetupTestServerWithClient(lsp.Config{
				NewRoot: func() *symbol.Symbol {
					return symbol.NewRoot(nil, append(stl.NewSymbols(), &symbol.Symbol{
						Name: "gate",
						Kind: symbol.KindChannel,
						Type: types.Chan(types.U8()),
						ID:   1,
					}))
				},
			})
		})

		DescribeTable(
			"should make the condition a bool",
			func(ctx SpecContext, format, numeric, fixed string) {
				content := fmt.Sprintf(format, numeric)
				OpenArcDocument(server, ctx, uri, content)
				actions := asCodeActions(MustSucceed(
					server.CodeAction(ctx, codeActionParams(client.Diagnostics())),
				))
				Expect(actions).To(HaveLen(1))
				action := actions[0]
				Expect(action.Title).To(Equal(fmt.Sprintf("Change to `%s`", fixed)))
				Expect(
					action.Kind,
				).To(HaveValue(Equal(protocol.CodeActionKindQuickFix)))
				Expect(action.IsPreferred).To(HaveValue(BeTrue()))
				Expect(action.Diagnostics).To(HaveLen(1))
				Expect(DiagnosticCode(action.Diagnostics[0])).
					To(Equal(string(codes.TruthyCondition.Code)))
				edits := action.Edit.Changes[uri]
				Expect(edits).To(HaveLen(1))
				Expect(xlsp.ApplyIncrementalChange(
					content,
					&protocol.TextDocumentContentChangePartial{
						Range: edits[0].Range,
						Text:  edits[0].NewText,
					},
				)).To(Equal(fmt.Sprintf(format, fixed)))
			},
			Entry("a nonzero integer transition", transition, "1", "true"),
			Entry("a zero integer transition", transition, "0", "false"),
			Entry("a zero float transition", transition, "0.0", "false"),
			Entry("a unit literal transition", transition, "5s", "true"),
			Entry("a string transition", transition, `"hello"`, "true"),
			Entry("an empty string transition", transition, `""`, "false"),
			Entry("a channel transition", transition, "gate", "gate != 0"),
			Entry(
				"a channel calculation transition",
				transition,
				"gate * 2",
				"gate * 2 != 0",
			),
			Entry("a nonzero integer if", ifStmt, "2", "true"),
			Entry("a negative integer if", ifStmt, "-3", "true"),
			Entry("a zero integer if", ifStmt, "0", "false"),
			Entry("a variable if", ifStmt, "x", "x != 0"),
			Entry("a calculation if", ifStmt, "x + 1", "x + 1 != 0"),
			Entry("a nonzero integer else if", elseIfStmt, "4", "true"),
			Entry("a variable else if", elseIfStmt, "x", "x != 0"),
		)

		It(
			"should return no action for a warning at another range",
			func(ctx SpecContext) {
				OpenArcDocument(server, ctx, uri, fmt.Sprintf(transition, "1"))
				diags := client.Diagnostics()
				Expect(diags).To(HaveLen(1))
				diags[0].Range.End.Character++
				actions := asCodeActions(
					MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
				)
				Expect(actions).To(BeEmpty())
			},
		)
	})

	Describe("Analyzer fixes", func() {
		It("should move the fix to the positions of a block document",
			func(ctx SpecContext) {
				const block = "arc://block/test"
				OpenArcDocument(server, ctx, block, "if 1 { return 1 }\nreturn 0")
				var diags []protocol.Diagnostic
				for _, d := range client.Diagnostics() {
					if DiagnosticCode(d) == string(codes.TruthyCondition.Code) {
						diags = append(diags, d)
					}
				}
				actions := asCodeActions(MustSucceed(
					server.CodeAction(ctx, &protocol.CodeActionParams{
						TextDocument: protocol.TextDocumentIdentifier{URI: block},
						Context:      protocol.CodeActionContext{Diagnostics: diags},
					}),
				))
				Expect(actions).To(HaveLen(1))
				Expect(actions[0].Edit.Changes[block]).To(Equal([]protocol.TextEdit{{
					Range: protocol.Range{
						Start: protocol.Position{Character: 3},
						End:   protocol.Position{Character: 4},
					},
					NewText: "true",
				}}))
			})

		It("should give no fix from the analysis of older content",
			func(ctx SpecContext) {
				OpenArcDocument(
					server,
					ctx,
					uri,
					"sequence main {\n    stage a { 1 => next }\n    stage b {}\n}\n",
				)
				diags := client.Diagnostics()
				Expect(diags).To(HaveLen(1))
				Expect(server.DidChange(ctx, &protocol.DidChangeTextDocumentParams{
					TextDocument: protocol.VersionedTextDocumentIdentifier{
						URI:     uri,
						Version: 2,
					},
					ContentChanges: []protocol.TextDocumentContentChangeEvent{
						&protocol.TextDocumentContentChangeWholeDocument{
							Text: "sequence main {\n    stage a { 12 => next }\n" +
								"    stage b {}\n}\n",
						},
					},
				})).To(Succeed())
				actions := asCodeActions(
					MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
				)
				Expect(actions).To(BeEmpty())
			})
	})

	Describe("Unused import quick fix", func() {
		It(
			"should delete the whole statement when it imports a single module",
			func(ctx SpecContext) {
				content := "import time\n\nfunc test() i64 { return 0 }\n"
				OpenArcDocument(server, ctx, uri, content)

				diags := client.Diagnostics()
				Expect(diags).To(HaveLen(1))
				Expect(
					DiagnosticCode(diags[0]),
				).To(Equal(string(codes.UnusedImport.Code)))

				actions := asCodeActions(
					MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
				)
				Expect(actions).To(HaveLen(1))

				action := actions[0]
				Expect(action.Title).To(Equal("Remove unused import"))
				Expect(
					action.Kind,
				).To(HaveValue(Equal(protocol.CodeActionKindQuickFix)))
				Expect(action.IsPreferred).To(HaveValue(BeTrue()))
				Expect(action.Diagnostics).To(ConsistOf(diags[0]))

				Expect(action.Edit).ToNot(BeNil())
				edits := action.Edit.Changes[uri]
				Expect(edits).To(HaveLen(1))
				Expect(edits[0].NewText).To(BeEmpty())
				Expect(
					edits[0].Range.Start,
				).To(Equal(protocol.Position{Line: 0, Character: 0}))
				Expect(
					edits[0].Range.End,
				).To(Equal(protocol.Position{Line: 1, Character: 0}))
			},
		)

		It(
			"should delete just the item from a parenthesized multi-import block",
			func(ctx SpecContext) {
				content := "import (time control)\n\nfunc test() i64 { return 0 }\n"
				OpenArcDocument(server, ctx, uri, content)

				diags := client.Diagnostics()
				Expect(diags).To(HaveLen(2))
				for _, d := range diags {
					Expect(DiagnosticCode(d)).To(Equal(string(codes.UnusedImport.Code)))
				}

				actions := asCodeActions(
					MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
				)
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
				Expect(
					timeEdit.Range.End,
				).To(Equal(protocol.Position{Line: 0, Character: 13}))

				controlEdit, ok := byStart[12]
				Expect(ok).To(BeTrue())
				Expect(
					controlEdit.Range.End,
				).To(Equal(protocol.Position{Line: 0, Character: 20}))
			},
		)

		It(
			"should delete a middle item from a 3-item parenthesized block",
			func(ctx SpecContext) {
				content := "import (time control math)\n\nfunc test() i64 { return 0 }\n"
				OpenArcDocument(server, ctx, uri, content)

				diags := client.Diagnostics()
				Expect(diags).To(HaveLen(3))
				for _, d := range diags {
					Expect(DiagnosticCode(d)).To(Equal(string(codes.UnusedImport.Code)))
				}

				actions := asCodeActions(
					MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
				)
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
				Expect(
					timeEdit.Range.End,
				).To(Equal(protocol.Position{Line: 0, Character: 13}))

				// Middle item eats forward: start of "control" -> start of "math".
				controlEdit, ok := byStart[13]
				Expect(ok).To(BeTrue())
				Expect(
					controlEdit.Range.End,
				).To(Equal(protocol.Position{Line: 0, Character: 21}))

				// Last item eats backward: end of "control" -> end of "math".
				mathEdit, ok := byStart[20]
				Expect(ok).To(BeTrue())
				Expect(
					mathEdit.Range.End,
				).To(Equal(protocol.Position{Line: 0, Character: 25}))
			},
		)

		It(
			"should return no actions when context has no diagnostics",
			func(ctx SpecContext) {
				OpenArcDocument(
					server,
					ctx,
					uri,
					"import time\n\nfunc test() i64 { return 0 }\n",
				)

				actions := asCodeActions(
					MustSucceed(server.CodeAction(ctx, codeActionParams(nil))),
				)
				Expect(actions).To(BeEmpty())
			},
		)

		It("should ignore diagnostics for unrelated codes", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, uri, "func test() i64 { return missing }\n")

			diags := client.Diagnostics()
			Expect(diags).ToNot(BeEmpty())

			actions := asCodeActions(
				MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
			)
			Expect(actions).To(BeEmpty())
		})

		It("should return nil when the document is not open", func(ctx SpecContext) {
			actions := asCodeActions(
				MustSucceed(server.CodeAction(ctx, &protocol.CodeActionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: "file:///missing.arc",
					},
					Context: protocol.CodeActionContext{},
				})),
			)
			Expect(actions).To(BeEmpty())
		})
	})

	Describe("Deprecated symbol quick fix", func() {
		filterByCode := func(diags []protocol.Diagnostic, code string) []protocol.Diagnostic {
			out := make([]protocol.Diagnostic, 0, len(diags))
			for _, d := range diags {
				if DiagnosticCode(d) == code {
					out = append(out, d)
				}
			}
			return out
		}

		It(
			"should replace a bare deprecated call and add the missing import",
			func(ctx SpecContext) {
				content := "func test() i64 {\n    return now()\n}\n"
				OpenArcDocument(server, ctx, uri, content)

				diags := filterByCode(
					client.Diagnostics(),
					string(codes.DeprecatedSymbol.Code),
				)
				Expect(diags).ToNot(BeEmpty())

				actions := asCodeActions(
					MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
				)
				Expect(actions).To(HaveLen(1))

				action := actions[0]
				Expect(action.Title).To(Equal("Replace 'now' with 'time.now'"))
				Expect(
					action.Kind,
				).To(HaveValue(Equal(protocol.CodeActionKindQuickFix)))
				Expect(action.IsPreferred).To(HaveValue(BeTrue()))
				Expect(action.Edit).ToNot(BeNil())

				edits := action.Edit.Changes[uri]
				Expect(edits).To(HaveLen(2))

				importEdit := edits[0]
				Expect(importEdit.NewText).To(ContainSubstring("import time"))
				Expect(
					importEdit.Range.Start,
				).To(Equal(protocol.Position{Line: 0, Character: 0}))

				renameEdit := edits[1]
				Expect(renameEdit.NewText).To(Equal("time.now"))
				Expect(
					renameEdit.Range.Start,
				).To(Equal(protocol.Position{Line: 1, Character: 11}))
				Expect(
					renameEdit.Range.End,
				).To(Equal(protocol.Position{Line: 1, Character: 14}))
			},
		)

		It(
			"should skip the import edit when the module is already imported",
			func(ctx SpecContext) {
				content := "import time\n\nfunc test() i64 {\n    return now()\n}\n"
				OpenArcDocument(server, ctx, uri, content)

				diags := filterByCode(
					client.Diagnostics(),
					string(codes.DeprecatedSymbol.Code),
				)
				Expect(diags).ToNot(BeEmpty())

				actions := asCodeActions(
					MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
				)
				Expect(actions).To(HaveLen(1))

				edits := actions[0].Edit.Changes[uri]
				Expect(edits).To(HaveLen(1))
				Expect(edits[0].NewText).To(Equal("time.now"))
			},
		)

		It(
			"should reuse an existing alias when the module is imported under one",
			func(ctx SpecContext) {
				content := "import time as t\n\nfunc test() i64 {\n    return now()\n}\n"
				OpenArcDocument(server, ctx, uri, content)

				diags := filterByCode(
					client.Diagnostics(),
					string(codes.DeprecatedSymbol.Code),
				)
				Expect(diags).ToNot(BeEmpty())

				actions := asCodeActions(
					MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
				)
				Expect(actions).To(HaveLen(1))
				Expect(actions[0].Title).To(Equal("Replace 'now' with 't.now'"))

				edits := actions[0].Edit.Changes[uri]
				Expect(edits).To(HaveLen(1))
				Expect(edits[0].NewText).To(Equal("t.now"))
			},
		)

		It(
			"should not produce an action when the deprecated identifier is part of a qualified path",
			func(ctx SpecContext) {
				content := "import time\n\nfunc test() i64 {\n    return time.now()\n}\n"
				OpenArcDocument(server, ctx, uri, content)

				deprecated := filterByCode(
					client.Diagnostics(),
					string(codes.DeprecatedSymbol.Code),
				)
				Expect(deprecated).To(BeEmpty())
			},
		)
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

		It(
			"should add an import when a qualified call references an unimported ambient module",
			func(ctx SpecContext) {
				content := "sequence main {\n    math.avg()\n}\n"
				OpenArcDocument(server, ctx, uri, content)

				diags := client.Diagnostics()
				Expect(diags).ToNot(BeEmpty())

				actions := asCodeActions(
					MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
				)
				action := findActionByTitle(actions, "Add import 'math'")
				Expect(action).ToNot(BeNil())
				Expect(
					action.Kind,
				).To(HaveValue(Equal(protocol.CodeActionKindQuickFix)))
				Expect(action.IsPreferred).To(HaveValue(BeTrue()))

				edits := action.Edit.Changes[uri]
				Expect(edits).To(HaveLen(1))
				Expect(edits[0].NewText).To(ContainSubstring("import math"))
				Expect(
					edits[0].Range.Start,
				).To(Equal(protocol.Position{Line: 0, Character: 0}))
			},
		)

		It(
			"should not offer the fix when the module is already imported",
			func(ctx SpecContext) {
				content := "import math\n\nsequence main {\n    math.avg()\n}\n"
				OpenArcDocument(server, ctx, uri, content)

				diags := client.Diagnostics()
				actions := asCodeActions(
					MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
				)
				Expect(findActionByTitle(actions, "Add import 'math'")).To(BeNil())
			},
		)

		It(
			"should not offer the fix for an undefined symbol that is not an ambient module",
			func(ctx SpecContext) {
				content := "func test() i64 {\n    return notARealModule\n}\n"
				OpenArcDocument(server, ctx, uri, content)

				diags := client.Diagnostics()
				Expect(diags).ToNot(BeEmpty())

				actions := asCodeActions(
					MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
				)
				for _, a := range actions {
					Expect(a.Title).ToNot(HavePrefix("Add import"))
				}
			},
		)

		Describe("User modules", func() {
			BeforeEach(func() {
				sources := map[string]string{
					"lib.m":   "export func f() i64 { return 1 }",
					"lib.n":   "export func f() i64 { return 1 }",
					"other.m": "export func f() i64 { return 1 }",
				}
				server, uri, client = SetupTestServerWithClient(lsp.Config{
					NewRoot: func() *symbol.Symbol { return NewModuleRoot(sources) },
				})
			})
			actionsFor := func(ctx SpecContext, content string) []protocol.CodeAction {
				GinkgoHelper()
				OpenArcDocument(server, ctx, uri, content)
				diags := client.Diagnostics()
				Expect(diags).ToNot(BeEmpty())
				return asCodeActions(
					MustSucceed(server.CodeAction(ctx, codeActionParams(diags))),
				)
			}

			It("should import the one module with the name", func(ctx SpecContext) {
				actions := actionsFor(ctx, "func g() i64 {\n    return n.f()\n}\n")
				action := findActionByTitle(actions, "Add import 'lib.n'")
				Expect(action).ToNot(BeNil())
				Expect(action.IsPreferred).To(HaveValue(BeTrue()))
				Expect(
					action.Edit.Changes[uri][0].NewText,
				).To(Equal("import lib.n\n\n"))
			})

			It(
				"should offer each module with the name and prefer none",
				func(ctx SpecContext) {
					actions := actionsFor(ctx, "func g() i64 {\n    return m.f()\n}\n")
					for _, title := range []string{
						"Add import 'lib.m'",
						"Add import 'other.m'",
					} {
						action := findActionByTitle(actions, title)
						Expect(action).ToNot(BeNil())
						Expect(action.IsPreferred).To(HaveValue(BeFalse()))
					}
				},
			)

			It(
				"should list the modules once per request",
				func(ctx SpecContext) {
					lists := 0
					loader := text.NewLoader(text.LoaderConfig{
						Paths: func(context.Context) ([]string, error) {
							lists++
							return []string{"lib.n", "lib.k"}, nil
						},
					})
					server, uri, client = SetupTestServerWithClient(lsp.Config{
						NewRoot: func() *symbol.Symbol {
							return symbol.NewRoot(
								nil,
								stl.NewSymbols(),
								symbol.WithModules(loader),
							)
						},
					})
					actions := actionsFor(
						ctx,
						"func g() i64 {\n    return n.f() + k.f()\n}\n",
					)
					Expect(
						findActionByTitle(actions, "Add import 'lib.n'"),
					).ToNot(BeNil())
					Expect(
						findActionByTitle(actions, "Add import 'lib.k'"),
					).ToNot(BeNil())
					Expect(lists).To(Equal(1))
				},
			)

			It(
				"should keep the other fixes when the module listing fails",
				func(ctx SpecContext) {
					loader := text.NewLoader(text.LoaderConfig{
						Paths: func(context.Context) ([]string, error) {
							return nil, errors.New("cat")
						},
					})
					server, uri, client = SetupTestServerWithClient(lsp.Config{
						NewRoot: func() *symbol.Symbol {
							return symbol.NewRoot(
								nil,
								stl.NewSymbols(),
								symbol.WithModules(loader),
							)
						},
					})
					actions := actionsFor(
						ctx,
						"func g() f64 {\n    return n.f() + math.sqrt(4.0)\n}\n",
					)
					Expect(findActionByTitle(actions, "Add import 'math'")).
						ToNot(BeNil())
					for _, a := range actions {
						Expect(a.Title).ToNot(ContainSubstring("'n'"))
					}
				},
			)

			It(
				"should keep the path and alias of each import in the merged block",
				func(ctx SpecContext) {
					actions := actionsFor(
						ctx,
						"import lib.m as k\n\nfunc g() i64 {\n    return k.f() + n.f()\n}\n",
					)
					action := findActionByTitle(actions, "Add import 'lib.n'")
					Expect(action).ToNot(BeNil())
					Expect(action.Edit.Changes[uri][0].NewText).
						To(Equal("import (\n    lib.m as k\n    lib.n\n)\n"))
				},
			)
		})

		It(
			"should return no action when context has no diagnostics",
			func(ctx SpecContext) {
				OpenArcDocument(
					server,
					ctx,
					uri,
					"sequence main {\n    math.avg()\n}\n",
				)

				actions := asCodeActions(
					MustSucceed(server.CodeAction(ctx, codeActionParams(nil))),
				)
				Expect(actions).To(BeEmpty())
			},
		)
	})
})
