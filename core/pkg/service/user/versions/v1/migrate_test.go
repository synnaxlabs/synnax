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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v1"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration", func() {
	It("Should drop the username and keep every other field", func(ctx SpecContext) {
		db := gorp.Wrap(DeferClose(memkv.New()))
		old := v0.User{
			Key:       uuid.New(),
			Username:  "alice",
			FirstName: "Alice",
			LastName:  "Liddell",
			RootUser:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.WrapWriter[v0.Key, v0.User](tx).Set(ctx, old)
		})).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "User",
			Migrations: []migrate.Migration{v1.Migration},
		})).To(Succeed())
		var migrated v1.User
		Expect(gorp.NewRetrieve[v1.Key, v1.User]().
			Where(gorp.MatchKeys[v1.Key, v1.User](old.Key)).
			Entry(&migrated).
			Exec(ctx, db)).To(Succeed())
		Expect(migrated).To(Equal(v1.User{
			Key:       old.Key,
			FirstName: "Alice",
			LastName:  "Liddell",
			RootUser:  true,
		}))
	})
})
