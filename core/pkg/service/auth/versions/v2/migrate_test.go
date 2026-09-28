// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v2"
	userv0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = gorp.Wrap(DeferClose(memkv.New())) })

	createCredentials := func(ctx SpecContext, creds ...v1.SecureCredentials) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.WrapWriter[string, v1.SecureCredentials](tx).Set(ctx, creds...)
		})).To(Succeed())
	}

	createUsers := func(ctx SpecContext, users ...userv0.User) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.WrapWriter[userv0.Key, userv0.User](tx).Set(ctx, users...)
		})).To(Succeed())
	}

	runMigration := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "User",
			Migrations: []migrate.Migration{
				v2.NewMigration(v2.MigrationConfig{UserKeys: userv0.KeysByUsername}),
			},
		})).To(Succeed())
	}

	retrieve := func(ctx SpecContext, key v2.Key) v2.SecureCredentials {
		GinkgoHelper()
		var creds v2.SecureCredentials
		Expect(gorp.NewRetrieve[v2.Key, v2.SecureCredentials]().
			Where(gorp.MatchKeys[v2.Key, v2.SecureCredentials](key)).
			Entry(&creds).
			Exec(ctx, db)).To(Succeed())
		return creds
	}

	It(
		"Should re-key each credential by its user's key and keep the username",
		func(ctx SpecContext) {
			alice := userv0.User{Key: uuid.New(), Username: "alice"}
			bob := userv0.User{Key: uuid.New(), Username: "bob"}
			createUsers(ctx, alice, bob)
			createCredentials(
				ctx,
				v1.SecureCredentials{Username: "alice", Password: []byte("alice-hash")},
				v1.SecureCredentials{Username: "bob", Password: []byte("bob-hash")},
			)
			runMigration(ctx)
			Expect(retrieve(ctx, alice.Key)).To(Equal(v2.SecureCredentials{
				Key:      alice.Key,
				Username: "alice",
				Password: []byte("alice-hash"),
			}))
			Expect(retrieve(ctx, bob.Key)).To(Equal(v2.SecureCredentials{
				Key:      bob.Key,
				Username: "bob",
				Password: []byte("bob-hash"),
			}))
		},
	)

	It("Should delete every username-keyed row", func(ctx SpecContext) {
		alice := userv0.User{Key: uuid.New(), Username: "alice"}
		createUsers(ctx, alice)
		createCredentials(
			ctx,
			v1.SecureCredentials{Username: "alice", Password: []byte("alice-hash")},
			v1.SecureCredentials{Username: "stranded", Password: []byte("old-hash")},
		)
		runMigration(ctx)
		Expect(gorp.NewRetrieve[string, v1.SecureCredentials]().
			Where(gorp.MatchKeys[string, v1.SecureCredentials]("alice", "stranded")).
			Exists(ctx, db)).To(BeFalse())
		var all []v2.SecureCredentials
		Expect(gorp.NewRetrieve[v2.Key, v2.SecureCredentials]().
			Entries(&all).
			Exec(ctx, db)).To(Succeed())
		Expect(all).To(HaveLen(1))
	})

	It("Should quarantine credentials that belong to no user", func(ctx SpecContext) {
		stranded := v1.SecureCredentials{
			Username: "stranded",
			Password: []byte("old-hash"),
		}
		createCredentials(ctx, stranded)
		runMigration(ctx)
		b, closer := MustSucceed2(db.Get(ctx, v2.QuarantineKVKey("stranded")))
		defer func() { Expect(closer.Close()).To(Succeed()) }()
		var staged v1.SecureCredentials
		Expect(json.Unmarshal(b, &staged)).To(Succeed())
		Expect(staged).To(Equal(stranded))
	})

	It("Should not create credentials for a user that had none",
		func(ctx SpecContext) {
			alice := userv0.User{Key: uuid.New(), Username: "alice"}
			createUsers(ctx, alice)
			createCredentials(ctx, v1.SecureCredentials{
				Username: "bob",
				Password: []byte("bob-hash"),
			})
			runMigration(ctx)
			Expect(gorp.NewRetrieve[v2.Key, v2.SecureCredentials]().
				Where(gorp.MatchKeys[v2.Key, v2.SecureCredentials](alice.Key)).
				Entry(&v2.SecureCredentials{}).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		})
})
