// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/compare"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// Retrieve is used to retrieve statuses from the cluster using a builder pattern.
type Retrieve[D any] struct {
	gorp gorp.Retrieve[Key, Status[D]]
	// indexes are on the table, whose entries hold details of any type.
	indexes indexes
}

// Filter is a per-service filter that is bound to the Retrieve when passed to Where.
// Use Match to construct one from a closure.
type Filter[D any] func(r Retrieve[D]) gorp.Filter[Key, Status[D]]

// Match wraps a closure that needs the Retrieve into a Filter.
func Match[D any](
	f func(gorp.Context, Retrieve[D], *Status[D]) (bool, error),
) Filter[D] {
	return func(r Retrieve[D]) gorp.Filter[Key, Status[D]] {
		return gorp.Match(func(ctx gorp.Context, s *Status[D]) (bool, error) {
			return f(ctx, r, s)
		})
	}
}

// And returns a filter that matches when all provided filters match.
func And[D any](fs ...Filter[D]) Filter[D] {
	return func(r Retrieve[D]) gorp.Filter[Key, Status[D]] {
		inner := make([]gorp.Filter[Key, Status[D]], len(fs))
		for i, f := range fs {
			inner[i] = f(r)
		}
		return gorp.And(inner...)
	}
}

// Or returns a filter that matches when any provided filter matches.
func Or[D any](fs ...Filter[D]) Filter[D] {
	return func(r Retrieve[D]) gorp.Filter[Key, Status[D]] {
		inner := make([]gorp.Filter[Key, Status[D]], len(fs))
		for i, f := range fs {
			inner[i] = f(r)
		}
		return gorp.Or(inner...)
	}
}

// Not returns a filter that inverts the provided filter.
func Not[D any](f Filter[D]) Filter[D] {
	return func(r Retrieve[D]) gorp.Filter[Key, Status[D]] {
		return gorp.Not(f(r))
	}
}

// Entry binds the Status that Retrieve will fill results into. If multiple results
// match the query, only the first result will be filled into the provided Status.
func (r Retrieve[D]) Entry(s *Status[D]) Retrieve[D] {
	r.gorp = r.gorp.Entry(s)
	return r
}

// Limit sets the maximum number of results that Retrieve will return.
func (r Retrieve[D]) Limit(limit int) Retrieve[D] {
	r.gorp = r.gorp.Limit(limit)
	return r
}

// After starts the query strictly past key in key order. It panics on a query with an
// OrderBy, since an Order carries its own cursor.
func (r Retrieve[D]) After(key Key) Retrieve[D] {
	r.gorp = r.gorp.After(key)
	return r
}

// SkipMissingKeys makes Exec treat a key with no status as a non-match instead of
// returning query.ErrNotFound.
func (r Retrieve[D]) SkipMissingKeys() Retrieve[D] {
	r.gorp = r.gorp.SkipMissingKeys()
	return r
}

// Entries binds a slice that Retrieve will fill results into.
func (r Retrieve[D]) Entries(s *[]Status[D]) Retrieve[D] {
	r.gorp = r.gorp.Entries(s)
	return r
}

// Where applies the provided filter to the query. A status must match every chained
// Where.
func (r Retrieve[D]) Where(filter Filter[D]) Retrieve[D] {
	r.gorp = r.gorp.Where(filter(r))
	return r
}

// MatchKeys returns a filter that restricts results to statuses whose key
// matches any of the provided values.
func MatchKeys[D any](keys ...Key) Filter[D] {
	return func(Retrieve[D]) gorp.Filter[Key, Status[D]] {
		return gorp.MatchKeys[Key, Status[D]](keys...)
	}
}

// MatchNames returns a filter for statuses whose name matches any of the provided
// values.
func MatchNames[D any](names ...string) Filter[D] {
	keys := make([]string, len(names))
	for i, n := range names {
		keys[i] = compare.TextKey(n)
	}
	exact := set.New(names...)
	named := gorp.Match(func(_ gorp.Context, s *Status[D]) (bool, error) {
		return exact.Contains(s.Name), nil
	})
	return func(r Retrieve[D]) gorp.Filter[Key, Status[D]] {
		return gorp.And(
			gorp.MatchKeysFunc[Key, Status[D]](func(ctx gorp.Context) ([]Key, error) {
				found, err := r.indexes.name.Get(ctx, ctx.Snapshot, keys...)
				if !errors.Is(err, gorp.ErrIndexInvalid) {
					return found, err
				}
				// Scan when the index failed to populate, as an index Filter does.
				var matches []Status[D]
				if err = gorp.NewRetrieve[Key, Status[D]]().
					Where(named).
					Entries(&matches).
					Exec(ctx, ctx.Snapshot); err != nil {
					return nil, err
				}
				return lo.Map(
					matches,
					func(s Status[D], _ int) Key { return s.Key },
				), nil
			}),
			named,
		)
	}
}

// MatchVariants returns a filter for statuses with the given variants.
func MatchVariants[D any](variants ...Variant) Filter[D] {
	return func(Retrieve[D]) gorp.Filter[Key, Status[D]] {
		return gorp.Match(func(_ gorp.Context, s *Status[D]) (bool, error) {
			return slices.Contains(variants, s.Variant), nil
		})
	}
}

// MatchLabels returns a filter for statuses that have any of the provided labels.
func MatchLabels[D any](labels ...label.Key) Filter[D] {
	return func(r Retrieve[D]) gorp.Filter[Key, Status[D]] {
		return r.indexes.labels.FilterAs[Status[D]](labels...)
	}
}

// MatchOwners returns a filter for the statuses that the provided resources own.
func MatchOwners[D any](owners ...resource.ID) Filter[D] {
	targets := make([]gorp.Target, len(owners))
	for i, o := range owners {
		targets[i] = resourceTarget(o)
	}
	return func(r Retrieve[D]) gorp.Filter[Key, Status[D]] {
		return r.indexes.owner.FilterAs[Status[D]](targets...)
	}
}

// Exec executes the query against snap and fills the results into the provided Status
// or slice of Statuses. It panics when snap is nil.
func (r Retrieve[D]) Exec(ctx context.Context, snap gorp.Snapshot) error {
	return r.gorp.Exec(ctx, snap)
}
