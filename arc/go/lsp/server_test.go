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
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/lsp/testutil"
	"github.com/synnaxlabs/x/observe"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

var _ = Describe("Server Diagnostics", func() {
	var (
		server *lsp.Server
		docURI uri.URI
		client *MockClient
	)

	BeforeEach(func() {
		server, docURI, client = SetupTestServerWithClient()
	})

	Describe("Diagnostic Range", func() {
		It(
			"Should publish diagnostics with correct end position for undefined symbol",
			func(ctx SpecContext) {
				OpenArcDocument(
					server,
					ctx,
					docURI,
					"func test() {\n\tx := undefined_var\n}",
				)

				Expect(client.Diagnostics()).To(HaveLen(1))
				diag := client.Diagnostics()[0]
				Expect(
					DiagnosticMessage(diag),
				).To(ContainSubstring("undefined symbol: undefined_var"))
				Expect(diag.Range.Start.Line).To(Equal(uint32(1)))
				Expect(diag.Range.Start.Character).To(Equal(uint32(6)))
				Expect(diag.Range.End.Line).To(Equal(uint32(1)))
				Expect(diag.Range.End.Character).To(Equal(uint32(19)))
			},
		)

		It(
			"Should publish diagnostics with correct end position for short identifier",
			func(ctx SpecContext) {
				OpenArcDocument(server, ctx, docURI, "func test() {\n\tx := y\n}")

				Expect(client.Diagnostics()).To(HaveLen(1))
				diag := client.Diagnostics()[0]
				Expect(
					DiagnosticMessage(diag),
				).To(ContainSubstring("undefined symbol: y"))
				Expect(diag.Range.Start.Line).To(Equal(uint32(1)))
				Expect(diag.Range.Start.Character).To(Equal(uint32(6)))
				Expect(diag.Range.End.Line).To(Equal(uint32(1)))
				Expect(diag.Range.End.Character).To(Equal(uint32(7)))
			},
		)

		It(
			"Should publish diagnostics with fallback end position when no stop token",
			func(ctx SpecContext) {
				OpenArcDocument(server, ctx, docURI, "func test() i32 {\n\tx := 1\n}")

				Expect(client.Diagnostics()).To(HaveLen(1))
				diag := client.Diagnostics()[0]
				Expect(DiagnosticMessage(diag)).To(ContainSubstring("must return"))
				Expect(
					diag.Range.End.Line,
				).To(BeNumerically(">=", diag.Range.Start.Line))
				Expect(
					diag.Range.End.Character,
				).To(BeNumerically(">=", diag.Range.Start.Character))
			},
		)

		It(
			"Should handle multiple diagnostics with correct ranges",
			func(ctx SpecContext) {
				OpenArcDocument(
					server,
					ctx,
					docURI,
					"func test() {\n\ta := undefined1\n\tb := undefined2\n}",
				)

				Expect(client.Diagnostics()).To(HaveLen(2))

				diag1 := client.Diagnostics()[0]
				Expect(
					DiagnosticMessage(diag1),
				).To(ContainSubstring("undefined symbol: undefined1"))
				Expect(diag1.Range.Start.Line).To(Equal(uint32(1)))
				Expect(diag1.Range.End.Line).To(Equal(uint32(1)))
				Expect(diag1.Range.End.Character).To(Equal(uint32(16)))

				diag2 := client.Diagnostics()[1]
				Expect(
					DiagnosticMessage(diag2),
				).To(ContainSubstring("undefined symbol: undefined2"))
				Expect(diag2.Range.Start.Line).To(Equal(uint32(2)))
				Expect(diag2.Range.End.Line).To(Equal(uint32(2)))
				Expect(diag2.Range.End.Character).To(Equal(uint32(16)))
			},
		)

		It(
			"Should handle block URI diagnostics with correct ranges",
			func(ctx SpecContext) {
				blockURI := uri.URI("arc://block/test")
				OpenArcDocument(server, ctx, blockURI, "x := undefined_var")

				Expect(client.Diagnostics()).To(HaveLen(1))
				diag := client.Diagnostics()[0]
				Expect(
					DiagnosticMessage(diag),
				).To(ContainSubstring("undefined symbol: undefined_var"))
				Expect(
					diag.Range.End.Character,
				).To(BeNumerically(">", diag.Range.Start.Character))
			},
		)
	})

	Describe("Diagnostic Severity", func() {
		It("Should set correct severity for errors", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func test() {\n\tx := undefined\n}")

			Expect(client.Diagnostics()).To(HaveLen(1))
			Expect(
				client.Diagnostics()[0].Severity,
			).To(Equal(protocol.DiagnosticSeverityError))
		})
	})

	Describe("Diagnostic Error Codes", func() {
		It(
			"Should include error code for function argument count mismatch",
			func(ctx SpecContext) {
				OpenArcDocument(
					server,
					ctx,
					docURI,
					"func add(x i64, y i64) i64 { return x + y }\nfunc test() { z := add(1) }",
				)

				Expect(client.Diagnostics()).To(HaveLen(1))
				Expect(DiagnosticCode(client.Diagnostics()[0])).To(Equal("ARC3001"))
			},
		)

		It(
			"Should include error code for function argument type mismatch",
			func(ctx SpecContext) {
				OpenArcDocument(
					server,
					ctx,
					docURI,
					"func process(x i32) i32 { return x }\nfunc test() { z := process(\"hello\") }",
				)

				Expect(client.Diagnostics()).To(HaveLen(1))
				Expect(DiagnosticCode(client.Diagnostics()[0])).To(Equal("ARC3002"))
			},
		)
	})

	Describe("Diagnostic Related Information", func() {
		It(
			"Should include function signature in related information for argument errors",
			func(ctx SpecContext) {
				OpenArcDocument(
					server,
					ctx,
					docURI,
					"func add(x i64, y i64) i64 { return x + y }\nfunc test() { z := add(1) }",
				)

				Expect(client.Diagnostics()).To(HaveLen(1))
				Expect(client.Diagnostics()[0].RelatedInformation).To(HaveLen(1))
				Expect(
					client.Diagnostics()[0].RelatedInformation[0].Message,
				).To(ContainSubstring("add(x i64, y i64) i64"))
			},
		)
	})
})

var _ = Describe("Debounced Diagnostics", func() {
	var (
		server *lsp.Server
		docURI uri.URI
		client *MockClient
	)

	BeforeEach(func() {
		server, docURI, client = SetupTestServerWithClient(lsp.Config{
			DebounceDelay:    20 * time.Millisecond,
			MaxDebounceDelay: 200 * time.Millisecond,
		})
	})

	It("Should publish diagnostics after debounce delay", func(ctx SpecContext) {
		OpenArcDocument(server, ctx, docURI, "func test() {}")
		baseline := client.PublishCount()

		ChangeDocument(server, ctx, docURI, "func test() {\n\tx := undefined\n}", 2)

		Expect(client.WaitForDiagnostics(baseline, 500*time.Millisecond)).To(BeTrue())
		Expect(client.Diagnostics()).To(HaveLen(1))
		Expect(
			DiagnosticMessage(client.Diagnostics()[0]),
		).To(ContainSubstring("undefined symbol"))
	})

	It("Should coalesce rapid changes into a single publish", func(ctx SpecContext) {
		OpenArcDocument(server, ctx, docURI, "func test() {}")
		baseline := client.PublishCount()

		for i := 2; i <= 6; i++ {
			ChangeDocument(
				server,
				ctx,
				docURI,
				"func test() {\n\tx := undefined\n}",
				int32(i),
			)
		}

		Expect(client.WaitForDiagnostics(baseline, 500*time.Millisecond)).To(BeTrue())
		time.Sleep(50 * time.Millisecond)
		// Should have far fewer publishes than changes
		Expect(client.PublishCount() - baseline).To(BeNumerically("<=", 2))
	})

	It("Should force-flush on DidSave", func(ctx SpecContext) {
		OpenArcDocument(server, ctx, docURI, "func test() {}")
		baseline := client.PublishCount()

		ChangeDocument(server, ctx, docURI, "func test() {\n\tx := undefined\n}", 2)
		// Immediately save - should flush without waiting for debounce
		Expect(server.DidSave(ctx, &protocol.DidSaveTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		})).To(Succeed())

		Expect(client.PublishCount()).To(BeNumerically(">", baseline))
		Expect(client.Diagnostics()).To(HaveLen(1))
	})

	It(
		"Should refresh semantic tokens after debounced analysis",
		func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func test() {}")
			baseline := client.SemanticRefreshCount()

			ChangeDocument(server, ctx, docURI, "func dog() {}", 2)

			Expect(
				client.WaitForSemanticRefresh(baseline, 500*time.Millisecond),
			).To(BeTrue())
		},
	)

	It("Should cancel stale analysis when new change arrives", func(ctx SpecContext) {
		OpenArcDocument(server, ctx, docURI, "func test() {}")
		baseline := client.PublishCount()

		// Send invalid code, then quickly send valid code
		ChangeDocument(server, ctx, docURI, "func test() {\n\tx := undefined\n}", 2)
		ChangeDocument(server, ctx, docURI, "func test() {\n\tx := 42\n}", 3)

		Expect(client.WaitForDiagnostics(baseline, 500*time.Millisecond)).To(BeTrue())
		time.Sleep(50 * time.Millisecond)
		// The final diagnostics should be clean (from the valid code)
		Expect(client.Diagnostics()).To(BeEmpty())
	})
})

var _ = Describe("Incremental Sync", func() {
	var (
		server *lsp.Server
		docURI uri.URI
		client *MockClient
	)

	BeforeEach(func() {
		server, docURI, client = SetupTestServerWithClient(lsp.Config{
			DebounceDelay:    5 * time.Millisecond,
			MaxDebounceDelay: 50 * time.Millisecond,
		})
	})

	It("Should apply incremental changes correctly", func(ctx SpecContext) {
		OpenArcDocument(server, ctx, docURI, "func test() {\n\tx := 42\n}")
		baseline := client.PublishCount()

		// Send an incremental change: replace "42" with "undefined"
		// In "\tx := 42", tab=0, x=1, ' '=2, :=3, ==4, ' '=5, 4=6, 2=7
		Expect(server.DidChange(ctx, &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: docURI},
				Version:                2,
			},
			ContentChanges: []protocol.TextDocumentContentChangeEvent{
				&protocol.TextDocumentContentChangePartial{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 6},
						End:   protocol.Position{Line: 1, Character: 8},
					},
					Text: "undefined",
				},
			},
		})).To(Succeed())

		Expect(client.WaitForDiagnostics(baseline, 500*time.Millisecond)).To(BeTrue())
		Expect(client.Diagnostics()).To(HaveLen(1))
		Expect(
			DiagnosticMessage(client.Diagnostics()[0]),
		).To(ContainSubstring("undefined symbol"))
	})

	It(
		"Should not treat a newline insertion at position (0,0) as a full replacement",
		func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func test() {\n\tx := 42\n}")
			Expect(client.Diagnostics()).To(BeEmpty())
			baseline := client.PublishCount()

			// Insert a newline at the very start of the document. The editor sends
			// Range{(0,0)-(0,0)} with Text="\n". Because the protocol library
			// deserializes an absent range to the same zero value, IsFullReplacement
			// incorrectly treats this as a full replacement, wiping the document
			// content to just "\n".
			Expect(server.DidChange(ctx, &protocol.DidChangeTextDocumentParams{
				TextDocument: protocol.VersionedTextDocumentIdentifier{
					TextDocumentIdentifier: protocol.TextDocumentIdentifier{
						URI: docURI,
					},
					Version: 2,
				},
				ContentChanges: []protocol.TextDocumentContentChangeEvent{
					&protocol.TextDocumentContentChangePartial{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 0},
						},
						Text: "\n",
					},
				},
			})).To(Succeed())

			Expect(
				client.WaitForDiagnostics(baseline, 500*time.Millisecond),
			).To(BeTrue())
			// The document should now be "\nfunc test() {\n\tx := 42\n}". If
			// IsFullReplacement incorrectly fires, it becomes just "\n" and semantic
			// tokens will be empty.
			tokens := SemanticTokens(server, ctx, docURI)
			Expect(tokens).ToNot(BeNil())
			Expect(tokens.Data).ToNot(BeEmpty())
		},
	)

	It(
		"Should not break when selecting and replacing the first line",
		func(ctx SpecContext) {
			program := "sequence main {\n    stage first {\n         1 -> ox_mpv_cmd\n    }\n}"
			OpenArcDocument(server, ctx, docURI, program)
			baseline := client.PublishCount()

			// Simulate selecting from col 0 to the end of the first line and pressing
			// Enter (replacing the selection with a newline).
			Expect(server.DidChange(ctx, &protocol.DidChangeTextDocumentParams{
				TextDocument: protocol.VersionedTextDocumentIdentifier{
					TextDocumentIdentifier: protocol.TextDocumentIdentifier{
						URI: docURI,
					},
					Version: 2,
				},
				ContentChanges: []protocol.TextDocumentContentChangeEvent{
					&protocol.TextDocumentContentChangePartial{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 16},
						},
						Text: "\n",
					},
				},
			})).To(Succeed())

			Expect(
				client.WaitForDiagnostics(baseline, 500*time.Millisecond),
			).To(BeTrue())

			// After the edit the document is "\n\n    stage first {...". The exact
			// diagnostics don't matter as much as verifying that the server still
			// produces them (analysis didn't silently break). With the bug, the
			// document would be wiped to just "\n".
			tokens := SemanticTokens(server, ctx, docURI)
			Expect(tokens).ToNot(BeNil())
			Expect(tokens.Data).ToNot(BeEmpty())
		},
	)
})

var _ = Describe("External Change Notifications", func() {
	var (
		server   *lsp.Server
		docURI   uri.URI
		client   *MockClient
		resolver StaticResolver
		observer observe.Observer[struct{}]
	)

	BeforeEach(func() {
		resolver = StaticResolver{}
		observer = observe.New[struct{}]()
		server, docURI, client = SetupTestServerWithClient(lsp.Config{
			NewRoot:          func() *symbol.Symbol { return NewRoot(resolver) },
			OnExternalChange: observer,
		})
		DeferCleanup(func(ctx SpecContext) {
			Expect(server.Shutdown(ctx)).To(Succeed())
		})
	})

	It(
		"Should republish diagnostics when external state changes",
		func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func test() {\n\tx := my_channel\n}")
			Expect(client.Diagnostics()).To(HaveLen(1))
			Expect(
				DiagnosticMessage(client.Diagnostics()[0]),
			).To(ContainSubstring("undefined symbol: my_channel"))
			resolver.Add(symbol.Symbol{
				Name: "my_channel",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F32()),
			})
			observer.Notify(ctx, struct{}{})
			Eventually(
				func() []protocol.Diagnostic { return client.Diagnostics() },
			).Should(BeEmpty())
		},
	)

	It(
		"Should refresh semantic tokens when external state changes",
		func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func test() {\n\tx := my_channel\n}")
			baseline := client.SemanticRefreshCount()

			resolver.Add(symbol.Symbol{
				Name: "my_channel",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F32()),
			})
			observer.Notify(ctx, struct{}{})

			Eventually(
				func() int { return client.SemanticRefreshCount() },
			).Should(BeNumerically(">", baseline))
		},
	)

	It(
		"Should show errors when a previously valid symbol is removed",
		func(ctx SpecContext) {
			resolver.Add(symbol.Symbol{
				Name: "sensor",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
			})
			OpenArcDocument(server, ctx, docURI, "func test() {\n\tx := sensor\n}")
			Expect(client.Diagnostics()).To(BeEmpty())
			resolver.Remove("sensor")
			observer.Notify(ctx, struct{}{})
			Eventually(func() int { return len(client.Diagnostics()) }).Should(Equal(1))
			Expect(
				DiagnosticMessage(client.Diagnostics()[0]),
			).To(ContainSubstring("undefined symbol: sensor"))
		},
	)

	It(
		"Should republish diagnostics for multiple open documents",
		func(ctx SpecContext) {
			docURI2 := uri.URI("file:///test2.arc")
			OpenArcDocument(server, ctx, docURI, "func test1() {\n\tx := channel_a\n}")
			OpenArcDocument(server, ctx, docURI2, "func test2() {\n\ty := channel_b\n}")
			Expect(client.Diagnostics()).To(HaveLen(1))
			resolver.Add(symbol.Symbol{
				Name: "channel_a",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.I32()),
			})
			resolver.Add(symbol.Symbol{
				Name: "channel_b",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.I64()),
			})
			observer.Notify(ctx, struct{}{})
			Eventually(
				func() []protocol.Diagnostic { return client.Diagnostics() },
			).Should(BeEmpty())
		},
	)

	It(
		"Should not race feature queries against a concurrent republish",
		func(ctx SpecContext) {
			resolver.Add(symbol.Symbol{
				Name: "my_channel",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F32()),
			})
			OpenArcDocument(server, ctx, docURI, "func test() {\n\tx := my_channel\n}")

			const workers = 8
			var wg sync.WaitGroup
			done := make(chan struct{})
			wg.Add(workers)
			for range workers {
				go func() {
					defer wg.Done()
					params := &protocol.PrepareRenameParams{
						TextDocumentPositionParams: protocol.TextDocumentPositionParams{
							TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
							Position:     protocol.Position{Line: 1, Character: 6},
						},
					}
					for {
						select {
						case <-done:
							return
						default:
							// Errors are irrelevant here; the assertion is the race
							// detector observing no concurrent access to the IR.
							_, _ = server.PrepareRename(ctx, params)
						}
					}
				}()
			}
			for range 50 {
				observer.Notify(ctx, struct{}{})
			}
			close(done)
			wg.Wait()
		},
	)
})

var _ = Describe("Server Lifecycle", func() {
	var (
		server *lsp.Server
		docURI uri.URI
		client *MockClient
	)

	BeforeEach(func() {
		server, docURI, client = SetupTestServerWithClient()
	})

	Describe("New", func() {
		It("Should reject a config without NewRoot", func() {
			Expect(lsp.New()).Error().To(MatchError(ContainSubstring("new_root")))
		})
	})

	Describe("Initialized", func() {
		It("Should accept the initialized notification", func(ctx SpecContext) {
			Expect(server.Initialized(ctx, &protocol.InitializedParams{})).To(Succeed())
		})
	})

	Describe("Symbol requests", func() {
		It("Should return an empty documentSymbol result", func(ctx SpecContext) {
			Expect(
				MustSucceed(server.DocumentSymbol(ctx, &protocol.DocumentSymbolParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
				})),
			).To(BeNil())
		})

		It("Should return an empty workspace/symbol result", func(ctx SpecContext) {
			Expect(MustSucceed(server.Symbols(ctx, &protocol.WorkspaceSymbolParams{
				Query: "test",
			}))).To(BeNil())
		})
	})

	Describe("DidClose", func() {
		It("Should drop the document and clear its diagnostics", func(ctx SpecContext) {
			OpenArcDocument(
				server,
				ctx,
				docURI,
				"func test() {\n\tx := undefined_var\n}",
			)
			Expect(client.Diagnostics()).ToNot(BeEmpty())
			Expect(server.DidClose(ctx, &protocol.DidCloseTextDocumentParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			})).To(Succeed())
			Expect(client.Diagnostics()).To(BeEmpty())
		})

		It("Should tolerate closing an unopened document", func(ctx SpecContext) {
			Expect(server.DidClose(ctx, &protocol.DidCloseTextDocumentParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "file:///unknown.arc",
				},
			})).To(Succeed())
		})
	})

	Describe("Unopened documents", func() {
		It("Should ignore a change for an unopened document", func(ctx SpecContext) {
			Expect(server.DidChange(ctx, &protocol.DidChangeTextDocumentParams{
				TextDocument: protocol.VersionedTextDocumentIdentifier{
					TextDocumentIdentifier: protocol.TextDocumentIdentifier{
						URI: "file:///unknown.arc",
					},
					Version: 2,
				},
				ContentChanges: []protocol.TextDocumentContentChangeEvent{
					&protocol.TextDocumentContentChangeWholeDocument{Text: "x"},
				},
			})).To(Succeed())
		})

		It("Should ignore a save for an unopened document", func(ctx SpecContext) {
			Expect(server.DidSave(ctx, &protocol.DidSaveTextDocumentParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "file:///unknown.arc",
				},
			})).To(Succeed())
		})
	})
})
