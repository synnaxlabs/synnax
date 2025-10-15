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

// Completion handles completion requests
func (s *Server) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	s.mu.RLock()
	doc, ok := s.documents[params.TextDocument.URI]
	s.mu.RUnlock()

	if !ok {
		return nil, nil
	}

	// Resolve the current line and prefix
	lines := strings.Split(doc.Content, "\n")
	if int(params.Position.Line) >= len(lines) {
		return &protocol.CompletionList{}, nil
	}

	line := lines[params.Position.Line]
	prefix := ""
	if int(params.Position.Character) <= len(line) {
		// FindChild the start of the current word
		start := int(params.Position.Character)
		for start > 0 && isWordChar(line[start-1]) {
			start--
		}
		prefix = line[start:params.Position.Character]
	}

	// Generate completions based on context
	items := s.getCompletionItems(prefix, line, params.Position)

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// getCompletionItems generates completion items based on context
func (s *Server) getCompletionItems(prefix string, line string, pos protocol.Position) []protocol.CompletionItem {
	items := []protocol.CompletionItem{}

	// Check if we're at the start of a line or after whitespace
	beforeCursor := line[:pos.Character]
	trimmed := strings.TrimSpace(beforeCursor)

	// Keywords
	keywords := []struct {
		label  string
		detail string
		doc    string
		insert string
	}{
		{"func", "Function declaration", "Declares a function", "func ${1:name}($2) $3 {\n\t$0\n}"},
		{"stage", "func declaration", "Declares a reactive stage", "func ${1:name}{\n\t$2\n} ($3) $4 {\n\t$0\n}"},
		{"if", "If statement", "Conditional statement", "if ${1:condition} {\n\t$0\n}"},
		{"else", "Else clause", "Alternative branch", "else {\n\t$0\n}"},
		{"else if", "Else-if clause", "Alternative conditional branch", "else if ${1:condition} {\n\t$0\n}"},
		{"return", "Return statement", "Returns a value", "return $0"},
	}

	for _, kw := range keywords {
		if strings.HasPrefix(kw.label, prefix) {
			item := protocol.CompletionItem{
				Label:            kw.label,
				Kind:             protocol.CompletionItemKindKeyword,
				Detail:           kw.detail,
				Documentation:    kw.doc,
				InsertText:       kw.insert,
				InsertTextFormat: protocol.InsertTextFormatSnippet,
			}
			items = append(items, item)
		}
	}

	// Types
	types := []struct {
		label  string
		detail string
		doc    string
	}{
		{"i8", "Signed 8-bit integer", "Range: -128 to 127"},
		{"i16", "Signed 16-bit integer", "Range: -32768 to 32767"},
		{"i32", "Signed 32-bit integer", "Range: -2147483648 to 2147483647"},
		{"i64", "Signed 64-bit integer", "Range: -9223372036854775808 to 9223372036854775807"},
		{"u8", "Unsigned 8-bit integer", "Range: 0 to 255"},
		{"u16", "Unsigned 16-bit integer", "Range: 0 to 65535"},
		{"u32", "Unsigned 32-bit integer", "Range: 0 to 4294967295"},
		{"u64", "Unsigned 64-bit integer", "Range: 0 to 18446744073709551615"},
		{"f32", "32-bit float", "Single precision floating point"},
		{"f64", "64-bit float", "Double precision floating point"},
		{"string", "String type", "Immutable UTF-8 string"},
		{"timestamp", "Timestamp type", "Nanoseconds since Unix epoch"},
		{"timespan", "Timespan type", "Duration in nanoseconds"},
		{"series", "Series type", "Homogeneous array of values"},
		{"chan", "Channel type", "Communication channel"},
	}

	// Check if we're in a type position (after identifier or colon)
	inTypePosition := strings.HasSuffix(trimmed, ":") ||
		strings.Contains(trimmed, " ") && !strings.Contains(trimmed, "=")

	if inTypePosition || trimmed == "" {
		for _, t := range types {
			if strings.HasPrefix(t.label, prefix) {
				item := protocol.CompletionItem{
					Label:         t.label,
					Kind:          protocol.CompletionItemKindClass,
					Detail:        t.detail,
					Documentation: t.doc,
				}
				items = append(items, item)
			}
		}
	}

	// Built-in functions
	builtins := []struct {
		label  string
		detail string
		doc    string
		insert string
	}{
		{"len", "len(series) i64", "Returns the length of a series", "len($0)"},
		{"now", "now() timestamp", "Returns the current timestamp", "now()"},
	}

	for _, b := range builtins {
		if strings.HasPrefix(b.label, prefix) {
			item := protocol.CompletionItem{
				Label:            b.label,
				Kind:             protocol.CompletionItemKindFunction,
				Detail:           b.detail,
				Documentation:    b.doc,
				InsertText:       b.insert,
				InsertTextFormat: protocol.InsertTextFormatSnippet,
			}
			items = append(items, item)
		}
	}

	// Channel operations
	if strings.Contains(line, "->") || strings.Contains(line, "<-") {
		// Suggest channel-related completions
		if strings.HasPrefix("chan", prefix) {
			items = append(items, protocol.CompletionItem{
				Label:  "chan",
				Kind:   protocol.CompletionItemKindClass,
				Detail: "Bidirectional channel",
			})
		}
		if strings.HasPrefix("<-chan", prefix) {
			items = append(items, protocol.CompletionItem{
				Label:  "<-chan",
				Kind:   protocol.CompletionItemKindClass,
				Detail: "Read-only channel",
			})
		}
		if strings.HasPrefix("->chan", prefix) {
			items = append(items, protocol.CompletionItem{
				Label:  "->chan",
				Kind:   protocol.CompletionItemKindClass,
				Detail: "Write-only channel",
			})
		}
	}

	// Temporal units
	temporalUnits := []string{"ns", "us", "ms", "s", "m", "h", "hz", "khz", "mhz"}
	for _, unit := range temporalUnits {
		if strings.HasPrefix(unit, prefix) {
			item := protocol.CompletionItem{
				Label:  unit,
				Kind:   protocol.CompletionItemKindUnit,
				Detail: "Temporal/frequency unit",
			}
			items = append(items, item)
		}
	}

	return items
}

// Definition handles go-to-definition requests
func (s *Server) Definition(ctx context.Context, params *protocol.DefinitionParams) ([]protocol.Location, error) {
	// TODO: Implement go-to-definition
	// This would require building a symbol table from the parsed AST
	return nil, nil
}

// DocumentSymbol handles document symbol requests
func (s *Server) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) ([]interface{}, error) {
	// TODO: Implement document symbols
	// This would extract functions, tasks, and variables from the parsed AST
	return []interface{}{}, nil
}
