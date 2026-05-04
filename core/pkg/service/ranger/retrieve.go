// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

// Retrieve is used to retrieve ranges from the cluster using a builder pattern.
type Retrieve struct {
	baseTX     gorp.Tx
	gorp       gorp.Retrieve[uuid.UUID, Range]
	search     *search.Index
	label      *label.Service
	searchTerm string
}

// MatchOverlap returns a filter that matches ranges whose TimeRange overlaps
// with the provided range.
func MatchOverlap(tr telem.TimeRange) Filter {
	return func(_ Retrieve) gorp.Filter[uuid.UUID, Range] {
		return gorp.Match(func(_ gorp.Context, rng *Range) (bool, error) {
			return rng.TimeRange.OverlapsWith(tr), nil
		})
	}
}

// MatchLabels returns a filter that matches ranges that have any of the
// provided labels.
func MatchLabels(matchLabels ...label.Key) Filter {
	return Match(func(ctx gorp.Context, r Retrieve, rng *Range) (bool, error) {
		labels, err := r.label.RetrieveFor(ctx, rng.OntologyID(), ctx.Tx)
		if err != nil {
			return false, err
		}
		labelKeys := lo.Map(labels, func(l label.Label, _ int) label.Key { return l.Key })
		return lo.ContainsBy(labelKeys, func(l label.Key) bool {
			return lo.Contains(matchLabels, l)
		}), nil
	})
}
