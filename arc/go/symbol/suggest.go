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

	"github.com/synnaxlabs/x/compare"
)

func (s *Scope) SuggestSimilar(ctx context.Context, name string, maxSuggestions int) []string {
	results, err := s.Search(ctx, name)
	if err != nil || len(results) == 0 {
		return nil
	}

	type suggestion struct {
		name     string
		distance int
	}

	var suggestions []suggestion
	for _, r := range results {
		if r.Name == name {
			continue // skip exact match
		}
		suggestions = append(suggestions, suggestion{
			name:     r.Name,
			distance: compare.LevenshteinDistance(name, r.Name),
		})
	}

	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].distance != suggestions[j].distance {
			return suggestions[i].distance < suggestions[j].distance
		}
		return suggestions[i].name < suggestions[j].name
	})

	result := make([]string, 0, maxSuggestions)
	for i := 0; i < len(suggestions) && i < maxSuggestions; i++ {
		result = append(result, suggestions[i].name)
	}
	return result
}
