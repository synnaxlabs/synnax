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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/builtin"
	v0 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/migrations/v0"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	policyv0 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func roleOf(
	ctx context.Context,
	tx gorp.Tx,
	users *user.Service,
	key user.Key,
) role.Key {
	GinkgoHelper()
	var u user.User
	Expect(users.NewRetrieve().
		Where(user.MatchKeys(key)).
		Entry(&u).
		Exec(ctx, tx)).To(Succeed())
	return u.Role
}

// storeLegacyUser writes u raw, as a Core before roles stored it: with no role.
func storeLegacyUser(ctx context.Context, db *gorp.DB, u user.User) user.User {
	GinkgoHelper()
	u.Key = uuid.New()
	Expect(gorp.WrapWriter[user.Key, user.User](db).Set(ctx, u)).To(Succeed())
	return u
}

var _ = Describe("Legacy Permission Migration", func() {
	It(
		"Should migrate users with legacy policies to correct roles",
		func(ctx SpecContext) {
			// Set up a fresh DB with legacy data pre-seeded. Legacy policies are
			// written through legacyDB (plain MessagePack over the same store), the
			// format pre-Orc servers persisted.
			kvDB := DeferClose(memkv.New())
			db := gorp.Wrap(kvDB)
			legacyDB := gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec))
			passwordSvc := MustOpen(
				password.OpenService(ctx, password.ServiceConfig{DB: db}),
			)
			userSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
				DB:       db,
				Password: passwordSvc,
			}))
			rootUser := storeLegacyUser(
				ctx,
				db,
				user.User{Name: "root", RootUser: true},
			)
			adminUser := storeLegacyUser(ctx, db, user.User{Name: "admin"})
			schematicUser := storeLegacyUser(ctx, db, user.User{Name: "schematicuser"})
			regularUser := storeLegacyUser(ctx, db, user.User{Name: "regular"})

			// Seed legacy policies with Subjects field
			adminPolicy := policyv0.Policy{
				Key:      uuid.New(),
				Subjects: []resource.ID{adminUser.ResourceID()},
				Objects: []resource.ID{
					{Type: resource.TypeUser},
					{Type: "policy"},
				},
				Actions: []access.Action{"all"},
			}
			schematicPolicy := policyv0.Policy{
				Key:      uuid.New(),
				Subjects: []resource.ID{schematicUser.ResourceID()},
				Objects:  []resource.ID{{Type: "schematic"}},
				Actions:  []access.Action{"all"},
			}
			writer := gorp.WrapWriter[uuid.UUID, policyv0.Policy](legacyDB)
			Expect(writer.Set(ctx, adminPolicy)).To(Succeed())
			Expect(writer.Set(ctx, schematicPolicy)).To(Succeed())

			roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
				DB: db,
			}))
			policySvc := MustOpen(policy.OpenService(ctx, policy.ServiceConfig{
				DB:   db,
				Role: roleSvc,
			}))
			builtinRoles := MustSucceed(builtin.Provision(ctx, db, policySvc, roleSvc))
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:        db,
				Namespace: "RBAC",
				Migrations: []gorp.Migration{
					v0.NewMigration(v0.MigrationConfig{
						User:  userSvc,
						Roles: builtinRoles,
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

			Expect(roleOf(ctx, tx2, userSvc, rootUser.Key)).To(Equal(ownerRole.Key))
			Expect(roleOf(ctx, tx2, userSvc, adminUser.Key)).To(Equal(ownerRole.Key))
			Expect(roleOf(ctx, tx2, userSvc, schematicUser.Key)).
				To(Equal(engineerRole.Key))
			Expect(roleOf(ctx, tx2, userSvc, regularUser.Key)).
				To(Equal(operatorRole.Key))

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
		passwordSvc := MustOpen(
			password.OpenService(ctx, password.ServiceConfig{DB: db}),
		)
		userSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
			DB:       db,
			Password: passwordSvc,
		}))
		root := storeLegacyUser(ctx, db, user.User{Name: "root", RootUser: true})
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
			DB: db,
		}))
		policySvc := MustOpen(policy.OpenService(ctx, policy.ServiceConfig{
			DB:   db,
			Role: roleSvc,
		}))
		run := func() error {
			builtinRoles, err := builtin.Provision(ctx, db, policySvc, roleSvc)
			if err != nil {
				return err
			}
			return gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:        db,
				Namespace: "RBAC",
				Migrations: []gorp.Migration{
					v0.NewMigration(v0.MigrationConfig{
						User:  userSvc,
						Roles: builtinRoles,
					}),
				},
			})
		}
		Expect(run()).To(Succeed())
		Expect(run()).To(Succeed())
		var owner role.Role
		Expect(roleSvc.NewRetrieve().
			Where(role.MatchNames("Owner")).
			Entry(&owner).
			Exec(ctx, db)).To(Succeed())
		Expect(roleOf(ctx, db, userSvc, root.Key)).To(Equal(owner.Key))
	})

	It("Should keep the roles of a store that already converted to roles", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		passwordSvc := MustOpen(
			password.OpenService(ctx, password.ServiceConfig{DB: db}),
		)
		userSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
			DB:       db,
			Password: passwordSvc,
		}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
			DB: db,
		}))
		policySvc := MustOpen(policy.OpenService(ctx, policy.ServiceConfig{
			DB:   db,
			Role: roleSvc,
		}))
		builtinRoles := MustSucceed(builtin.Provision(ctx, db, policySvc, roleSvc))
		MustSucceed2(user.ProvisionRoot(
			ctx,
			userSvc,
			passwordSvc,
			auth.Credentials{Username: "root", Password: "p"},
			builtinRoles.OwnerKey,
		))
		// Cores up to v0.53 converted permissions to roles and set this flag.
		Expect(db.Set(ctx, []byte("sy_rbac_migration_performed"), []byte{1})).
			To(Succeed())
		var viewer user.User
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			viewer, err = userSvc.NewWriter(tx).Create(ctx, user.User{
				Name: "viewer",
				Role: builtinRoles.ViewerKey,
			})
			return err
		})).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "RBAC",
			Migrations: []gorp.Migration{v0.NewMigration(v0.MigrationConfig{
				User:  userSvc,
				Roles: builtinRoles,
			})},
		})).To(Succeed())
		Expect(roleOf(ctx, db, userSvc, viewer.Key)).To(Equal(builtinRoles.ViewerKey))
	})
})
