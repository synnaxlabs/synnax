// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v1 gives each stored user one role.
package v1

import (
	"context"
	"slices"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// MigrationConfig is the configuration for NewMigration.
type MigrationConfig struct {
	// User holds the users that the migration gives a role.
	User *user.Service
	// Role holds the roles that the users held.
	Role *role.Service
	// Policy holds the policies of those roles.
	Policy *policy.Service
	// Owner is the key of the Owner role, which each root user gets.
	Owner role.Key
}

// NoAccessKey is the key of the role that the migration gives each user that held no
// role.
var NoAccessKey = uuid.MustParse("3f6b2a1e-8c4d-4e7a-9b5f-1d2c3e4f5a6b")

// NewMigration returns the migration that gives one role to each stored user with no
// role. A root user gets the Owner role. A user that held roles through ontology edges
// keeps the one that keeps the most of its access, and a tie goes to the lowest key. A
// user that held no role gets the No access role, which grants nothing, so the upgrade
// changes no user's access. The migration deletes every edge from a role to a user.
func NewMigration(cfg MigrationConfig) gorp.Migration {
	return migrate.New(
		"v1.user_role",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			return migrateUsers(ctx, tx, cfg)
		},
	)
}

func migrateUsers(ctx context.Context, tx gorp.Tx, cfg MigrationConfig) error {
	edges, err := ontology.RetrieveRelationships(
		ctx,
		tx,
		func(r ontology.Relationship) bool {
			return r.Type == ontology.RelationshipTypeParentOf &&
				r.From.Type == resource.TypeRole &&
				r.To.Type == resource.TypeUser
		},
	)
	if err != nil {
		return err
	}
	held, err := heldRoles(ctx, tx, cfg.Role, edges)
	if err != nil {
		return err
	}
	var users []user.User
	if err = cfg.User.NewRetrieve().Entries(&users).Exec(ctx, tx); err != nil {
		return err
	}
	byRole := make(map[role.Key][]user.Key)
	for _, u := range users {
		if u.Role != uuid.Nil() {
			continue
		}
		r, err := pickRole(ctx, tx, cfg, u, held[u.Key])
		if err != nil {
			return err
		}
		byRole[r] = append(byRole[r], u.Key)
	}
	if _, ok := byRole[NoAccessKey]; ok {
		if err = cfg.Role.NewWriter(tx, false).Create(ctx, &role.Role{
			Key:  NoAccessKey,
			Name: "No access",
			Description: "Grants no permissions. Holds the users that had no role " +
				"before the upgrade.",
		}); err != nil {
			return err
		}
	}
	w := cfg.User.NewWriter(tx)
	for r, keys := range byRole {
		if err = w.SetRole(ctx, r, nil, keys...); err != nil {
			return err
		}
	}
	return ontology.DeleteRelationships(ctx, tx, edges...)
}

// heldRoles returns the keys of the roles that each user holds through edges, leaving
// out the roles that no longer exist.
func heldRoles(
	ctx context.Context,
	tx gorp.Tx,
	roles *role.Service,
	edges []ontology.Relationship,
) (map[user.Key][]role.Key, error) {
	held := make(map[user.Key][]role.Key)
	for _, e := range edges {
		u, err := uuid.Parse(e.To.Key)
		if err != nil {
			return nil, errors.Wrapf(err, "user of edge %s", e.GorpKey())
		}
		r, err := uuid.Parse(e.From.Key)
		if err != nil {
			return nil, errors.Wrapf(err, "role of edge %s", e.GorpKey())
		}
		exists, err := roles.NewRetrieve().Where(role.MatchKeys(r)).Exists(ctx, tx)
		if err != nil {
			return nil, err
		}
		if exists && !slices.Contains(held[u], r) {
			held[u] = append(held[u], r)
		}
	}
	return held, nil
}

func pickRole(
	ctx context.Context,
	tx gorp.Tx,
	cfg MigrationConfig,
	u user.User,
	held []role.Key,
) (role.Key, error) {
	switch {
	case u.RootUser:
		return cfg.Owner, nil
	case len(held) == 0:
		return NoAccessKey, nil
	case len(held) == 1:
		return held[0], nil
	}
	var policies []policy.Policy
	if err := cfg.Policy.NewRetrieve().
		Where(policy.MatchRoles(held...)).
		Entries(&policies).
		Exec(ctx, tx); err != nil {
		return uuid.Nil(), err
	}
	return mostAccess(held, policies), nil
}

type grant struct {
	action access.Action
	object resource.ID
}

// mostAccess returns the role in held whose policies cover the most of the grants that
// all of held's policies give together. A tie goes to the lowest key.
func mostAccess(held []role.Key, policies []policy.Policy) role.Key {
	all := make(set.Set[grant])
	for _, p := range policies {
		for _, a := range p.Actions {
			for _, o := range p.Objects {
				all.Add(grant{action: a, object: o})
			}
		}
	}
	byRole := lo.GroupBy(policies, func(p policy.Policy) role.Key { return p.Role })
	slices.SortFunc(held, func(a, b role.Key) int { return a.Compare(b) })
	best, bestScore := held[0], -1
	for _, r := range held {
		score := 0
		for g := range all {
			if covers(byRole[r], g) {
				score++
			}
		}
		if score > bestScore {
			best, bestScore = r, score
		}
	}
	return best
}

// covers reports whether a policy allows the grant's action on its object, by the
// object's ID or by its type.
func covers(policies []policy.Policy, g grant) bool {
	return slices.ContainsFunc(policies, func(p policy.Policy) bool {
		return slices.Contains(p.Actions, g.action) &&
			slices.ContainsFunc(p.Objects, func(o resource.ID) bool {
				return o.Type == g.object.Type && (o.IsType() || o.Key == g.object.Key)
			})
	})
}
