// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"context"
	"slices"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
)

// unknownRangesMigration drops key references to ranges the Core does not hold, such
// as a range kept only in one Console's local session or a deleted range. It decodes
// line plots and ranges with their current types, so it runs
// after the line plot version chain and after the ranger table opens.
var unknownRangesMigration = migrate.New(
	"v61_lineplot_unknown_ranges",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		var plots []LinePlot
		if err := gorp.NewRetrieve[Key, LinePlot]().
			Entries(&plots).
			Exec(ctx, tx); err != nil {
			return err
		}
		changed := make([]LinePlot, 0, len(plots))
		for _, p := range plots {
			next, err := dropUnknownRanges(ctx, tx, p)
			if err != nil {
				return err
			}
			if len(next.Ranges) != len(p.Ranges) {
				changed = append(changed, next)
			}
		}
		return gorp.NewCreate[Key, LinePlot]().Entries(&changed).Exec(ctx, tx)
	},
)

// migrations is the line plot version chain followed by unknownRangesMigration.
var migrations = append(
	slices.Clone(versions.Migrations),
	unknownRangesMigration,
)

func dropUnknownRanges(ctx context.Context, tx gorp.Tx, p LinePlot) (LinePlot, error) {
	kept := make([]ranger.Ref, 0, len(p.Ranges))
	for _, r := range p.Ranges {
		k, ok := r.Variant.(ranger.KeyRef)
		if !ok {
			kept = append(kept, r)
			continue
		}
		exists, err := rangeExists(ctx, tx, k.Key)
		if err != nil {
			return LinePlot{}, err
		}
		if exists {
			kept = append(kept, r)
		}
	}
	p.Ranges = kept
	return p, nil
}

// rangeExists reports whether the Core holds a saved range with the given key.
func rangeExists(ctx context.Context, tx gorp.Tx, key ranger.Key) (bool, error) {
	return gorp.NewRetrieve[ranger.Key, ranger.Range]().
		Where(gorp.MatchKeys[ranger.Key, ranger.Range](key)).
		Exists(ctx, tx)
}
