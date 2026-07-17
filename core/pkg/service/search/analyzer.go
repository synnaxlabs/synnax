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
	"slices"
	"unicode"

	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/registry"
	"github.com/samber/lo"
)

func init() {
	lo.Must0(registry.RegisterTokenizer(separatorTokenizer, func(map[string]any, *registry.Cache) (analysis.Tokenizer, error) {
		return &SepTokenizer{}, nil
	}))
}

const (
	separatorTokenizer = "separator"
	separatorAnalyzer  = "separator"
)

var validSeparators = []byte{'_', '-', '.'}

// SepTokenizer splits tokens by common separators.
type SepTokenizer struct{}

// Tokenize splits the input text by common separators.
func (t *SepTokenizer) Tokenize(input []byte) analysis.TokenStream {
	var (
		start  int
		tokens analysis.TokenStream
	)
	for i, r := range input {
		if unicode.IsSpace(rune(r)) || slices.Contains(validSeparators, r) {
			if start != i {
				tokens = append(tokens, &analysis.Token{
					Term:     input[start:i],
					Start:    start,
					End:      i,
					Position: start + 1,
				})
			}
			start = i + 1
		}
	}
	if start != len(input) {
		tokens = append(tokens, &analysis.Token{
			Term:     input[start:],
			Start:    start,
			End:      len(input),
			Position: start + 1,
			KeyWord:  true,
		})
	}
	return tokens
}

func registerSeparatorAnalyzer(mapping *mapping.IndexMappingImpl) error {
	return mapping.AddCustomAnalyzer(separatorAnalyzer, map[string]any{
		"type":          custom.Name,
		"tokenizer":     separatorTokenizer,
		"token_filters": []string{"to_lower"},
	})
}
