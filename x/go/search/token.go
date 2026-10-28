// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package search

import (
	"strings"
	"unicode"
)

// maxTokenRunes is the number of runes that tokenize keeps of each token, so the trie
// holds a bounded number of nodes for each token.
const maxTokenRunes = 255

// tokenize splits s into lowercase tokens at whitespace, '_', '-', and '.', and cuts
// each token to its first maxTokenRunes runes.
func tokenize(s string) []string {
	tokens := strings.FieldsFunc(strings.ToLower(s), isSeparator)
	for i, tok := range tokens {
		tokens[i] = truncate(tok)
	}
	return tokens
}

// firstWords returns the start of s that holds its first n words.
func firstWords(s string, n int) string {
	words, inWord := 0, false
	for i, r := range s {
		if isSeparator(r) {
			inWord = false
			continue
		}
		if !inWord {
			if words == n {
				return s[:i]
			}
			words, inWord = words+1, true
		}
	}
	return s
}

func truncate(token string) string {
	if len(token) <= maxTokenRunes {
		return token
	}
	runes := 0
	for i := range token {
		if runes == maxTokenRunes {
			return token[:i]
		}
		runes++
	}
	return token
}

func isSeparator(r rune) bool {
	return unicode.IsSpace(r) || r == '_' || r == '-' || r == '.'
}
