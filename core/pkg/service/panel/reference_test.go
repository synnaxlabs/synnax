// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Tab references", func() {
	retrieve := func(ctx context.Context, db *gorp.DB, key panel.Key) panel.Panel {
		GinkgoHelper()
		var res panel.Panel
		Expect(gorp.NewRetrieve[panel.Key, panel.Panel]().
			Where(gorp.MatchKeys[panel.Key, panel.Panel](key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}

	It("Should refuse a tab whose resource does not exist", func(ctx SpecContext) {
		key := uuid.New()
		p := panel.Panel{Name: "missing", Root: leafNode(tab(key)), Project: &proj.Key}
		Expect(svc.NewWriter(nil).Create(ctx, &p)).To(MatchError(ContainSubstring(
			"root.tabs.resource: lineplot " + key.String() + " does not exist",
		)))
	})

	It(
		"Should remove the tab of a deleted resource and collapse its leaf",
		func(ctx SpecContext) {
			kept, deleted := uuid.New(), uuid.New()
			createPlot(ctx, nil, kept)
			createPlot(ctx, nil, deleted)
			p := panel.Panel{
				Name: "detach",
				Root: splitNode(
					spatial.DirectionX,
					0.5,
					leafNode(tab(kept), viewTab(uuid.New(), "docs")),
					leafNode(tab(deleted)),
				),
				Project: &proj.Key,
			}
			Expect(writer.Create(ctx, &p)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(writer.Delete(ctx, p.Key)).To(Succeed())
			})
			Expect(plotSvc.NewWriter(nil).Delete(ctx, deleted)).To(Succeed())
			leaf := MustBeOk(asLeaf(retrieve(ctx, db, p.Key).Root))
			Expect(leaf.Tabs).To(HaveLen(2))
			Expect(leaf.Tabs[0].Key()).To(Equal(kept))
		},
	)

	It(
		"Should remove the tabs of resources the Core does not hold on migration",
		func(ctx SpecContext) {
			kept := uuid.New()
			createPlot(ctx, nil, kept)
			stored := panel.Panel{
				Key:  uuid.New(),
				Name: "stored",
				Root: splitNode(
					spatial.DirectionY,
					0.5,
					leafNode(tab(uuid.New())),
					leafNode(tab(kept)),
				),
				Project: &proj.Key,
			}
			Expect(gorp.NewCreate[panel.Key, panel.Panel]().
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(writer.Delete(ctx, stored.Key)).To(Succeed())
			})
			migrations := svc.CompositionMigrations()
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "panel_dangling",
				Migrations: migrations[len(migrations)-1:],
			})).To(Succeed())
			leaf := MustBeOk(asLeaf(retrieve(ctx, db, stored.Key).Root))
			Expect(leaf.Tabs).To(HaveLen(1))
			Expect(leaf.Tabs[0].Key()).To(Equal(kept))
		},
	)
})
