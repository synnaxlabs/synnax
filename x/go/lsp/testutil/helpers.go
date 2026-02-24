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
	xutil "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

// OpenDocument is a helper to open a document in the LSP server.
func OpenDocument(
	server protocol.Server,
	ctx context.Context,
	uri protocol.DocumentURI,
	content string,
	languageID string,
) {
	gomega.Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: protocol.LanguageIdentifier(languageID),
			Version:    1,
			Text:       content,
		},
	})).To(gomega.Succeed())
}

// ChangeDocument sends a full-content DidChange to the server.
func ChangeDocument(
	server protocol.Server,
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

// Hover returns hover information at the given position.
func Hover(
	server protocol.Server,
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

// Definition returns definition locations at the given position.
func Definition(
	server protocol.Server,
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

// Completion returns completion items at the given position.
func Completion(
	server protocol.Server,
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

// SemanticTokens returns semantic tokens for the given document.
func SemanticTokens(
	server protocol.Server,
	ctx context.Context,
	uri protocol.DocumentURI,
) *protocol.SemanticTokens {
	return xutil.MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	}))
}

// FindCompletion finds a completion item by label in the given items slice.
func FindCompletion(
	items []protocol.CompletionItem,
	label string,
) (protocol.CompletionItem, bool) {
	return lo.Find(items, func(item protocol.CompletionItem) bool {
		return item.Label == label
	})
}

// HasCompletion reports whether a completion item with the given label exists.
func HasCompletion(items []protocol.CompletionItem, label string) bool {
	_, found := FindCompletion(items, label)
	return found
}
