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
	"github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// OpenDocument is a helper to open a document in the LSP server.
func OpenDocument(
	server protocol.Server,
	ctx context.Context,
	uri uri.URI,
	content,
	languageID string,
) {
	gomega.Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: protocol.LanguageKind(languageID),
			Version:    1,
			Text:       content,
		},
	})).To(gomega.Succeed())
}

// ChangeDocument sends a full-content DidChange to the server.
func ChangeDocument(
	server protocol.Server,
	ctx context.Context,
	uri uri.URI,
	content string,
	version int32,
) {
	gomega.Expect(server.DidChange(ctx, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                version,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			&protocol.TextDocumentContentChangeWholeDocument{Text: content},
		},
	})).To(gomega.Succeed())
}

// Hover returns hover information at the given position.
func Hover(
	server protocol.Server,
	ctx context.Context,
	uri uri.URI,
	line, char uint32,
) *protocol.Hover {
	return testutil.MustSucceed(server.Hover(ctx, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: line, Character: char},
		},
	}))
}

// HoverContents extracts the markup value from a hover result, handling the
// HoverContents union. Returns the empty string when hover is nil or the contents carry
// no text.
func HoverContents(hover *protocol.Hover) string {
	if hover == nil {
		return ""
	}
	switch c := hover.Contents.(type) {
	case *protocol.MarkupContent:
		return c.Value
	case protocol.String:
		return string(c)
	default:
		return ""
	}
}

// Definition returns definition locations at the given position, unwrapping the
// DefinitionResult union into a flat location slice.
func Definition(
	server protocol.Server,
	ctx context.Context,
	uri uri.URI,
	line, char uint32,
) []protocol.Location {
	result := testutil.MustSucceed(server.Definition(ctx, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: line, Character: char},
		},
	}))
	switch r := result.(type) {
	case protocol.LocationSlice:
		return r
	case *protocol.Location:
		return []protocol.Location{*r}
	default:
		return nil
	}
}

// Completion returns completion items at the given position, unwrapping the
// CompletionResult union into a completion list.
func Completion(
	server protocol.Server,
	ctx context.Context,
	uri uri.URI,
	line, char uint32,
) *protocol.CompletionList {
	result := testutil.MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: line, Character: char},
		},
	}))
	switch r := result.(type) {
	case *protocol.CompletionList:
		return r
	case protocol.CompletionItemSlice:
		return &protocol.CompletionList{Items: r}
	default:
		return nil
	}
}

// SemanticTokens returns semantic tokens for the given document.
func SemanticTokens(
	server protocol.Server,
	ctx context.Context,
	uri uri.URI,
) *protocol.SemanticTokens {
	return testutil.MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
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

// ItemDetail returns the detail string of a completion item, or the empty string when
// unset.
func ItemDetail(item protocol.CompletionItem) string {
	v, _ := item.Detail.Get()
	return v
}

// ItemInsertText returns the insert text of a completion item, or the empty string when
// unset.
func ItemInsertText(item protocol.CompletionItem) string {
	v, _ := item.InsertText.Get()
	return v
}

// ItemTextEdit returns the completion item's text edit, or nil when unset or not a
// plain TextEdit.
func ItemTextEdit(item protocol.CompletionItem) *protocol.TextEdit {
	edit, _ := item.TextEdit.(*protocol.TextEdit)
	return edit
}

// DiagnosticCode returns the string code of a diagnostic, or the empty string when
// unset or numeric.
func DiagnosticCode(d protocol.Diagnostic) string {
	code, _ := d.Code.(protocol.String)
	return string(code)
}

// DiagnosticMessage returns the message text of a diagnostic, handling the union
// between plain strings and markup content.
func DiagnosticMessage(d protocol.Diagnostic) string {
	switch m := d.Message.(type) {
	case protocol.String:
		return string(m)
	case *protocol.MarkupContent:
		return m.Value
	default:
		return ""
	}
}
