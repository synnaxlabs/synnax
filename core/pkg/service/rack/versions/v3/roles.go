// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"
	"slices"
	"strconv"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	role "github.com/synnaxlabs/synnax/pkg/service/access/rbac/role/versions/v0"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

// RolesMigration adds each stored role to the roles of the stored racks that a parent
// edge from the role points at, then deletes every parent edge from a role to a rack.
// An edge from a deleted role adds nothing.
var RolesMigration = migrate.New("v61_rack_roles", migrateRoles)

func migrateRoles(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	edges, err := collect(ctx, tx, func(r ontology.Relationship) bool {
		return r.Type == ontology.RelationshipTypeParentOf &&
			r.From.Type == resource.TypeRole &&
			r.To.Type == resource.TypeRack
	})
	if err != nil || len(edges) == 0 {
		return err
	}
	racks, err := collect(ctx, tx, func(Rack) bool { return true })
	if err != nil {
		return err
	}
	byKey := make(map[Key]*Rack, len(racks))
	for i := range racks {
		byKey[racks[i].Key] = &racks[i]
	}
	roles := gorp.WrapReader[role.Key, role.Role](tx)
	changed := make(set.Set[Key])
	for _, e := range edges {
		r, err := uuid.Parse(e.From.Key)
		if err != nil {
			return errors.Wrapf(err, "role of edge %s", e.GorpKey())
		}
		k, err := strconv.ParseUint(e.To.Key, 10, 32)
		if err != nil {
			return errors.Wrapf(err, "rack of edge %s", e.GorpKey())
		}
		rck, ok := byKey[Key(k)]
		if !ok || slices.Contains(rck.Roles, r) {
			continue
		}
		if _, err = roles.Get(ctx, r); errors.Is(err, query.ErrNotFound) {
			continue
		} else if err != nil {
			return err
		}
		rck.Roles = append(rck.Roles, r)
		changed.Add(rck.Key)
	}
	updated := make([]Rack, 0, len(changed))
	for k := range changed {
		updated = append(updated, *byKey[k])
	}
	if err = gorp.WrapWriter[Key, Rack](tx).Set(ctx, updated...); err != nil {
		return err
	}
	keys := make([]string, len(edges))
	for i, e := range edges {
		keys[i] = e.GorpKey()
	}
	return gorp.WrapWriter[string, ontology.Relationship](tx).Delete(ctx, keys...)
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
