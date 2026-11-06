// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"
	"slices"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	role "github.com/synnaxlabs/synnax/pkg/service/access/rbac/role/versions/v0"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	xuuid "github.com/synnaxlabs/x/uuid"
)

// Migration re-encodes each stored policy with an empty role.
var Migration = migrate.NewEntry("v61_add_role", autoMigratePolicy)

// RoleMigration moves each policy's parent role edge into its role field and deletes
// the edge. A policy under several roles is copied under each extra role, with a key
// that every node derives the same way from the policy and the role. A policy under no
// role is deleted.
var RoleMigration = migrate.New("v61_policy_role", migrateRoles)

// copyNamespace is the UUIDv5 namespace of the keys that copyKey derives. Changing it
// gives each node a different key for the same copy.
var copyNamespace = uuid.MustParse("6f1d2a8c-3b4e-5c7a-9d21-8e4f0b6a1c35")

func copyKey(policy Key, r role.Key) Key {
	return xuuid.NewV5(copyNamespace, policy.String()+"/"+r.String())
}

func migrateRoles(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	edges, err := ontology.RetrieveRelationships(
		ctx,
		tx,
		func(r ontology.Relationship) bool {
			return r.Type == ontology.RelationshipTypeParentOf &&
				r.From.Type == resource.TypeRole &&
				r.To.Type == resource.TypePolicy
		},
	)
	if err != nil {
		return err
	}
	var policies []Policy
	if err = gorp.NewRetrieve[Key, Policy]().Entries(&policies).
		Exec(ctx, tx); err != nil {
		return err
	}
	if len(policies) == 0 && len(edges) == 0 {
		return nil
	}
	roles := gorp.WrapReader[role.Key, role.Role](tx)
	byPolicy := make(map[Key][]role.Key, len(policies))
	for _, e := range edges {
		p, err := uuid.Parse(e.To.Key)
		if err != nil {
			return errors.Wrapf(err, "policy of edge %s", e.GorpKey())
		}
		r, err := uuid.Parse(e.From.Key)
		if err != nil {
			return errors.Wrapf(err, "role of edge %s", e.GorpKey())
		}
		if _, err = roles.Get(ctx, r); errors.Is(err, query.ErrNotFound) {
			continue
		} else if err != nil {
			return err
		}
		byPolicy[p] = append(byPolicy[p], r)
	}
	var (
		set     []Policy
		deleted []Key
	)
	for _, p := range policies {
		held := byPolicy[p.Key]
		if len(held) == 0 {
			deleted = append(deleted, p.Key)
			continue
		}
		slices.SortFunc(held, func(a, b role.Key) int { return a.Compare(b) })
		held = slices.Compact(held)
		for i, r := range held {
			c := p
			c.Role = r
			c.Objects = slices.Clone(p.Objects)
			c.Actions = slices.Clone(p.Actions)
			if i > 0 {
				c.Key = copyKey(p.Key, r)
			}
			set = append(set, c)
		}
	}
	if err = gorp.WrapWriter[Key, Policy](tx).Set(ctx, set...); err != nil {
		return err
	}
	if err = gorp.WrapWriter[Key, Policy](tx).Delete(ctx, deleted...); err != nil {
		return err
	}
	return ontology.DeleteRelationships(ctx, tx, edges...)
}
