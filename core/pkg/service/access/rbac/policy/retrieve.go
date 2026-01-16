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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/set"
)

// Retrieve is a query builder for retrieving policies from the database.
type Retrieve struct {
	baseTx   gorp.Tx
	gorp     gorp.Retrieve[uuid.UUID, Policy]
	ontology *ontology.Ontology
	subject  ontology.ID
}

// NewRetrieve opens a new Retrieve query to fetch policies.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		baseTx:   s.cfg.DB,
		gorp:     gorp.NewRetrieve[uuid.UUID, Policy](),
		ontology: s.cfg.Ontology,
	}
}

// WhereKeys filters the policies by the given keys.
func (r Retrieve) WhereKeys(keys ...uuid.UUID) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}

// WhereNames filters the policies by the given names.
func (r Retrieve) WhereNames(names ...string) Retrieve {
	namesSet := set.FromSlice(names)
	r.gorp = r.gorp.Where(func(_ gorp.Context, p *Policy) (bool, error) {
		return namesSet.Contains(p.Name), nil
	})
	return r
}

// WhereSubject filters for policies that are associated with the given subject.
func (r Retrieve) WhereSubject(subject ontology.ID) Retrieve {
	r.subject = subject
	return r
}

// Limit limits the number of policies returned.
func (r Retrieve) Limit(limit int) Retrieve { r.gorp = r.gorp.Limit(limit); return r }

// Offset gives the number of policies to skip before returning results.
func (r Retrieve) Offset(offset int) Retrieve {
	r.gorp = r.gorp.Offset(offset)
	return r
}

// Exec executes the query against the provided transaction.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTx, tx)
	if !r.subject.IsZero() {
		var policies []ontology.Resource
		if err := r.ontology.NewRetrieve().WhereIDs(r.subject).
			ExcludeFieldData(true).
			TraverseTo(ontology.Parents).
			WhereTypes(role.OntologyType).
			TraverseTo(ontology.Children).
			WhereTypes(OntologyType).
			Entries(&policies).
			Exec(ctx, tx); err != nil {
			return err
		}
		keys, err := KeysFromOntologyIDs(ontology.ResourceIDs(policies))
		if err != nil {
			return err
		}
		r = r.WhereKeys(keys...)
	}
	return r.gorp.Exec(ctx, tx)
}

// Entry binds the given policy to the query.
func (r Retrieve) Entry(p *Policy) Retrieve { r.gorp = r.gorp.Entry(p); return r }

// Entries binds the given slice of policies to the query.
func (r Retrieve) Entries(policies *[]Policy) Retrieve {
	r.gorp = r.gorp.Entries(policies)
	return r
}
