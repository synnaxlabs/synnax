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
	v1 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/migrations/v1"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	ontologyv0 "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func parentOf(from, to resource.ID) ontologyv0.Relationship {
	return ontologyv0.Relationship{
		From: from,
		To:   to,
		Type: ontologyv0.RelationshipTypeParentOf,
	}
}

var _ = Describe("User roles migration", func() {
	var (
		db    *gorp.DB
		users *user.Service
		roles *role.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
		users = MustOpen(user.OpenService(ctx, user.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
		roles = MustOpen(role.OpenService(ctx, role.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
	})
	createRole := func(ctx SpecContext, name string) role.Role {
		GinkgoHelper()
		r := role.Role{Name: name}
		Expect(roles.NewWriter(nil, true).Create(ctx, &r)).To(Succeed())
		return r
	}
	createUser := func(ctx SpecContext, held ...role.Key) user.User {
		GinkgoHelper()
		return MustSucceed(users.NewWriter(nil).Create(ctx, user.User{
			Name:  "user-" + uuid.New().String(),
			Roles: held,
		}))
	}
	setEdges := func(ctx SpecContext, edges ...ontologyv0.Relationship) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[string, ontologyv0.Relationship](db).
			Set(ctx, edges...)).To(Succeed())
	}
	migrate := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "RBAC",
			Migrations: []gorp.Migration{
				v1.NewMigration(v1.MigrationConfig{User: users}),
			},
		})).To(Succeed())
	}
	rolesOf := func(ctx SpecContext, key user.Key) []role.Key {
		GinkgoHelper()
		var u user.User
		Expect(users.NewRetrieve().
			Where(user.MatchKeys(key)).
			Entry(&u).
			Exec(ctx, nil)).To(Succeed())
		return u.Roles
	}
	edgeExists := func(ctx SpecContext, edge ontologyv0.Relationship) bool {
		GinkgoHelper()
		reader := gorp.WrapReader[string, ontologyv0.Relationship](db)
		_, err := reader.Get(ctx, edge.GorpKey())
		if err != nil {
			Expect(err).To(MatchError(query.ErrNotFound))
		}
		return err == nil
	}

	It("Should give each user the roles that point at it", func(ctx SpecContext) {
		engineer, operator := createRole(ctx, "Engineer"), createRole(ctx, "Operator")
		u := createUser(ctx)
		edges := []ontologyv0.Relationship{
			parentOf(role.ResourceID(engineer.Key), u.ResourceID()),
			parentOf(role.ResourceID(operator.Key), u.ResourceID()),
		}
		setEdges(ctx, edges...)
		migrate(ctx)
		Expect(rolesOf(ctx, u.Key)).To(ConsistOf(engineer.Key, operator.Key))
		for _, e := range edges {
			Expect(edgeExists(ctx, e)).To(BeFalse())
		}
	})

	It("Should keep the roles a user already holds once each", func(ctx SpecContext) {
		owner, viewer := createRole(ctx, "Owner"), createRole(ctx, "Viewer")
		u := createUser(ctx, owner.Key)
		setEdges(
			ctx,
			parentOf(role.ResourceID(owner.Key), u.ResourceID()),
			parentOf(role.ResourceID(viewer.Key), u.ResourceID()),
		)
		migrate(ctx)
		Expect(rolesOf(ctx, u.Key)).To(Equal([]role.Key{owner.Key, viewer.Key}))
	})

	It("Should drop an edge from a deleted role", func(ctx SpecContext) {
		u := createUser(ctx)
		edge := parentOf(role.ResourceID(uuid.New()), u.ResourceID())
		setEdges(ctx, edge)
		migrate(ctx)
		Expect(rolesOf(ctx, u.Key)).To(BeEmpty())
		Expect(edgeExists(ctx, edge)).To(BeFalse())
	})

	It("Should leave edges from a role to other resources", func(ctx SpecContext) {
		r := createRole(ctx, "Driver")
		edge := parentOf(
			role.ResourceID(r.Key),
			resource.ID{Type: resource.TypeRack, Key: "1"},
		)
		setEdges(ctx, edge)
		migrate(ctx)
		Expect(edgeExists(ctx, edge)).To(BeTrue())
	})
})
