// Copyright 2026 Synnax Labs, Inc.
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
		{
			Label:  "struct",
			Kind:   protocol.CompletionItemKindKeyword,
			Detail: protocol.NewOptional("Declare a data structure"),
		},
		{
			Label:  "enum",
			Kind:   protocol.CompletionItemKindKeyword,
			Detail: protocol.NewOptional("Declare an enumeration type"),
		},
		{
			Label:  "union",
			Kind:   protocol.CompletionItemKindKeyword,
			Detail: protocol.NewOptional("Declare a discriminated union"),
		},
		{
			Label:  "import",
			Kind:   protocol.CompletionItemKindKeyword,
			Detail: protocol.NewOptional("Import another schema file"),
		},
		{
			Label:  "extends",
			Kind:   protocol.CompletionItemKindKeyword,
			Detail: protocol.NewOptional("Inherit fields from another struct"),
		},
		{
			Label:  "map",
			Kind:   protocol.CompletionItemKindKeyword,
			Detail: protocol.NewOptional("Declare a map type: map<K, V>"),
		},
		{
			Label:  "action",
			Kind:   protocol.CompletionItemKindKeyword,
			Detail: protocol.NewOptional("Declare a wire mutation on a struct"),
		},
	}

	primitiveTypeCompletions = []protocol.CompletionItem{
		{
			Label:  "uuid",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("UUID type"),
		},
		{
			Label:  "string",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("String type"),
		},
		{
			Label:  "bool",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("Boolean type"),
		},
		{
			Label:  "int8",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("8-bit signed integer"),
		},
		{
			Label:  "int16",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("16-bit signed integer"),
		},
		{
			Label:  "int32",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("32-bit signed integer"),
		},
		{
			Label:  "int64",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("64-bit signed integer"),
		},
		{
			Label:  "uint8",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("8-bit unsigned integer"),
		},
		{
			Label:  "uint16",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("16-bit unsigned integer"),
		},
		{
			Label:  "uint32",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("32-bit unsigned integer"),
		},
		{
			Label:  "uint64",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("64-bit unsigned integer"),
		},
		{
			Label:  "float32",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("32-bit floating-point number"),
		},
		{
			Label:  "float64",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("64-bit floating-point number"),
		},
		{
			Label:  "record",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("Record type (untyped key-value map)"),
		},
		{
			Label:  "bytes",
			Kind:   protocol.CompletionItemKindClass,
			Detail: protocol.NewOptional("Byte array type"),
		},
	}

	domainNameCompletions = []protocol.CompletionItem{
		{
			Label:  "doc",
			Kind:   protocol.CompletionItemKindProperty,
			Detail: protocol.NewOptional("Documentation for the declaration"),
		},
		{
			Label:  "key",
			Kind:   protocol.CompletionItemKindProperty,
			Detail: protocol.NewOptional("Marks the field as the primary key"),
		},
		{
			Label:  "validate",
			Kind:   protocol.CompletionItemKindProperty,
			Detail: protocol.NewOptional("Validation constraints"),
		},
		{
			Label:  "filter",
			Kind:   protocol.CompletionItemKindProperty,
			Detail: protocol.NewOptional("Retrieve requests can filter on the field"),
		},
		{
			Label:  "ontology",
			Kind:   protocol.CompletionItemKindProperty,
			Detail: protocol.NewOptional("Ontology resource type"),
		},
		{
			Label:  "create",
			Kind:   protocol.CompletionItemKindProperty,
			Detail: protocol.NewOptional("Generate a create endpoint"),
		},
		{
			Label:  "retrieve",
			Kind:   protocol.CompletionItemKindProperty,
			Detail: protocol.NewOptional("Generate a retrieve endpoint"),
		},
		{
			Label:  "search",
			Kind:   protocol.CompletionItemKindProperty,
			Detail: protocol.NewOptional("Index the resource for search"),
		},
		{
			Label:  "index",
			Kind:   protocol.CompletionItemKindProperty,
			Detail: protocol.NewOptional("Index configuration for the field"),
		},
		{
			Label:  "go",
			Kind:   protocol.CompletionItemKindProperty,
			Detail: protocol.NewOptional("Go output configuration"),
		},
		{
			Label:  "ts",
			Kind:   protocol.CompletionItemKindProperty,
			Detail: protocol.NewOptional("TypeScript output configuration"),
		},
		{
			Label:  "py",
			Kind:   protocol.CompletionItemKindProperty,
			Detail: protocol.NewOptional("Python output configuration"),
		},
		{
			Label:  "cpp",
			Kind:   protocol.CompletionItemKindProperty,
			Detail: protocol.NewOptional("C++ output configuration"),
		},
		{
			Label:  "pb",
			Kind:   protocol.CompletionItemKindProperty,
			Detail: protocol.NewOptional("Protobuf output configuration"),
		},
	}

	validateExpressionCompletions = []protocol.CompletionItem{
		{
			Label:  "required",
			Kind:   protocol.CompletionItemKindValue,
			Detail: protocol.NewOptional("Field must be set"),
		},
		{
			Label:  "skip",
			Kind:   protocol.CompletionItemKindValue,
			Detail: protocol.NewOptional("Skip validation for this field"),
		},
		{
			Label:  "min_length",
			Kind:   protocol.CompletionItemKindValue,
			Detail: protocol.NewOptional("Minimum string length"),
		},
		{
			Label:  "max_length",
			Kind:   protocol.CompletionItemKindValue,
			Detail: protocol.NewOptional("Maximum string length"),
		},
		{
			Label:  "min",
			Kind:   protocol.CompletionItemKindValue,
			Detail: protocol.NewOptional("Minimum numeric value"),
		},
		{
			Label:  "max",
			Kind:   protocol.CompletionItemKindValue,
			Detail: protocol.NewOptional("Maximum numeric value"),
		},
		{
			Label:  "pattern",
			Kind:   protocol.CompletionItemKindValue,
			Detail: protocol.NewOptional("Regex pattern constraint"),
		},
	}

	outputExpressionCompletions = []protocol.CompletionItem{
		{
			Label:  "output",
			Kind:   protocol.CompletionItemKindValue,
			Detail: protocol.NewOptional("Output path for generated code"),
		},
		{
			Label:  "omit",
			Kind:   protocol.CompletionItemKindValue,
			Detail: protocol.NewOptional("Skip code generation for this declaration"),
		},
	}

	ontologyExpressionCompletions = []protocol.CompletionItem{
		{
			Label:  "type",
			Kind:   protocol.CompletionItemKindValue,
			Detail: protocol.NewOptional("Ontology type name"),
		},
	}

	tsExpressionCompletions = []protocol.CompletionItem{
		{
			Label:  "output",
			Kind:   protocol.CompletionItemKindValue,
			Detail: protocol.NewOptional("Output path for generated code"),
		},
		{
			Label:  "use_input",
			Kind:   protocol.CompletionItemKindValue,
			Detail: protocol.NewOptional("Use z.input instead of z.infer for type"),
		},
		{
			Label:  "name",
			Kind:   protocol.CompletionItemKindValue,
			Detail: protocol.NewOptional("Override generated type/schema name"),
		},
	}
)

// Completion handles completion requests.
func (s *Server) Completion(
	_ context.Context,
	params *protocol.CompletionParams,
) (protocol.CompletionResult, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return &protocol.CompletionList{Items: []protocol.CompletionItem{}}, nil
	}

	lines := strings.Split(doc.Content, "\n")
	if int(params.Position.Line) >= len(lines) {
		return &protocol.CompletionList{Items: []protocol.CompletionItem{}}, nil
	}
	line := lines[params.Position.Line]
	col := min(int(params.Position.Character), len(line))
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
	trimmed := strings.TrimLeft(linePrefix, " \t")
	if _, rest, ok := strings.CutLast(trimmed, "@"); ok {
		switch {
		case strings.HasPrefix(rest, "validate "):
			return validateExpressionCompletions
		case strings.HasPrefix(rest, "ts "):
			return tsExpressionCompletions
		case strings.HasPrefix(rest, "ontology "):
			return ontologyExpressionCompletions
		case strings.HasPrefix(rest, "go "),
			strings.HasPrefix(rest, "py "),
			strings.HasPrefix(rest, "cpp "),
			strings.HasPrefix(rest, "pb "):
			return outputExpressionCompletions
		default:
			return domainNameCompletions
		}
	}
	if strings.TrimSpace(trimmed) == "" {
		return keywordCompletions
	}
	all := make([]protocol.CompletionItem, 0)
	all = append(all, keywordCompletions...)
	all = append(all, primitiveTypeCompletions...)
	return all
}

func extractPrefix(linePrefix string) string {
	for i := len(linePrefix) - 1; i >= 0; i-- {
		ch := linePrefix[i]
		if ch == ' ' || ch == '\t' || ch == '{' || ch == '}' || ch == '[' ||
			ch == ']' || ch == '@' {
			return linePrefix[i+1:]
		}
	}
	return linePrefix
}
