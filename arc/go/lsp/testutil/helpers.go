// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"context"

	"github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/lsp"
	xutil "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

func SetupTestServer(cfgs ...lsp.Config) (*lsp.Server, protocol.DocumentURI) {
	server := xutil.MustSucceed(lsp.New(cfgs...))
	uri := protocol.DocumentURI("file:///test.arc")
	server.SetClient(&MockClient{})
	return server, uri
}

func SetupTestServerWithClient(cfgs ...lsp.Config) (*lsp.Server, protocol.DocumentURI, *MockClient) {
	server := xutil.MustSucceed(lsp.New(cfgs...))
	uri := protocol.DocumentURI("file:///test.arc")
	client := &MockClient{}
	server.SetClient(client)
	return server, uri, client
}

// OpenDocument is a helper to open a document in the LSP server.
func OpenDocument(
	server *lsp.Server,
	ctx context.Context,
	uri protocol.DocumentURI,
	content string,
) {
	gomega.Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "arc",
			Version:    1,
			Text:       content,
		},
	})).To(gomega.Succeed())
}

func Hover(
	server *lsp.Server,
	ctx context.Context,
	uri protocol.DocumentURI,
	line, char uint32,
) *protocol.Hover {
	return xutil.MustSucceed(server.Hover(ctx, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: line, Character: char},
		},
	}))
}

func Definition(
	server *lsp.Server,
	ctx context.Context,
	uri protocol.DocumentURI,
	line, char uint32,
) []protocol.Location {
	return xutil.MustSucceed(server.Definition(ctx, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: line, Character: char},
		},
	}))
}

func Completion(
	server *lsp.Server,
	ctx context.Context,
	uri protocol.DocumentURI,
	line, char uint32,
) *protocol.CompletionList {
	return xutil.MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: line, Character: char},
		},
	}))
}

func SemanticTokens(
	server *lsp.Server,
	ctx context.Context,
	uri protocol.DocumentURI,
) *protocol.SemanticTokens {
	return xutil.MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	}))
}

func FindCompletion(
	items []protocol.CompletionItem,
	label string,
) (protocol.CompletionItem, bool) {
	return lo.Find(items, func(item protocol.CompletionItem) bool {
		return item.Label == label
	})
}

func HasCompletion(items []protocol.CompletionItem, label string) bool {
	_, found := FindCompletion(items, label)
	return found
}

// ChangeDocument sends a full-content DidChange to the server.
func ChangeDocument(
	server *lsp.Server,
	ctx context.Context,
	uri protocol.DocumentURI,
	content string,
	version int32,
) {
	gomega.Expect(server.DidChange(ctx, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                version,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			{Text: content},
		},
	})).To(gomega.Succeed())
}
