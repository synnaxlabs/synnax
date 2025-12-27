// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/gorp"
)

// Retrieve is used to retrieve devices from the device service using a builder pattern
// for querying the database.
type Retrieve struct {
	baseTX     gorp.Tx
	otg        *ontology.Ontology
	gorp       gorp.Retrieve[string, Device]
	searchTerm string
}

// Search sets a fuzzy search term that Retrieve will use to filter results.
func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }

// WhereKeys filters for devices whose key matches the provided keys.
func (r Retrieve) WhereKeys(keys ...string) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

// WhereRacks filters for devices whose rack matches the provided racks.
func (r Retrieve) WhereRacks(racks ...rack.Key) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, d *Device) (bool, error) {
		return lo.Contains(racks, d.Rack), nil
	}, gorp.Required())
	return r
}

func (r Retrieve) WhereMakes(make ...string) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, d *Device) (bool, error) {
		return lo.Contains(make, d.Make), nil
	}, gorp.Required())
	return r
}

// WhereLocations filters for devices whose location matches the provided locations.
func (r Retrieve) WhereLocations(locations ...string) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, d *Device) (bool, error) {
		return lo.Contains(locations, d.Location), nil
	}, gorp.Required())
	return r
}

// WhereModels filters for devices whose model matches the provided models.
func (r Retrieve) WhereModels(models ...string) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, d *Device) (bool, error) {
		return lo.Contains(models, d.Model), nil
	}, gorp.Required())
	return r
}

// WhereNames filters for devices whose name matches the provided names.
func (r Retrieve) WhereNames(names ...string) Retrieve {
	r.gorp = r.gorp.Where(func(ctx gorp.Context, d *Device) (bool, error) {
		return lo.Contains(names, d.Name), nil
	})
	return r
}

// Entry binds the provided device as the result container for the query. If multiple
// devices are found, the first one will be used.
func (r Retrieve) Entry(device *Device) Retrieve {
	r.gorp = r.gorp.Entry(device)
	return r
}

// Entries binds the provided slice as the result container for the query. If multiple
// devices are found, they will be appended to the slice.
func (r Retrieve) Entries(devices *[]Device) Retrieve {
	r.gorp = r.gorp.Entries(devices)
	return r
}

// Limit sets the maximum number of results that Retrieve will return.
func (r Retrieve) Limit(limit int) Retrieve { r.gorp = r.gorp.Limit(limit); return r }

// Offset sets the starting index of the results that Retrieve will return.
func (r Retrieve) Offset(offset int) Retrieve {
	r.gorp = r.gorp.Offset(offset)
	return r
}

func (r Retrieve) execSearch(ctx context.Context) (Retrieve, error) {
	if r.searchTerm == "" {
		return r, nil
	}
	ids, err := r.otg.SearchIDs(ctx, ontology.SearchRequest{
		Type: OntologyType,
		Term: r.searchTerm,
	})
	if err != nil {
		return Retrieve{}, err
	}
	keys := KeysFromOntologyIDs(ids)
	return r.WhereKeys(keys...), nil
}

// Exec executes the query against the provided transaction.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	var err error
	if r, err = r.execSearch(ctx); err != nil {
		return err
	}
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}

// Count returns the number of results that the query will return.
func (r Retrieve) Count(ctx context.Context, tx gorp.Tx) (int, error) {
	var err error
	if r, err = r.execSearch(ctx); err != nil {
		return 0, err
	}
	return r.gorp.Count(ctx, gorp.OverrideTx(r.baseTX, tx))
}

// Exists checks if the query has any results matching its parameters.
func (r Retrieve) Exists(ctx context.Context, tx gorp.Tx) (bool, error) {
	var err error
	if r, err = r.execSearch(ctx); err != nil {
		return false, err
	}
	return r.gorp.Exists(ctx, gorp.OverrideTx(r.baseTX, tx))
}
