// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	authv0 "github.com/synnaxlabs/synnax/pkg/service/auth/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	groupv0 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	lineplotversions "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions"
	lineplotv0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	logv0 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v0"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	panelversions "github.com/synnaxlabs/synnax/pkg/service/panel/versions"
	panelv0 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	projectv0 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	schematicv0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	tablev0 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskv0 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v0"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	userv0 "github.com/synnaxlabs/synnax/pkg/service/user/versions/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"golang.org/x/crypto/bcrypt"
)

var _ = Describe("Layer", func() {
	It("Should cap external channels at the license's limit", func(ctx SpecContext) {
		signer := svctestutil.NewSigner()
		lic := svctestutil.NewLicense()
		lic.Channels = 50
		node := testutil.NewNode(ctx)
		layer := MustOpen(
			svctestutil.OpenLayer(ctx, node, service.LayerConfig{
				License: license.ServiceConfig{
					Key:     signer.Sign(lic),
					Anchors: signer.Anchors,
				},
			}),
		)
		var existing []channel.Channel
		Expect(layer.Channel.NewRetrieve().Entries(&existing).Exec(ctx, node.DB)).
			To(Succeed())
		external := lo.CountBy(existing, func(ch channel.Channel) bool {
			return !ch.Internal && !ch.Virtual
		})
		newIndices := func(n int) []channel.Channel {
			return lo.Times(n, func(int) channel.Channel {
				return channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.TimestampT,
					IsIndex:     true,
					Leaseholder: 1,
				}
			})
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).
				CreateMany(ctx, new(newIndices(50-external)))
		})).To(Succeed())
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).CreateMany(ctx, new(newIndices(1)))
		})).To(And(
			MatchError(license.ErrTooMany),
			MatchError(ContainSubstring("limit is 50 channels")),
		))
	})

	It("Should refuse a framer writer on a change channel", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		layer := MustOpen(svctestutil.OpenLayer(ctx, node))
		var ch channel.Channel
		Expect(layer.Channel.NewRetrieve().
			Where(channel.MatchNames("sy_range_set")).
			Entry(&ch).
			Exec(ctx, node.DB)).To(Succeed())
		Expect(layer.Framer.NewStreamWriter(ctx, framer.WriterConfig{
			Keys:  channel.Keys{ch.Key()},
			Start: telem.SecondTS,
		})).Error().To(MatchError(fmt.Sprintf(
			"channel %d holds changes that only the Core writes: validation error",
			ch.Key(),
		)))
	})

	It("Should delete a stored snapshot task", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		legacyDB := gorp.Wrap(node.DB.KV(), gorp.WithCodec(msgpack.Codec))
		stored := []taskv0.Task{
			{
				Key:      taskv0.Key(uint64(65537)<<32 | 1),
				Name:     "Snapshot",
				Type:     pagerduty.AlertTaskType,
				Config:   msgpack.EncodedJSON{"routing_key": "rk"},
				Snapshot: true,
			},
			{
				Key:    taskv0.Key(uint64(65537)<<32 | 2),
				Name:   "Live",
				Type:   pagerduty.AlertTaskType,
				Config: msgpack.EncodedJSON{"routing_key": "rk"},
			},
		}
		for _, t := range stored {
			Expect(gorp.WrapWriter[taskv0.Key, taskv0.Task](legacyDB).Set(ctx, t)).
				To(Succeed())
		}
		layer := MustOpen(svctestutil.OpenLayer(ctx, node))
		Expect(layer.Task.NewRetrieve().
			Where(task.MatchNames("Snapshot")).
			Exists(ctx, node.DB)).To(BeFalse())
		Expect(layer.Task.NewRetrieve().
			Where(task.MatchNames("Live")).
			Exists(ctx, node.DB)).To(BeTrue())
	})

	It(
		"Should end root sessions only when the root password changes",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			open := func(pwd string) *service.Layer {
				GinkgoHelper()
				return MustSucceed(svctestutil.OpenLayer(ctx, node, service.LayerConfig{
					RootCredentials: auth.Credentials{Username: "root", Password: pwd},
				}))
			}
			first := open("p1")
			_, token := MustSucceed2(first.Auth.Login(ctx, auth.Proof{
				Variant: auth.PasswordProof{
					Credentials: auth.Credentials{Username: "root", Password: "p1"},
				},
			}))
			Expect(first.Close()).To(Succeed())
			second := open("p1")
			MustSucceed(second.Auth.Authenticate(ctx, token))
			Expect(second.Close()).To(Succeed())
			third := open("p2")
			Expect(third.Auth.Authenticate(ctx, token)).Error().
				To(MatchError(auth.ErrInvalidToken))
			Expect(third.Close()).To(Succeed())
		},
	)

	It(
		"Should log in a user whose password a Core before 0.59 stored",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			alice := userv0.User{Key: uuid.New(), Username: "alice"}
			hash := MustSucceed(
				bcrypt.GenerateFromPassword([]byte("p"), bcrypt.MinCost),
			)
			Expect(gorp.WrapWriter[string, ontology.Resource](node.DB).
				Set(ctx, ontology.Resource{ID: alice.ResourceID()})).To(Succeed())
			legacyDB := gorp.Wrap(node.DB.KV(), gorp.WithCodec(msgpack.Codec))
			Expect(gorp.WrapWriter[userv0.Key, userv0.User](legacyDB).Set(ctx, alice)).
				To(Succeed())
			Expect(gorp.WrapWriter[string, authv0.SecureCredentials](legacyDB).Set(
				ctx,
				authv0.SecureCredentials{Username: "alice", Password: hash},
			)).To(Succeed())
			layer := MustOpen(svctestutil.OpenLayer(ctx, node))
			session, _ := MustSucceed2(layer.Auth.Login(ctx, auth.Proof{
				Variant: auth.PasswordProof{
					Credentials: auth.Credentials{Username: "alice", Password: "p"},
				},
			}))
			Expect(session.Subject).To(Equal(alice.ResourceID()))
		},
	)

	// Regression: the composition pass converts data staged by services that open
	// after the panel table (project layouts, task re-key mapping). Running the
	// conversion inside the panel table's own chain silently produced zero panels,
	// because the chain ran before the producers staged anything.
	It(
		"Should adopt a legacy workspace layout as a panel after all tables open",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			db := node.DB
			plot := lineplot.LinePlot{Key: uuid.New(), Name: "Pressures"}
			lpKey := plot.Key.String()
			lpID := resource.ID{Type: resource.TypeLineplot, Key: lpKey}
			Expect(gorp.WrapWriter[string, ontology.Resource](db).
				Set(ctx, ontology.Resource{ID: lpID})).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "LinePlot",
				Migrations: lineplotversions.Migrations,
			})).To(Succeed())
			Expect(gorp.NewCreate[lineplot.Key, lineplot.LinePlot]().
				Entry(&plot).
				Exec(ctx, db)).To(Succeed())
			ws := projectv0.Workspace{
				Key:  uuid.New(),
				Name: "TPC",
				Layout: msgpack.EncodedJSON{
					"mosaics": map[string]any{
						"main": map[string]any{
							"root": map[string]any{
								"key": 1,
								"tabs": []any{
									map[string]any{"tabKey": lpKey},
								},
							},
						},
					},
					"layouts": map[string]any{
						lpKey: map[string]any{
							"key":      lpKey,
							"type":     "lineplot",
							"name":     "Pressures",
							"location": "mosaic",
						},
						"main": map[string]any{
							"key":      "main",
							"type":     "main",
							"name":     "Main",
							"location": "window",
						},
					},
				},
			}
			Expect(gorp.WrapWriter[projectv0.Key, projectv0.Workspace](db).
				Set(ctx, ws)).To(Succeed())
			wsID := resource.ID{Type: "workspace", Key: ws.Key.String()}
			Expect(gorp.WrapWriter[string, ontology.Resource](db).
				Set(ctx, ontology.Resource{ID: wsID})).To(Succeed())

			MustOpen(svctestutil.OpenLayer(ctx, node))

			seq, closer := MustSucceed2(gorp.
				WrapReader[panelversions.Key, panelversions.Panel](db).
				OpenNexter(ctx))
			DeferClose(closer)
			var panels []panelversions.Panel
			for p := range seq {
				panels = append(panels, p)
			}
			Expect(panels).To(HaveLen(1))
			Expect(panels[0].Name).To(Equal("Main"))
			lf, ok := panels[0].Root.Variant.(panelversions.LeafNode)
			Expect(ok).To(BeTrue())
			Expect(lf.Tabs).To(HaveLen(1))
			rt, ok := lf.Tabs[0].Variant.(panelversions.ResourceTab)
			Expect(ok).To(BeTrue())
			Expect(rt.Resource).To(Equal(lpID))

			By("Giving the panel to the migrated project")
			Expect(panels[0].Project).To(HaveValue(Equal(ws.Key)))
			rel := ontology.Relationship{
				From: resource.ID{Type: resource.TypeProject, Key: ws.Key.String()},
				Type: ontology.RelationshipTypeParentOf,
				To:   panels[0].ResourceID(),
			}
			Expect(MustSucceed(gorp.NewRetrieve[string, ontology.Relationship]().
				Where(gorp.MatchKeys[string, ontology.Relationship](rel.GorpKey())).
				Exists(ctx, db))).To(BeFalse())
		},
	)

	It(
		"Should delete a group on a v0.58 Core that stores panels",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			tab := func(t resource.Type) panelv0.Tab {
				return panelv0.Tab{Variant: panelv0.ResourceTab{
					Key:      uuid.New(),
					Resource: resource.ID{Type: t, Key: uuid.New().String()},
				}}
			}
			Expect(gorp.WrapWriter[panelv0.Key, panelv0.Panel](node.DB).Set(
				ctx,
				panelv0.Panel{
					Key:  uuid.New(),
					Name: "Main",
					Root: panelv0.Node{Variant: panelv0.SplitNode{
						Direction: spatial.DirectionX,
						Size:      0.5,
						First: panelv0.Node{
							Variant: panelv0.LeafNode{Tabs: []panelv0.Tab{
								{Variant: panelv0.ViewTab{
									Key:  uuid.New(),
									Type: "docs",
									Args: msgpack.EncodedJSON{"page": "intro"},
								}},
								tab(resource.TypeLog),
							}},
						},
						Last: panelv0.Node{Variant: panelv0.LeafNode{
							Tabs: []panelv0.Tab{tab(resource.TypeTable)},
						}},
					}},
				},
			)).To(Succeed())
			layer := MustOpen(svctestutil.OpenLayer(ctx, node))
			g := group.Group{Name: "Valves", Scope: resource.TypeChannel}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Group.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Group.NewWriter(tx).Delete(ctx, g.Key)
			})).To(Succeed())
		},
	)
})

var _ = Describe("Workspace-era upgrade", func() {
	It(
		"Should keep every visualization in the project its workspace becomes",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			legacyDB := gorp.Wrap(node.DB.KV(), gorp.WithCodec(msgpack.Codec))
			ws := projectv0.Workspace{Key: uuid.New(), Name: "Workspace"}
			Expect(gorp.WrapWriter[projectv0.Key, projectv0.Workspace](legacyDB).
				Set(ctx, ws)).To(Succeed())
			g := groupv0.Group{Key: uuid.New(), Name: "Group"}
			Expect(gorp.WrapWriter[groupv0.Key, groupv0.Group](legacyDB).Set(ctx, g)).
				To(Succeed())
			loose := schematicv0.Schematic{Key: uuid.New(), Name: "Loose"}
			grouped := schematicv0.Schematic{Key: uuid.New(), Name: "Grouped"}
			Expect(gorp.WrapWriter[schematicv0.Key, schematicv0.Schematic](legacyDB).
				Set(ctx, loose, grouped)).To(Succeed())
			lp := lineplotv0.LinePlot{Key: uuid.New(), Name: "Line Plot"}
			Expect(gorp.WrapWriter[lineplotv0.Key, lineplotv0.LinePlot](legacyDB).
				Set(ctx, lp)).To(Succeed())
			lg := logv0.Log{Key: uuid.New(), Name: "Log"}
			Expect(gorp.WrapWriter[logv0.Key, logv0.Log](legacyDB).Set(ctx, lg)).
				To(Succeed())
			tbl := tablev0.Table{Key: uuid.New(), Name: "Table"}
			Expect(gorp.WrapWriter[tablev0.Key, tablev0.Table](legacyDB).Set(ctx, tbl)).
				To(Succeed())
			var (
				wsID      = resource.ID{Type: "workspace", Key: ws.Key.String()}
				groupID   = resource.ID{Type: resource.TypeGroup, Key: g.Key.String()}
				groupedID = schematic.ResourceID(grouped.Key)
				owned     = []resource.ID{
					groupID,
					schematic.ResourceID(loose.Key),
					lineplot.ResourceID(lp.Key),
					log.ResourceID(lg.Key),
					table.ResourceID(tbl.Key),
				}
				resources = []ontology.Resource{{ID: wsID}, {ID: groupedID}}
				edges     = []ontology.Relationship{{
					From: groupID,
					Type: ontology.RelationshipTypeParentOf,
					To:   groupedID,
				}}
			)
			for _, id := range owned {
				resources = append(resources, ontology.Resource{ID: id})
				edges = append(edges, ontology.Relationship{
					From: wsID,
					Type: ontology.RelationshipTypeParentOf,
					To:   id,
				})
			}
			Expect(gorp.WrapWriter[string, ontology.Resource](node.DB).
				Set(ctx, resources...)).To(Succeed())
			Expect(gorp.WrapWriter[string, ontology.Relationship](node.DB).
				Set(ctx, edges...)).To(Succeed())

			layer := MustOpen(svctestutil.OpenLayer(ctx, node))

			var schematics []schematic.Schematic
			Expect(layer.Schematic.NewRetrieve().
				Where(schematic.MatchKeys(loose.Key, grouped.Key)).
				Entries(&schematics).
				Exec(ctx, node.DB)).To(Succeed())
			Expect(schematics).To(HaveLen(2))
			for _, s := range schematics {
				Expect(s.Project).To(HaveValue(Equal(ws.Key)))
			}
			var plot lineplot.LinePlot
			Expect(layer.LinePlot.NewRetrieve().
				Where(lineplot.MatchKeys(lp.Key)).
				Entry(&plot).
				Exec(ctx, node.DB)).To(Succeed())
			Expect(plot.Project).To(HaveValue(Equal(ws.Key)))
			var l log.Log
			Expect(layer.Log.NewRetrieve().
				Where(log.MatchKeys(lg.Key)).
				Entry(&l).
				Exec(ctx, node.DB)).To(Succeed())
			Expect(l.Project).To(HaveValue(Equal(ws.Key)))
			var t table.Table
			Expect(layer.Table.NewRetrieve().
				Where(table.MatchKeys(tbl.Key)).
				Entry(&t).
				Exec(ctx, node.DB)).To(Succeed())
			Expect(t.Project).To(HaveValue(Equal(ws.Key)))
		},
	)
})

var _ = Describe("Resource search", Ordered, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
	})

	DescribeTable(
		"Should search every type that names records",
		func(ctx SpecContext, t resource.Type) {
			MustSucceed(layer.Resource.Search(ctx, db, resource.SearchQuery{
				Term:  "pump",
				Types: []resource.Type{t},
			}))
		},
		Entry("Arc", resource.TypeArc),
		Entry("Channel", resource.TypeChannel),
		Entry("Device", resource.TypeDevice),
		Entry("Group", resource.TypeGroup),
		Entry("Label", resource.TypeLabel),
		Entry("Lineplot", resource.TypeLineplot),
		Entry("Log", resource.TypeLog),
		Entry("Node", resource.TypeNode),
		Entry("Panel", resource.TypePanel),
		Entry("Policy", resource.TypePolicy),
		Entry("Project", resource.TypeProject),
		Entry("Rack", resource.TypeRack),
		Entry("Range", resource.TypeRange),
		Entry("RangeAlias", resource.TypeRangeAlias),
		Entry("Role", resource.TypeRole),
		Entry("Schematic", resource.TypeSchematic),
		Entry("SchematicSymbol", resource.TypeSchematicSymbol),
		Entry("Status", resource.TypeStatus),
		Entry("Table", resource.TypeTable),
		Entry("Task", resource.TypeTask),
		Entry("User", resource.TypeUser),
		Entry("View", resource.TypeView),
	)
})

var _ = Describe("Group moves", Ordered, func() {
	var (
		l      *service.Layer
		db     *gorp.DB
		src    project.Project
		dest   project.Project
		g      group.Group
		owners func(ctx SpecContext) []*project.Key
	)

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		l = MustOpen(svctestutil.OpenLayer(ctx, node))
		src = project.Project{Name: "Source"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Project.NewWriter(tx).Create(ctx, &src)
		})).To(Succeed())
		dest = project.Project{Name: "Destination"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Project.NewWriter(tx).Create(ctx, &dest)
		})).To(Succeed())
		g = group.Group{Name: "Moved", Parent: new(project.ResourceID(src.Key))}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Group.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		sch := schematic.Schematic{Name: "Schematic", Group: &g.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Schematic.NewWriter(tx).Create(ctx, &sch)
		})).To(Succeed())
		lp := lineplot.LinePlot{Name: "Plot", Group: &g.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.LinePlot.NewWriter(tx).Create(ctx, &lp)
		})).To(Succeed())
		lg := log.Log{Name: "Log", Group: &g.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Log.NewWriter(tx).Create(ctx, &lg)
		})).To(Succeed())
		tb := table.Table{Name: "Table", Group: &g.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Table.NewWriter(tx).Create(ctx, &tb)
		})).To(Succeed())
		pn := panel.Panel{Name: "Panel", Group: &g.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Panel.NewWriter(tx).Create(ctx, &pn)
		})).To(Succeed())
		owners = func(ctx SpecContext) []*project.Key {
			GinkgoHelper()
			Expect(l.Schematic.NewRetrieve().
				Where(schematic.MatchKeys(sch.Key)).Entry(&sch).Exec(ctx, db)).
				To(Succeed())
			Expect(l.LinePlot.NewRetrieve().
				Where(lineplot.MatchKeys(lp.Key)).Entry(&lp).Exec(ctx, db)).
				To(Succeed())
			Expect(l.Log.NewRetrieve().
				Where(log.MatchKeys(lg.Key)).Entry(&lg).Exec(ctx, db)).
				To(Succeed())
			Expect(l.Table.NewRetrieve().
				Where(table.MatchKeys(tb.Key)).Entry(&tb).Exec(ctx, db)).
				To(Succeed())
			Expect(l.Panel.NewRetrieve().
				Where(panel.MatchKeys(pn.Key)).Entry(&pn).Exec(ctx, db)).
				To(Succeed())
			return []*project.Key{
				sch.Project,
				lp.Project,
				lg.Project,
				tb.Project,
				pn.Project,
			}
		}
	})

	It(
		"Should create each visualization in the project of its group",
		func(ctx SpecContext) { Expect(owners(ctx)).To(HaveEach(Equal(&src.Key))) },
	)

	It(
		"Should move the visualizations of a group into the project it moves under",
		func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return l.Group.NewWriter(tx).
					SetParent(ctx, new(project.ResourceID(dest.Key)), g.Key)
			})).To(Succeed())
			Expect(owners(ctx)).To(HaveEach(Equal(&dest.Key)))
		},
	)

	It(
		"Should reject a move of a group that holds visualizations to the top level",
		func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return l.Group.NewWriter(tx).SetParent(ctx, nil, g.Key)
			})).To(MatchError(
				"group: a schematic sits only in a group under a project: validation error",
			))
			Expect(owners(ctx)).To(HaveEach(Equal(&dest.Key)))
		},
	)
})
