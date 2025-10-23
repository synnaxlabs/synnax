// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate_test

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	policyv0 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/v0"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/version"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Policy Migration V0 to V1", Ordered, func() {
	var (
		db  *gorp.DB
		otg *ontology.Ontology
		tx  gorp.Tx
	)

	BeforeEach(func() {
		db = gorp.Wrap(memkv.New())
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
		tx = db.OpenTx()
	})

	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})

	Describe("Idempotency", func() {
		It("Should not duplicate roles on multiple runs", func() {
			user1 := ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(createUserInOntology(ctx, otg, user1)).To(Succeed())
			pol1 := createV0Policy(user1, "channel", access.ActionRetrieve)
			Expect(writeV0Policy(ctx, db, pol1)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			// Run migration twice
			Expect(migrate.MigratePolicies(ctx, db, otg)).To(Succeed())
			tx = db.OpenTx()
			var rolesAfterFirst []role.Role
			Expect(gorp.NewRetrieve[uuid.UUID, role.Role]().
				Entries(&rolesAfterFirst).
				Exec(ctx, tx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			Expect(migrate.MigratePolicies(ctx, db, otg)).To(Succeed())
			tx = db.OpenTx()
			var rolesAfterSecond []role.Role
			Expect(gorp.NewRetrieve[uuid.UUID, role.Role]().
				Entries(&rolesAfterSecond).
				Exec(ctx, tx)).To(Succeed())

			Expect(len(rolesAfterFirst)).To(Equal(len(rolesAfterSecond)))
		})
	})

	Describe("Single User, Single Policy", func() {
		It("Should create one role and assign it", func() {
			user := ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(createUserInOntology(ctx, otg, user)).To(Succeed())
			pol := createV0Policy(user, "channel", access.ActionRetrieve)
			Expect(writeV0Policy(ctx, db, pol)).To(Succeed())

			// DEBUG: Verify policy was written
			var checkPolicies []policyv0.Policy
			Expect(gorp.NewRetrieve[uuid.UUID, policyv0.Policy]().
				Entries(&checkPolicies).
				Exec(ctx, tx)).To(Succeed())
			println("[TEST DEBUG] Wrote policy, found", len(checkPolicies), "policies in tx before close")

			Expect(tx.Close()).To(Succeed())

			// DEBUG: Check if policy is visible after commit
			tx2 := db.OpenTx()
			var checkPolicies2 []policyv0.Policy
			Expect(gorp.NewRetrieve[uuid.UUID, policyv0.Policy]().
				Entries(&checkPolicies2).
				Exec(ctx, tx2)).To(Succeed())
			println("[TEST DEBUG] After close, found", len(checkPolicies2), "policies in new tx")
			Expect(tx2.Close()).To(Succeed())

			Expect(migrate.MigratePolicies(ctx, db, otg)).To(Succeed())
			tx = db.OpenTx()

			// Verify role created
			var roles []role.Role
			Expect(gorp.NewRetrieve[uuid.UUID, role.Role]().
				Entries(&roles).
				Exec(ctx, tx)).To(Succeed())
			Expect(len(roles)).To(Equal(1))
			Expect(roles[0].Policies).To(ContainElement(pol.Key))

			// Verify role assigned to user via ontology
			var roleIDs []ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(user).
				ExcludeFieldData(true).
				TraverseTo(role.Roles).
				Entries(&roleIDs).
				ExcludeFieldData(true).
				Exec(ctx, tx)).To(Succeed())
			Expect(len(roleIDs)).To(Equal(1))
		})
	})

	Describe("Multiple Users, Same Permissions (Deduplication)", func() {
		It("Should create one shared role for identical permission sets", func() {
			user1 := ontology.ID{Type: "user", Key: uuid.New().String()}
			user2 := ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(createUserInOntology(ctx, otg, user1)).To(Succeed())
			Expect(createUserInOntology(ctx, otg, user2)).To(Succeed())
			pol1 := createV0PolicyWithKey(uuid.New(), []ontology.ID{user1, user2}, "channel", access.ActionRetrieve)
			Expect(writeV0Policy(ctx, db, pol1)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			Expect(migrate.MigratePolicies(ctx, db, otg)).To(Succeed())
			tx = db.OpenTx()

			// Verify only one role created
			var roles []role.Role
			Expect(gorp.NewRetrieve[uuid.UUID, role.Role]().
				Entries(&roles).
				Exec(ctx, tx)).To(Succeed())
			Expect(len(roles)).To(Equal(1))

			// Verify both users assigned same role
			var user1Roles []ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(user1).
				ExcludeFieldData(true).
				TraverseTo(role.Roles).
				Entries(&user1Roles).
				ExcludeFieldData(true).
				Exec(ctx, tx)).To(Succeed())

			var user2Roles []ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(user2).
				ExcludeFieldData(true).
				TraverseTo(role.Roles).
				Entries(&user2Roles).
				ExcludeFieldData(true).
				Exec(ctx, tx)).To(Succeed())

			Expect(user1Roles[0].ID).To(Equal(user2Roles[0].ID))
		})
	})

	Describe("Permission Preservation", func() {
		It("Should preserve exact permissions after migration", func() {
			user := ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(createUserInOntology(ctx, otg, user)).To(Succeed())
			channelObj := ontology.ID{Type: "channel", Key: "chan-123"}
			pol := policyv0.Policy{
				Key:      uuid.New(),
				Subjects: []ontology.ID{user},
				Objects:  []ontology.ID{channelObj},
				Actions:  []access.Action{access.ActionRetrieve, access.ActionCreate},
			}
			Expect(writeV0Policy(ctx, db, pol)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			Expect(migrate.MigratePolicies(ctx, db, otg)).To(Succeed())
			tx = db.OpenTx()

			// Verify policy still has correct objects/actions
			var v1Policy policy.Policy
			Expect(gorp.NewRetrieve[uuid.UUID, policy.Policy]().
				WhereKeys(pol.Key).
				Entry(&v1Policy).
				Exec(ctx, tx)).To(Succeed())

			Expect(v1Policy.Objects).To(Equal(pol.Objects))
			Expect(v1Policy.Actions).To(ContainElements(access.ActionRetrieve, access.ActionCreate))
			Expect(v1Policy.Version).To(Equal(version.V1))
		})
	})

	Describe("Empty Database", func() {
		It("Should handle empty database gracefully", func() {
			// No policies exist, just run migration
			Expect(migrate.MigratePolicies(ctx, db, otg)).To(Succeed())

			// Should complete without error
			var roles []role.Role
			Expect(gorp.NewRetrieve[uuid.UUID, role.Role]().
				Entries(&roles).
				Exec(ctx, tx)).To(Succeed())
			Expect(len(roles)).To(Equal(0))
		})
	})

	Describe("User with Multiple Policies", func() {
		It("Should create one role with all policies", func() {
			user := ontology.ID{Type: "user", Key: uuid.New().String()}
			Expect(createUserInOntology(ctx, otg, user)).To(Succeed())
			pol1 := createV0Policy(user, "channel", access.ActionRetrieve)
			pol2 := createV0Policy(user, "workspace", access.ActionCreate)
			pol3 := createV0Policy(user, "schematic", access.ActionDelete)
			Expect(writeV0Policy(ctx, db, pol1)).To(Succeed())
			Expect(writeV0Policy(ctx, db, pol2)).To(Succeed())
			Expect(writeV0Policy(ctx, db, pol3)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			Expect(migrate.MigratePolicies(ctx, db, otg)).To(Succeed())
			tx = db.OpenTx()

			// Should create one role with all three policies
			var roles []role.Role
			Expect(gorp.NewRetrieve[uuid.UUID, role.Role]().
				Entries(&roles).
				Exec(ctx, tx)).To(Succeed())

			Expect(len(roles)).To(Equal(1))
			Expect(len(roles[0].Policies)).To(Equal(3))
			Expect(roles[0].Policies).To(ContainElements(pol1.Key, pol2.Key, pol3.Key))
		})
	})
})

// Helper functions
func createUserInOntology(ctx context.Context, otg *ontology.Ontology, userID ontology.ID) error {
	// Register the user as a resource in the ontology so it can have relationships
	w := otg.NewWriter(nil)
	return w.DefineResource(ctx, userID)
}

func createV0Policy(subject ontology.ID, objType ontology.Type, action access.Action) policyv0.Policy {
	return policyv0.Policy{
		Key:      uuid.New(),
		Subjects: []ontology.ID{subject},
		Objects:  []ontology.ID{{Type: objType, Key: ""}},
		Actions:  []access.Action{action},
	}
}

func createV0PolicyWithKey(key uuid.UUID, subjects []ontology.ID, objType ontology.Type, action access.Action) policyv0.Policy {
	return policyv0.Policy{
		Key:      key,
		Subjects: subjects,
		Objects:  []ontology.ID{{Type: objType, Key: ""}},
		Actions:  []access.Action{action},
	}
}

func writeV0Policy(ctx context.Context, db *gorp.DB, pol policyv0.Policy) error {
	// Write directly to DB to avoid transaction isolation issues with memkv
	return gorp.NewCreate[uuid.UUID, policyv0.Policy]().Entry(&pol).Exec(ctx, db)
}
