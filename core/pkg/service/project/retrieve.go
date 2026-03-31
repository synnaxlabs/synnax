// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/x/gorp"
)

type Retrieve struct {
	baseTX     gorp.Tx
	search     *search.Index
	gorp       gorp.Retrieve[uuid.UUID, Project]
	searchTerm string
}

func (r Retrieve) Search(term string) Retrieve {
	r.searchTerm = term
	return r
}

func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

func (r Retrieve) Entry(p *Project) Retrieve {
	r.gorp = r.gorp.Entry(p)
	return r
}

func (r Retrieve) Entries(ps *[]Project) Retrieve {
	r.gorp = r.gorp.Entries(ps)
	return r
}

func (r Retrieve) Limit(limit int) Retrieve {
	r.gorp = r.gorp.Limit(limit)
	return r
}

func (r Retrieve) Offset(offset int) Retrieve {
	r.gorp = r.gorp.Offset(offset)
	return r
}

func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	if r.searchTerm != "" {
		ids, err := r.search.Search(ctx, search.Request{
			Type: ontology.ResourceTypeProject,
			Term: r.searchTerm,
		})
		if err != nil {
			return err
		}
		keys, err := KeysFromOntologyIDs(ids)
		if err != nil {
			return err
		}
		r = r.WhereKeys(keys...)
	}
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}
