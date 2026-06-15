// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/x/gorp"
)

// Filter restricts the set of arcs a Retrieve returns.
type Filter = gorp.Filter[Key, Arc]

// Retrieve is a builder for querying arcs. In addition to the standard filtering and
// pagination options, IncludeTask and IncludeStatus opt into resolving the arc's
// deployed task (and that task's status) from the resource graph at Exec time.
type Retrieve struct {
	baseTX        gorp.Tx
	gorp          gorp.Retrieve[Key, Arc]
	search        *search.Index
	searchTerm    string
	svc           *Service
	includeTask   bool
	includeStatus bool
	entry         *Arc
	entries       *[]Arc
}

// MatchKeys returns a filter that restricts results to arcs whose key matches any of the
// provided values.
func MatchKeys(keys ...Key) Filter { return gorp.MatchKeys[Key, Arc](keys...) }

// MatchNames returns a filter for arcs whose Name matches any of the provided values.
func MatchNames(names ...string) Filter {
	return gorp.Match(func(_ gorp.Context, a *Arc) (bool, error) {
		return lo.Contains(names, a.Name), nil
	})
}

// Search sets a fuzzy search term that Retrieve will use to filter results.
func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }

// Where applies the provided filter to the query.
func (r Retrieve) Where(filter Filter) Retrieve { r.gorp = r.gorp.Where(filter); return r }

// Entry binds the arc that Retrieve will fill the first result into.
func (r Retrieve) Entry(a *Arc) Retrieve {
	r.entry = a
	r.gorp = r.gorp.Entry(a)
	return r
}

// Entries binds the slice that Retrieve will fill results into.
func (r Retrieve) Entries(a *[]Arc) Retrieve {
	r.entries = a
	r.gorp = r.gorp.Entries(a)
	return r
}

// Limit sets the maximum number of arcs to return.
func (r Retrieve) Limit(limit int) Retrieve { r.gorp = r.gorp.Limit(limit); return r }

// Offset sets the starting index of the arcs to return.
func (r Retrieve) Offset(offset int) Retrieve { r.gorp = r.gorp.Offset(offset); return r }

// IncludeTask sets whether Exec resolves each arc's deployed task key into Arc.Task and
// the rack it runs on into Arc.Rack.
func (r Retrieve) IncludeTask(v bool) Retrieve { r.includeTask = v; return r }

// IncludeStatus sets whether Exec resolves the status of each arc's deployed task into
// Arc.Status. Enabling it also resolves the task key (see IncludeTask).
func (r Retrieve) IncludeStatus(v bool) Retrieve { r.includeStatus = v; return r }

func (r Retrieve) execSearch(ctx context.Context) (Retrieve, error) {
	if r.searchTerm == "" {
		return r, nil
	}
	ids, err := r.search.Search(ctx, search.Request{
		Type: ontology.ResourceTypeArc,
		Term: r.searchTerm,
	})
	if err != nil {
		return Retrieve{}, err
	}
	keys, err := KeysFromOntologyIDs(ids)
	if err != nil {
		return Retrieve{}, err
	}
	return r.Where(MatchKeys(keys...)), nil
}

// Exec executes the query against the provided transaction, then resolves the task and
// status fields of the results if IncludeTask or IncludeStatus were set.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTX, tx)
	var err error
	if r, err = r.execSearch(ctx); err != nil {
		return err
	}
	if err = r.gorp.Exec(ctx, tx); err != nil {
		return err
	}
	if !r.includeTask && !r.includeStatus {
		return nil
	}
	var resolved []*Arc
	if r.entries != nil {
		resolved = make([]*Arc, len(*r.entries))
		for i := range *r.entries {
			resolved[i] = &(*r.entries)[i]
		}
	} else if r.entry != nil {
		resolved = []*Arc{r.entry}
	}
	return r.svc.resolve(ctx, tx, resolved, r.includeStatus)
}

// Count returns the number of arcs matching the query.
func (r Retrieve) Count(ctx context.Context, tx gorp.Tx) (int, error) {
	var err error
	if r, err = r.execSearch(ctx); err != nil {
		return 0, err
	}
	return r.gorp.Count(ctx, gorp.OverrideTx(r.baseTX, tx))
}

// Exists checks whether any arcs match the query.
func (r Retrieve) Exists(ctx context.Context, tx gorp.Tx) (bool, error) {
	var err error
	if r, err = r.execSearch(ctx); err != nil {
		return false, err
	}
	return r.gorp.Exists(ctx, gorp.OverrideTx(r.baseTX, tx))
}
