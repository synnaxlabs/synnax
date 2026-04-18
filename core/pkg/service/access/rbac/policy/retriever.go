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
	"slices"
	"sync"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

type Retrieve struct {
	baseTX   gorp.Tx
	gorp     gorp.Retrieve[uuid.UUID, Policy]
	ontology *ontology.Ontology
}

// MatchSubjects returns a filter that matches policies attached to any of the
// given subjects via the ontology. On first evaluation the filter resolves the
// subjects to policy keys through a parent→role→child→policy traversal, caches
// the result, and then tests key membership for each subsequent entry.
func MatchSubjects(subjects ...ontology.ID) Filter {
	var (
		keys   []uuid.UUID
		resErr error
		once   sync.Once
	)
	return Match(func(ctx gorp.Context, r Retrieve, p *Policy) (bool, error) {
		once.Do(func() {
			var policyResources []ontology.Resource
			if err := r.ontology.NewRetrieve().WhereIDs(subjects...).
				ExcludeFieldData(true).
				TraverseTo(ontology.ParentsTraverser).
				WhereTypes(ontology.ResourceTypeRole).
				TraverseTo(ontology.ChildrenTraverser).
				WhereTypes(ontology.ResourceTypePolicy).
				Entries(&policyResources).
				Exec(ctx, ctx.Tx); err != nil {
				resErr = err
				return
			}
			k, err := KeysFromOntologyIDs(ontology.ResourceIDs(policyResources))
			if err != nil {
				resErr = err
				return
			}
			keys = k
		})
		if resErr != nil {
			return false, resErr
		}
		return slices.Contains(keys, p.Key), nil
	})
}
