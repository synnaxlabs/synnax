// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v3 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v4"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should delete each range-owned Log and lift the rest", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v3.Key, v3.Log]{DB: db}))
		proj := uuid.New()
		owned := v3.Log{Key: uuid.New(), Name: "Owned", Project: &proj}
		ranged := v3.Log{Key: uuid.New(), Name: "Ranged", Range: new(uuid.New())}
		Expect(gorp.NewCreate[v3.Key, v3.Log]().
			Entries(&[]v3.Log{owned, ranged}).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Log",
			Migrations: []gorp.Migration{v4.RangeMigration, v4.Migration},
		})).To(Succeed())
		var stored []v4.Log
		Expect(gorp.NewRetrieve[v4.Key, v4.Log]().
			Entries(&stored).
			Exec(ctx, db)).To(Succeed())
		Expect(stored).To(ConsistOf(SatisfyAll(
			HaveField("Key", owned.Key),
			HaveField("Project", HaveValue(Equal(proj))),
		)))
	})
})
