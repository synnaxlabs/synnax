// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	role "github.com/synnaxlabs/synnax/pkg/service/access/rbac/role/versions/v0"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	v3 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("RolesMigration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	createRole := func(ctx SpecContext) role.Key {
		GinkgoHelper()
		r := role.Role{Key: uuid.New(), Name: "role"}
		Expect(gorp.NewCreate[role.Key, role.Role]().Entry(&r).Exec(ctx, db)).
			To(Succeed())
		return r.Key
	}
	parentOf := func(r role.Key, k v3.Key) ontology.Relationship {
		return ontology.Relationship{
			From: resource.ID{Type: resource.TypeRole, Key: r.String()},
			To:   k.ResourceID(),
			Type: ontology.RelationshipTypeParentOf,
		}
	}
	migrate := func(ctx SpecContext, rck v3.Rack, edges ...ontology.Relationship) v3.Rack {
		GinkgoHelper()
		Expect(gorp.NewCreate[v3.Key, v3.Rack]().Entry(&rck).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.WrapWriter[string, ontology.Relationship](db).
			Set(ctx, edges...)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Rack",
			Migrations: []gorp.Migration{v3.RolesMigration},
		})).To(Succeed())
		var remaining []ontology.Relationship
		Expect(gorp.NewRetrieve[string, ontology.Relationship]().
			Entries(&remaining).
			Exec(ctx, db)).To(Succeed())
		Expect(remaining).To(BeEmpty())
		var out v3.Rack
		Expect(gorp.NewRetrieve[v3.Key, v3.Rack]().
			Where(gorp.MatchKeys[v3.Key, v3.Rack](rck.Key)).
			Entry(&out).
			Exec(ctx, db)).To(Succeed())
		return out
	}

	It("Should give a rack the roles that point at it", func(ctx SpecContext) {
		driver, viewer := createRole(ctx), createRole(ctx)
		rck := v3.Rack{Key: 65537, Name: "rack", Roles: []role.Key{driver}}
		out := migrate(
			ctx,
			rck,
			parentOf(driver, rck.Key),
			parentOf(viewer, rck.Key),
		)
		Expect(out.Roles).To(Equal([]role.Key{driver, viewer}))
	})

	It("Should drop an edge from a deleted role", func(ctx SpecContext) {
		rck := v3.Rack{Key: 65537, Name: "rack"}
		Expect(migrate(ctx, rck, parentOf(uuid.New(), rck.Key)).Roles).
			To(BeEmpty())
	})
})
