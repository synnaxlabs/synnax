// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access_test

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/cmd/access"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	svcAccess "github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("Access", Ordered, func() {
	var (
		tx    gorp.Tx
		roles access.ProvisionResult
	)
	BeforeAll(func() {
		tx = db.OpenTx()
		var err error
		roles, err = access.Provision(ctx, tx, svc.RBAC)
		Expect(err).ToNot(HaveOccurred())
	})
	AfterAll(func() {
		Expect(tx.Close()).To(Succeed())
	})

	Describe("Provision", func() {
		It("Should create all built-in roles", func() {
			Expect(roles.OwnerKey).ToNot(Equal(uuid.Nil))
			Expect(roles.EngineerKey).ToNot(Equal(uuid.Nil))
			Expect(roles.OperatorKey).ToNot(Equal(uuid.Nil))
			Expect(roles.ViewerKey).ToNot(Equal(uuid.Nil))
		})
		It("Should create policies for each role", func() {
			for _, roleKey := range []uuid.UUID{
				roles.OwnerKey,
				roles.EngineerKey,
				roles.OperatorKey,
				roles.ViewerKey,
			} {
				var policies []ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(role.OntologyID(roleKey)).
					TraverseTo(ontology.Children).
					Entries(&policies).
					Exec(ctx, tx)).To(Succeed())
				Expect(policies).ToNot(BeEmpty())
			}
		})
		It("Should be idempotent", func() {
			roles2, err := access.Provision(ctx, tx, svc.RBAC)
			Expect(err).ToNot(HaveOccurred())
			Expect(roles2.OwnerKey).To(Equal(roles.OwnerKey))
			Expect(roles2.EngineerKey).To(Equal(roles.EngineerKey))
			Expect(roles2.OperatorKey).To(Equal(roles.OperatorKey))
			Expect(roles2.ViewerKey).To(Equal(roles.ViewerKey))
		})
	})

	Describe("MigratePermissions", func() {
		Describe("Migration tracking", func() {
			It("Should only run migration once", func() {
				u := &user.User{Username: "testuser1"}
				Expect(svc.User.NewWriter(tx).Create(ctx, u)).To(Succeed())
				Expect(access.MigratePermissions(ctx, tx, dist, svc, roles)).To(Succeed())
				hasRole := userHasSpecificRole(ctx, tx, user.OntologyID(u.Key), roles.OperatorKey)
				Expect(hasRole).To(BeTrue())
				Expect(access.MigratePermissions(ctx, tx, dist, svc, roles)).To(Succeed())
			})
		})
		Describe("Role assignment", func() {
			It("Should assign Owner role to user with RootUser=true", func() {
				u := &user.User{Username: "rootuser", RootUser: true}
				Expect(svc.User.NewWriter(tx).Create(ctx, u)).To(Succeed())
				Expect(tx.Delete(ctx, []byte("sy_rbac_migration_performed"))).To(Succeed())
				Expect(access.MigratePermissions(ctx, tx, dist, svc, roles)).To(Succeed())
				hasOwner := userHasSpecificRole(ctx, tx, user.OntologyID(u.Key), roles.OwnerKey)
				Expect(hasOwner).To(BeTrue())
			})
			It("Should assign Owner role to user with admin policy", func() {
				u := &user.User{Username: "adminuser"}
				Expect(svc.User.NewWriter(tx).Create(ctx, u)).To(Succeed())
				adminPolicy := access.LegacyPolicy{
					Key:      uuid.New(),
					Subjects: []ontology.ID{user.OntologyID(u.Key)},
					Objects: []ontology.ID{
						{Type: user.OntologyType},
						{Type: "policy"},
					},
					Actions: []svcAccess.Action{"all"},
				}
				Expect(gorp.NewCreate[uuid.UUID, access.LegacyPolicy]().
					Entry(&adminPolicy).
					Exec(ctx, tx)).To(Succeed())
				Expect(tx.Delete(ctx, []byte("sy_rbac_migration_performed"))).To(Succeed())
				Expect(access.MigratePermissions(ctx, tx, dist, svc, roles)).To(Succeed())
				hasOwner := userHasSpecificRole(ctx, tx, user.OntologyID(u.Key), roles.OwnerKey)
				Expect(hasOwner).To(BeTrue())
			})
			It("Should assign Engineer role to user with schematic policy", func() {
				u := &user.User{Username: "schematicuser"}
				Expect(svc.User.NewWriter(tx).Create(ctx, u)).To(Succeed())
				schematicPolicy := access.LegacyPolicy{
					Key:      uuid.New(),
					Subjects: []ontology.ID{user.OntologyID(u.Key)},
					Objects: []ontology.ID{
						{Type: "schematic"},
					},
					Actions: []svcAccess.Action{"all"},
				}
				Expect(gorp.NewCreate[uuid.UUID, access.LegacyPolicy]().
					Entry(&schematicPolicy).
					Exec(ctx, tx)).To(Succeed())
				Expect(tx.Delete(ctx, []byte("sy_rbac_migration_performed"))).To(Succeed())
				Expect(access.MigratePermissions(ctx, tx, dist, svc, roles)).To(Succeed())
				hasEngineer := userHasSpecificRole(ctx, tx, user.OntologyID(u.Key), roles.EngineerKey)
				Expect(hasEngineer).To(BeTrue())
			})
			It("Should assign Operator role to user with no special permissions", func() {
				u := &user.User{Username: "regularuser"}
				Expect(svc.User.NewWriter(tx).Create(ctx, u)).To(Succeed())
				Expect(tx.Delete(ctx, []byte("sy_rbac_migration_performed"))).To(Succeed())
				Expect(access.MigratePermissions(ctx, tx, dist, svc, roles)).To(Succeed())
				hasOperator := userHasSpecificRole(ctx, tx, user.OntologyID(u.Key), roles.OperatorKey)
				Expect(hasOperator).To(BeTrue())
			})
			It("Should prioritize RootUser over policies", func() {
				u := &user.User{Username: "rootwithschematic", RootUser: true}
				Expect(svc.User.NewWriter(tx).Create(ctx, u)).To(Succeed())
				schematicPolicy := access.LegacyPolicy{
					Key:      uuid.New(),
					Subjects: []ontology.ID{user.OntologyID(u.Key)},
					Objects:  []ontology.ID{{Type: "schematic"}},
					Actions:  []svcAccess.Action{"all"},
				}
				Expect(gorp.NewCreate[uuid.UUID, access.LegacyPolicy]().
					Entry(&schematicPolicy).
					Exec(ctx, tx)).To(Succeed())
				Expect(tx.Delete(ctx, []byte("sy_rbac_migration_performed"))).To(Succeed())
				Expect(access.MigratePermissions(ctx, tx, dist, svc, roles)).To(Succeed())
				hasOwner := userHasSpecificRole(ctx, tx, user.OntologyID(u.Key), roles.OwnerKey)
				Expect(hasOwner).To(BeTrue())
			})
		})
		Describe("Legacy group relationship cleanup", func() {
			It("Should remove old UsersGroup -> ParentOf -> User relationship", func() {
				u := &user.User{Username: "legacygroupuser"}
				Expect(svc.User.NewWriter(tx).Create(ctx, u)).To(Succeed())
				userOntologyID := user.OntologyID(u.Key)

				// Simulate legacy behavior: add user directly under Users group
				var usersGroup group.Group
				Expect(g.NewRetrieve().WhereNames("Users").Entry(&usersGroup).Exec(ctx, tx)).To(Succeed())
				Expect(otg.NewWriter(tx).DefineRelationship(
					ctx,
					usersGroup.OntologyID(),
					ontology.ParentOf,
					userOntologyID,
				)).To(Succeed())

				// Verify user is under the Users group
				var groupParents []ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(userOntologyID).
					TraverseTo(ontology.Parents).
					WhereTypes("group").
					Entries(&groupParents).
					Exec(ctx, tx)).To(Succeed())
				Expect(groupParents).To(HaveLen(1))

				// Run migration
				Expect(tx.Delete(ctx, []byte("sy_rbac_migration_performed"))).To(Succeed())
				Expect(access.MigratePermissions(ctx, tx, dist, svc, roles)).To(Succeed())

				// Verify user is no longer under the Users group
				var groupParentsAfter []ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(userOntologyID).
					TraverseTo(ontology.Parents).
					WhereTypes("group").
					Entries(&groupParentsAfter).
					Exec(ctx, tx)).To(Succeed())
				Expect(groupParentsAfter).To(BeEmpty())

				// Verify user is now under a role
				hasRole := userHasSpecificRole(ctx, tx, userOntologyID, roles.OperatorKey)
				Expect(hasRole).To(BeTrue())
			})
		})
		Describe("Legacy policy cleanup", func() {
			It("Should delete legacy policies after migration", func() {
				u := &user.User{Username: "cleanupuser"}
				Expect(svc.User.NewWriter(tx).Create(ctx, u)).To(Succeed())
				legacyPolicy := access.LegacyPolicy{
					Key:      uuid.New(),
					Subjects: []ontology.ID{user.OntologyID(u.Key)},
					Objects:  []ontology.ID{{Type: "schematic"}},
					Actions:  []svcAccess.Action{"all"},
				}
				Expect(gorp.NewCreate[uuid.UUID, access.LegacyPolicy]().
					Entry(&legacyPolicy).
					Exec(ctx, tx)).To(Succeed())
				Expect(tx.Delete(ctx, []byte("sy_rbac_migration_performed"))).To(Succeed())
				Expect(access.MigratePermissions(ctx, tx, dist, svc, roles)).To(Succeed())
				var policies []access.LegacyPolicy
				Expect(gorp.NewRetrieve[uuid.UUID, access.LegacyPolicy]().
					Entries(&policies).
					Exec(ctx, tx)).To(Succeed())
				legacyPolicies := make([]access.LegacyPolicy, 0)
				for _, p := range policies {
					if len(p.Subjects) > 0 {
						legacyPolicies = append(legacyPolicies, p)
					}
				}
				Expect(legacyPolicies).To(BeEmpty())
			})
		})
	})
})

func userHasSpecificRole(
	ctx context.Context,
	tx gorp.Tx,
	userID ontology.ID,
	roleKey uuid.UUID,
) bool {
	var roles []ontology.Resource
	if err := otg.NewRetrieve().
		WhereIDs(userID).
		TraverseTo(ontology.Parents).
		WhereTypes(role.OntologyType).
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
