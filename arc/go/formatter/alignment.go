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
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
)

type declarationGroup struct {
	startIdx     int
	endIdx       int
	maxNameLen   int
	maxTypeLen   int
	declarations []declaration
}

type declaration struct {
	nameToken antlr.Token
	typeToken antlr.Token
	opToken   antlr.Token
}

func findDeclarationGroups(tokens []antlr.Token) []declarationGroup {
	var groups []declarationGroup
	var current *declarationGroup

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.GetTokenType() != parser.ArcLexerIDENTIFIER {
			continue
		}

		if i+2 >= len(tokens) {
			continue
		}

		nextTok := tokens[i+1]
		nextType := nextTok.GetTokenType()

		if nextType == parser.ArcLexerDECLARE || nextType == parser.ArcLexerSTATE_DECLARE {
			decl := declaration{
				nameToken: tok,
				opToken:   nextTok,
			}
			if current == nil {
				current = &declarationGroup{startIdx: i}
			}
			current.declarations = append(current.declarations, decl)
			nameLen := len(tok.GetText())
			if nameLen > current.maxNameLen {
				current.maxNameLen = nameLen
			}
			current.endIdx = i + 2
		} else if isTypeToken(nextType) && i+3 < len(tokens) {
			opTok := tokens[i+2]
			opType := opTok.GetTokenType()
			if opType == parser.ArcLexerDECLARE || opType == parser.ArcLexerSTATE_DECLARE {
				decl := declaration{
					nameToken: tok,
					typeToken: nextTok,
					opToken:   opTok,
				}
				if current == nil {
					current = &declarationGroup{startIdx: i}
				}
				current.declarations = append(current.declarations, decl)
				nameLen := len(tok.GetText())
				typeLen := len(nextTok.GetText())
				if nameLen > current.maxNameLen {
					current.maxNameLen = nameLen
				}
				if typeLen > current.maxTypeLen {
					current.maxTypeLen = typeLen
				}
				current.endIdx = i + 3
			}
		}

		if shouldBreakGroup(tokens, i) && current != nil {
			groups = append(groups, *current)
			current = nil
		}
	}

	if current != nil && len(current.declarations) > 0 {
		groups = append(groups, *current)
	}

	return groups
}

func isTypeToken(tokType int) bool {
	switch tokType {
	case parser.ArcLexerI8, parser.ArcLexerI16, parser.ArcLexerI32, parser.ArcLexerI64,
		parser.ArcLexerU8, parser.ArcLexerU16, parser.ArcLexerU32, parser.ArcLexerU64,
		parser.ArcLexerF32, parser.ArcLexerF64, parser.ArcLexerSTR,
		parser.ArcLexerSERIES, parser.ArcLexerCHAN:
		return true
	}
	return false
}

func shouldBreakGroup(tokens []antlr.Token, idx int) bool {
	if idx+1 >= len(tokens) {
		return true
	}
	currentLine := tokens[idx].GetLine()
	nextLine := tokens[idx+1].GetLine()
	return nextLine > currentLine+1
}
