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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should keep each panel's tree and leave its icon and color absent", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v1.Key, v1.Panel]{DB: db}))
		old := v1.Panel{
			Key:  uuid.New(),
			Name: "Main",
			Root: v1.Node{Variant: v1.LeafNode{Tabs: []v1.Tab{
				{Variant: v1.ViewTab{
					TabBase: v1.TabBase{Key: uuid.New()},
					View:    v1.View{Type: "docs"},
				}},
			}}},
		}
		Expect(gorp.NewCreate[v1.Key, v1.Panel]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Composition",
			Migrations: []gorp.Migration{v2.Migration},
		})).To(Succeed())
		var got v2.Panel
		Expect(gorp.NewRetrieve[v2.Key, v2.Panel]().
			Where(gorp.MatchKeys[v2.Key, v2.Panel](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got).To(Equal(v2.Panel{Key: old.Key, Name: "Main", Root: old.Root}))
	})
})
