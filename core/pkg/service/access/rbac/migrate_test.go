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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/migrations/v0"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func userHasSpecificRole(
	ctx context.Context,
	tx gorp.Tx,
	otg *ontology.Ontology,
	userID ontology.ID,
	roleKey uuid.UUID,
) bool {
	var roles []ontology.Resource
	if err := otg.NewRetrieve().
		WhereIDs(userID).
		TraverseTo(ontology.ParentsTraverser).
		WhereTypes(ontology.ResourceTypeRole).
		Entries(&roles).
		Exec(ctx, tx); err != nil {
		return false
	}
	for _, r := range roles {
		if r.ID.Key == roleKey.String() {
			return true
		}
	}
	return false
}

var _ = Describe("Legacy Permission Migration", func() {
	It("Should migrate users with legacy policies to correct roles", func(ctx SpecContext) {
		// Set up a fresh DB with legacy data pre-seeded
		testDB := DeferClose(gorp.Wrap(memkv.New()))
		testOtg := MustOpen(ontology.Open(ctx, ontology.Config{DB: testDB}))
		testSearch := MustOpen(search.Open())
		testGroup := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       testDB,
			Ontology: testOtg,
			Search:   testSearch,
		}))
		testUserSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
			DB:       testDB,
			Ontology: testOtg,
			Group:    testGroup,
			Search:   testSearch,
		}))

		// Create test users
		tx := testDB.OpenTx()
		rootUser := &user.User{Username: "root", RootUser: true}
		Expect(testUserSvc.NewWriter(tx).Create(ctx, rootUser)).To(Succeed())
		adminUser := &user.User{Username: "admin"}
		Expect(testUserSvc.NewWriter(tx).Create(ctx, adminUser)).To(Succeed())
		schematicUser := &user.User{Username: "schematicuser"}
		Expect(testUserSvc.NewWriter(tx).Create(ctx, schematicUser)).To(Succeed())
		regularUser := &user.User{Username: "regular"}
		Expect(testUserSvc.NewWriter(tx).Create(ctx, regularUser)).To(Succeed())

		// Seed legacy policies with Subjects field
		adminPolicy := v0.Policy{
			Key:      uuid.New(),
			Subjects: []ontology.ID{user.OntologyID(adminUser.Key)},
			Objects: []ontology.ID{
				{Type: ontology.ResourceTypeUser},
				{Type: "policy"},
			},
			Actions: []access.Action{"all"},
		}
		schematicPolicy := v0.Policy{
			Key:      uuid.New(),
			Subjects: []ontology.ID{user.OntologyID(schematicUser.Key)},
			Objects:  []ontology.ID{{Type: "schematic"}},
			Actions:  []access.Action{"all"},
		}
		writer := gorp.WrapWriter[uuid.UUID, v0.Policy](tx)
		Expect(writer.Set(ctx, adminPolicy)).To(Succeed())
		Expect(writer.Set(ctx, schematicPolicy)).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())

		// Open RBAC service, which runs Phase 1 (extraction) and Phase 2 (assignment)
		testRBAC := MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
			DB:       testDB,
			Ontology: testOtg,
			Group:    testGroup,
			Search:   testSearch,
			User:     testUserSvc,
		}))

		// Look up the built-in role keys
		tx2 := testDB.OpenTx()
		defer func() { Expect(tx2.Close()).To(Succeed()) }()

		var ownerRole role.Role
		Expect(testRBAC.Role.NewRetrieve().Where(role.MatchNames("Owner")).Entry(&ownerRole).Exec(ctx, tx2)).To(Succeed())
		var engineerRole role.Role
		Expect(testRBAC.Role.NewRetrieve().Where(role.MatchNames("Engineer")).Entry(&engineerRole).Exec(ctx, tx2)).To(Succeed())
		var operatorRole role.Role
		Expect(testRBAC.Role.NewRetrieve().Where(role.MatchNames("Operator")).Entry(&operatorRole).Exec(ctx, tx2)).To(Succeed())

		// Root user -> Owner
		Expect(userHasSpecificRole(ctx, tx2, testOtg, user.OntologyID(rootUser.Key), ownerRole.Key)).To(BeTrue())
		// Admin policy user -> Owner
		Expect(userHasSpecificRole(ctx, tx2, testOtg, user.OntologyID(adminUser.Key), ownerRole.Key)).To(BeTrue())
		// Schematic policy user -> Engineer
		Expect(userHasSpecificRole(ctx, tx2, testOtg, user.OntologyID(schematicUser.Key), engineerRole.Key)).To(BeTrue())
		// Regular user -> Operator
		Expect(userHasSpecificRole(ctx, tx2, testOtg, user.OntologyID(regularUser.Key), operatorRole.Key)).To(BeTrue())

		// Legacy policies should be deleted
		reader := gorp.WrapReader[uuid.UUID, v0.Policy](tx2)
		iter := MustOpen(reader.OpenIterator(gorp.IterOptions{}))
		legacyCount := 0
		for iter.First(); iter.Valid(); iter.Next() {
			v := iter.Value(ctx)
			if v != nil && len(v.Subjects) > 0 {
				legacyCount++
			}
		}
		Expect(legacyCount).To(Equal(0))
	})

	It("Should be idempotent across multiple service opens", func(ctx SpecContext) {
		testDB := DeferClose(gorp.Wrap(memkv.New()))
		testOtg := MustOpen(ontology.Open(ctx, ontology.Config{DB: testDB}))
		testSearch := MustOpen(search.Open())
		testGroup := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       testDB,
			Ontology: testOtg,
			Search:   testSearch,
		}))
		testUserSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
			DB:       testDB,
			Ontology: testOtg,
			Group:    testGroup,
			Search:   testSearch,
		}))

		// First open
		svc1 := MustSucceed(rbac.OpenService(ctx, rbac.ServiceConfig{
			DB:       testDB,
			Ontology: testOtg,
			Group:    testGroup,
			Search:   testSearch,
			User:     testUserSvc,
		}))
		Expect(svc1.Close()).To(Succeed())

		// Second open should not fail
		svc2 := MustSucceed(rbac.OpenService(ctx, rbac.ServiceConfig{
			DB:       testDB,
			Ontology: testOtg,
			Group:    testGroup,
			Search:   testSearch,
			User:     testUserSvc,
		}))
		Expect(svc2.Close()).To(Succeed())
	})
})
