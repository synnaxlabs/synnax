// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	"context"
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/panel/versions"
	v0 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v5"
	project "github.com/synnaxlabs/synnax/pkg/service/project/versions/v1"
	projectv3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	color "github.com/synnaxlabs/x/color/versions/v1"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migrations", func() {
	var (
		ctx context.Context
		db  *gorp.DB
	)
	BeforeEach(func(sCtx SpecContext) {
		ctx = sCtx
		db = DeferClose(gorp.Wrap(memkv.New()))
	})
	// recordComposition marks the given migrations as run by the composition pass.
	recordComposition := func(migrations ...gorp.Migration) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Composition",
			Migrations: migrations,
		})).To(Succeed())
	}
	openTable := func() *gorp.Table[versions.Key, versions.Panel] {
		GinkgoHelper()
		return MustOpen(
			gorp.OpenTable(ctx, gorp.TableConfig[versions.Key, versions.Panel]{
				DB:         db,
				Migrations: versions.Migrations,
			}),
		)
	}
	// own stores a project and its edge to the panel with key, and returns its key.
	own := func(key versions.Key) uuid.UUID {
		GinkgoHelper()
		owner := uuid.New()
		Expect(gorp.WrapWriter[projectv3.Key, projectv3.Project](db).
			Set(ctx, projectv3.Project{Key: owner, Name: "Owner"})).To(Succeed())
		Expect(gorp.WrapWriter[string, ontology.Relationship](db).Set(
			ctx,
			ontology.Relationship{
				From: project.Project{Key: owner}.ResourceID(),
				Type: ontology.RelationshipTypeParentOf,
				To:   resource.ID{Type: resource.TypePanel, Key: key.String()},
			},
		)).To(Succeed())
		return owner
	}
	retrieve := func(key versions.Key) versions.Panel {
		GinkgoHelper()
		var p versions.Panel
		Expect(gorp.NewRetrieve[versions.Key, versions.Panel]().
			Where(gorp.MatchKeys[versions.Key, versions.Panel](key)).
			Entry(&p).
			Exec(ctx, db)).To(Succeed())
		return p
	}

	Describe("Table chain", func() {
		It("Should lift the v0 panels of a v0.58 Core to the latest version", func() {
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Panel]{
				DB:         db,
				Migrations: []gorp.Migration{v0.Migration},
			}))
			old := v0.Panel{
				Key:  uuid.New(),
				Name: "Main",
				Root: v0.Node{Variant: v0.SplitNode{
					Direction: spatial.DirectionX,
					Size:      0.5,
					First: v0.Node{Variant: v0.LeafNode{Tabs: []v0.Tab{
						{Variant: v0.ViewTab{
							Key:  uuid.New(),
							Type: "docs",
							Args: msgpack.EncodedJSON{"page": "intro"},
						}},
						{Variant: v0.ResourceTab{
							Key:      uuid.New(),
							Resource: resource.ID{Type: resource.TypeLog, Key: "a"},
						}},
					}}},
					Last: v0.Node{Variant: v0.LeafNode{Tabs: []v0.Tab{
						{Variant: v0.ResourceTab{
							Key:      uuid.New(),
							Resource: resource.ID{Type: resource.TypeTable, Key: "b"},
						}},
					}}},
				}},
			}
			Expect(gorp.WrapWriter[v0.Key, v0.Panel](db).Set(ctx, old)).To(Succeed())
			owner := own(old.Key)
			recordComposition(v0.ResourceTabTypesMigration)
			openTable()
			p1 := MustSucceed(v1.MigratePanel(ctx, old))
			p3 := MustSucceed(
				v3.MigratePanel(ctx, MustSucceed(v2.MigratePanel(ctx, p1))),
			)
			want := MustSucceed(
				v5.MigratePanel(ctx, MustSucceed(v4.MigratePanel(ctx, p3))),
			)
			want.Project = &owner
			Expect(retrieve(old.Key)).To(Equal(want))
		})

		It("Should not lift again the panels of a v0.60 Core", func() {
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v2.Key, v2.Panel]{
				DB:         db,
				Migrations: []gorp.Migration{v0.Migration},
			}))
			icon := "valve"
			stored := v2.Panel{
				Key:   uuid.New(),
				Name:  "Main",
				Icon:  &icon,
				Color: &color.Color{R: 1, A: 0.5},
				Root: v2.Node{Variant: v2.LeafNode{Tabs: []v2.Tab{
					{Variant: v2.ViewTab{
						TabBase: v2.TabBase{Key: uuid.New()},
						View:    v2.View{Type: "docs"},
					}},
				}}},
			}
			recordComposition(
				v0.ResourceTabTypesMigration,
				v1.Migration,
				v2.Migration,
			)
			Expect(gorp.WrapWriter[v2.Key, v2.Panel](db).Set(ctx, stored)).
				To(Succeed())
			owner := own(stored.Key)
			openTable()
			want := MustSucceed(v5.MigratePanel(
				ctx,
				MustSucceed(
					v4.MigratePanel(ctx, MustSucceed(v3.MigratePanel(ctx, stored))),
				),
			))
			want.Project = &owner
			got := retrieve(stored.Key)
			Expect(got).To(Equal(want))
			Expect(got.Color).To(HaveValue(Equal(colorv2.Color{R: 1, A: 128})))
		})

		It("Should remove the Arc explorer tabs of a v0.58 Core", func() {
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Panel]{
				DB:         db,
				Migrations: []gorp.Migration{v0.Migration},
			}))
			docs := uuid.New()
			old := v0.Panel{
				Key:  uuid.New(),
				Name: "Main",
				Root: v0.Node{Variant: v0.LeafNode{Tabs: []v0.Tab{
					{Variant: v0.ViewTab{Key: uuid.New(), Type: "arc_explorer"}},
					{Variant: v0.ViewTab{Key: docs, Type: "docs"}},
				}}},
			}
			Expect(gorp.WrapWriter[v0.Key, v0.Panel](db).Set(ctx, old)).To(Succeed())
			own(old.Key)
			recordComposition(v0.ResourceTabTypesMigration)
			openTable()
			tabs := retrieve(old.Key).Root.Variant.(versions.LeafNode).Tabs
			Expect(tabs).To(HaveLen(1))
			Expect(tabs[0].Variant.(versions.ViewTab).Key).To(Equal(docs))
		})

		It("Should remove the Arc explorer tabs of a v0.60 Core", func() {
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v2.Key, v2.Panel]{
				DB:         db,
				Migrations: []gorp.Migration{v0.Migration},
			}))
			docs := v2.Tab{Variant: v2.ViewTab{
				TabBase: v2.TabBase{Key: uuid.New()},
				View:    v2.View{Type: "docs"},
			}}
			stored := v2.Panel{
				Key:  uuid.New(),
				Name: "Main",
				Root: v2.Node{Variant: v2.SplitNode{
					Direction: spatial.DirectionX,
					Size:      0.5,
					First: v2.Node{Variant: v2.LeafNode{Tabs: []v2.Tab{
						{Variant: v2.ViewTab{
							TabBase: v2.TabBase{Key: uuid.New()},
							View:    v2.View{Type: "arc_explorer"},
						}},
					}}},
					Last: v2.Node{Variant: v2.LeafNode{Tabs: []v2.Tab{docs}}},
				}},
			}
			recordComposition(
				v0.ResourceTabTypesMigration,
				v1.Migration,
				v2.Migration,
			)
			Expect(gorp.WrapWriter[v2.Key, v2.Panel](db).Set(ctx, stored)).
				To(Succeed())
			own(stored.Key)
			openTable()
			root := retrieve(stored.Key).Root.Variant.(versions.SplitNode)
			Expect(root.First.Variant.(versions.LeafNode).Tabs).To(BeEmpty())
			Expect(root.Last.Variant.(versions.LeafNode).Tabs).
				To(Equal([]versions.Tab{docs}))
		})
	})

	Describe("Composition", func() {
		It("Should store each panel of a legacy layout at the latest version", func() {
			lpKey := uuid.New().String()
			lpID := resource.ID{Type: resource.TypeLineplot, Key: lpKey}
			Expect(gorp.WrapWriter[string, ontology.Resource](db).
				Set(ctx, ontology.Resource{ID: lpID})).To(Succeed())
			projectKey := uuid.New()
			blob := MustSucceed(json.Marshal(map[string]any{
				"mosaics": map[string]any{
					"main": map[string]any{"root": map[string]any{
						"key":  1,
						"tabs": []any{map[string]any{"tabKey": lpKey}},
					}},
				},
				"layouts": map[string]any{
					lpKey: map[string]any{"key": lpKey, "type": "lineplot"},
				},
			}))
			Expect(db.Set(
				ctx, []byte(project.LegacyLayoutKVPrefix+projectKey.String()), blob,
			)).To(Succeed())
			table := openTable()
			recordComposition(versions.CompositionMigrations(table)...)

			var panels []versions.Panel
			Expect(table.NewRetrieve().Entries(&panels).Exec(ctx, db)).To(Succeed())
			Expect(panels).To(HaveLen(1))
			lf := panels[0].Root.Variant.(versions.LeafNode)
			Expect(lf.Tabs).To(HaveLen(1))
			Expect(lf.Tabs[0].Variant.(versions.ResourceTab).Resource).To(Equal(lpID))
			Expect(panels[0].Project).To(Equal(&projectKey))
		})
	})

	// Applied migrations are tracked by key, so renaming or reordering an entry
	// re-runs or skips it on stores that already migrated. The chain is pinned
	// verbatim; extend it for a new version, never edit shipped entries.
	It("Should pin the migration chain keys in order", func() {
		Expect(lo.Map(
			versions.Migrations,
			func(m gorp.Migration, _ int) string { return m.Key() },
		)).To(Equal([]string{
			"msgpack_to_orc",
			"v56_strip_invalid_resource_tabs",
			"v60_drop_view_args",
			"v60_add_icon_and_color",
			"v61_add_group",
			"v61_panel_group_membership",
			"v61_panel_owner",
			"v61_strip_arc_explorer_tabs",
			"v62_panel_delete_range_owned",
			"v62_panel_drop_range",
			"v61_color_alpha_byte",
		}))
	})
})
