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

// WhereOverlapsWith filters for ranges whose TimeRange overlaps with the provided range.
func (r Retrieve) WhereOverlapsWith(tr telem.TimeRange) Retrieve {
	r.gorp = r.gorp.Where(func(_ gorp.Context, rng *Range) (bool, error) {
		return rng.TimeRange.OverlapsWith(tr), nil
	})
	return r
}

// WhereHasLabels filters for ranges that have all of the provided labels.
func (r Retrieve) WhereHasLabels(matchLabels ...label.Key) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, rng *Range) (bool, error) {
		labels, err := r.label.RetrieveFor(ctx, rng.OntologyID(), ctx.Tx)
		if err != nil {
			return false, err
		}
		labelKeys := lo.Map(labels, func(l label.Label, _ int) label.Key { return l.Key })
		return lo.ContainsBy(labelKeys, func(l label.Key) bool {
			return lo.Contains(matchLabels, l)
		}), nil
	})
	return r
}
