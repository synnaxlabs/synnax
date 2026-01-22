// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package formatter

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
)

const (
	indentWidth   = 4
	maxBlankLines = 2
)

func SplitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.Split(content, "\n")
}

func Format(content string) string {
	if content == "" {
		return ""
	}

	input := antlr.NewInputStream(content)
	lexer := parser.NewArcLexer(input)
	tokens := lexer.GetAllTokens()

	hasVisibleTokens := false
	for _, tok := range tokens {
		tokType := tok.GetTokenType()
		if tokType != parser.ArcLexerWS && tokType != antlr.TokenEOF {
			hasVisibleTokens = true
			break
		}
	}
	if !hasVisibleTokens {
		return content
	}

	p := newPrinter()
	return p.print(tokens)
}

func FormatRange(content string, startLine, endLine int) string {
	lines := SplitLines(content)
	if startLine < 0 || endLine >= len(lines) || startLine > endLine {
		return content
	}
	rangeContent := strings.Join(lines[startLine:endLine+1], "\n")
	formatted := Format(rangeContent)
	formatted = strings.TrimSuffix(formatted, "\n")
	var result strings.Builder
	for i := 0; i < startLine; i++ {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(lines[i])
	}
	if startLine > 0 {
		result.WriteString("\n")
	}
	result.WriteString(formatted)
	for i := endLine + 1; i < len(lines); i++ {
		result.WriteString("\n")
		result.WriteString(lines[i])
	}
	return result.String()
}
