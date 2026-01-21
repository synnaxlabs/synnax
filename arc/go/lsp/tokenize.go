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
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
)

func TokenizeContent(content string) []antlr.Token {
	input := antlr.NewInputStream(content)
	lexer := parser.NewArcLexer(input)
	lexer.RemoveErrorListeners()
	return lexer.GetAllTokens()
}

func TokenizeContentWithComments(content string) []antlr.Token {
	input := antlr.NewInputStream(content)
	lexer := parser.NewArcLexer(input)
	lexer.RemoveErrorListeners()
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	stream.Fill()
	return stream.GetAllTokens()
}

func SplitLines(content string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			end := i
			if i > 0 && content[i-1] == '\r' {
				end--
			}
			lines = append(lines, content[start:end])
			start = i + 1
		}
	}
	if start <= len(content) {
		lines = append(lines, content[start:])
	}
	return lines
}
