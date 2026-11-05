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
	"crypto/sha1"
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
)

// Migration re-encodes each stored policy with an empty role.
var Migration = migrate.NewEntry("v61_add_role", autoMigratePolicy)

// RoleMigration sets the role of each stored policy from the parent edges that point
// at it from stored roles, then deletes every parent edge from a role to a policy. A
// policy under more than one role gets a copy under each role after the first, keyed
// by CopyKey. A policy under no stored role grants nothing, so the migration deletes
// it.
var RoleMigration = migrate.New("v61_policy_role", migrateRoles)

// copyNamespace is the UUIDv5 namespace of the keys that CopyKey derives. Changing it
// gives each node a different key for the same copy.
var copyNamespace = uuid.MustParse("6f1d2a8c-3b4e-5c7a-9d21-8e4f0b6a1c35")

// CopyKey returns the key of the copy of the policy with key policy that the migration
// makes under the role with key r.
func CopyKey(policy Key, r role.Key) Key {
	h := sha1.New()
	h.Write(copyNamespace[:])
	h.Write(policy[:])
	h.Write(r[:])
	var k uuid.UUID
	copy(k[:], h.Sum(nil))
	k[6] = (k[6] & 0x0f) | 0x50
	k[8] = (k[8] & 0x3f) | 0x80
	return k
}

func migrateRoles(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	edges, err := collect(ctx, tx, func(r ontology.Relationship) bool {
		return r.Type == ontology.RelationshipTypeParentOf &&
			r.From.Type == resource.TypeRole &&
			r.To.Type == resource.TypePolicy
	})
	if err != nil {
		return err
	}
	policies, err := collect(ctx, tx, func(Policy) bool { return true })
	if err != nil || (len(policies) == 0 && len(edges) == 0) {
		return err
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
				c.Key = CopyKey(p.Key, r)
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
