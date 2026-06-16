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
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Project layout to panel migration", func() {
	openPanelTable := func(
		ctx context.Context, db *gorp.DB,
	) *gorp.Table[panel.Key, panel.Panel] {
		return MustOpen(gorp.OpenTable[panel.Key, panel.Panel](
			ctx, gorp.TableConfig[panel.Key, panel.Panel]{
				DB: db,
				Migrations: []migrate.Migration{
					gorp.CodecMigration[panel.Key, panel.Panel]("msgpack_to_orc"),
					migrate.WithAddedDeps(
						gorp.NewMigration(
							"v56_migrate_project_layouts_to_panels",
							panel.MigrateProjectLayouts(),
						),
						"msgpack_to_orc",
					),
				},
			},
		))
	}
	// stageLayout stages a project's layout blob under its staging key, mirroring the
	// project migration's Phase 1 so the panel migration finds the layout to convert.
	stageLayout := func(ctx context.Context, db *gorp.DB, p project.Project) {
		if len(p.Layout) == 0 {
			return
		}
		blob := MustSucceed(json.Marshal(p.Layout))
		Expect(db.Set(ctx, project.LegacyLayoutKVKey(p.Key), blob)).To(Succeed())
	}
	seedResources := func(ctx context.Context, db *gorp.DB, ids ...ontology.ID) {
		table := MustOpen(gorp.OpenTable[string, ontology.Resource](
			ctx, gorp.TableConfig[string, ontology.Resource]{DB: db},
		))
		for _, id := range ids {
			Expect(table.NewCreate().Entry(&ontology.Resource{ID: id}).
				Exec(ctx, db)).To(Succeed())
		}
	}
	collectPanels := func(ctx context.Context, db *gorp.DB) []panel.Panel {
		seq, closer := MustSucceed2(
			gorp.WrapReader[panel.Key, panel.Panel](db).OpenNexter(ctx),
		)
		DeferClose(closer)
		var out []panel.Panel
		for p := range seq {
			out = append(out, p)
		}
		return out
	}
	findPanel := func(panels []panel.Panel, name string) panel.Panel {
		for _, p := range panels {
			if p.Name == name {
				return p
			}
		}
		Fail("no panel named " + name)
		return panel.Panel{}
	}
	// zeroTabKeys asserts every tab in the tree was assigned a fresh key and zeroes
	// the keys so trees can be compared structurally.
	var zeroTabKeys func(n *panel.Node)
	zeroTabKeys = func(n *panel.Node) {
		if n == nil {
			return
		}
		switch v := n.Variant.(type) {
		case panel.NodeLeaf:
			for i := range v.Tabs {
				Expect(v.Tabs[i].Key).ToNot(Equal(uuid.Nil))
				v.Tabs[i].Key = uuid.Nil
			}
			n.Variant = v
		case panel.NodeSplit:
			zeroTabKeys(&v.First)
			zeroTabKeys(&v.Last)
			n.Variant = v
		}
	}
	rel := func(from, to ontology.ID) ontology.Relationship {
		return ontology.Relationship{
			From: from,
			Type: ontology.RelationshipTypeParentOf,
			To:   to,
		}
	}
	hasRel := func(ctx context.Context, db *gorp.DB, r ontology.Relationship) bool {
		return MustSucceed(gorp.NewRetrieve[string, ontology.Relationship]().
			Where(gorp.MatchKeys[string, ontology.Relationship](r.GorpKey())).
			Exists(ctx, db))
	}
	resourceTab := func(t ontology.ResourceType, key string) panel.Tab {
		return panel.Tab{
			Type: string(t),
			Args: msgpack.EncodedJSON{"resourceKey": key},
		}
	}
	leaf := func(tabs ...panel.Tab) *panel.Node {
		return &panel.Node{Variant: panel.NodeLeaf{Leaf: panel.Leaf{Tabs: tabs}}}
	}
	mosaicTab := func(tabKey string) map[string]any {
		return map[string]any{"tabKey": tabKey, "name": "Tab " + tabKey}
	}
	vizLayout := func(key, layoutType string) map[string]any {
		return map[string]any{
			"key":      key,
			"type":     layoutType,
			"name":     "Viz " + key,
			"location": "mosaic",
		}
	}

	It("Should convert mosaics into panels parented under the project", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		projectKey := uuid.New()
		lpKey, scKey, logKey, tblKey := uuid.NewString(), uuid.NewString(),
			uuid.NewString(), uuid.NewString()
		staleKey := uuid.NewString()
		seedResources(ctx, db,
			ontology.ID{Type: ontology.ResourceTypeLineplot, Key: lpKey},
			ontology.ID{Type: ontology.ResourceTypeSchematic, Key: scKey},
			ontology.ID{Type: ontology.ResourceTypeLog, Key: logKey},
			ontology.ID{Type: ontology.ResourceTypeTable, Key: tblKey},
		)
		stageLayout(ctx, db, project.Project{
			Key:  projectKey,
			Name: "Ops",
			Layout: msgpack.EncodedJSON{
				"mosaics": map[string]any{
					"main": map[string]any{
						"activeTab": lpKey,
						"root": map[string]any{
							"key":       1,
							"direction": "x",
							"size":      0.25,
							"first": map[string]any{
								"key":       2,
								"direction": "y",
								"first": map[string]any{
									"key":  4,
									"tabs": []any{mosaicTab(lpKey)},
								},
								"last": map[string]any{
									"key":  5,
									"tabs": []any{mosaicTab(scKey)},
								},
							},
							"last": map[string]any{
								"key": 3,
								"tabs": []any{
									mosaicTab(logKey),
									// Inline app view: no backing resource, dropped.
									mosaicTab("docs"),
									// Viz whose resource was deleted: dropped.
									mosaicTab(staleKey),
									// Tab with no layout entry at all: dropped.
									mosaicTab("orphan"),
								},
							},
						},
					},
					"tableWin": map[string]any{
						"activeTab": tblKey,
						"root": map[string]any{
							"key":  1,
							"tabs": []any{mosaicTab(tblKey)},
						},
					},
				},
				"layouts": map[string]any{
					lpKey:    vizLayout(lpKey, "lineplot"),
					scKey:    vizLayout(scKey, "schematic"),
					logKey:   vizLayout(logKey, "log"),
					tblKey:   vizLayout(tblKey, "table"),
					staleKey: vizLayout(staleKey, "lineplot"),
					"docs": map[string]any{
						"key":      "docs",
						"type":     "docs",
						"name":     "Documentation",
						"location": "mosaic",
					},
					"main": map[string]any{
						"key":      "main",
						"type":     "main",
						"name":     "Main",
						"location": "window",
					},
					"tableWin": map[string]any{
						"key":      "tableWin",
						"type":     "mosaic",
						"name":     "Table Window",
						"location": "window",
					},
				},
			},
		})

		openPanelTable(ctx, db)
		panels := collectPanels(ctx, db)
		Expect(panels).To(HaveLen(2))

		main := findPanel(panels, "Main")
		zeroTabKeys(&main.Root)
		Expect(main.Root).To(Equal(panel.Node{Variant: panel.NodeSplit{Split: panel.Split{
			Direction: spatial.DirectionX,
			Size:      0.25,
			First: panel.Node{Variant: panel.NodeSplit{Split: panel.Split{
				Direction: spatial.DirectionY,
				Size:      0.5,
				First:     *leaf(resourceTab(ontology.ResourceTypeLineplot, lpKey)),
				Last:      *leaf(resourceTab(ontology.ResourceTypeSchematic, scKey)),
			}}},
			Last: *leaf(resourceTab(ontology.ResourceTypeLog, logKey)),
		}}}))

		tableWin := findPanel(panels, "Table Window")
		zeroTabKeys(&tableWin.Root)
		Expect(tableWin.Root).To(Equal(
			*leaf(resourceTab(ontology.ResourceTypeTable, tblKey)),
		))

		By("Defining an ontology resource and parent relationship for each panel")
		for _, p := range panels {
			panelID := panel.OntologyID(p.Key)
			Expect(MustSucceed(gorp.NewRetrieve[string, ontology.Resource]().
				Where(gorp.MatchKeys[string, ontology.Resource](panelID.String())).
				Exists(ctx, db))).To(BeTrue())
			Expect(hasRel(ctx, db, rel(project.OntologyID(projectKey), panelID))).
				To(BeTrue())
		}

		By("Deleting the staging entry once it has been consumed")
		Expect(db.Get(ctx, project.LegacyLayoutKVKey(projectKey))).Error().
			To(MatchError(query.ErrNotFound))
	})

	It("Should collapse splits whose sides lose all of their tabs", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		lpKey, staleKey := uuid.NewString(), uuid.NewString()
		seedResources(ctx, db, ontology.ID{
			Type: ontology.ResourceTypeLineplot, Key: lpKey,
		})
		stageLayout(ctx, db, project.Project{
			Key:  uuid.New(),
			Name: "Ops",
			Layout: msgpack.EncodedJSON{
				"mosaics": map[string]any{
					"main": map[string]any{
						"root": map[string]any{
							"key":       1,
							"direction": "x",
							"size":      0.7,
							"first": map[string]any{
								"key":  2,
								"tabs": []any{mosaicTab(lpKey)},
							},
							"last": map[string]any{
								"key":  3,
								"tabs": []any{mosaicTab(staleKey)},
							},
						},
					},
				},
				"layouts": map[string]any{
					lpKey:    vizLayout(lpKey, "lineplot"),
					staleKey: vizLayout(staleKey, "lineplot"),
				},
			},
		})

		openPanelTable(ctx, db)
		panels := collectPanels(ctx, db)
		Expect(panels).To(HaveLen(1))
		// No "main" layout entry was seeded, so the panel name falls back to the
		// window key.
		Expect(panels[0].Name).To(Equal("main"))
		root := panels[0].Root
		zeroTabKeys(&root)
		Expect(root).To(Equal(*leaf(resourceTab(ontology.ResourceTypeLineplot, lpKey))))
	})

	It("Should skip corrupt or unmigratable staged layouts", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		stageLayout(ctx, db, project.Project{
			Key:    uuid.New(),
			Name:   "Corrupt",
			Layout: msgpack.EncodedJSON{"mosaics": "garbage"},
		})
		stageLayout(ctx, db, project.Project{
			Key:  uuid.New(),
			Name: "Unmigratable",
			Layout: msgpack.EncodedJSON{
				"mosaics": map[string]any{
					"main": map[string]any{
						"root": map[string]any{
							"key":  1,
							"tabs": []any{mosaicTab("docs")},
						},
					},
				},
				"layouts": map[string]any{
					"docs": map[string]any{
						"key":      "docs",
						"type":     "docs",
						"name":     "Documentation",
						"location": "mosaic",
					},
				},
			},
		})

		openPanelTable(ctx, db)
		Expect(collectPanels(ctx, db)).To(BeEmpty())
	})

	It("Should be a no-op on a cluster with no projects", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		openPanelTable(ctx, db)
		Expect(collectPanels(ctx, db)).To(BeEmpty())
	})
})
