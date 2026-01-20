// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"
	"sort"
)

// LevenshteinDistance calculates the minimum number of single-character edits
// (insertions, deletions, or substitutions) required to change one string into another.
func LevenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create two rows for the DP table (only need current and previous)
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	// Initialize the first row
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			// Minimum of deletion, insertion, or substitution
			curr[j] = min(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		// Swap rows
		prev, curr = curr, prev
	}

	return prev[len(b)]
}

// suggestion holds a symbol name and its edit distance for sorting
type suggestion struct {
	name     string
	distance int
}

// SuggestSimilar finds symbols in the scope that are similar to the given name.
// It returns up to maxSuggestions symbols with an edit distance <= maxDistance,
// sorted by distance (closest first).
func (s *Scope) SuggestSimilar(ctx context.Context, name string, maxSuggestions, maxDistance int) []string {
	seen := make(map[string]bool)
	var suggestions []suggestion

	// Collect symbols from the entire scope hierarchy
	s.collectSimilarSymbols(ctx, name, maxDistance, seen, &suggestions)

	// Sort by distance, then alphabetically for ties
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].distance != suggestions[j].distance {
			return suggestions[i].distance < suggestions[j].distance
		}
		return suggestions[i].name < suggestions[j].name
	})

	// Return up to maxSuggestions names
	result := make([]string, 0, maxSuggestions)
	for i := 0; i < len(suggestions) && i < maxSuggestions; i++ {
		result = append(result, suggestions[i].name)
	}
	return result
}

// collectSimilarSymbols recursively gathers similar symbols from scope hierarchy
func (s *Scope) collectSimilarSymbols(
	ctx context.Context,
	name string,
	maxDistance int,
	seen map[string]bool,
	suggestions *[]suggestion,
) {
	// Check children in current scope
	for _, child := range s.Children {
		if child.Name != "" && !seen[child.Name] {
			dist := LevenshteinDistance(name, child.Name)
			if dist <= maxDistance && dist > 0 { // dist > 0 to exclude exact matches
				seen[child.Name] = true
				*suggestions = append(*suggestions, suggestion{name: child.Name, distance: dist})
			}
		}
	}

	// Check global resolver
	if s.GlobalResolver != nil {
		// Try to get all symbols with empty prefix (if supported)
		if symbols, err := s.GlobalResolver.ResolvePrefix(ctx, ""); err == nil {
			for _, sym := range symbols {
				if sym.Name != "" && !seen[sym.Name] {
					dist := LevenshteinDistance(name, sym.Name)
					if dist <= maxDistance && dist > 0 {
						seen[sym.Name] = true
						*suggestions = append(*suggestions, suggestion{name: sym.Name, distance: dist})
					}
				}
			}
		}
	}

	// Recurse to parent scope
	if s.Parent != nil {
		s.Parent.collectSimilarSymbols(ctx, name, maxDistance, seen, suggestions)
	}
}
