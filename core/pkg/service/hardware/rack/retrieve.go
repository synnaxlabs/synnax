// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	"github.com/synnaxlabs/x/gorp"
)

type Retrieve struct {
	baseTX       gorp.Tx
	otg          *ontology.Ontology
	gorp         gorp.Retrieve[Key, Rack]
	hostProvider cluster.HostProvider
	searchTerm   string
}

// Search applies a fuzzy search filter to the query. This will be executed before
// all other filters are applied.
func (r Retrieve) Search(term string) Retrieve {
	r.searchTerm = term
	return r
}

// WhereKeys filters racks by their keys.
func (r Retrieve) WhereKeys(keys ...Key) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

// WhereNames filters racks by their names.
func (r Retrieve) WhereNames(names ...string) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, rack *Rack) (bool, error) {
		return lo.Contains(names, rack.Name), nil
	})
	return r
}

// WhereEmbedded filters for racks that are embedded within the Synnax server.
func (r Retrieve) WhereEmbedded(embedded bool, opts ...gorp.FilterOption) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, rack *Rack) (bool, error) {
		return rack.Embedded == embedded, nil
	}, opts...)
	return r
}

// WhereNodeIsHost filters for racks that are bound to the provided node and are
// a gateway.
func (r Retrieve) WhereNodeIsHost(opts ...gorp.FilterOption) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, rack *Rack) (bool, error) {
		return rack.Key.Node() == r.hostProvider.HostKey(), nil
	}, opts...)
	return r
}

// Entry binds the provided entry as the result container for the query. If multiple
// entries are found, the first one will be used.
func (r Retrieve) Entry(rack *Rack) Retrieve {
	r.gorp = r.gorp.Entry(rack)
	return r
}

// Entries binds the provided slice as the result container for the query. If multiple
// entries are found, they will be appended to the slice.
func (r Retrieve) Entries(racks *[]Rack) Retrieve {
	r.gorp = r.gorp.Entries(racks)
	return r
}

// Limit sets the maximum number of entries to return.
func (r Retrieve) Limit(limit int) Retrieve {
	r.gorp = r.gorp.Limit(limit)
	return r
}

// Offset sets the starting index of the entries to return.
func (r Retrieve) Offset(offset int) Retrieve {
	r.gorp = r.gorp.Offset(offset)
	return r
}

// WhereNode filters for racks that are embedded within the provided node.
func (r Retrieve) WhereNode(node cluster.NodeKey, opts ...gorp.FilterOption) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, rack *Rack) (bool, error) {
		return rack.Key.Node() == node, nil
	}, opts...)
	return r
}

func (r Retrieve) execSearch(ctx context.Context) (Retrieve, error) {
	if r.searchTerm == "" {
		return r, nil
	}
	ids, err := r.otg.SearchIDs(ctx, search.Request{
		Type: OntologyType,
		Term: r.searchTerm,
	})
	if err != nil {
		return r, err
	}
	keys, err := KeysFromOntologyIds(ids)
	if err != nil {
		return r, err
	}
	r = r.WhereKeys(keys...)
	return r, err
}

// Exec executes the query against the provided transaction.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	var err error
	r, err = r.execSearch(ctx)
	if err != nil {
		return err
	}
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}
