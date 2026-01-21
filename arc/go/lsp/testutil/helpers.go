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
	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

func SetupTestServer(cfgs ...lsp.Config) (*lsp.Server, protocol.DocumentURI) {
	server := testutil.MustSucceed(lsp.New(cfgs...))
	uri := protocol.DocumentURI("file:///test.arc")
	server.SetClient(&MockClient{})
	return server, uri
}

func SetupTestServerWithClient(cfgs ...lsp.Config) (*lsp.Server, protocol.DocumentURI, *MockClient) {
	server := testutil.MustSucceed(lsp.New(cfgs...))
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
	result, err := server.Hover(ctx, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: line, Character: char},
		},
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return result
}

func Definition(
	server *lsp.Server,
	ctx context.Context,
	uri protocol.DocumentURI,
	line, char uint32,
) []protocol.Location {
	result, err := server.Definition(ctx, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: line, Character: char},
		},
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return result
}

func Completion(
	server *lsp.Server,
	ctx context.Context,
	uri protocol.DocumentURI,
	line, char uint32,
) *protocol.CompletionList {
	result, err := server.Completion(ctx, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: line, Character: char},
		},
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return result
}

func SemanticTokens(
	server *lsp.Server,
	ctx context.Context,
	uri protocol.DocumentURI,
) *protocol.SemanticTokens {
	result, err := server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return result
}
