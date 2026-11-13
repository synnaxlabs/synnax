// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"context"
	"slices"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Retrieve is used to retrieve Task records from the database using a
// builder pattern for constructing queries.
type Retrieve struct {
	gorp      gorp.Retrieve[Key, Task]
	indexes   indexes
	configs   config.Registry
	matchArcs func(arcs ...uuid.UUID) Filter
	entry     *Task
	entries   *[]Task
}

// Filter is a per-service filter that is bound to the Retrieve when passed to Where.
// Pure filters ignore the Retrieve argument; service-bound filters read from it (e.g.
// r.indexes, r.configs) to evaluate. Use Match to construct one from a closure.
//
// Filter is a type alias for gorp.BoundFilter[Retrieve, K, E] so the composition
// helpers (Match / And / Or / Not) can be one-line wrappers around their gorp.*Bound
// counterparts instead of re-emitting closure plumbing per service.
type Filter = gorp.BoundFilter[Retrieve, Key, Task]

// Match wraps a closure that needs the Retrieve into a Filter. The Retrieve
// value is supplied by Retrieve.Where at evaluation time.
func Match(f func(gorp.Context, Retrieve, *Task) (bool, error)) Filter {
	return gorp.MatchBound(f)
}

// And returns a filter that matches when all provided filters match.
func And(fs ...Filter) Filter { return gorp.AndBound(fs...) }

// Or returns a filter that matches when any provided filter matches.
func Or(fs ...Filter) Filter { return gorp.OrBound(fs...) }

// Not returns a filter that inverts the provided filter.
func Not(f Filter) Filter { return gorp.NotBound(f) }

// MatchKeys returns a filter that restricts results to tasks whose key matches any of
// the provided values. Composing MatchKeys at the top level of a Where clause (i.e.
// r.Where(MatchKeys(...))) dispatches Exec to the multi-get fast path; composing
// inside Or / Not falls back to a full scan.
func MatchKeys(keys ...Key) Filter {
	return func(_ Retrieve) gorp.Filter[Key, Task] {
		return gorp.MatchKeys[Key, Task](keys...)
	}
}

// MatchRacks returns a filter for tasks whose Rack matches any of the provided values.
// A draft on no rack matches the value 0.
func MatchRacks(vals ...rack.Key) Filter {
	return func(r Retrieve) gorp.Filter[Key, Task] {
		if !slices.Contains(vals, 0) {
			return r.indexes.rack.Filter(vals...)
		}
		return gorp.Or(
			r.indexes.rack.Filter(vals...),
			r.indexes.rack.FilterHolding(false),
		)
	}
}

// MatchArcs returns a filter for the tasks that run any of the Arcs with the given
// keys.
func MatchArcs(arcs ...uuid.UUID) Filter {
	return func(r Retrieve) gorp.Filter[Key, Task] {
		if r.matchArcs == nil {
			return gorp.MatchKeysFunc[Key, Task](func(gorp.Context) ([]Key, error) {
				return nil, errors.New("this task service cannot filter tasks by Arc")
			})
		}
		return r.matchArcs(arcs...)(r)
	}
}

// MatchNames returns a filter for tasks whose Name matches any of the provided values.
func MatchNames(vals ...string) Filter {
	return func(r Retrieve) gorp.Filter[Key, Task] {
		return gorp.Match(func(_ gorp.Context, e *Task) (bool, error) {
			return lo.Contains(vals, e.Name), nil
		})
	}
}

// MatchTypes returns a filter for tasks whose Type matches any of the provided values.
func MatchTypes(vals ...string) Filter {
	return func(r Retrieve) gorp.Filter[Key, Task] {
		return gorp.Match(func(_ gorp.Context, e *Task) (bool, error) {
			return lo.Contains(vals, e.Type), nil
		})
	}
}

// MatchInternal returns a filter for tasks by their Internal field.
func MatchInternal(v bool) Filter {
	return func(r Retrieve) gorp.Filter[Key, Task] {
		return gorp.Match(func(_ gorp.Context, e *Task) (bool, error) {
			return e.Internal == v, nil
		})
	}
}

// Where applies the provided filter to the query, binding it to the Retrieve so
// service-bound filters can read from r.indexes, r.configs, etc. To compose multiple
// filters, chain Where calls or pass a combined filter via And / Or.
func (r Retrieve) Where(filter Filter) Retrieve {
	r.gorp = r.gorp.Where(filter(r))
	return r
}

// Entry binds the provided task as the result container for the query. If multiple
// tasks match, the first one is used.
func (r Retrieve) Entry(e *Task) Retrieve {
	r.entry = e
	r.gorp = r.gorp.Entry(e)
	return r
}

// Entries binds the provided slice of tasks as the result container for the query.
func (r Retrieve) Entries(es *[]Task) Retrieve {
	r.entries = es
	r.gorp = r.gorp.Entries(es)
	return r
}

// Limit sets the maximum number of tasks to return.
func (r Retrieve) Limit(limit int) Retrieve { r.gorp = r.gorp.Limit(limit); return r }

// After starts the query strictly past key in key order. It panics on a query with an
// OrderBy, since an Order carries its own cursor.
func (r Retrieve) After(key Key) Retrieve {
	r.gorp = r.gorp.After(key)
	return r
}

// SkipMissingKeys makes Exec treat a key with no task as a non-match instead of
// returning query.ErrNotFound.
func (r Retrieve) SkipMissingKeys() Retrieve {
	r.gorp = r.gorp.SkipMissingKeys()
	return r
}

// results returns pointers to the tasks the executed query populated.
func (r Retrieve) results() []*Task {
	if r.entries != nil {
		res := make([]*Task, len(*r.entries))
		for i := range *r.entries {
			res[i] = &(*r.entries)[i]
		}
		return res
	}
	if r.entry != nil {
		return []*Task{r.entry}
	}
	return nil
}

// composeConfigs fills the Config of each retrieved task from the record stored
// under the task's key in the task type's config store. Tasks whose type has no
// registered store or no stored record keep a nil Config.
func (r Retrieve) composeConfigs(
	ctx context.Context,
	snap gorp.Snapshot,
	tasks []*Task,
) error {
	for _, t := range tasks {
		store, ok := r.configs.Store(t.Type)
		if !ok {
			continue
		}
		data, err := store.Read(ctx, snap, t.Key)
		if err != nil {
			if errors.Is(err, query.ErrNotFound) {
				continue
			}
			return err
		}
		t.Config = data
	}
	return nil
}

// Exec executes the query against snap. It panics when snap is nil.
func (r Retrieve) Exec(ctx context.Context, snap gorp.Snapshot) error {
	if err := r.gorp.Exec(ctx, snap); err != nil {
		return err
	}
	return r.composeConfigs(ctx, snap, r.results())
}

// Count returns the number of tasks in snap that match the query. It panics when snap
// is nil.
func (r Retrieve) Count(ctx context.Context, snap gorp.Snapshot) (int, error) {
	return r.gorp.Count(ctx, snap)
}

// Exists checks whether any tasks in snap match the query. It panics when snap is nil.
func (r Retrieve) Exists(ctx context.Context, snap gorp.Snapshot) (bool, error) {
	return r.gorp.Exists(ctx, snap)
}
