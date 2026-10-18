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
	v0 "github.com/synnaxlabs/synnax/pkg/service/auth/password/versions/v0"
	authv1 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Password", func() {
	It("Should use its key as the gorp key", func() {
		key := uuid.New()
		Expect(v0.Password{Key: key}.GorpKey()).To(Equal(key))
	})
	It("Should return no options", func() {
		Expect(v0.Password{}.SetOptions()).To(BeNil())
	})
})

var _ = Describe("NewMove", func() {
	var (
		db    *gorp.DB
		alice = ontology.ID{Type: ontology.ResourceTypeUser, Key: uuid.New().String()}
	)
	lookup := func(_ context.Context, _ gorp.Tx, username string) (ontology.ID, error) {
		if username == "alice" {
			return alice, nil
		}
		return ontology.ID{}, query.ErrNotFound
	}
	run := func(ctx context.Context, cfg v0.MoveConfig) error {
		return gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "test",
			Migrations: []migrate.Migration{v0.NewMove(cfg)},
		})
	}
	passwords := func(ctx context.Context) []v0.Password {
		GinkgoHelper()
		var ps []v0.Password
		Expect(gorp.NewRetrieve[v0.Key, v0.Password]().Entries(&ps).Exec(ctx, db)).
			To(Succeed())
		return ps
	}
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		Expect(gorp.WrapWriter[string, authv1.SecureCredentials](db).Set(ctx,
			authv1.SecureCredentials{Username: "alice", Password: []byte("hash")},
			authv1.SecureCredentials{Username: "ghost", Password: []byte("hash")},
		)).To(Succeed())
	})
	It(
		"Should move each credential to its subject and keep the hash",
		func(ctx SpecContext) {
			Expect(run(ctx, v0.MoveConfig{SubjectByUsername: lookup})).To(Succeed())
			ps := passwords(ctx)
			Expect(ps).To(HaveLen(1))
			Expect(ps[0].Subject).To(Equal(alice))
			Expect(ps[0].Username).To(Equal("alice"))
			Expect(ps[0].Hash).To(Equal([]byte("hash")))
			Expect(ps[0].Key).ToNot(Equal(uuid.Nil()))
			Expect(ps[0].Created).ToNot(BeZero())
		},
	)
	It("Should delete every credential row, owned or not", func(ctx SpecContext) {
		Expect(run(ctx, v0.MoveConfig{SubjectByUsername: lookup})).To(Succeed())
		Expect(gorp.NewRetrieve[string, authv1.SecureCredentials]().
			Exists(ctx, db)).To(BeFalse())
	})
	It(
		"Should fail when credentials exist and no lookup is set",
		func(ctx SpecContext) {
			Expect(run(ctx, v0.MoveConfig{})).
				To(MatchError(ContainSubstring("subject lookup")))
		},
	)
	It("Should succeed with no lookup when nothing is stored", func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		Expect(run(ctx, v0.MoveConfig{})).To(Succeed())
		Expect(passwords(ctx)).To(BeEmpty())
	})
})
