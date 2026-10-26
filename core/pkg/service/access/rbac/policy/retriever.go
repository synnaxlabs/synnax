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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

type Retrieve struct {
	baseTX   gorp.Tx
	gorp     gorp.Retrieve[Key, Policy]
	indexes  indexes
	ontology *ontology.Ontology
}

// ResolveSubjects walks the ontology from each subject up to its parent roles and back
// down to the policies attached to those roles, returning the policy keys.
func (s *Service) ResolveSubjects(
	ctx context.Context,
	tx gorp.Tx,
	subjects ...ontology.ID,
) ([]Key, error) {
	return resolveSubjects(ctx, s.cfg.Ontology, gorp.OverrideTx(s.cfg.DB, tx), subjects)
}

// MatchSubjects returns a filter that matches the policies of the roles that any of
// the subjects hold.
func MatchSubjects(subjects ...ontology.ID) Filter {
	return func(r Retrieve) gorp.Filter[Key, Policy] {
		return gorp.MatchKeysFunc[Key, Policy](
			func(ctx context.Context, tx gorp.Tx) ([]Key, error) {
				return resolveSubjects(ctx, r.ontology, tx, subjects)
			},
		)
	}
}

func resolveSubjects(
	ctx context.Context,
	otg *ontology.Ontology,
	tx gorp.Tx,
	subjects []ontology.ID,
) ([]Key, error) {
	var policyResources []ontology.Resource
	if err := otg.NewRetrieve().WhereIDs(subjects...).
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
