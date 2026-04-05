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
	"github.com/synnaxlabs/x/gorp"
)

type Retrieve struct {
	baseTX        gorp.Tx
	gorp          gorp.Retrieve[uuid.UUID, Policy]
	ontology      *ontology.Ontology
	whereSubjects []ontology.ID
}

// WhereSubjects accumulates subject IDs for ontology traversal during Exec.
func (r Retrieve) WhereSubjects(subjects ...ontology.ID) Retrieve {
	r.whereSubjects = append(r.whereSubjects, subjects...)
	return r
}

func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	tx = gorp.OverrideTx(r.baseTX, tx)
	if len(r.whereSubjects) > 0 {
		var policyResources []ontology.Resource
		if err := r.ontology.NewRetrieve().WhereIDs(r.whereSubjects...).
			ExcludeFieldData(true).
			TraverseTo(ontology.ParentsTraverser).
			WhereTypes(ontology.ResourceTypeRole).
			TraverseTo(ontology.ChildrenTraverser).
			WhereTypes(ontology.ResourceTypePolicy).
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
