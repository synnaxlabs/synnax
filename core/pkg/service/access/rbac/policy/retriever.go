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
	baseTX   gorp.Tx
	gorp     gorp.Retrieve[uuid.UUID, Policy]
	ontology *ontology.Ontology
}

// ResolveSubjects walks the ontology from each subject up to its parent roles
// and back down to the policies attached to those roles, returning the policy
// keys. The result is the input to MatchKeys when retrieving policies for a
// subject; performing the resolution eagerly lets the policy retrieve hit the
// keyed multi-get fast path instead of scanning the full policy table.
func (s *Service) ResolveSubjects(
	ctx context.Context,
	tx gorp.Tx,
	subjects ...ontology.ID,
) ([]uuid.UUID, error) {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	var policyResources []ontology.Resource
	if err := s.cfg.Ontology.NewRetrieve().WhereIDs(subjects...).
		ExcludeFieldData(true).
		TraverseTo(ontology.ParentsTraverser).
		WhereTypes(ontology.ResourceTypeRole).
		TraverseTo(ontology.ChildrenTraverser).
		WhereTypes(ontology.ResourceTypePolicy).
		Entries(&policyResources).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	return KeysFromOntologyIDs(ontology.ResourceIDs(policyResources))
}
