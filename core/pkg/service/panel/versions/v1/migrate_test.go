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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v0 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v1"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("MigrateView", func() {
	It("Should keep the view's type and drop its args", func(ctx SpecContext) {
		Expect(v1.MigrateView(ctx, v0.View{
			Type: "docs",
			Args: map[string]any{"url": "/reference"},
		})).To(Equal(v1.View{Type: "docs"}))
	})
})

var _ = Describe("Migration", func() {
	It("Should drop the args of every view tab in the tree", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Panel]{DB: db}))
		viewKey, resourceKey, nestedKey := uuid.New(), uuid.New(), uuid.New()
		resource := ontology.ID{Type: ontology.ResourceTypeLineplot, Key: "lp"}
		seed := v0.Panel{
			Key:  uuid.New(),
			Name: "Main",
			Root: v0.Node{Variant: v0.SplitNode{
				Direction: spatial.DirectionX,
				Size:      0.25,
				First: v0.Node{Variant: v0.LeafNode{Tabs: []v0.Tab{
					{Variant: v0.ViewTab{
						TabBase: v0.TabBase{Key: viewKey},
						View: v0.View{
							Type: "docs",
							Args: map[string]any{"url": "/reference"},
						},
					}},
					{Variant: v0.ResourceTab{
						TabBase:  v0.TabBase{Key: resourceKey},
						Resource: resource,
					}},
				}}},
				Last: v0.Node{Variant: v0.SplitNode{
					Direction: spatial.DirectionY,
					Size:      0.5,
					First: v0.Node{Variant: v0.LeafNode{Tabs: []v0.Tab{
						{Variant: v0.ViewTab{
							TabBase: v0.TabBase{Key: nestedKey},
							View: v0.View{
								Type: "range_explorer",
								Args: map[string]any{"filter": "all"},
							},
						}},
					}}},
					Last: v0.Node{Variant: v0.LeafNode{Tabs: []v0.Tab{}}},
				}},
			}},
		}
		Expect(gorp.NewCreate[v0.Key, v0.Panel]().Entry(&seed).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Composition",
			Migrations: []gorp.Migration{v1.Migration},
		})).To(Succeed())
		var got v1.Panel
		Expect(gorp.NewRetrieve[v1.Key, v1.Panel]().
			Where(gorp.MatchKeys[v1.Key, v1.Panel](seed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got).To(Equal(v1.Panel{
			Key:  seed.Key,
			Name: "Main",
			Root: v1.Node{Variant: v1.SplitNode{
				Direction: spatial.DirectionX,
				Size:      0.25,
				First: v1.Node{Variant: v1.LeafNode{Tabs: []v1.Tab{
					{Variant: v1.ViewTab{
						TabBase: v1.TabBase{Key: viewKey},
						View:    v1.View{Type: "docs"},
					}},
					{Variant: v1.ResourceTab{
						TabBase:  v1.TabBase{Key: resourceKey},
						Resource: resource,
					}},
				}}},
				Last: v1.Node{Variant: v1.SplitNode{
					Direction: spatial.DirectionY,
					Size:      0.5,
					First: v1.Node{Variant: v1.LeafNode{Tabs: []v1.Tab{
						{Variant: v1.ViewTab{
							TabBase: v1.TabBase{Key: nestedKey},
							View:    v1.View{Type: "range_explorer"},
						}},
					}}},
					Last: v1.Node{Variant: v1.LeafNode{Tabs: []v1.Tab{}}},
				}},
			}},
		}))
	})
})
