// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/builtin"
	v0 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/migrations/v0"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	policyv0 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
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
	It(
		"Should migrate users with legacy policies to correct roles",
		func(ctx SpecContext) {
			// Set up a fresh DB with legacy data pre-seeded
			db := DeferClose(gorp.Wrap(memkv.New()))
			otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
			searchIdx := MustOpen(search.OpenIndex())
			groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   searchIdx,
			}))
			authSvc := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{DB: db}))
			userSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
				DB:              db,
				Ontology:        otg,
				Group:           groupSvc,
				Search:          searchIdx,
				Auth:            authSvc,
				RootCredentials: auth.Credentials{Username: "root", Password: "p"},
			}))

			// Create test users
			tx := DeferClose(db.OpenTx())
			w := userSvc.NewWriter(tx)
			var rootUser user.User
			Expect(userSvc.NewRetrieve().
				Where(user.MatchUsernames("root")).
				Entry(&rootUser).
				Exec(ctx, tx)).To(Succeed())
			adminUser := MustSucceed(w.Create(ctx, user.User{Username: "admin"}))
			schematicUser := MustSucceed(w.Create(ctx, user.User{
				Username: "schematicuser",
			}))
			regularUser := MustSucceed(w.Create(ctx, user.User{Username: "regular"}))

			// Seed legacy policies with Subjects field
			adminPolicy := policyv0.Policy{
				Key:      uuid.New(),
				Subjects: []ontology.ID{adminUser.OntologyID()},
				Objects: []ontology.ID{
					{Type: ontology.ResourceTypeUser},
					{Type: "policy"},
				},
				Actions: []access.Action{"all"},
			}
			schematicPolicy := policyv0.Policy{
				Key:      uuid.New(),
				Subjects: []ontology.ID{schematicUser.OntologyID()},
				Objects:  []ontology.ID{{Type: "schematic"}},
				Actions:  []access.Action{"all"},
			}
			writer := gorp.WrapWriter[uuid.UUID, policyv0.Policy](tx)
			Expect(writer.Set(ctx, adminPolicy)).To(Succeed())
			Expect(writer.Set(ctx, schematicPolicy)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())

			policySvc := MustOpen(policy.OpenService(ctx, policy.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   searchIdx,
			}))
			roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    groupSvc,
				Search:   searchIdx,
			}))
			builtinRoles := MustSucceed(builtin.Provision(ctx, db, policySvc, roleSvc))
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:        db,
				Namespace: "RBAC",
				Migrations: []migrate.Migration{
					v0.NewMigration(v0.MigrationConfig{
						User:     userSvc,
						Ontology: otg,
						Role:     roleSvc,
						Roles:    builtinRoles,
					}),
				},
			})).To(Succeed())

			// Look up the built-in role keys
			tx2 := DeferClose(db.OpenTx())

			var ownerRole role.Role
			Expect(
				roleSvc.NewRetrieve().
					Where(role.MatchNames("Owner")).
					Entry(&ownerRole).
					Exec(ctx, tx2),
			).To(Succeed())
			var engineerRole role.Role
			Expect(
				roleSvc.NewRetrieve().
					Where(role.MatchNames("Engineer")).
					Entry(&engineerRole).
					Exec(ctx, tx2),
			).To(Succeed())
			var operatorRole role.Role
			Expect(
				roleSvc.NewRetrieve().
					Where(role.MatchNames("Operator")).
					Entry(&operatorRole).
					Exec(ctx, tx2),
			).To(Succeed())

			// Root user -> Owner
			Expect(
				userHasSpecificRole(
					ctx,
					tx2,
					otg,
					rootUser.OntologyID(),
					ownerRole.Key,
				),
			).To(BeTrue())
			// Admin policy user -> Owner
			Expect(
				userHasSpecificRole(
					ctx,
					tx2,
					otg,
					adminUser.OntologyID(),
					ownerRole.Key,
				),
			).To(BeTrue())
			// Schematic policy user -> Engineer
			Expect(
				userHasSpecificRole(
					ctx,
					tx2,
					otg,
					schematicUser.OntologyID(),
					engineerRole.Key,
				),
			).To(BeTrue())
			// Regular user -> Operator
			Expect(
				userHasSpecificRole(
					ctx,
					tx2,
					otg,
					regularUser.OntologyID(),
					operatorRole.Key,
				),
			).To(BeTrue())

			// Legacy policies should be deleted
			reader := gorp.WrapReader[uuid.UUID, policyv0.Policy](tx2)
			iter := MustOpen(reader.OpenIterator(gorp.IterOptions{}))
			legacyCount := 0
			for iter.First(); iter.Valid(); iter.Next() {
				v := iter.Value(ctx)
				if v != nil && len(v.Subjects) > 0 {
					legacyCount++
				}
			}
			Expect(legacyCount).To(Equal(0))
		},
	)

	It("Should be idempotent across repeated runs", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.OpenIndex())
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		authSvc := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{DB: db}))
		userSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
			DB:              db,
			Ontology:        otg,
			Group:           groupSvc,
			Search:          searchIdx,
			Auth:            authSvc,
			RootCredentials: auth.Credentials{Username: "suite-root", Password: "p"},
		}))

		policySvc := MustOpen(policy.OpenService(ctx, policy.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Search:   searchIdx,
		}))
		run := func() error {
			builtinRoles, err := builtin.Provision(ctx, db, policySvc, roleSvc)
			if err != nil {
				return err
			}
			return gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:        db,
				Namespace: "RBAC",
				Migrations: []migrate.Migration{
					v0.NewMigration(v0.MigrationConfig{
						User:     userSvc,
						Ontology: otg,
						Role:     roleSvc,
						Roles:    builtinRoles,
					}),
				},
			})
		}
		Expect(run()).To(Succeed())
		Expect(run()).To(Succeed())
	})
})
