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
	role "github.com/synnaxlabs/synnax/pkg/service/access/rbac/role/versions/v0"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	v1 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v1"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func parentOf(from, to resource.ID) ontology.Relationship {
	return ontology.Relationship{
		From: from,
		To:   to,
		Type: ontology.RelationshipTypeParentOf,
	}
}

func roleID(k role.Key) resource.ID {
	return resource.ID{Type: resource.TypeRole, Key: k.String()}
}

var _ = Describe("Roles migration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	createRole := func(ctx SpecContext) role.Key {
		GinkgoHelper()
		r := role.Role{Key: uuid.New(), Name: "role"}
		Expect(gorp.WrapWriter[role.Key, role.Role](db).Set(ctx, r)).To(Succeed())
		return r.Key
	}
	createUser := func(ctx SpecContext, held ...role.Key) v1.User {
		GinkgoHelper()
		u := v1.User{Key: uuid.New(), Name: "user", Roles: held}
		Expect(gorp.WrapWriter[v1.Key, v1.User](db).Set(ctx, u)).To(Succeed())
		return u
	}
	setEdges := func(ctx SpecContext, edges ...ontology.Relationship) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[string, ontology.Relationship](db).
			Set(ctx, edges...)).To(Succeed())
	}
	migrate := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "User",
			Migrations: []gorp.Migration{v1.RolesMigration},
		})).To(Succeed())
	}
	rolesOf := func(ctx SpecContext, key v1.Key) []role.Key {
		GinkgoHelper()
		var u v1.User
		Expect(gorp.NewRetrieve[v1.Key, v1.User]().
			Where(gorp.MatchKeys[v1.Key, v1.User](key)).
			Entry(&u).
			Exec(ctx, db)).To(Succeed())
		return u.Roles
	}
	edgeExists := func(ctx SpecContext, edge ontology.Relationship) bool {
		GinkgoHelper()
		reader := gorp.WrapReader[string, ontology.Relationship](db)
		_, err := reader.Get(ctx, edge.GorpKey())
		if err != nil {
			Expect(err).To(MatchError(query.ErrNotFound))
		}
		return err == nil
	}

	It("Should give each user the roles that point at it", func(ctx SpecContext) {
		a, b := createRole(ctx), createRole(ctx)
		u := createUser(ctx)
		edges := []ontology.Relationship{
			parentOf(roleID(a), u.ResourceID()),
			parentOf(roleID(b), u.ResourceID()),
		}
		setEdges(ctx, edges...)
		migrate(ctx)
		Expect(rolesOf(ctx, u.Key)).To(ConsistOf(a, b))
		for _, e := range edges {
			Expect(edgeExists(ctx, e)).To(BeFalse())
		}
	})

	It("Should keep the roles a user already holds once each", func(ctx SpecContext) {
		held, added := createRole(ctx), createRole(ctx)
		u := createUser(ctx, held)
		setEdges(
			ctx,
			parentOf(roleID(held), u.ResourceID()),
			parentOf(roleID(added), u.ResourceID()),
		)
		migrate(ctx)
		Expect(rolesOf(ctx, u.Key)).To(Equal([]role.Key{held, added}))
	})

	It("Should drop an edge from a deleted role", func(ctx SpecContext) {
		u := createUser(ctx)
		edge := parentOf(roleID(uuid.New()), u.ResourceID())
		setEdges(ctx, edge)
		migrate(ctx)
		Expect(rolesOf(ctx, u.Key)).To(BeEmpty())
		Expect(edgeExists(ctx, edge)).To(BeFalse())
	})

	It("Should leave edges from a role to other resources", func(ctx SpecContext) {
		edge := parentOf(
			roleID(createRole(ctx)),
			resource.ID{Type: resource.TypeRack, Key: "1"},
		)
		setEdges(ctx, edge)
		migrate(ctx)
		Expect(edgeExists(ctx, edge)).To(BeTrue())
	})
})
