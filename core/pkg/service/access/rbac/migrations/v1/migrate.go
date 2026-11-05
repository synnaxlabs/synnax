// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"slices"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

// MigrationConfig is the configuration for NewMigration.
type MigrationConfig struct {
	// User writes the roles of each migrated user.
	User *user.Service
}

// NewMigration returns a migration that adds each stored role to the roles of the
// stored users that a parent edge from the role points at, then deletes every parent
// edge from a role to a user. An edge from a deleted role adds nothing.
func NewMigration(cfg MigrationConfig) gorp.Migration {
	return migrate.New(
		"v61_user_roles",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			edges, err := collect(ctx, tx, func(r ontology.Relationship) bool {
				return r.Type == ontology.RelationshipTypeParentOf &&
					r.From.Type == resource.TypeRole &&
					r.To.Type == resource.TypeUser
			})
			if err != nil || len(edges) == 0 {
				return err
			}
			if err = assignRoles(ctx, tx, cfg.User, edges); err != nil {
				return err
			}
			keys := make([]string, len(edges))
			for i, e := range edges {
				keys[i] = e.GorpKey()
			}
			return gorp.WrapWriter[string, ontology.Relationship](tx).
				Delete(ctx, keys...)
		},
	)
}

func assignRoles(
	ctx context.Context,
	tx gorp.Tx,
	users *user.Service,
	edges []ontology.Relationship,
) error {
	var stored []user.User
	if err := users.NewRetrieve().Entries(&stored).Exec(ctx, tx); err != nil {
		return err
	}
	byKey := make(map[user.Key]*user.User, len(stored))
	for i := range stored {
		byKey[stored[i].Key] = &stored[i]
	}
	roles := gorp.WrapReader[role.Key, role.Role](tx)
	changed := make(set.Set[user.Key])
	for _, e := range edges {
		r, err := uuid.Parse(e.From.Key)
		if err != nil {
			return errors.Wrapf(err, "role of edge %s", e.GorpKey())
		}
		k, err := uuid.Parse(e.To.Key)
		if err != nil {
			return errors.Wrapf(err, "user of edge %s", e.GorpKey())
		}
		u, ok := byKey[k]
		if !ok || slices.Contains(u.Roles, r) {
			continue
		}
		if _, err = roles.Get(ctx, r); errors.Is(err, query.ErrNotFound) {
			continue
		} else if err != nil {
			return err
		}
		u.Roles = append(u.Roles, r)
		changed.Add(k)
	}
	w := users.NewWriter(tx)
	for k := range changed {
		if err := w.SetRoles(ctx, byKey[k].Roles, k); err != nil {
			return err
		}
	}
	return nil
}

func collect[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	tx gorp.Tx,
	keep func(E) bool,
) (out []E, err error) {
	iter, err := gorp.WrapReader[K, E](tx).OpenIterator(gorp.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		if e := iter.Value(ctx); e != nil && keep(*e) {
			out = append(out, *e)
		}
	}
	return out, iter.Error()
}
