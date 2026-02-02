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
	"struct": `**struct**

Defines a data structure in the Oracle schema.

` + "```oracle" + `
struct User {
    field key uuid { domain id }
    field name string
}
` + "```",

	"field": `**field**

Defines a field within a struct. Fields have a name and type, and optionally domain blocks.

` + "```oracle" + `
field name string {
    domain validate { required }
}
` + "```",

	"domain": `**domain**

Defines a domain block containing rules or configuration for a field or struct.

Common domains:
- ` + "`id`" + ` - Marks field as primary key
- ` + "`validate`" + ` - Validation constraints
- ` + "`query`" + ` - Query operators
- ` + "`index`" + ` - Index configuration
- ` + "`go`" + `, ` + "`ts`" + `, ` + "`python`" + ` - Output configuration`,

	"enum": `**enum**

Defines an enumeration type with integer or string values.

` + "```oracle" + `
enum Status {
    Active = 1
    Inactive = 2
    Pending = 3
}
` + "```",

	"import": `**import**

Imports another schema file.

` + "```oracle" + `
import "schema/core/label"
` + "```",

	"uuid":       "**uuid**\n\nUniversally unique identifier (128-bit).",
	"string":     "**string**\n\nUTF-8 encoded text.",
	"bool":       "**bool**\n\nBoolean value: `true` or `false`.",
	"int8":       "**int8**\n\n8-bit signed integer. Range: -128 to 127.",
	"int16":      "**int16**\n\n16-bit signed integer. Range: -32,768 to 32,767.",
	"int32":      "**int32**\n\n32-bit signed integer. Range: -2^31 to 2^31-1.",
	"int64":      "**int64**\n\n64-bit signed integer. Range: -2^63 to 2^63-1.",
	"uint8":      "**uint8**\n\n8-bit unsigned integer. Range: 0 to 255.",
	"uint16":     "**uint16**\n\n16-bit unsigned integer. Range: 0 to 65,535.",
	"uint32":     "**uint32**\n\n32-bit unsigned integer. Range: 0 to 2^32-1.",
	"uint64":     "**uint64**\n\n64-bit unsigned integer. Range: 0 to 2^64-1.",
	"float32":    "**float32**\n\n32-bit IEEE 754 floating-point number.",
	"float64":    "**float64**\n\n64-bit IEEE 754 floating-point number.",
	"timestamp":  "**timestamp**\n\nPoint in time with nanosecond precision.",
	"timespan":   "**timespan**\n\nDuration or elapsed time.",
	"time_range": "**time_range**\n\nTime range with start and end timestamps.",
	"json":       "**json**\n\nArbitrary JSON data.",
	"bytes":      "**bytes**\n\nByte array / binary data.",

	"id":        "**domain id**\n\nMarks this field as the primary key for the struct.",
	"validate":  "**domain validate**\n\nValidation constraints for the field.\n\nExpressions: `required`, `min_length`, `max_length`, `pattern`, `min`, `max`, `email`, `url`, `default`, `immutable`",
	"query":     "**domain query**\n\nQuery operators available for this field.\n\nExpressions: `eq`, `neq`, `contains`, `starts_with`, `ends_with`, `has_any`, `has_all`, `overlaps`, `between`, `lt`, `lte`, `gt`, `gte`",
	"index":     "**domain index**\n\nIndex configuration for the field.\n\nExpressions: `lookup`, `sorted`, `range`, `composite`",
	"relation":  "**domain relation**\n\nDefines a relationship to another struct.",
	"sort":      "**domain sort**\n\nMarks this field as sortable in queries.",
	"go":        "**domain go**\n\nGo output configuration.\n\nUse `output \"path\"` to specify the output directory.",
	"ts":        "**domain ts**\n\nTypeScript output configuration.\n\nExpressions:\n- `output \"path\"` - Output directory\n- `use_input` - Use `z.input` instead of `z.infer` for type derivation\n- `name \"TypeName\"` - Override the generated TypeScript type/schema name",
	"use_input": "**use_input**\n\nUse `z.input` instead of `z.infer` for the generated TypeScript type.\n\nUseful for schemas with transforms where you want the input type.",
	"name": "**name** *\"TypeName\"*\n\nOverride the generated TypeScript type and schema name.\n\n" + "```oracle" + `
struct New {
    domain ts {
        name "WorkspaceNew"
    }
}
` + "```",
	"python": "**domain python**\n\nPython output configuration.\n\nUse `output \"path\"` to specify the output directory.",
	"zod":    "**domain zod**\n\nZod schema output configuration.\n\nUse `output \"path\"` to specify the output directory.",

	"required":   "**required**\n\nField must have a non-null value.",
	"min_length": "**min_length** *value*\n\nMinimum string length.",
	"max_length": "**max_length** *value*\n\nMaximum string length.",
	"pattern":    "**pattern** *\"regex\"*\n\nRegex pattern constraint.",
	"default":    "**default** *value*\n\nDefault value if not provided.",
	"immutable":  "**immutable**\n\nField cannot be modified after creation.",
	"eq":         "**eq**\n\nEquals operator for queries.",
	"neq":        "**neq**\n\nNot equals operator for queries.",
	"contains":   "**contains**\n\nString contains operator.",
	"lookup":     "**lookup**\n\nLookup index for exact match queries.",
	"sorted":     "**sorted**\n\nSorted index for range queries.",
	"output":     "**output** *\"path\"*\n\nOutput path for generated code.",
}

func (s *Server) Hover(_ context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
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
			Contents: protocol.MarkupContent{Kind: protocol.Markdown, Value: docStr},
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
	col := int(pos.Character)
	if col > len(line) {
		col = len(line)
	}

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
