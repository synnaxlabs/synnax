// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
)

type Retriever struct {
	baseTx        gorp.Tx
	gorp          gorp.Retrieve[uuid.UUID, Policy]
	ontology      *ontology.Ontology
	whereSubjects []ontology.ID
}

func (r Retriever) WhereKeys(keys ...uuid.UUID) Retriever {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

func (r Retriever) WhereNames(names ...string) Retriever {
	r.gorp = r.gorp.Where(func(_ gorp.Context, e *Policy) (bool, error) {
		return lo.Contains(names, e.Name), nil
	})
	return r
}

func (r Retriever) WhereSubjects(subjects ...ontology.ID) Retriever {
	r.whereSubjects = append(r.whereSubjects, subjects...)
	return r
}

func (r Retriever) Limit(limit int) Retriever { r.gorp = r.gorp.Limit(limit); return r }

func (r Retriever) Offset(offset int) Retriever {
	r.gorp = r.gorp.Offset(offset)
	return r
}

func (r Retriever) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTx, tx)
	if len(r.whereSubjects) > 0 {
		var policyResources []ontology.Resource
		if err := r.ontology.NewRetrieve().WhereIDs(r.whereSubjects...).
			ExcludeFieldData(true).
			TraverseTo(ontology.Parents).
			WhereTypes(role.OntologyType).
			TraverseTo(ontology.Children).
			WhereTypes(OntologyType).
			Entries(&policyResources).
			Exec(ctx, tx); err != nil {
			return err
		}
		keys, err := KeysFromOntologyIDs(ontology.ResourceIDs(policyResources))
		if err != nil {
			return err
		}
		r = r.WhereKeys(keys...)
	}
	return r.gorp.Exec(ctx, tx)
}

func (r Retriever) Entry(p *Policy) Retriever { r.gorp = r.gorp.Entry(p); return r }

func (r Retriever) Entries(policies *[]Policy) Retriever {
	r.gorp = r.gorp.Entries(policies)
	return r
}
