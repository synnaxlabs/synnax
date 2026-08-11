// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v1"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	// seedV0 writes a credential row in the untagged v0 shape, exactly as the
	// pre-Oracle server persisted it.
	seedV0 := func(ctx SpecContext, db *gorp.DB, sc v0.SecureCredentials) {
		GinkgoHelper()
		t := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[string, v0.SecureCredentials]{DB: db},
		))
		Expect(t.NewCreate().Entry(&sc).Exec(ctx, db)).To(Succeed())
	}

	// retrieveMigrated opens the current credentials table with the migration wired
	// in, driving the v0 -> Orc lift end-to-end through gorp.
	retrieveMigrated := func(
		ctx SpecContext,
		db *gorp.DB,
		username string,
	) v1.SecureCredentials {
		GinkgoHelper()
		t := MustOpen(gorp.OpenTable(
			ctx, gorp.TableConfig[string, v1.SecureCredentials]{
				DB:         db,
				Migrations: []migrate.Migration{v1.Migration},
			},
		))
		var got v1.SecureCredentials
		Expect(t.NewRetrieve().
			Where(gorp.MatchKeys[string, v1.SecureCredentials](username)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	It("Should lift a legacy row into the current shape", func(ctx SpecContext) {
		kvDB := DeferClose(memkv.New())
		db := gorp.Wrap(kvDB)
		seedV0(ctx, db, v0.SecureCredentials{
			Username: "root",
			Password: []byte("bcrypt-hash"),
		})
		got := retrieveMigrated(ctx, db, "root")
		Expect(got.Username).To(Equal("root"))
		Expect(got.Password).To(Equal([]byte("bcrypt-hash")))
	})

	It("Should re-encode rows so they decode without the MessagePack fallback",
		func(ctx SpecContext) {
			kvDB := DeferClose(memkv.New())
			db := gorp.Wrap(kvDB)
			seedV0(ctx, db, v0.SecureCredentials{
				Username: "root",
				Password: []byte("bcrypt-hash"),
			})
			retrieveMigrated(ctx, db, "root")
			orcDB := gorp.Wrap(kvDB, gorp.WithCodec(orc.Codec))
			t := MustOpen(gorp.OpenTable(
				ctx, gorp.TableConfig[string, v1.SecureCredentials]{
					DB:         orcDB,
					Migrations: []migrate.Migration{v1.Migration},
				},
			))
			var got v1.SecureCredentials
			Expect(t.NewRetrieve().
				Where(gorp.MatchKeys[string, v1.SecureCredentials]("root")).
				Entry(&got).Exec(ctx, orcDB)).To(Succeed())
			Expect(got.Password).To(Equal([]byte("bcrypt-hash")))
		})
})
