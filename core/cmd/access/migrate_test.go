// Copyright 2025 Synnax Labs, Inc.
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
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/cmd/access"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service"
	svcAccess "github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	ctx = context.Background()
	db  *gorp.DB
	otg *ontology.Ontology
	g   *group.Service
	svc *service.Layer
)

var _ = BeforeSuite(func() {
	db = gorp.Wrap(memkv.New())
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
	g = MustSucceed(group.OpenService(ctx, group.Config{DB: db, Ontology: otg}))
	userSvc := MustSucceed(user.NewService(ctx, user.Config{
		DB:       db,
		Ontology: otg,
		Group:    g,
	}))
	rbacSvc := MustSucceed(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
	}))
	svc = &service.Layer{
		User: userSvc,
		RBAC: rbacSvc,
	}
})

var _ = AfterSuite(func() {
	Expect(svc.RBAC.Close()).To(Succeed())
	Expect(g.Close()).To(Succeed())
	Expect(otg.Close()).To(Succeed())
	Expect(db.Close()).To(Succeed())
})

func TestMigration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Migration Suite")
}

var _ = Describe("MigratePermissions", Ordered, func() {
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

	Describe("Migration tracking", func() {
		It("Should only run migration once", func() {
			// Create a user
			u := &user.User{Username: "testuser1"}
			Expect(svc.User.NewWriter(tx).Create(ctx, u)).To(Succeed())

			// Run migration first time
			Expect(access.MigratePermissions(ctx, tx, svc, roles)).To(Succeed())

			// Verify user has Operator role (default)
			hasRole := userHasSpecificRole(ctx, tx, otg, user.OntologyID(u.Key), roles.OperatorKey)
			Expect(hasRole).To(BeTrue())

			// Run migration second time - should be idempotent
			Expect(access.MigratePermissions(ctx, tx, svc, roles)).To(Succeed())
		})
	})

	Describe("Role assignment", func() {
		It("Should assign Owner role to user with RootUser=true", func() {
			u := &user.User{Username: "rootuser", RootUser: true}
			Expect(svc.User.NewWriter(tx).Create(ctx, u)).To(Succeed())

			// Clear migration flag to allow re-run
			Expect(tx.Delete(ctx, []byte("sy_rbac_migration_performed"))).To(Succeed())

			Expect(access.MigratePermissions(ctx, tx, svc, roles)).To(Succeed())

			hasOwner := userHasSpecificRole(ctx, tx, otg, user.OntologyID(u.Key), roles.OwnerKey)
			Expect(hasOwner).To(BeTrue())
		})

		It("Should assign Owner role to user with admin policy", func() {
			u := &user.User{Username: "adminuser"}
			Expect(svc.User.NewWriter(tx).Create(ctx, u)).To(Succeed())

			// Create legacy admin policy with Subjects field
			adminPolicy := access.LegacyPolicy{
				Key:      uuid.New(),
				Subjects: []ontology.ID{user.OntologyID(u.Key)},
				Objects: []ontology.ID{
					{Type: user.OntologyType},
					{Type: "policy"},
				},
				Actions: []svcAccess.Action{svcAccess.ActionAll},
			}
			Expect(gorp.NewCreate[uuid.UUID, access.LegacyPolicy]().
				Entry(&adminPolicy).
				Exec(ctx, tx)).To(Succeed())

			// Clear migration flag
			Expect(tx.Delete(ctx, []byte("sy_rbac_migration_performed"))).To(Succeed())

			Expect(access.MigratePermissions(ctx, tx, svc, roles)).To(Succeed())

			hasOwner := userHasSpecificRole(ctx, tx, otg, user.OntologyID(u.Key), roles.OwnerKey)
			Expect(hasOwner).To(BeTrue())
		})

		It("Should assign Engineer role to user with schematic policy", func() {
			u := &user.User{Username: "schematicuser"}
			Expect(svc.User.NewWriter(tx).Create(ctx, u)).To(Succeed())

			// Create legacy schematic policy with Subjects field
			schematicPolicy := access.LegacyPolicy{
				Key:      uuid.New(),
				Subjects: []ontology.ID{user.OntologyID(u.Key)},
				Objects: []ontology.ID{
					{Type: "schematic"},
				},
				Actions: []svcAccess.Action{svcAccess.ActionAll},
			}
			Expect(gorp.NewCreate[uuid.UUID, access.LegacyPolicy]().
				Entry(&schematicPolicy).
				Exec(ctx, tx)).To(Succeed())

			// Clear migration flag
			Expect(tx.Delete(ctx, []byte("sy_rbac_migration_performed"))).To(Succeed())

			Expect(access.MigratePermissions(ctx, tx, svc, roles)).To(Succeed())

			hasEngineer := userHasSpecificRole(ctx, tx, otg, user.OntologyID(u.Key), roles.EngineerKey)
			Expect(hasEngineer).To(BeTrue())
		})

		It("Should assign Operator role to user with no special permissions", func() {
			u := &user.User{Username: "regularuser"}
			Expect(svc.User.NewWriter(tx).Create(ctx, u)).To(Succeed())

			// Clear migration flag
			Expect(tx.Delete(ctx, []byte("sy_rbac_migration_performed"))).To(Succeed())

			Expect(access.MigratePermissions(ctx, tx, svc, roles)).To(Succeed())

			hasOperator := userHasSpecificRole(ctx, tx, otg, user.OntologyID(u.Key), roles.OperatorKey)
			Expect(hasOperator).To(BeTrue())
		})

		It("Should prioritize RootUser over policies", func() {
			u := &user.User{Username: "rootwithschematic", RootUser: true}
			Expect(svc.User.NewWriter(tx).Create(ctx, u)).To(Succeed())

			// Create schematic policy - but RootUser should take precedence
			schematicPolicy := access.LegacyPolicy{
				Key:      uuid.New(),
				Subjects: []ontology.ID{user.OntologyID(u.Key)},
				Objects:  []ontology.ID{{Type: "schematic"}},
				Actions:  []svcAccess.Action{svcAccess.ActionAll},
			}
			Expect(gorp.NewCreate[uuid.UUID, access.LegacyPolicy]().
				Entry(&schematicPolicy).
				Exec(ctx, tx)).To(Succeed())

			// Clear migration flag
			Expect(tx.Delete(ctx, []byte("sy_rbac_migration_performed"))).To(Succeed())

			Expect(access.MigratePermissions(ctx, tx, svc, roles)).To(Succeed())

			hasOwner := userHasSpecificRole(ctx, tx, otg, user.OntologyID(u.Key), roles.OwnerKey)
			Expect(hasOwner).To(BeTrue())
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
				Actions:  []svcAccess.Action{svcAccess.ActionAll},
			}
			Expect(gorp.NewCreate[uuid.UUID, access.LegacyPolicy]().
				Entry(&legacyPolicy).
				Exec(ctx, tx)).To(Succeed())

			// Clear migration flag
			Expect(tx.Delete(ctx, []byte("sy_rbac_migration_performed"))).To(Succeed())

			Expect(access.MigratePermissions(ctx, tx, svc, roles)).To(Succeed())

			// Verify legacy policy was deleted by checking all legacy policies
			var policies []access.LegacyPolicy
			Expect(gorp.NewRetrieve[uuid.UUID, access.LegacyPolicy]().
				Entries(&policies).
				Exec(ctx, tx)).To(Succeed())

			// Filter to only policies with Subjects (legacy format)
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

// userHasSpecificRole checks if a user has a specific role assigned via ontology.
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
