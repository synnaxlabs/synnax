// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v3 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func viewTab(viewType string) v3.Tab {
	return v3.Tab{Variant: v3.ViewTab{
		TabBase: v3.TabBase{Key: uuid.New()},
		View:    v3.View{Type: viewType},
	}}
}

func leaf(tabs ...v3.Tab) v3.Node {
	return v3.Node{Variant: v3.LeafNode{Tabs: tabs}}
}

func split(first, last v3.Node) v3.Node {
	return v3.Node{Variant: v3.SplitNode{
		Direction: spatial.DirectionX,
		Size:      0.5,
		First:     first,
		Last:      last,
	}}
}

var _ = Describe("ArcExplorerTabsMigration", func() {
	var db *gorp.DB
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v3.Key, v3.Panel]{DB: db}))
	})

	migrate := func(ctx SpecContext, panels ...v3.Panel) []v3.Panel {
		GinkgoHelper()
		Expect(gorp.NewCreate[v3.Key, v3.Panel]().Entries(&panels).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Panel",
			Migrations: []gorp.Migration{v3.ArcExplorerTabsMigration},
		})).To(Succeed())
		got := make([]v3.Panel, len(panels))
		for i, p := range panels {
			Expect(gorp.NewRetrieve[v3.Key, v3.Panel]().
				Where(gorp.MatchKeys[v3.Key, v3.Panel](p.Key)).
				Entry(&got[i]).
				Exec(ctx, db)).To(Succeed())
		}
		return got
	}

	It("Should remove each Arc explorer tab and keep every other tab", func(
		ctx SpecContext,
	) {
		rangeExplorer, docs := viewTab("range_explorer"), viewTab("docs")
		log := v3.Tab{Variant: v3.ResourceTab{
			TabBase:  v3.TabBase{Key: uuid.New()},
			Resource: resource.ID{Type: resource.TypeLog, Key: uuid.New().String()},
		}}
		p := v3.Panel{
			Key:  uuid.New(),
			Name: "Ops",
			Root: split(
				leaf(viewTab("arc_explorer"), rangeExplorer, log),
				split(leaf(viewTab("arc_explorer")), leaf(docs)),
			),
		}
		got := migrate(ctx, p)
		Expect(got[0].Name).To(Equal("Ops"))
		root := got[0].Root.Variant.(v3.SplitNode)
		Expect(root.First).To(Equal(leaf(rangeExplorer, log)))
		last := root.Last.Variant.(v3.SplitNode)
		Expect(last.First.Variant.(v3.LeafNode).Tabs).To(BeEmpty())
		Expect(last.Last).To(Equal(leaf(docs)))
	})

	It("Should leave a panel with no Arc explorer tab unchanged", func(
		ctx SpecContext,
	) {
		p := v3.Panel{
			Key:  uuid.New(),
			Name: "Ops",
			Root: leaf(viewTab("status_explorer"), viewTab("selector")),
		}
		Expect(migrate(ctx, p)).To(Equal([]v3.Panel{p}))
	})
})

var _ = Describe("StripArcExplorerTabs", func() {
	It("Should remove the Arc explorer tabs of a leaf", func() {
		docs := viewTab("docs")
		n := leaf(viewTab("arc_explorer"), docs)
		Expect(v3.StripArcExplorerTabs(&n)).To(BeTrue())
		Expect(n).To(Equal(leaf(docs)))
	})

	It("Should change only the side of a split that holds one", func() {
		docs, selector := viewTab("docs"), viewTab("selector")
		n := split(leaf(docs), leaf(viewTab("arc_explorer"), selector))
		Expect(v3.StripArcExplorerTabs(&n)).To(BeTrue())
		Expect(n).To(Equal(split(leaf(docs), leaf(selector))))
	})

	It("Should report no change for a tree with no Arc explorer tab", func() {
		n := split(leaf(viewTab("docs")), leaf(viewTab("selector")))
		want := n
		Expect(v3.StripArcExplorerTabs(&n)).To(BeFalse())
		Expect(n).To(Equal(want))
	})
})
