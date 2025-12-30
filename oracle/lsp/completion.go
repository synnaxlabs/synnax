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

// Static completion items for Oracle schema language
var (
	keywordCompletions = []protocol.CompletionItem{
		{Label: "struct", Kind: protocol.CompletionItemKindKeyword, Detail: "Define a data structure"},
		{Label: "field", Kind: protocol.CompletionItemKindKeyword, Detail: "Define a field within a struct"},
		{Label: "domain", Kind: protocol.CompletionItemKindKeyword, Detail: "Define a domain block with rules"},
		{Label: "enum", Kind: protocol.CompletionItemKindKeyword, Detail: "Define an enumeration type"},
		{Label: "import", Kind: protocol.CompletionItemKindKeyword, Detail: "Import another schema file"},
	}

	primitiveTypeCompletions = []protocol.CompletionItem{
		{Label: "uuid", Kind: protocol.CompletionItemKindClass, Detail: "UUID type"},
		{Label: "string", Kind: protocol.CompletionItemKindClass, Detail: "String type"},
		{Label: "bool", Kind: protocol.CompletionItemKindClass, Detail: "Boolean type"},
		{Label: "int8", Kind: protocol.CompletionItemKindClass, Detail: "8-bit signed integer"},
		{Label: "int16", Kind: protocol.CompletionItemKindClass, Detail: "16-bit signed integer"},
		{Label: "int32", Kind: protocol.CompletionItemKindClass, Detail: "32-bit signed integer"},
		{Label: "int64", Kind: protocol.CompletionItemKindClass, Detail: "64-bit signed integer"},
		{Label: "uint8", Kind: protocol.CompletionItemKindClass, Detail: "8-bit unsigned integer"},
		{Label: "uint16", Kind: protocol.CompletionItemKindClass, Detail: "16-bit unsigned integer"},
		{Label: "uint32", Kind: protocol.CompletionItemKindClass, Detail: "32-bit unsigned integer"},
		{Label: "uint64", Kind: protocol.CompletionItemKindClass, Detail: "64-bit unsigned integer"},
		{Label: "float32", Kind: protocol.CompletionItemKindClass, Detail: "32-bit floating point"},
		{Label: "float64", Kind: protocol.CompletionItemKindClass, Detail: "64-bit floating point"},
		{Label: "timestamp", Kind: protocol.CompletionItemKindClass, Detail: "Timestamp type"},
		{Label: "timespan", Kind: protocol.CompletionItemKindClass, Detail: "Duration/timespan type"},
		{Label: "time_range", Kind: protocol.CompletionItemKindClass, Detail: "Time range type (start, end)"},
		{Label: "json", Kind: protocol.CompletionItemKindClass, Detail: "JSON type"},
		{Label: "bytes", Kind: protocol.CompletionItemKindClass, Detail: "Byte array type"},
	}

	domainNameCompletions = []protocol.CompletionItem{
		{Label: "id", Kind: protocol.CompletionItemKindProperty, Detail: "Marks field as primary key"},
		{Label: "validate", Kind: protocol.CompletionItemKindProperty, Detail: "Validation constraints"},
		{Label: "query", Kind: protocol.CompletionItemKindProperty, Detail: "Query operators for this field"},
		{Label: "index", Kind: protocol.CompletionItemKindProperty, Detail: "Index configuration"},
		{Label: "relation", Kind: protocol.CompletionItemKindProperty, Detail: "Relationship to other structs"},
		{Label: "sort", Kind: protocol.CompletionItemKindProperty, Detail: "Sortable field configuration"},
		{Label: "go", Kind: protocol.CompletionItemKindProperty, Detail: "Go output configuration"},
		{Label: "ts", Kind: protocol.CompletionItemKindProperty, Detail: "TypeScript output configuration"},
		{Label: "python", Kind: protocol.CompletionItemKindProperty, Detail: "Python output configuration"},
		{Label: "zod", Kind: protocol.CompletionItemKindProperty, Detail: "Zod schema output configuration"},
	}

	validateExpressionCompletions = []protocol.CompletionItem{
		{Label: "required", Kind: protocol.CompletionItemKindValue, Detail: "Field is required"},
		{Label: "min_length", Kind: protocol.CompletionItemKindValue, Detail: "Minimum string length"},
		{Label: "max_length", Kind: protocol.CompletionItemKindValue, Detail: "Maximum string length"},
		{Label: "pattern", Kind: protocol.CompletionItemKindValue, Detail: "Regex pattern constraint"},
		{Label: "min", Kind: protocol.CompletionItemKindValue, Detail: "Minimum numeric value"},
		{Label: "max", Kind: protocol.CompletionItemKindValue, Detail: "Maximum numeric value"},
		{Label: "email", Kind: protocol.CompletionItemKindValue, Detail: "Email format validation"},
		{Label: "url", Kind: protocol.CompletionItemKindValue, Detail: "URL format validation"},
		{Label: "default", Kind: protocol.CompletionItemKindValue, Detail: "Default value"},
		{Label: "immutable", Kind: protocol.CompletionItemKindValue, Detail: "Field cannot be modified after creation"},
	}

	queryExpressionCompletions = []protocol.CompletionItem{
		{Label: "eq", Kind: protocol.CompletionItemKindValue, Detail: "Equals operator"},
		{Label: "neq", Kind: protocol.CompletionItemKindValue, Detail: "Not equals operator"},
		{Label: "contains", Kind: protocol.CompletionItemKindValue, Detail: "String contains operator"},
		{Label: "starts_with", Kind: protocol.CompletionItemKindValue, Detail: "String starts with operator"},
		{Label: "ends_with", Kind: protocol.CompletionItemKindValue, Detail: "String ends with operator"},
		{Label: "has_any", Kind: protocol.CompletionItemKindValue, Detail: "Array has any of values"},
		{Label: "has_all", Kind: protocol.CompletionItemKindValue, Detail: "Array has all values"},
		{Label: "overlaps", Kind: protocol.CompletionItemKindValue, Detail: "Ranges overlap"},
		{Label: "between", Kind: protocol.CompletionItemKindValue, Detail: "Value is between range"},
		{Label: "lt", Kind: protocol.CompletionItemKindValue, Detail: "Less than"},
		{Label: "lte", Kind: protocol.CompletionItemKindValue, Detail: "Less than or equal"},
		{Label: "gt", Kind: protocol.CompletionItemKindValue, Detail: "Greater than"},
		{Label: "gte", Kind: protocol.CompletionItemKindValue, Detail: "Greater than or equal"},
	}

	indexExpressionCompletions = []protocol.CompletionItem{
		{Label: "lookup", Kind: protocol.CompletionItemKindValue, Detail: "Lookup index for exact match queries"},
		{Label: "sorted", Kind: protocol.CompletionItemKindValue, Detail: "Sorted index for range queries"},
		{Label: "range", Kind: protocol.CompletionItemKindValue, Detail: "Range index"},
		{Label: "composite", Kind: protocol.CompletionItemKindValue, Detail: "Composite index with other fields"},
	}

	outputExpressionCompletions = []protocol.CompletionItem{
		{Label: "output", Kind: protocol.CompletionItemKindValue, Detail: "Output path for generated code"},
	}
)

// Completion handles completion requests.
func (s *Server) Completion(_ context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return &protocol.CompletionList{Items: []protocol.CompletionItem{}}, nil
	}

	lines := strings.Split(doc.Content, "\n")
	if int(params.Position.Line) >= len(lines) {
		return &protocol.CompletionList{Items: []protocol.CompletionItem{}}, nil
	}
	line := lines[params.Position.Line]
	col := int(params.Position.Character)
	if col > len(line) {
		col = len(line)
	}
	linePrefix := line[:col]

	items := getCompletionsForContext(linePrefix)

	prefix := extractPrefix(linePrefix)
	if prefix != "" {
		filtered := make([]protocol.CompletionItem, 0)
		for _, item := range items {
			if strings.HasPrefix(strings.ToLower(item.Label), strings.ToLower(prefix)) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}

func getCompletionsForContext(linePrefix string) []protocol.CompletionItem {
	trimmed := strings.TrimSpace(linePrefix)

	if strings.Contains(linePrefix, "domain validate") || isInsideDomain(linePrefix, "validate") {
		return validateExpressionCompletions
	}
	if strings.Contains(linePrefix, "domain query") || isInsideDomain(linePrefix, "query") {
		return queryExpressionCompletions
	}
	if strings.Contains(linePrefix, "domain index") || isInsideDomain(linePrefix, "index") {
		return indexExpressionCompletions
	}
	if strings.Contains(linePrefix, "domain go") || strings.Contains(linePrefix, "domain ts") ||
		strings.Contains(linePrefix, "domain python") || strings.Contains(linePrefix, "domain zod") ||
		isInsideDomain(linePrefix, "go") || isInsideDomain(linePrefix, "ts") ||
		isInsideDomain(linePrefix, "python") || isInsideDomain(linePrefix, "zod") {
		return outputExpressionCompletions
	}

	if strings.HasSuffix(trimmed, "domain") {
		return domainNameCompletions
	}

	if strings.Contains(trimmed, "field ") && !strings.Contains(trimmed, "{") {
		parts := strings.Fields(trimmed)
		if len(parts) >= 2 && parts[0] == "field" {
			return primitiveTypeCompletions
		}
	}

	if trimmed == "" || !strings.Contains(trimmed, "{") {
		return keywordCompletions
	}

	all := make([]protocol.CompletionItem, 0)
	all = append(all, keywordCompletions...)
	all = append(all, primitiveTypeCompletions...)
	all = append(all, domainNameCompletions...)
	return all
}

func isInsideDomain(linePrefix, domainName string) bool {
	return strings.Contains(linePrefix, domainName) && strings.Contains(linePrefix, "{")
}

func extractPrefix(linePrefix string) string {
	for i := len(linePrefix) - 1; i >= 0; i-- {
		ch := linePrefix[i]
		if ch == ' ' || ch == '\t' || ch == '{' || ch == '}' || ch == '[' || ch == ']' {
			return linePrefix[i+1:]
		}
	}
	return linePrefix
}
