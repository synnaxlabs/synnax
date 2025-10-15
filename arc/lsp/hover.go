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
	"fmt"
	"strings"

	"go.lsp.dev/protocol"
)

// Hover handles hover requests
func (s *Server) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	return nil, nil
}

// getWordAtPosition extracts the word at the given position
func (s *Server) getWordAtPosition(content string, pos protocol.Position) string {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return ""
	}

	line := lines[pos.Line]
	if int(pos.Character) >= len(line) {
		return ""
	}

	// FindChild word boundaries
	start := int(pos.Character)
	end := int(pos.Character)

	// Expand left
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	// Expand right
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	return line[start:end]
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// getHoverContents returns hover documentation for known keywords and types
func (s *Server) getHoverContents(word string) string {
	switch word {
	// Keywords
	case "func":
		return "## func\nDeclares a function.\n\n```arc\nfunc name(param type) returnType {\n    // body\n}\n```"
	case "stage":
		return "## stage\nDeclares a reactive stage.\n\n```arc\nfunc name{\n    config type\n} (runtime type) returnType {\n    // body\n}\n```"
	case "if":
		return "## if\nConditional statement.\n\n```arc\nif condition {\n    // body\n}\n```"
	case "else":
		return "## else\nAlternative branch for if statement.\n\n```arc\nif condition {\n    // body\n} else {\n    // alternative\n}\n```"
	case "return":
		return "## return\nReturns a value from a function or stage."

	// Types
	case "i8", "i16", "i32", "i64":
		bits := word[1:]
		return fmt.Sprintf("## %s\nSigned %s-bit integer.\n\nRange: -%d to %d", word, bits, 1<<(parseInt(bits)-1), (1<<(parseInt(bits)-1))-1)
	case "u8", "u16", "u32", "u64":
		bits := word[1:]
		return fmt.Sprintf("## %s\nUnsigned %s-bit integer.\n\nRange: 0 to %d", word, bits, (1<<parseInt(bits))-1)
	case "f32":
		return "## f32\n32-bit floating point number (single precision)."
	case "f64":
		return "## f64\n64-bit floating point number (double precision)."
	case "string":
		return "## string\nImmutable UTF-8 encoded string."
	case "timestamp":
		return "## timestamp\nPoint in time represented as nanoseconds since Unix epoch."
	case "timespan":
		return "## timespan\nDuration represented as nanoseconds."
	case "series":
		return "## series\nHomogeneous array of values.\n\n```arc\nseries f64\n```"
	case "chan":
		return "## chan\nBidirectional channel for communication.\n\n```arc\nchan f64\n```"

	// Built-in functions
	case "len":
		return "## len\nReturns the length of a series.\n\n```arc\nlength := len(data)\n```"
	case "now":
		return "## now\nReturns the current timestamp.\n\n```arc\ntime := now()\n```"

	default:
		return ""
	}
}

func parseInt(s string) int {
	switch s {
	case "8":
		return 8
	case "16":
		return 16
	case "32":
		return 32
	case "64":
		return 64
	default:
		return 0
	}
}
