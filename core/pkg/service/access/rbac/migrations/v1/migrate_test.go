// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/builtin"
	v1 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/migrations/v1"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("User role migration", func() {
	var (
		db       *gorp.DB
		users    *user.Service
		roles    *role.Service
		policies *policy.Service
		builtins builtin.ProvisionResult
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		passwords := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
		users = MustOpen(user.OpenService(ctx, user.ServiceConfig{
			DB:       db,
			Password: passwords,
		}))
		roles = MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		policies = MustOpen(policy.OpenService(ctx, policy.ServiceConfig{
			DB:   db,
			Role: roles,
		}))
		builtins = MustSucceed(builtin.Provision(ctx, db, policies, roles))
	})

	// storeUser writes u raw, as a v0.60 Core stored it: with no role.
	storeUser := func(ctx SpecContext, u user.User) user.User {
		GinkgoHelper()
		u.Key = uuid.New()
		Expect(gorp.WrapWriter[user.Key, user.User](db).Set(ctx, u)).To(Succeed())
		return u
	}
	createRole := func(ctx SpecContext, key role.Key, grants ...policy.Policy) role.Key {
		GinkgoHelper()
		r := role.Role{Key: key, Name: "role"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return roles.NewWriter(tx, false).Create(ctx, &r)
		})).To(Succeed())
		for _, p := range grants {
			p.Role = r.Key
			p.Name = "policy"
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return policies.NewWriter(tx, false).Create(ctx, &p)
			})).To(Succeed())
		}
		return r.Key
	}
	allow := func(action access.Action, objects ...resource.ID) policy.Policy {
		return policy.Policy{Actions: []access.Action{action}, Objects: objects}
	}
	channels := resource.ID{Type: resource.TypeChannel}
	channel := func(key string) resource.ID {
		return resource.ID{Type: resource.TypeChannel, Key: key}
	}
	storeEdges := func(ctx SpecContext, u user.User, held ...role.Key) {
		GinkgoHelper()
		for _, r := range held {
			Expect(gorp.WrapWriter[string, ontology.Relationship](db).Set(
				ctx,
				ontology.Relationship{
					From: role.ResourceID(r),
					To:   u.ResourceID(),
					Type: ontology.RelationshipTypeParentOf,
				},
			)).To(Succeed())
		}
	}
	migrate := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "RBAC",
			Migrations: []gorp.Migration{v1.NewMigration(v1.MigrationConfig{
				User:   users,
				Role:   roles,
				Policy: policies,
				Owner:  builtins.OwnerKey,
			})},
		})).To(Succeed())
	}
	roleOf := func(ctx SpecContext, u user.User) role.Key {
		GinkgoHelper()
		var stored user.User
		Expect(users.NewRetrieve().
			Where(user.MatchKeys(u.Key)).
			Entry(&stored).
			Exec(ctx, nil)).To(Succeed())
		return stored.Role
	}
	noAccessExists := func(ctx SpecContext) bool {
		GinkgoHelper()
		return MustSucceed(roles.NewRetrieve().
			Where(role.MatchKeys(v1.NoAccessKey)).
			Exists(ctx, nil))
	}

	It("Should give each root user the Owner role", func(ctx SpecContext) {
		root := storeUser(ctx, user.User{Name: "root", RootUser: true})
		storeEdges(ctx, root, builtins.ViewerKey)
		migrate(ctx)
		Expect(roleOf(ctx, root)).To(Equal(builtins.OwnerKey))
	})

	It("Should give a user the one role it held", func(ctx SpecContext) {
		u := storeUser(ctx, user.User{Name: "u"})
		storeEdges(ctx, u, builtins.EngineerKey)
		migrate(ctx)
		Expect(roleOf(ctx, u)).To(Equal(builtins.EngineerKey))
	})

	It("Should keep the held role that grants the most access", func(ctx SpecContext) {
		narrow := createRole(
			ctx,
			uuid.Nil(),
			allow(access.ActionRetrieve, channel("1")),
		)
		wide := createRole(ctx, uuid.Nil(), allow(access.ActionRetrieve, channels))
		u := storeUser(ctx, user.User{Name: "u"})
		storeEdges(ctx, u, narrow, wide)
		migrate(ctx)
		Expect(roleOf(ctx, u)).To(Equal(wide))
	})

	It("Should count each action that a role grants", func(ctx SpecContext) {
		reads := createRole(ctx, uuid.Nil(), allow(access.ActionRetrieve, channels))
		writes := createRole(
			ctx,
			uuid.Nil(),
			allow(access.ActionUpdate, channels),
			allow(access.ActionDelete, channels),
		)
		u := storeUser(ctx, user.User{Name: "u"})
		storeEdges(ctx, u, reads, writes)
		migrate(ctx)
		Expect(roleOf(ctx, u)).To(Equal(writes))
	})

	It("Should break a tie with the lowest role key", func(ctx SpecContext) {
		grant := allow(access.ActionRetrieve, channels)
		high := createRole(
			ctx,
			uuid.MustParse("ffffffff-0000-4000-8000-000000000000"),
			grant,
		)
		low := createRole(
			ctx,
			uuid.MustParse("00000000-0000-4000-8000-000000000001"),
			grant,
		)
		u := storeUser(ctx, user.User{Name: "u"})
		storeEdges(ctx, u, high, low)
		migrate(ctx)
		Expect(roleOf(ctx, u)).To(Equal(low))
	})

	It(
		"Should give a user that held no role the No access role",
		func(ctx SpecContext) {
			u := storeUser(ctx, user.User{Name: "u"})
			migrate(ctx)
			Expect(roleOf(ctx, u)).To(Equal(v1.NoAccessKey))
			var noAccess role.Role
			Expect(roles.NewRetrieve().
				Where(role.MatchKeys(v1.NoAccessKey)).
				Entry(&noAccess).
				Exec(ctx, nil)).To(Succeed())
			Expect(noAccess.Name).To(Equal("No access"))
			Expect(noAccess.Internal).To(BeFalse())
			Expect(policies.NewRetrieve().
				Where(policy.MatchRoles(v1.NoAccessKey)).
				Exists(ctx, nil)).To(BeFalse())
		},
	)

	It("Should not create the No access role when every user held a role", func(
		ctx SpecContext,
	) {
		u := storeUser(ctx, user.User{Name: "u"})
		storeEdges(ctx, u, builtins.ViewerKey)
		migrate(ctx)
		Expect(noAccessExists(ctx)).To(BeFalse())
	})

	It(
		"Should ignore an edge from a role that no longer exists",
		func(ctx SpecContext) {
			u := storeUser(ctx, user.User{Name: "u"})
			storeEdges(ctx, u, uuid.New())
			migrate(ctx)
			Expect(roleOf(ctx, u)).To(Equal(v1.NoAccessKey))
		},
	)

	It("Should keep the role of a user that already holds one", func(ctx SpecContext) {
		var u user.User
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			u, err = users.NewWriter(tx).Create(ctx, user.User{
				Name: "u",
				Role: builtins.OperatorKey,
			})
			return err
		})).To(Succeed())
		storeEdges(ctx, u, builtins.OwnerKey)
		migrate(ctx)
		Expect(roleOf(ctx, u)).To(Equal(builtins.OperatorKey))
	})

	It("Should delete each edge from a role to a user", func(ctx SpecContext) {
		u := storeUser(ctx, user.User{Name: "u"})
		storeEdges(ctx, u, builtins.ViewerKey, builtins.EngineerKey)
		kept := ontology.Relationship{
			From: resource.ID{Type: resource.TypeGroup, Key: uuid.New().String()},
			To:   u.ResourceID(),
			Type: ontology.RelationshipTypeParentOf,
		}
		Expect(gorp.WrapWriter[string, ontology.Relationship](db).Set(ctx, kept)).
			To(Succeed())
		migrate(ctx)
		Expect(ontology.RetrieveRelationships(
			ctx,
			db,
			func(ontology.Relationship) bool { return true },
		)).To(ConsistOf(kept))
	})
})
