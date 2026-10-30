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
	v0 "github.com/synnaxlabs/synnax/pkg/service/auth/password/versions/v0"
	authv1 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	userv0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Password", func() {
	It("Should use its key as the gorp key", func() {
		key := uuid.New()
		Expect(v0.Password{Key: key}.GorpKey()).To(Equal(key))
	})
	It("Should return no options", func() {
		Expect(v0.Password{}.SetOptions()).To(BeNil())
	})
	It("Should validate its subject", func() {
		Expect(v0.Password{}.Validate()).To(MatchError(ContainSubstring("subject")))
		Expect(v0.Password{Subject: ontology.ID{
			Type: ontology.ResourceTypeUser,
			Key:  uuid.New().String(),
		}}.Validate()).To(Succeed())
	})
})

var _ = Describe("NewMove", func() {
	var (
		db    *gorp.DB
		alice userv0.User
	)
	run := func(ctx context.Context) error {
		return gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "test",
			Migrations: []gorp.Migration{v0.NewMove()},
		})
	}
	passwords := func(ctx context.Context) []v0.Password {
		GinkgoHelper()
		var ps []v0.Password
		Expect(gorp.NewRetrieve[v0.Key, v0.Password]().Entries(&ps).Exec(ctx, db)).
			To(Succeed())
		return ps
	}
	create := func(ctx context.Context, usernames ...string) {
		GinkgoHelper()
		db = DeferClose(gorp.Wrap(memkv.New()))
		Expect(gorp.WrapWriter[userv0.Key, userv0.User](db).Set(ctx, alice)).
			To(Succeed())
		creds := make([]authv1.SecureCredentials, len(usernames))
		for i, u := range usernames {
			creds[i] = authv1.SecureCredentials{Username: u, Password: []byte("hash")}
		}
		Expect(
			gorp.WrapWriter[string, authv1.SecureCredentials](db).Set(ctx, creds...),
		).
			To(Succeed())
	}
	BeforeEach(func(ctx SpecContext) {
		alice = userv0.User{Key: uuid.New(), Username: "alice"}
		create(ctx, "alice", "ghost")
	})
	It(
		"Should move each credential to the user that holds its username",
		func(ctx SpecContext) {
			Expect(run(ctx)).To(Succeed())
			ps := passwords(ctx)
			Expect(ps).To(HaveLen(1))
			Expect(ps[0].Subject).To(Equal(alice.OntologyID()))
			Expect(ps[0].Username).To(Equal("alice"))
			Expect(ps[0].Hash).To(Equal([]byte("hash")))
			Expect(ps[0].Created).ToNot(BeZero())
		},
	)
	It("Should derive the same key on every node", func(ctx SpecContext) {
		Expect(run(ctx)).To(Succeed())
		first := passwords(ctx)
		create(ctx, "alice")
		Expect(run(ctx)).To(Succeed())
		second := passwords(ctx)
		Expect(first).To(HaveLen(1))
		Expect(second).To(HaveLen(1))
		Expect(second[0].Key).To(Equal(first[0].Key))
		Expect(second[0].Key.String()[14]).To(Equal(byte('5')))
	})
	It("Should delete every credential row, owned or not", func(ctx SpecContext) {
		Expect(run(ctx)).To(Succeed())
		Expect(gorp.NewRetrieve[string, authv1.SecureCredentials]().
			Exists(ctx, db)).To(BeFalse())
	})
	It("Should succeed when nothing is stored", func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		Expect(run(ctx)).To(Succeed())
		Expect(passwords(ctx)).To(BeEmpty())
	})
})
