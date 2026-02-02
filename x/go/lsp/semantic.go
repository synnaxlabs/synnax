// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp

// Token represents a semantic token for LSP semantic highlighting.
type Token struct {
	Line      uint32
	StartChar uint32
	Length    uint32
	TokenType uint32
}

// EncodeSemanticTokens encodes a slice of tokens into the delta-encoded format
// required by the LSP semantic tokens specification.
// The format is: [deltaLine, deltaStartChar, length, tokenType, tokenModifiers]
func EncodeSemanticTokens(tokens []Token) []uint32 {
	if len(tokens) == 0 {
		return []uint32{}
	}
	encoded := make([]uint32, 0, len(tokens)*5)
	prevLine := uint32(0)
	prevChar := uint32(0)
	for _, t := range tokens {
		deltaLine := t.Line - prevLine
		var deltaChar uint32
		if deltaLine == 0 {
			deltaChar = t.StartChar - prevChar
		} else {
			deltaChar = t.StartChar
		}
		// 5 values per token: deltaLine, deltaStartChar, length, tokenType, tokenModifiers
		encoded = append(encoded, deltaLine, deltaChar, t.Length, t.TokenType, 0)
		prevLine = t.Line
		prevChar = t.StartChar
	}
	return encoded
}
