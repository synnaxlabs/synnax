// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password/versions"
	authv0 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v0"
	authv1 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v1"
	userversions "github.com/synnaxlabs/synnax/pkg/service/user/versions"
	userv0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"golang.org/x/crypto/bcrypt"
)

var _ = Describe("MoveLegacy", func() {
	var (
		db       *gorp.DB
		legacyDB *gorp.DB
		users    []userv0.User
	)
	login := func(
		ctx context.Context,
		svc *password.Service,
		username string,
	) (auth.Identity, error) {
		return svc.Authenticate(ctx, auth.Proof{Variant: auth.PasswordProof{
			Credentials: auth.Credentials{Username: username, Password: "p"},
		}})
	}
	BeforeEach(func(ctx SpecContext) {
		kvDB := DeferClose(memkv.New())
		db = gorp.Wrap(kvDB)
		users = []userv0.User{
			{Key: uuid.New(), Username: "alice"},
			{Key: uuid.New(), Username: "bob"},
		}
		Expect(gorp.WrapWriter[userv0.Key, userv0.User](db).Set(ctx, users...)).
			To(Succeed())
		hash := MustSucceed(bcrypt.GenerateFromPassword([]byte("p"), bcrypt.MinCost))
		// the untagged MessagePack shape a Core before 0.59 stored
		legacyDB = gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec))
		Expect(gorp.WrapWriter[string, authv0.SecureCredentials](legacyDB).Set(ctx,
			authv0.SecureCredentials{Username: "alice", Password: hash},
			authv0.SecureCredentials{Username: "bob", Password: hash},
			authv0.SecureCredentials{Username: "ghost", Password: hash},
		)).To(Succeed())
	})
	It("Should give each user its password and drop the orphan", func(ctx SpecContext) {
		Expect(versions.MoveLegacy(ctx, db, alamos.Instrumentation{})).To(Succeed())
		svc := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
		for _, u := range users {
			Expect(MustSucceed(login(ctx, svc, u.Username)).Subject).
				To(Equal(u.OntologyID()))
		}
		Expect(login(ctx, svc, "ghost")).Error().
			To(MatchError(auth.ErrInvalidCredentials))
		Expect(gorp.NewRetrieve[string, authv1.SecureCredentials]().Exists(ctx, db)).
			To(BeFalse())
	})
	It("Should move credentials for users a Core before 0.59 stored", func(
		ctx SpecContext,
	) {
		Expect(gorp.WrapWriter[userv0.Key, userv0.User](legacyDB).Set(ctx, users...)).
			To(Succeed())
		Expect(versions.MoveLegacy(ctx, db, alamos.Instrumentation{})).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "User",
			Migrations: userversions.Migrations,
		})).To(Succeed())
		svc := MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
		for _, u := range users {
			Expect(MustSucceed(login(ctx, svc, u.Username)).Subject).
				To(Equal(u.OntologyID()))
			var migrated userversions.User
			Expect(gorp.NewRetrieve[userversions.Key, userversions.User]().
				Where(gorp.MatchKeys[userversions.Key, userversions.User](u.Key)).
				Entry(&migrated).
				Exec(ctx, db)).To(Succeed())
			Expect(migrated.Name).To(Equal(u.Username))
		}
	})
	It("Should run once", func(ctx SpecContext) {
		Expect(versions.MoveLegacy(ctx, db, alamos.Instrumentation{})).To(Succeed())
		var first []password.Password
		Expect(gorp.NewRetrieve[password.Key, password.Password]().
			Entries(&first).Exec(ctx, db)).To(Succeed())
		Expect(gorp.WrapWriter[string, authv1.SecureCredentials](db).Set(ctx,
			authv1.SecureCredentials{Username: "bob", Password: []byte("other")},
		)).To(Succeed())
		Expect(versions.MoveLegacy(ctx, db, alamos.Instrumentation{})).To(Succeed())
		var second []password.Password
		Expect(gorp.NewRetrieve[password.Key, password.Password]().
			Entries(&second).Exec(ctx, db)).To(Succeed())
		Expect(second).To(ConsistOf(first))
	})
})
