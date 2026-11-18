// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/union"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a Schematic", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			s := schematic.Schematic{Project: &proj.Key, Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			Expect(s.Key).ToNot(Equal(uuid.Nil()))
		})

		DescribeTable("Should validate the scale of a decoded schematic",
			func(ctx SpecContext, scale string, matcher OmegaMatcher) {
				tx := DeferClose(db.OpenTx())
				var s schematic.Schematic
				Expect(json.Unmarshal([]byte(
					`{"name":"valves","nodes":[{"key":"v1","config":`+
						`{"variant":"valve"`+scale+`}}]}`,
				), &s)).To(Succeed())
				s.Project = &proj.Key
				Expect(svc.NewWriter(tx).Create(ctx, &s)).To(matcher)
			},
			Entry("omitted scale takes the default", "", Succeed()),
			Entry(
				"zero scale is rejected",
				`,"scale":0`,
				SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring("scale"),
					),
				),
			),
		)

		DescribeTable("Should keep the gauge bounds of a decoded schematic",
			func(ctx SpecContext, config string, expected spatial.Bounds) {
				tx := DeferClose(db.OpenTx())
				var s schematic.Schematic
				Expect(json.Unmarshal([]byte(
					`{"name":"gauges","nodes":[{"key":"g1","config":`+
						config+`}]}`,
				), &s)).To(Succeed())
				s.Project = &proj.Key
				Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
				var res schematic.Schematic
				Expect(svc.NewRetrieve().
					Where(schematic.MatchKeys(s.Key)).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				gauge, ok := res.Nodes[0].Config.Variant.(schematic.GaugeNodeConfig)
				Expect(ok).To(BeTrue())
				Expect(gauge.Bounds).To(Equal(expected))
			},
			Entry(
				"explicit zero upper bound stays",
				`{"variant":"gauge","bounds":{"lower":-10,"upper":0}}`,
				spatial.Bounds{Lower: -10, Upper: 0},
			),
			Entry(
				"omitted bounds take the default",
				`{"variant":"gauge"}`,
				spatial.Bounds{Lower: 0, Upper: 100},
			),
		)
		It(
			"Should return a validation error when the name is empty",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				s := schematic.Schematic{Project: &proj.Key}
				Expect(svc.NewWriter(tx).Create(ctx, &s)).
					To(MatchError(ContainSubstring("name: required")))
			},
		)
		It("Should refuse a schematic whose key exists", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			stored := schematic.Schematic{Project: &proj.Key, Name: "stored"}
			Expect(svc.NewWriter(tx).Create(ctx, &stored)).To(Succeed())
			again := schematic.Schematic{
				Key:     stored.Key,
				Project: &proj.Key,
				Name:    "refused",
			}
			Expect(svc.NewWriter(tx).Create(ctx, &again)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("schematic %s already exists", stored.Key)),
			))
			var res schematic.Schematic
			Expect(svc.NewRetrieve().
				Where(schematic.MatchKeys(stored.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(stored))
		})

		It(
			"Should broadcast the create with the writer's dispatch key after commit",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				rec := &Recorder[schematic.Key, schematic.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				v := schematic.Schematic{Project: &proj.Key, Name: "broadcast"}
				Expect(
					svc.NewWriter(tx).WithDispatchKey("dk-1").Create(ctx, &v),
				).To(Succeed())
				Expect(rec.Snapshot()).To(BeEmpty())
				Expect(tx.Commit(ctx)).To(Succeed())
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(1))
				Expect(seen[0].Key).To(Equal(v.Key))
				Expect(seen[0].DispatchKey).To(Equal("dk-1"))
				Expect(seen[0].Actions).To(HaveLen(1))
				Expect(seen[0].Actions[0].Type).To(Equal(schematic.ActionTypeCreate))
			},
		)

		It(
			"Should not broadcast a create whose transaction closes uncommitted",
			func(ctx SpecContext) {
				rec := &Recorder[schematic.Key, schematic.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				uncommitted := db.OpenTx()
				v := schematic.Schematic{Project: &proj.Key, Name: "discarded"}
				Expect(
					svc.NewWriter(uncommitted).Create(ctx, &v),
				).To(Succeed())
				Expect(uncommitted.Close()).To(Succeed())
				Expect(rec.Snapshot()).To(BeEmpty())
			},
		)
	})
	Describe("CreateMany", func() {
		It("Should create multiple schematics", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			schematics := []schematic.Schematic{
				{Name: "schematic-1", Project: &proj.Key},
				{Name: "schematic-2", Project: &proj.Key},
			}
			Expect(
				svc.NewWriter(tx).CreateMany(ctx, &schematics),
			).To(Succeed())

			var retrieved []schematic.Schematic
			Expect(svc.NewRetrieve().Where(schematic.MatchKeys(
				schematics[0].Key,
				schematics[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})
	Describe("Apply of a project move", func() {
		It("Should move the schematic into the project", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			dest := project.Project{Name: "destination"}
			Expect(projectSvc.NewWriter(tx).Create(ctx, &dest)).To(Succeed())
			v := schematic.Schematic{Project: &proj.Key, Name: "moved"}
			Expect(svc.NewWriter(tx).Create(ctx, &v)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(ctx, v.Key, setProject(dest.Key))).
				To(BeEmpty())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().
				Where(schematic.MatchKeys(v.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Project).To(HaveValue(Equal(dest.Key)))
		})

		It("Should reject a project that does not exist", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			v := schematic.Schematic{Project: &proj.Key, Name: "stuck"}
			Expect(svc.NewWriter(tx).Create(ctx, &v)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(ctx, v.Key, setProject(uuid.New()))).
				Error().To(MatchError(ContainSubstring("does not exist")))
		})
	})

	Describe("Dispatch", func() {
		It(
			"Should reject an action that leaves the schematic invalid",
			func(ctx SpecContext) {
				s := schematic.Schematic{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				Expect(
					dispatch(ctx, s.Key, "session-1", []schematic.Action{
						schematic.NewSetAction(schematic.SetPayload{
							Path:  []string{"name"},
							Value: "",
						}),
					}),
				).To(MatchError(ContainSubstring("name: required")))
				var res schematic.Schematic
				Expect(
					svc.NewRetrieve().
						Where(schematic.MatchKeys(s.Key)).
						Entry(&res).
						Exec(ctx, db),
				).To(Succeed())
				Expect(res.Name).To(Equal("test"))
			},
		)
		It("Should reject a dispatch that sets a zero scale", func(ctx SpecContext) {
			var s schematic.Schematic
			Expect(json.Unmarshal([]byte(
				`{"name":"valves","nodes":[{"key":"v1",`+
					`"config":{"variant":"valve"}}]}`,
			), &s)).To(Succeed())
			s.Project = &proj.Key
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &s)
			})).To(Succeed())
			Expect(dispatch(ctx, s.Key, "session-1", []schematic.Action{
				schematic.NewSetAction(schematic.SetPayload{
					Path:  []string{"nodes", "v1", "config", "scale"},
					Value: 0.0,
				}),
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("scale"),
				),
			))
		})

		It("Should reject a create that sets a stored positive scale to zero", func(
			ctx SpecContext,
		) {
			var s schematic.Schematic
			Expect(json.Unmarshal([]byte(
				`{"name":"valves","nodes":[{"key":"v1",`+
					`"config":{"variant":"valve"}}]}`,
			), &s)).To(Succeed())
			s.Project = &proj.Key
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &s)
			})).To(Succeed())
			var edited schematic.Schematic
			Expect(json.Unmarshal([]byte(
				`{"name":"valves","nodes":[{"key":"v1",`+
					`"config":{"variant":"valve","scale":0}}]}`,
			), &edited)).To(Succeed())
			edited.Key = s.Key
			edited.Project = &proj.Key
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &edited)
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("scale"),
				),
			))
		})

		DescribeTable("Should reject a create whose config has no variant",
			func(ctx SpecContext, s schematic.Schematic, msg string) {
				s.Project = &proj.Key
				s.Name = "unconfigured"
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry(
				"on a node",
				schematic.Schematic{Nodes: []schematic.Node{{Key: "n1"}}},
				"nodes.0.config.variant: required",
			),
			Entry(
				"on an edge",
				schematic.Schematic{
					Nodes: []schematic.Node{
						{Key: "n1", Config: groupCfg()},
						{Key: "n2", Config: groupCfg()},
					},
					Edges: []schematic.Edge{{
						Key:    "e1",
						Source: schematic.Handle{Node: "n1"},
						Target: schematic.Handle{Node: "n2"},
					}},
				},
				"edges.0.config.variant: required",
			),
		)

		Describe("A dispatch to a stored valve", func() {
			var s schematic.Schematic
			BeforeEach(func(ctx SpecContext) {
				s = schematic.Schematic{}
				Expect(json.Unmarshal([]byte(
					`{"name":"valves","nodes":[`+
						`{"key":"v0","config":{"variant":"tank"}},`+
						`{"key":"v1","config":{"variant":"valve"}}]}`,
				), &s)).To(Succeed())
				s.Project = &proj.Key
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
			})

			DescribeTable("Should be rejected when it",
				func(ctx SpecContext, a schematic.Action, msg string) {
					Expect(dispatch(ctx, s.Key, "session-1", []schematic.Action{a})).
						To(SatisfyAll(
							MatchError(validate.ErrValidation),
							MatchError(ContainSubstring(msg)),
						))
				},
				Entry(
					"clears the config",
					schematic.NewSetAction(schematic.SetPayload{
						Path: []string{"nodes", "v1", "config"},
					}),
					"[nodes, v1, config]: the document requires a value",
				),
				Entry(
					"sets a node with no config",
					schematic.NewSetNodeAction(schematic.SetNodePayload{
						Node: schematic.Node{Key: "v1"},
					}),
					"nodes.1.config.variant: required",
				),
				Entry(
					"sets a value that fails the config's schema",
					schematic.NewSetAction(schematic.SetPayload{
						Path:  []string{"nodes", "v0", "config", "level"},
						Value: "h9",
					}),
					"nodes.0.config.level: invalid level: h9",
				),
			)
		})

		Describe("A dispatch to a stored edge", func() {
			var s schematic.Schematic
			BeforeEach(func(ctx SpecContext) {
				s = schematic.Schematic{}
				Expect(json.Unmarshal([]byte(
					`{"name":"edges","nodes":[`+
						`{"key":"a","config":{"variant":"cap"}},`+
						`{"key":"b","config":{"variant":"cap"}}],"edges":[`+
						`{"key":"e1","source":{"node":"a","param":"o"},`+
						`"target":{"node":"b","param":"i"},`+
						`"config":{"variant":"pipe",`+
						`"stroke_color":{"type":"fixed","value":"#ff0000"}}}]}`,
				), &s)).To(Succeed())
				s.Project = &proj.Key
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
			})

			retrieve := func(ctx SpecContext) schematic.SegmentedEdgeConfig {
				GinkgoHelper()
				var res schematic.Schematic
				Expect(svc.NewRetrieve().
					Where(schematic.MatchKeys(s.Key)).
					Entry(&res).
					Exec(ctx, db)).To(Succeed())
				pipe, ok := res.Edges[0].Config.Variant.(schematic.PipeEdgeConfig)
				Expect(ok).To(BeTrue())
				return pipe.SegmentedEdgeConfig
			}

			setStrokeWidth := func(w uint8) schematic.Action {
				return schematic.NewSetAction(schematic.SetPayload{
					Path:  []string{"edges", "e1", "config", "stroke_width"},
					Value: w,
				})
			}

			It("Should store the default stroke width", func(ctx SpecContext) {
				Expect(retrieve(ctx).StrokeWidth).To(BeEquivalentTo(2))
			})

			DescribeTable("Should set a stroke width within bounds",
				func(ctx SpecContext, w uint8) {
					Expect(dispatch(ctx, s.Key, "session-1", []schematic.Action{
						setStrokeWidth(w),
					})).To(Succeed())
					Expect(retrieve(ctx).StrokeWidth).To(Equal(w))
				},
				Entry("at 1", uint8(1)),
				Entry("at 20", uint8(20)),
			)

			DescribeTable("Should reject a stroke width out of bounds",
				func(ctx SpecContext, w uint8) {
					Expect(dispatch(ctx, s.Key, "session-1", []schematic.Action{
						setStrokeWidth(w),
					})).To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring("stroke_width")),
					))
					Expect(retrieve(ctx).StrokeWidth).To(BeEquivalentTo(2))
				},
				Entry("at 0", uint8(0)),
				Entry("at 21", uint8(21)),
			)

			It("Should clear the stroke color", func(ctx SpecContext) {
				Expect(retrieve(ctx).StrokeColor).ToNot(BeNil())
				Expect(dispatch(ctx, s.Key, "session-1", []schematic.Action{
					schematic.NewSetAction(schematic.SetPayload{
						Path: []string{"edges", "e1", "config", "stroke_color"},
					}),
				})).To(Succeed())
				Expect(retrieve(ctx).StrokeColor).To(BeNil())
			})
		})

		Describe("A dispatch to a stored group", func() {
			var s schematic.Schematic
			BeforeEach(func(ctx SpecContext) {
				s = schematic.Schematic{}
				Expect(json.Unmarshal([]byte(
					`{"name":"groups","nodes":[`+
						`{"key":"g1","config":{"variant":"group_box"}},`+
						`{"key":"g2","config":{"variant":"group_box"},"parent":"g1"},`+
						`{"key":"v1","config":{"variant":"valve"}}]}`,
				), &s)).To(Succeed())
				s.Project = &proj.Key
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
			})
			setParent := func(key, parent string) schematic.Action {
				return schematic.NewSetAction(schematic.SetPayload{
					Path:  []string{"nodes", key, "parent"},
					Value: parent,
				})
			}

			It("Should accept a parent that is a group box", func(ctx SpecContext) {
				Expect(dispatch(ctx, s.Key, "session-1", []schematic.Action{
					setParent("v1", "g2"),
				})).To(Succeed())
			})

			It("Should promote the children of a removed group", func(ctx SpecContext) {
				Expect(dispatch(ctx, s.Key, "session-1", []schematic.Action{
					setParent("v1", "g2"),
					schematic.NewRemoveNodeAction(
						schematic.RemoveNodePayload{Key: "g2"},
					),
				})).To(Succeed())
				var res schematic.Schematic
				Expect(svc.NewRetrieve().
					Where(schematic.MatchKeys(s.Key)).
					Entry(&res).
					Exec(ctx, db)).To(Succeed())
				Expect(res.Nodes[1].Key).To(Equal("v1"))
				Expect(res.Nodes[1].Parent).To(HaveValue(Equal("g1")))
			})

			DescribeTable("Should be rejected when it",
				func(ctx SpecContext, a schematic.Action, msg string) {
					Expect(dispatch(ctx, s.Key, "session-1", []schematic.Action{a})).
						To(SatisfyAll(
							MatchError(validate.ErrValidation),
							MatchError(ContainSubstring(msg)),
						))
				},
				Entry(
					"sets a parent that is not a group box",
					setParent("g1", "v1"),
					"nodes.0.parent: v1 is not a group box of the schematic",
				),
				Entry(
					"sets a parent that is not a node",
					setParent("v1", "ghost"),
					"nodes.2.parent: ghost is not a group box of the schematic",
				),
				Entry(
					"sets a node as its own parent",
					setParent("g1", "g1"),
					"nodes.0.parent: the parents of g1 form a cycle",
				),
				Entry(
					"closes a cycle of parents",
					setParent("g1", "g2"),
					"nodes.0.parent: the parents of g1 form a cycle",
				),
			)
		})

		It("Should reject a create whose parent is not a group box", func(
			ctx SpecContext,
		) {
			var s schematic.Schematic
			Expect(json.Unmarshal([]byte(
				`{"name":"valves","nodes":[`+
					`{"key":"v0","config":{"variant":"valve"}},`+
					`{"key":"v1","config":{"variant":"valve"},"parent":"v0"}]}`,
			), &s)).To(Succeed())
			s.Project = &proj.Key
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &s)
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("nodes.1.parent: v0 is not a group box"),
				),
			))
		})

		It("Should reject an action with no payload", func(ctx SpecContext) {
			s := schematic.Schematic{Project: &proj.Key, Name: "test"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &s)
			})).To(Succeed())
			Expect(
				dispatch(ctx, s.Key, "session-1", []schematic.Action{
					{Type: schematic.ActionTypeSet},
				}),
			).To(MatchError(union.ErrMissingPayload))
		})

		It(
			"Should reject a create without changing the schematic it names",
			func(ctx SpecContext) {
				target := schematic.Schematic{Project: &proj.Key, Name: "target"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &target)
				})).To(Succeed())
				other := schematic.Schematic{Project: &proj.Key, Name: "other"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &other)
				})).To(Succeed())
				Expect(dispatch(ctx, target.Key, "d1", []schematic.Action{
					schematic.NewCreateAction(schematic.CreatePayload{
						Schematic: schematic.Schematic{
							Key:  other.Key,
							Name: "overwritten",
						},
					}),
				})).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring("only the Core dispatches a schematic create"),
					),
				))
				var res schematic.Schematic
				Expect(svc.NewRetrieve().
					Where(schematic.MatchKeys(other.Key)).
					Entry(&res).
					Exec(ctx, db)).To(Succeed())
				Expect(res.Name).To(Equal("other"))
			},
		)

		It("Should apply a single SetNodePosition action", func(ctx SpecContext) {
			s := schematic.Schematic{
				Project: &proj.Key,
				Name:    "test",
				Nodes: []schematic.Node{
					{Key: "n1", Config: tankCfg("", "")},
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &s)
			})).To(Succeed())
			Expect(
				dispatch(ctx, s.Key, "session-1", []schematic.Action{
					schematic.NewSetNodePositionAction(schematic.SetNodePositionPayload{
						Key:      "n1",
						Position: spatial.XY{X: 100, Y: 200},
					}),
				}),
			).To(Succeed())
			var res schematic.Schematic
			Expect(
				svc.NewRetrieve().
					Where(schematic.MatchKeys(s.Key)).
					Entry(&res).
					Exec(ctx, db),
			).To(Succeed())
			Expect(res.Nodes).To(HaveLen(1))
			Expect(res.Nodes[0].Position).To(Equal(spatial.XY{X: 100, Y: 200}))
		})
		It(
			"Should apply a sequence of mixed actions atomically",
			func(ctx SpecContext) {
				s := schematic.Schematic{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				Expect(
					dispatch(ctx, s.Key, "session-1", []schematic.Action{
						schematic.NewSetNodeAction(schematic.SetNodePayload{
							Node: schematic.Node{
								Key:      "n1",
								Position: spatial.XY{X: 1, Y: 2},
								Config:   tankCfg("", ""),
							},
						}),
						schematic.NewSetNodeAction(schematic.SetNodePayload{
							Node: schematic.Node{
								Key:      "n2",
								Position: spatial.XY{X: 3, Y: 4},
								Config:   tankCfg("", ""),
							},
						}),
						schematic.NewAddEdgeAction(schematic.AddEdgePayload{
							Edge: schematic.Edge{
								Key:    "e1",
								Source: schematic.Handle{Node: "n1"},
								Target: schematic.Handle{Node: "n2"},
								Config: pipeCfg(""),
							},
						}),
					}),
				).To(Succeed())
				var res schematic.Schematic
				Expect(
					svc.NewRetrieve().
						Where(schematic.MatchKeys(s.Key)).
						Entry(&res).
						Exec(ctx, db),
				).To(Succeed())
				Expect(res.Nodes).To(HaveLen(2))
				Expect(res.Edges).To(HaveLen(1))
			},
		)
		It(
			"Should be a no-op when actions reference non-existent keys",
			func(ctx SpecContext) {
				s := schematic.Schematic{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				Expect(
					dispatch(ctx, s.Key, "session-1", []schematic.Action{
						schematic.NewRemoveNodeAction(
							schematic.RemoveNodePayload{Key: "ghost"},
						),
						schematic.NewRemoveEdgeAction(
							schematic.RemoveEdgePayload{Key: "ghost-edge"},
						),
					}),
				).To(Succeed())
				var res schematic.Schematic
				Expect(
					svc.NewRetrieve().
						Where(schematic.MatchKeys(s.Key)).
						Entry(&res).
						Exec(ctx, db),
				).To(Succeed())
				Expect(res.Nodes).To(BeEmpty())
				Expect(res.Edges).To(BeEmpty())
			},
		)

		It(
			"Should converge a 30-action drag storm to the final position",
			func(ctx SpecContext) {
				s := schematic.Schematic{
					Project: &proj.Key,
					Name:    "drag-storm",
					Nodes:   []schematic.Node{{Key: "pump", Config: tankCfg("", "")}},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				actions := make([]schematic.Action, 0, 30)
				for i := range 30 {
					actions = append(
						actions,
						schematic.NewSetNodePositionAction(
							schematic.SetNodePositionPayload{
								Key:      "pump",
								Position: spatial.XY{X: float64(i), Y: float64(i * 2)},
							},
						),
					)
				}
				Expect(
					dispatch(ctx, s.Key, "session-1", actions),
				).To(Succeed())
				var res schematic.Schematic
				Expect(
					svc.NewRetrieve().
						Where(schematic.MatchKeys(s.Key)).
						Entry(&res).
						Exec(ctx, db),
				).To(Succeed())
				Expect(res.Nodes[0].Position).To(Equal(spatial.XY{X: 29, Y: 58}))
			},
		)

		It(
			"Should build a graph atomically from an empty schematic in one Dispatch",
			func(ctx SpecContext) {
				s := schematic.Schematic{Project: &proj.Key, Name: "graph"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				Expect(
					dispatch(ctx, s.Key, "session-1", []schematic.Action{
						schematic.NewSetNodeAction(schematic.SetNodePayload{
							Node: schematic.Node{
								Key:      "pump",
								Position: spatial.XY{X: 0, Y: 0},
								Config:   tankCfg("", ""),
							},
						}),
						schematic.NewSetNodeAction(schematic.SetNodePayload{
							Node: schematic.Node{
								Key:      "valve",
								Position: spatial.XY{X: 100, Y: 0},
								Config:   tankCfg("", ""),
							},
						}),
						schematic.NewAddEdgeAction(
							schematic.AddEdgePayload{Edge: schematic.Edge{
								Key: "e1",
								Source: schematic.Handle{
									Node:  "pump",
									Param: "out",
								},
								Target: schematic.Handle{
									Node:  "valve",
									Param: "in",
								},
								Config: pipeCfg(""),
							}},
						),
						schematic.NewSetAction(schematic.SetPayload{
							Path: []string{
								"nodes",
								"pump",
								"config",
								"label",
								"label",
							},
							Value: "Main Pump",
						}),
					}),
				).To(Succeed())
				var res schematic.Schematic
				Expect(
					svc.NewRetrieve().
						Where(schematic.MatchKeys(s.Key)).
						Entry(&res).
						Exec(ctx, db),
				).To(Succeed())
				Expect(res.Nodes).To(HaveLen(2))
				Expect(res.Edges).To(HaveLen(1))
				Expect(res.Nodes[0].Config).To(Equal(tankCfg("Main Pump", "")))
			},
		)

		It(
			"Should notify subscribers with the dispatched ScopedAction on success",
			func(ctx SpecContext) {
				s := schematic.Schematic{Project: &proj.Key, Name: "observed"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				rec := &Recorder[schematic.Key, schematic.Action]{}
				disconnect := svc.OnAction(rec.Record)
				DeferCleanup(disconnect)
				actions := []schematic.Action{
					schematic.NewSetNodeAction(schematic.SetNodePayload{
						Node: schematic.Node{Key: "n1", Config: tankCfg("", "")},
					}),
					schematic.NewSetNodePositionAction(schematic.SetNodePositionPayload{
						Key:      "n1",
						Position: spatial.XY{X: 5, Y: 5},
					}),
				}
				Expect(
					dispatch(ctx, s.Key, "client-xyz", actions),
				).To(Succeed())
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(1))
				Expect(seen[0].Key).To(Equal(s.Key))
				Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
				Expect(seen[0].Seq).To(BeNumerically(">", uint64(0)))
				Expect(seen[0].Actions).To(HaveLen(2))
				Expect(seen[0].Actions[0].Type).To(Equal(schematic.ActionTypeSetNode))
				Expect(
					seen[0].Actions[1].Type,
				).To(Equal(schematic.ActionTypeSetNodePosition))
			},
		)

		It(
			"Should stamp strictly increasing Seq values onto successive Dispatch broadcasts",
			func(ctx SpecContext) {
				s := schematic.Schematic{Project: &proj.Key, Name: "seq-test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &s)
				})).To(Succeed())
				rec := &Recorder[schematic.Key, schematic.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				action := []schematic.Action{
					schematic.NewSetNodeAction(schematic.SetNodePayload{
						Node: schematic.Node{Key: "n1", Config: tankCfg("", "")},
					}),
				}
				for range 3 {
					Expect(
						dispatch(ctx, s.Key, "client-xyz", action),
					).To(Succeed())
				}
				seen := rec.Snapshot()
				Expect(seen).To(HaveLen(3))
				Expect(seen[1].Seq).To(BeNumerically(">", seen[0].Seq))
				Expect(seen[2].Seq).To(BeNumerically(">", seen[1].Seq))
			},
		)

		It(
			"Should fail with query.ErrNotFound and not notify subscribers when the target schematic does not exist",
			func(ctx SpecContext) {
				rec := &Recorder[schematic.Key, schematic.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				Expect(
					dispatch(ctx, uuid.New(), "client-xyz", []schematic.Action{
						schematic.NewRemoveNodeAction(
							schematic.RemoveNodePayload{Key: "n1"},
						),
					}),
				).To(MatchError(query.ErrNotFound))
				Expect(rec.Snapshot()).To(BeEmpty())
			},
		)
	})

	Describe("Copy", func() {
		It(
			"Should copy a Schematic with a new name under the same project",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				s := schematic.Schematic{Project: &proj.Key, Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
				var cpy schematic.Schematic
				Expect(
					svc.NewWriter(tx).Copy(ctx, s.Key, "test2", &cpy),
				).To(Succeed())
				Expect(cpy.Key).ToNot(Equal(s.Key))
				Expect(cpy.Name).To(Equal("test2"))
				Expect(cpy.Project).To(HaveValue(Equal(proj.Key)))
			},
		)
	})
})
