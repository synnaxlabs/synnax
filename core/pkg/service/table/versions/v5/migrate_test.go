// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v4 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v5"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should delete each range-owned Table and lift the rest", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v4.Key, v4.Table]{DB: db}))
		proj := uuid.New()
		owned := v4.Table{Key: uuid.New(), Name: "Owned", Project: &proj}
		ranged := v4.Table{Key: uuid.New(), Name: "Ranged", Range: new(uuid.New())}
		Expect(gorp.NewCreate[v4.Key, v4.Table]().
			Entries(&[]v4.Table{owned, ranged}).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Table",
			Migrations: []gorp.Migration{v5.RangeMigration, v5.Migration},
		})).To(Succeed())
		var stored []v5.Table
		Expect(gorp.NewRetrieve[v5.Key, v5.Table]().
			Entries(&stored).
			Exec(ctx, db)).To(Succeed())
		Expect(stored).To(ConsistOf(SatisfyAll(
			HaveField("Key", owned.Key),
			HaveField("Project", HaveValue(Equal(proj))),
		)))
	})
})
