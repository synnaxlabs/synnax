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

type Config struct {
	IndentWidth       int
	MaxLineLength     int
	MaxBlankLines     int
	TrailingCommas    bool
	AlignDeclarations bool
}

var DefaultConfig = Config{
	IndentWidth:       4,
	MaxLineLength:     88,
	MaxBlankLines:     2,
	TrailingCommas:    true,
	AlignDeclarations: true,
}

func Format(content string, cfg Config) string {
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

	p := newPrinter(cfg, content)
	return p.print(tokens)
}

func FormatRange(content string, startLine, endLine int, cfg Config) string {
	lines := strings.Split(content, "\n")
	if startLine < 0 || endLine >= len(lines) || startLine > endLine {
		return content
	}

	rangeContent := strings.Join(lines[startLine:endLine+1], "\n")
	formatted := Format(rangeContent, cfg)

	var result strings.Builder
	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}
		if i >= startLine && i <= endLine {
			if i == startLine {
				result.WriteString(formatted)
			}
		} else {
			result.WriteString(line)
		}
	}
	return result.String()
}
