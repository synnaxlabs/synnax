// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp

import (
	"context"
	"strings"

	"go.lsp.dev/protocol"
)

type CompletionInfo struct {
	Label        string
	Detail       string
	Doc          string
	Insert       string
	InsertFormat protocol.InsertTextFormat
	Kind         protocol.CompletionItemKind
}

var completions = []CompletionInfo{
	{
		Label:  "i8",
		Detail: "Signed 8-bit integer",
		Doc:    "Range: -128 to 172",
		Kind:   protocol.CompletionItemKindClass,
	},
	{
		Label:  "u8",
		Detail: "Unsigned 8-bit integer",
		Doc:    "Range: 0 to 255",
		Kind:   protocol.CompletionItemKindClass,
	},
	{
		Label:  "i16",
		Detail: "Signed 16-bit integer",
		Doc:    "Range: -32768 to 32767",
		Kind:   protocol.CompletionItemKindClass,
	},
	{
		Label:  "u16",
		Detail: "Unsigned 16-bit integer",
		Doc:    "Range: 0 to 65535",
		Kind:   protocol.CompletionItemKindClass,
	},
	{
		Label:  "i32",
		Detail: "Signed 32-bit integer",
		Doc:    "Range: -2147483648 to 2147483647",
		Kind:   protocol.CompletionItemKindClass,
	},
	{
		Label:  "u32",
		Detail: "Unsigned 32-bit integer",
		Doc:    "Range: 0 to 4294967295",
		Kind:   protocol.CompletionItemKindClass,
	},
	{
		Label:  "i64",
		Detail: "Signed 64-bit integer",
		Doc:    "Range: -9223372036854775808 to 9223372036854775807",
		Kind:   protocol.CompletionItemKindClass,
	},
	{
		Label:  "u64",
		Detail: "Unsigned 64-bit integer",
		Doc:    "Range: 0 to 18446744073709551615",
		Kind:   protocol.CompletionItemKindClass,
	},
	{
		Label:  "f32",
		Detail: "32-bit float",
		Doc:    "Single precision floating point",
		Kind:   protocol.CompletionItemKindClass,
	},
	{
		Label:  "f64",
		Detail: "64-bit float",
		Kind:   protocol.CompletionItemKindClass,
	},
	{
		Label:  "string",
		Detail: "String type",
		Doc:    "Immutable UTF-8 string",
		Kind:   protocol.CompletionItemKindClass,
	},
	{
		Label:  "timestamp",
		Detail: "Timestamp type",
		Kind:   protocol.CompletionItemKindClass,
	},
	{
		Label:  "series",
		Detail: "Series type",
		Doc:    "Homogeneous array of values",
		Kind:   protocol.CompletionItemKindClass,
	},
	{
		Label:  "chan",
		Detail: "Channel type",
		Doc:    "Communication channel",
		Kind:   protocol.CompletionItemKindClass,
	},
	{
		Label:        "len",
		Detail:       "len(series) i64",
		Doc:          "Returns the length of a series",
		Insert:       "len($0)",
		Kind:         protocol.CompletionItemKindFunction,
		InsertFormat: protocol.InsertTextFormatSnippet,
	},
	{
		Label:        "now",
		Detail:       "now() timestamp",
		Doc:          "Returns the current timestamp",
		Insert:       "now()",
		Kind:         protocol.CompletionItemKindFunction,
		InsertFormat: protocol.InsertTextFormatSnippet,
	},
	{
		Label:  "ns",
		Detail: "Nanoseconds",
		Doc:    "1/1000000000 seconds",
		Insert: "ns",
		Kind:   protocol.CompletionItemKindUnit,
	},
	{
		Label:  "us",
		Detail: "Microseconds",
		Doc:    "1/1000000 seconds",
		Insert: "us",
		Kind:   protocol.CompletionItemKindUnit,
	},
	{
		Label:  "ms",
		Detail: "Milliseconds",
		Doc:    "1/1000 seconds",
		Insert: "ms",
		Kind:   protocol.CompletionItemKindUnit,
	},
	{
		Label:  "s",
		Detail: "Seconds",
		Doc:    "1 second",
		Insert: "s",
		Kind:   protocol.CompletionItemKindUnit,
	},
	{
		Label:  "m",
		Detail: "Minutes",
		Kind:   protocol.CompletionItemKindUnit,
	},
	{
		Label:  "h",
		Detail: "Hours",
		Doc:    "1 hour",
		Insert: "h",
		Kind:   protocol.CompletionItemKindUnit,
	},
	{
		Label:  "hz",
		Detail: "Hertz",
		Kind:   protocol.CompletionItemKindUnit,
	},
	{
		Label:  "khz",
		Detail: "Kilohertz",
		Doc:    "1000 hertz",
		Insert: "khz",
		Kind:   protocol.CompletionItemKindUnit,
	},
	{
		Label:  "mhz",
		Detail: "Megahertz",
		Doc:    "1000000 hertz",
		Insert: "mhz",
		Kind:   protocol.CompletionItemKindUnit,
	},
	{
		Label:  "ghz",
		Detail: "Gigahertz",
		Doc:    "1000000 hertz",
		Insert: "ghz",
		Kind:   protocol.CompletionItemKindUnit,
	},
	{
		Label:        "func",
		Detail:       "func declaration",
		Doc:          "Declares a function",
		Insert:       "func ${1:name}($2) $3 {\n\t$0\n}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
	},
	{
		Label:        "if",
		Detail:       "if statement",
		Doc:          "Conditional statement",
		Insert:       "if ${1:condition} {\n\t$0\n}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
	},
	{
		Label:        "else",
		Detail:       "else clause",
		Doc:          "Alternative branch",
		Insert:       "else {\n\t$0\n}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
	},
	{
		Label:        "else if",
		Detail:       "else-if clause",
		Doc:          "Alternative conditional branch",
		Insert:       "else if ${1:condition} {\n\t$0\n}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
	},
	{
		Label:        "return",
		Detail:       "return statement",
		Doc:          "Returns a value",
		Insert:       "return $0",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
	},
}

// Completion handles completion requests
func (s *Server) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	s.mu.RLock()
	doc, ok := s.documents[params.TextDocument.URI]
	s.mu.RUnlock()

	if !ok {
		return nil, nil
	}

	lines := strings.Split(doc.Content, "\n")
	if int(params.Position.Line) >= len(lines) {
		return &protocol.CompletionList{}, nil
	}

	line := lines[params.Position.Line]
	prefix := ""
	if int(params.Position.Character) <= len(line) {
		start := int(params.Position.Character)
		for start > 0 && isWordChar(line[start-1]) {
			start--
		}
		prefix = line[start:params.Position.Character]
	}

	items := s.getCompletionItems(ctx, doc, prefix, line, params.Position)

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// getCompletionItems generates completion items based on context
func (s *Server) getCompletionItems(ctx context.Context, doc *Document, prefix string, line string, pos protocol.Position) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(completions))

	// Add built-in completions (keywords, types, functions, units)
	for _, c := range completions {
		if !strings.HasPrefix(c.Label, prefix) {
			continue
		}

		item := protocol.CompletionItem{
			Label:         c.Label,
			Kind:          c.Kind,
			Detail:        c.Detail,
			Documentation: c.Doc,
		}

		if c.Insert != "" {
			item.InsertText = c.Insert
			item.InsertTextFormat = c.InsertFormat
		}

		items = append(items, item)
	}

	// Add symbols from the document's symbol table using ResolvePrefix
	if doc.IR.Symbols != nil {
		// Map position to wrapped coordinates if this is a block expression
		searchPos := pos
		if doc.Wrapper != nil {
			searchPos = doc.Wrapper.MapOriginalToWrapped(pos)
		}

		scopeAtCursor := s.findScopeAtPosition(doc.IR.Symbols, searchPos)
		if scopeAtCursor != nil {
			// Use ResolvePrefix to get all matching symbols from children, GlobalResolver, and parents
			scopes, err := scopeAtCursor.ResolvePrefix(ctx, prefix)
			if err == nil {
				for _, scope := range scopes {
					var kind protocol.CompletionItemKind
					var detail string

					typeStr := scope.Type.String()
					if typeStr != "" {
						if strings.Contains(typeStr, "->") {
							kind = protocol.CompletionItemKindFunction
						} else {
							kind = protocol.CompletionItemKindVariable
						}
						detail = typeStr
					} else {
						kind = protocol.CompletionItemKindVariable
					}

					items = append(items, protocol.CompletionItem{
						Label:  scope.Name,
						Kind:   kind,
						Detail: detail,
					})
				}
			}
		}
	}

	return items
}
