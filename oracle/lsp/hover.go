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

var hoverDocs = map[string]string{
	"struct": "**struct**\n\nDeclares a data structure.\n\n```oracle\n" +
		"User struct {\n    key  uuid   @key\n    name string\n}\n```",
	"enum": "**enum**\n\nDeclares an enumeration type with integer or string " +
		"values.\n\n```oracle\nStatus enum {\n    active   = 1\n    " +
		"inactive = 2\n}\n```",
	"union": "**union**\n\nDeclares a discriminated union: a closed set of " +
		"struct variants distinguished by a shared discriminator field.",
	"import": "**import**\n\nImports another schema file.\n\n```oracle\n" +
		"import \"schemas/synnax/label\"\n```",
	"extends": "**extends**\n\nInherits the fields of another struct.",
	"action": "**action**\n\nDeclares a wire mutation on a struct. Actions are " +
		"never persisted.",

	"uuid":    "**uuid**\n\nUniversally unique identifier (128-bit).",
	"string":  "**string**\n\nUTF-8 encoded text.",
	"bool":    "**bool**\n\nBoolean value: `1` or `0`.",
	"int8":    "**int8**\n\n8-bit signed integer. Range: -128 to 127.",
	"int16":   "**int16**\n\n16-bit signed integer. Range: -32,768 to 32,767.",
	"int32":   "**int32**\n\n32-bit signed integer. Range: -2^31 to 2^31-1.",
	"int64":   "**int64**\n\n64-bit signed integer. Range: -2^63 to 2^63-1.",
	"uint8":   "**uint8**\n\n8-bit unsigned integer. Range: 0 to 255.",
	"uint12":  "**uint12**\n\n12-bit unsigned integer carried in a uint16.",
	"uint16":  "**uint16**\n\n16-bit unsigned integer. Range: 0 to 65,535.",
	"uint20":  "**uint20**\n\n20-bit unsigned integer carried in a uint32.",
	"uint32":  "**uint32**\n\n32-bit unsigned integer. Range: 0 to 2^32-1.",
	"uint64":  "**uint64**\n\n64-bit unsigned integer. Range: 0 to 2^64-1.",
	"float32": "**float32**\n\n32-bit IEEE 754 floating-point number.",
	"float64": "**float64**\n\n64-bit IEEE 754 floating-point number.",
	"record":  "**record**\n\nUntyped key-value record (map of string to any).",
	"bytes":   "**bytes**\n\nByte array / binary data.",

	"key": "**@key**\n\nMarks the field as the struct's primary key.",
	"doc": "**@doc**\n\nDocumentation for the declaration.\n\nUse " +
		"`@doc value \"...\"` on fields and declarations.",
	"validate": "**@validate**\n\nValidation constraints.\n\nExpressions: " +
		"`required`, `skip`, `min_length`, `max_length`, `min`, `max`, `pattern`",
	"filter": "**@filter**\n\nRetrieve requests can filter on this field.",
	"index": "**@index**\n\nIndex configuration for the field.\n\n" +
		"Expressions: `lookup`, `sorted`",
	"ontology": "**@ontology**\n\nOntology resource type.\n\nUse " +
		"`@ontology type \"name\"`.",
	"go": "**@go**\n\nGo output configuration.\n\nUse `output \"path\"` " +
		"to set the output directory; `marshal` to generate a codec; " +
		"`migrate` to generate migrations.",
	"ts": "**@ts**\n\nTypeScript output configuration.\n\nExpressions:\n- " +
		"`output \"path\"` - Output directory\n- `use_input` - Use `z.input` " +
		"instead of `z.infer`\n- `name \"TypeName\"` - Override the " +
		"generated name",
	"py":  "**@py**\n\nPython output configuration.",
	"cpp": "**@cpp**\n\nC++ output configuration.",
	"pb":  "**@pb**\n\nProtobuf output configuration.",

	"required":   "**required**\n\nField must be set.",
	"skip":       "**skip**\n\nSkip validation for this field.",
	"min_length": "**min_length** *value*\n\nMinimum string length.",
	"max_length": "**max_length** *value*\n\nMaximum string length.",
	"pattern":    "**pattern** *\"regex\"*\n\nRegex pattern constraint.",
	"use_input": "**use_input**\n\nUse `z.input` instead of `z.infer` for the " +
		"generated TypeScript type.",
	"name":   "**name** *\"TypeName\"*\n\nOverride the generated type name.",
	"lookup": "**lookup**\n\nLookup index for exact-match queries.",
	"sorted": "**sorted**\n\nSorted index for range queries.",
	"output": "**output** *\"path\"*\n\nOutput path for generated code.",
}

func (s *Server) Hover(
	_ context.Context,
	params *protocol.HoverParams,
) (*protocol.Hover, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	word := getWordAtPosition(doc.Content, params.Position)
	if word == "" {
		return nil, nil
	}

	if docStr, ok := hoverDocs[word]; ok {
		return &protocol.Hover{
			Contents: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: docStr,
			},
		}, nil
	}

	return nil, nil
}

func getWordAtPosition(content string, pos protocol.Position) string {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	col := min(int(pos.Character), len(line))

	start := col
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}
	end := col
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	if start == end {
		return ""
	}
	return line[start:end]
}

func isWordChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_'
}
