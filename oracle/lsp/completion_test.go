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
	"github.com/synnaxlabs/oracle/lsp"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

var _ = Describe("Completion", func() {
	var (
		server *lsp.Server
		ctx    context.Context
	)

	BeforeEach(func() {
		server = lsp.New()
		server.SetClient(&MockClient{})
		ctx = context.Background()
		Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:     "file:///test.oracle",
				Version: 1,
				Text:    "User struct {\n  name string\n  domain validate {\n    required\n  }\n}\n",
			},
		})).To(Succeed())
	})

	completionAt := func(line, col uint32) *protocol.CompletionList {
		return MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "file:///test.oracle",
				},
				Position: protocol.Position{Line: line, Character: col},
			},
		}))
	}

	completionFor := func(uri protocol.DocumentURI, line, col uint32) *protocol.CompletionList {
		return MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: line, Character: col},
			},
		}))
	}

	labels := func(items []protocol.CompletionItem) []string {
		result := make([]string, len(items))
		for i, item := range items {
			result[i] = item.Label
		}
		return result
	}

	openDoc := func(uri protocol.DocumentURI, text string) {
		Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI: uri, Version: 1, Text: text,
			},
		})).To(Succeed())
	}

	Describe("Keyword Completions", func() {
		It("should return keyword completions at the start of a line", func() {
			openDoc("file:///empty.oracle", "\n")
			list := completionFor("file:///empty.oracle", 0, 0)
			Expect(labels(list.Items)).To(ContainElements(
				"struct", "field", "domain", "enum", "import",
			))
		})
	})

	Describe("Type Completions", func() {
		It("should return primitive types after field keyword", func() {
			openDoc("file:///field.oracle", "  field name \n")
			list := completionFor("file:///field.oracle", 0, 13)
			Expect(labels(list.Items)).To(ContainElements(
				"string", "int32", "float64", "bool", "uuid",
			))
		})
	})

	Describe("Domain Name Completions", func() {
		It("should return domain names after domain keyword", func() {
			openDoc("file:///domain.oracle", "  domain \n")
			list := completionFor("file:///domain.oracle", 0, 9)
			Expect(labels(list.Items)).To(ContainElements(
				"id", "validate", "ontology", "doc", "go", "ts", "py",
			))
		})
	})

	Describe("Validate Expression Completions", func() {
		It("should return validate expressions inside validate domain", func() {
			list := completionAt(2, 20)
			Expect(labels(list.Items)).To(ContainElements(
				"required", "min_length", "max_length",
			))
		})
	})

	Describe("Go Output Completions", func() {
		It("should return output expressions inside go domain", func() {
			openDoc("file:///go-domain.oracle", "  domain go { \n")
			list := completionFor("file:///go-domain.oracle", 0, 14)
			Expect(labels(list.Items)).To(ContainElements("output", "omit"))
		})
	})

	Describe("TS Expression Completions", func() {
		It("should return ts expressions inside ts domain", func() {
			openDoc("file:///ts-domain.oracle", "  domain ts { \n")
			list := completionFor("file:///ts-domain.oracle", 0, 14)
			Expect(labels(list.Items)).To(ContainElements(
				"output", "use_input", "name",
			))
		})
	})

	Describe("Ontology Expression Completions", func() {
		It("should return ontology expressions inside ontology domain", func() {
			openDoc("file:///ontology-domain.oracle", "  domain ontology { \n")
			list := completionFor("file:///ontology-domain.oracle", 0, 20)
			Expect(labels(list.Items)).To(ContainElement("type"))
		})
	})

	Describe("Unknown Document", func() {
		It("should return empty completions for unknown URI", func() {
			list := completionFor("file:///unknown.oracle", 0, 0)
			Expect(list.Items).To(BeEmpty())
		})
	})

	Describe("Out of Range Position", func() {
		It("should return empty completions for line beyond document", func() {
			list := completionAt(100, 0)
			Expect(list.Items).To(BeEmpty())
		})
	})

	Describe("Prefix Filtering", func() {
		It("should filter completions by typed prefix", func() {
			openDoc("file:///prefix.oracle", "str\n")
			list := completionFor("file:///prefix.oracle", 0, 3)
			for _, item := range list.Items {
				Expect(item.Label).To(HavePrefix("str"))
			}
		})
	})
})

var _ = Describe("Initialize", func() {
	It("should return server capabilities", func() {
		server := lsp.New()
		result := MustSucceed(server.Initialize(
			context.Background(),
			&protocol.InitializeParams{},
		))
		Expect(result.ServerInfo.Name).To(Equal("oracle-lsp"))
		Expect(result.Capabilities.HoverProvider).To(BeTrue())
		Expect(result.Capabilities.DocumentFormattingProvider).To(BeTrue())
	})
})

var _ = Describe("Initialized", func() {
	It("should return nil", func() {
		server := lsp.New()
		Expect(server.Initialized(
			context.Background(),
			&protocol.InitializedParams{},
		)).To(Succeed())
	})
})

var _ = Describe("Shutdown", func() {
	It("should return nil", func() {
		server := lsp.New()
		Expect(server.Shutdown(context.Background())).To(Succeed())
	})
})

var _ = Describe("DidChange", func() {
	var (
		server *lsp.Server
		ctx    context.Context
	)

	BeforeEach(func() {
		server = lsp.New()
		server.SetClient(&MockClient{})
		ctx = context.Background()
	})

	It("should update document content", func() {
		Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI: "file:///change.oracle", Version: 1,
				Text: "User struct {}\n",
			},
		})).To(Succeed())

		Expect(server.DidChange(ctx, &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{
					URI: "file:///change.oracle",
				},
				Version: 2,
			},
			ContentChanges: []protocol.TextDocumentContentChangeEvent{
				{Text: "Status enum {}\n"},
			},
		})).To(Succeed())

		list := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "file:///change.oracle",
				},
				Position: protocol.Position{Line: 0, Character: 0},
			},
		}))
		Expect(list.Items).ToNot(BeEmpty())
	})

	It("should handle empty content changes", func() {
		Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI: "file:///empty-change.oracle", Version: 1,
				Text: "User struct {}\n",
			},
		})).To(Succeed())

		Expect(server.DidChange(ctx, &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{
					URI: "file:///empty-change.oracle",
				},
				Version: 2,
			},
			ContentChanges: []protocol.TextDocumentContentChangeEvent{},
		})).To(Succeed())
	})
})

var _ = Describe("DidClose", func() {
	It("should remove document and clear diagnostics", func() {
		server := lsp.New()
		server.SetClient(&MockClient{})
		ctx := context.Background()
		Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI: "file:///close.oracle", Version: 1,
				Text: "User struct {}\n",
			},
		})).To(Succeed())

		Expect(server.DidClose(ctx, &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///close.oracle",
			},
		})).To(Succeed())

		list := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "file:///close.oracle",
				},
				Position: protocol.Position{Line: 0, Character: 0},
			},
		}))
		Expect(list.Items).To(BeEmpty())
	})
})

var _ = Describe("Hover", func() {
	var (
		server *lsp.Server
		ctx    context.Context
	)

	BeforeEach(func() {
		server = lsp.New()
		server.SetClient(&MockClient{})
		ctx = context.Background()
		Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI: "file:///hover.oracle", Version: 1,
				Text: "User struct {\n    name string\n}\n",
			},
		})).To(Succeed())
	})

	hoverAt := func(uri protocol.DocumentURI, line, col uint32) *protocol.Hover {
		return MustSucceed(server.Hover(ctx, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: line, Character: col},
			},
		}))
	}

	It("should return hover docs for struct keyword", func() {
		hover := hoverAt("file:///hover.oracle", 0, 6)
		Expect(hover).ToNot(BeNil())
		Expect(hover.Contents.Value).To(ContainSubstring("struct"))
	})

	It("should return hover docs for string type", func() {
		hover := hoverAt("file:///hover.oracle", 1, 10)
		Expect(hover).ToNot(BeNil())
		Expect(hover.Contents.Value).To(ContainSubstring("string"))
	})

	It("should return nil for unknown document", func() {
		hover := hoverAt("file:///unknown.oracle", 0, 0)
		Expect(hover).To(BeNil())
	})

	It("should return nil for non-keyword word", func() {
		hover := hoverAt("file:///hover.oracle", 0, 2)
		Expect(hover).To(BeNil())
	})

	It("should return nil for whitespace position", func() {
		Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI: "file:///spaces.oracle", Version: 1,
				Text: "    \n",
			},
		})).To(Succeed())
		hover := hoverAt("file:///spaces.oracle", 0, 2)
		Expect(hover).To(BeNil())
	})

	It("should return nil for line beyond document", func() {
		hover := hoverAt("file:///hover.oracle", 100, 0)
		Expect(hover).To(BeNil())
	})
})

var _ = Describe("SemanticTokensFull", func() {
	var (
		server *lsp.Server
		ctx    context.Context
	)

	BeforeEach(func() {
		server = lsp.New()
		server.SetClient(&MockClient{})
		ctx = context.Background()
	})

	openDoc := func(uri protocol.DocumentURI, text string) {
		Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI: uri, Version: 1, Text: text,
			},
		})).To(Succeed())
	}

	tokensFor := func(uri protocol.DocumentURI) *protocol.SemanticTokens {
		return MustSucceed(server.SemanticTokensFull(
			ctx,
			&protocol.SemanticTokensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			},
		))
	}

	It("should return empty tokens for unknown document", func() {
		Expect(tokensFor("file:///unknown.oracle").Data).To(BeEmpty())
	})

	It("should return tokens for a schema with keywords and types", func() {
		openDoc("file:///tokens.oracle", "User struct {\n    name string\n}\n")
		Expect(tokensFor("file:///tokens.oracle").Data).ToNot(BeEmpty())
	})

	It("should tokenize comments", func() {
		openDoc("file:///comments.oracle", "// a comment\nUser struct {}\n")
		Expect(tokensFor("file:///comments.oracle").Data).ToNot(BeEmpty())
	})

	It("should tokenize string literals", func() {
		openDoc("file:///strings.oracle", "import \"common.oracle\"\n")
		Expect(tokensFor("file:///strings.oracle").Data).ToNot(BeEmpty())
	})

	It("should tokenize enum with number literals", func() {
		openDoc("file:///enum.oracle", "Status enum {\n    Active = 1\n}\n")
		Expect(tokensFor("file:///enum.oracle").Data).ToNot(BeEmpty())
	})
})

var _ = Describe("Formatting", func() {
	var (
		server *lsp.Server
		ctx    context.Context
	)

	BeforeEach(func() {
		server = lsp.New()
		server.SetClient(&MockClient{})
		ctx = context.Background()
	})

	openDoc := func(uri protocol.DocumentURI, text string) {
		Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI: uri, Version: 1, Text: text,
			},
		})).To(Succeed())
	}

	formatDoc := func(uri protocol.DocumentURI) []protocol.TextEdit {
		return MustSucceed(server.Formatting(ctx, &protocol.DocumentFormattingParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		}))
	}

	It("should return nil for unknown document", func() {
		Expect(formatDoc("file:///unknown.oracle")).To(BeNil())
	})

	It("should return nil when content is already formatted", func() {
		openDoc("file:///formatted.oracle", "User struct {}\n")
		Expect(formatDoc("file:///formatted.oracle")).To(BeNil())
	})

	It("should return text edit when formatting changes content", func() {
		openDoc("file:///unformatted.oracle", "User struct {\n  x int32\n  longName string\n}\n")
		edits := formatDoc("file:///unformatted.oracle")
		Expect(edits).To(HaveLen(1))
		Expect(edits[0].Range.Start.Line).To(Equal(uint32(0)))
		Expect(edits[0].NewText).To(ContainSubstring("x        int32"))
	})
})
