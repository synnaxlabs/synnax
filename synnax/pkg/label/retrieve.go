/*
 * Copyright 2023 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package label

import (
	"context"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	"github.com/synnaxlabs/x/gorp"
)

type Retrieve struct {
	baseTx     gorp.Tx
	gorp       gorp.Retrieve[uuid.UUID, Label]
	otg        *ontology.Ontology
	searchTerm string
}

func newRetrieve(tx gorp.Tx, otg *ontology.Ontology) Retrieve {
	return Retrieve{
		baseTx: tx,
		gorp:   gorp.NewRetrieve[uuid.UUID, Label](),
		otg:    otg,
	}
}

func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }

// Entry binds the Label that Retrieve will fill results into. If multiple results match
// the query, only the first result will be filled into the provided Label.
func (r Retrieve) Entry(label *Label) Retrieve { r.gorp.Entry(label); return r }

// Entries binds a slice that Retrieve will fill results into.
func (r Retrieve) Entries(labels *[]Label) Retrieve { r.gorp.Entries(labels); return r }

// WhereKeys filters for labels whose Name attribute matches the provided key.
func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve { r.gorp.WhereKeys(keys...); return r }

// WhereNames filters for labels whose Name attribute matches the provided name.
func (r Retrieve) WhereNames(names ...string) Retrieve {
	r.gorp.Where(func(label *Label) bool { return lo.Contains(names, label.Name) })
	return r
}

func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTx, tx)
	if r.searchTerm != "" {
		ids, err := r.otg.SearchIDs(ctx, search.Request{
			Type: ontologyType,
			Term: r.searchTerm,
		})
		if err != nil {
			return err
		}
		keys, err := KeysFromOntologyIds(ids)
		if err != nil {
			return err
		}
		r.gorp.WhereKeys(keys...)
	}
	return r.gorp.Exec(ctx, tx)
}
