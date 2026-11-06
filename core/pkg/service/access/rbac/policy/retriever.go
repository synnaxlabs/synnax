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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

type Retrieve struct {
	baseTX   gorp.Tx
	gorp     gorp.Retrieve[Key, Policy]
	indexes  indexes
	ontology *ontology.Ontology
}

// ResolveSubjects walks the ontology from each subject up to its parent roles and back
// down to the policies attached to those roles, returning the policy keys. A subject
// that does not exist has no policies.
func (s *Service) ResolveSubjects(
	ctx context.Context,
	tx gorp.Tx,
	subjects ...resource.ID,
) ([]Key, error) {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return resolveSubjects(ctx, tx, s.cfg.Ontology.NewRetrieve().WhereIDs(subjects...))
}

// MatchSubjects returns a filter that matches the policies of the roles that any of the
// subjects hold. A subject that does not exist holds no roles.
func MatchSubjects(subjects ...resource.ID) Filter {
	return func(r Retrieve) gorp.Filter[Key, Policy] {
		return gorp.MatchKeysFunc[Key, Policy](func(ctx gorp.Context) ([]Key, error) {
			start := r.ontology.NewRetrieve().WhereIDs(subjects...).SkipMissingIDs()
			return resolveSubjects(ctx, ctx.Tx, start)
		})
	}
}

func resolveSubjects(
	ctx context.Context,
	tx gorp.Tx,
	subjects ontology.Retrieve,
) ([]Key, error) {
	var policyResources []ontology.Resource
	if err := subjects.
		ExcludeFieldData(true).
		TraverseTo(ontology.ParentsTraverser).
		WhereTypes(resource.TypeRole).
		TraverseTo(ontology.ChildrenTraverser).
		WhereTypes(resource.TypePolicy).
		Entries(&policyResources).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	return KeysFromResourceIDs(ontology.ResourceIDs(policyResources))
}
