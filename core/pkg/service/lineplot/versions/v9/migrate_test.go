// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v9_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
	v9 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v9"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It(
		"Should delete each range-owned LinePlot and lift the rest",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v8.Key, v8.LinePlot]{DB: db}))
			proj := uuid.New()
			owned := v8.LinePlot{Key: uuid.New(), Name: "Owned", Project: &proj}
			ranged := v8.LinePlot{
				Key:   uuid.New(),
				Name:  "Ranged",
				Range: new(uuid.New()),
			}
			Expect(gorp.NewCreate[v8.Key, v8.LinePlot]().
				Entries(&[]v8.LinePlot{owned, ranged}).
				Exec(ctx, db)).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "LinePlot",
				Migrations: []gorp.Migration{v9.RangeMigration, v9.Migration},
			})).To(Succeed())
			var stored []v9.LinePlot
			Expect(gorp.NewRetrieve[v9.Key, v9.LinePlot]().
				Entries(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored).To(ConsistOf(SatisfyAll(
				HaveField("Key", owned.Key),
				HaveField("Project", HaveValue(Equal(proj))),
			)))
		},
	)
})
