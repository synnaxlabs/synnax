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
	v3 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v4"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should delete each range-owned Panel and lift the rest", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v3.Key, v3.Panel]{DB: db}))
		proj := uuid.New()
		root := v3.Node{Variant: v3.LeafNode{}}
		owned := v3.Panel{Key: uuid.New(), Name: "Owned", Project: &proj, Root: root}
		ranged := v3.Panel{
			Key:   uuid.New(),
			Name:  "Ranged",
			Range: new(uuid.New()),
			Root:  root,
		}
		Expect(gorp.NewCreate[v3.Key, v3.Panel]().
			Entries(&[]v3.Panel{owned, ranged}).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Panel",
			Migrations: []gorp.Migration{v4.RangeMigration, v4.Migration},
		})).To(Succeed())
		var stored []v4.Panel
		Expect(gorp.NewRetrieve[v4.Key, v4.Panel]().
			Entries(&stored).
			Exec(ctx, db)).To(Succeed())
		Expect(stored).To(ConsistOf(SatisfyAll(
			HaveField("Key", owned.Key),
			HaveField("Project", HaveValue(Equal(proj))),
		)))
	})
})
