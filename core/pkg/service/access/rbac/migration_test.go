// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac_test

import (
	"context"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	policyversions "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/versions"
	policyv1 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	roleversions "github.com/synnaxlabs/synnax/pkg/service/access/rbac/role/versions"
	rolev0 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/role/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	ontologyv0 "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	rackversions "github.com/synnaxlabs/synnax/pkg/service/rack/versions"
	rackv3 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	userversions "github.com/synnaxlabs/synnax/pkg/service/user/versions"
	userv1 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/types"
)

// markApplied records each migration in chain as applied in namespace without running
// it, unless its key is in pending, as on a store from before the pending migrations.
func markApplied(
	ctx context.Context,
	db *gorp.DB,
	namespace string,
	chain []gorp.Migration,
	pending ...string,
) {
	GinkgoHelper()
	noop := func(context.Context, gorp.Tx, alamos.Instrumentation) error { return nil }
	var applied []gorp.Migration
	for _, m := range chain {
		if !slices.Contains(pending, m.Key()) {
			applied = append(applied, migrate.New(m.Key(), noop))
		}
	}
	Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
		DB:         db,
		Namespace:  namespace,
		Migrations: applied,
	})).To(Succeed())
}

func parentOf(from, to resource.ID) ontologyv0.Relationship {
	return ontologyv0.Relationship{
		From: from,
		To:   to,
		Type: ontologyv0.RelationshipTypeParentOf,
	}
}

var _ = Describe("Migration chain", func() {
	It("Should move every role edge into a field in the open order of a Core", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		a := rolev0.Role{Key: uuid.New(), Name: "a"}
		b := rolev0.Role{Key: uuid.New(), Name: "b"}
		Expect(gorp.WrapWriter[rolev0.Key, rolev0.Role](db).Set(ctx, a, b)).
			To(Succeed())
		u := userv1.User{Key: uuid.New(), Name: "u"}
		Expect(gorp.WrapWriter[userv1.Key, userv1.User](db).Set(ctx, u)).
			To(Succeed())
		rck := rackv3.Rack{Key: rack.NewKey(1, 10), Name: "r"}
		Expect(gorp.WrapWriter[rackv3.Key, rackv3.Rack](db).Set(ctx, rck)).
			To(Succeed())
		newPolicy := func(name string) policyv1.Policy {
			return policyv1.Policy{
				Key:     uuid.New(),
				Name:    name,
				Objects: []resource.ID{{Type: resource.TypeChannel}},
				Actions: []access.Action{access.ActionRetrieve},
			}
		}
		shared, orphan := newPolicy("shared"), newPolicy("orphan")
		Expect(gorp.WrapWriter[policyv1.Key, policyv1.Policy](db).
			Set(ctx, shared, orphan)).To(Succeed())
		roleID := func(r rolev0.Role) resource.ID { return role.ResourceID(r.Key) }
		Expect(gorp.WrapWriter[string, ontologyv0.Relationship](db).Set(
			ctx,
			parentOf(roleID(a), u.ResourceID()),
			parentOf(roleID(a), rck.ResourceID()),
			parentOf(roleID(a), policy.ResourceID(shared.Key)),
			parentOf(roleID(b), policy.ResourceID(shared.Key)),
		)).To(Succeed())
		markApplied(ctx, db, types.Name[role.Role](), roleversions.Migrations)
		markApplied(
			ctx,
			db,
			types.Name[policy.Policy](),
			policyversions.Migrations,
			"v61_add_role",
			"v61_policy_role",
		)
		markApplied(
			ctx,
			db,
			types.Name[user.User](),
			userversions.Migrations,
			"v61_user_roles",
		)
		markApplied(
			ctx,
			db,
			types.Name[rack.Rack](),
			rackversions.NewMigrations(rackversions.MigrationsConfig{}),
			"v61_rack_roles",
		)
		markApplied(
			ctx,
			db,
			"rack.roles",
			[]gorp.Migration{migrate.New("v0.driver_role", nil)},
		)
		markApplied(
			ctx,
			db,
			"RBAC",
			[]gorp.Migration{migrate.New("v0.permission_assignment", nil)},
		)

		roles := MustOpen(rbac.OpenRoles(ctx, rbac.RolesConfig{DB: db}))
		passwords := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
		users := MustOpen(user.OpenService(ctx, user.ServiceConfig{
			DB:       db,
			Password: passwords,
		}))
		statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: db,
		}))
		racks := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:           db,
			HostProvider: testutil.NewStaticHostProvider(1),
			Status:       statuses,
			DriverRole:   roles.Builtin.DriverKey,
		}))
		svc := MustSucceed(rbac.NewService(ctx, rbac.ServiceConfig{
			DB:    db,
			Roles: roles,
			User:  users,
			Rack:  racks,
		}))

		Expect(svc.RolesOf(ctx, nil, u.ResourceID())).To(Equal([]role.Key{a.Key}))
		Expect(svc.RolesOf(ctx, nil, rck.ResourceID())).To(Equal([]role.Key{a.Key}))
		var policies []policy.Policy
		Expect(svc.Policy.NewRetrieve().
			Where(policy.MatchNames("shared", "orphan")).
			Entries(&policies).
			Exec(ctx, nil)).To(Succeed())
		Expect(
			lo.Map(policies, func(p policy.Policy, _ int) role.Key { return p.Role }),
		).
			To(ConsistOf(a.Key, b.Key))
		Expect(svc.Policy.NewRetrieve().
			Where(policy.MatchRoles(svc.Builtin.OwnerKey)).
			Exists(ctx, nil)).To(BeTrue())
		edges := MustSucceed(ontologyv0.RetrieveRelationships(
			ctx,
			db,
			func(rel ontologyv0.Relationship) bool {
				return rel.From.Type == resource.TypeRole
			},
		))
		Expect(edges).To(BeEmpty())
		req := access.Request{
			Subject: u.ResourceID(),
			Action:  access.ActionRetrieve,
			Objects: []resource.ID{{Type: resource.TypeChannel, Key: "1"}},
		}
		Expect(svc.NewEnforcer(nil).Enforce(ctx, req)).To(Succeed())
		req.Subject = rck.ResourceID()
		Expect(svc.NewEnforcer(nil).Enforce(ctx, req)).To(Succeed())
	})
})
