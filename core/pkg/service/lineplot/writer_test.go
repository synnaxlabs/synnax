// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/text"
	"github.com/synnaxlabs/x/union"
	"github.com/synnaxlabs/x/validate"
)

func keyRef(key string) ranger.Ref {
	return ranger.Ref{Variant: ranger.KeyRef{Key: uuid.MustParse(key)}}
}

// rk1 and rk2 are saved range keys for specs that need fixed keys.
const (
	rk1 = "00000000-0000-4000-8000-000000000001"
	rk2 = "00000000-0000-4000-8000-000000000002"
)

// recentRef is the window reference a v6 "recent" range key migrates to.
var recentRef = ranger.Ref{Variant: ranger.WindowRef{
	Window: telem.Window{Variant: telem.DynamicWindow{TimeSpan: 30 * telem.Second}},
}}

func refs(keys ...string) []ranger.Ref {
	out := make([]ranger.Ref, len(keys))
	for i, k := range keys {
		out[i] = keyRef(k)
	}
	return out
}

// lk is the key of the plot's y1 line plotting yChannel against xChannel.
func lk(plot lineplot.Key, xChannel, yChannel channel.Key) string {
	return v8.LineKey(plot, lineplot.YAxisKeyY1, yChannel, xChannel)
}

func lineKeysOf(lines []lineplot.Line) []string {
	keys := make([]string, len(lines))
	for i, l := range lines {
		keys[i] = l.Key
	}
	return keys
}

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a LinePlot", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, &plot)).To(Succeed())
			Expect(plot.Key).ToNot(Equal(uuid.Nil()))
		})
		It(
			"Should return a validation error when the name is empty",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				plot := lineplot.LinePlot{Project: &proj.Key}
				Expect(svc.NewWriter(tx).Create(ctx, &plot)).
					To(MatchError(ContainSubstring("name: required")))
			},
		)
		It("Should reject a line plot with no project", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			plot := lineplot.LinePlot{Name: "owner"}
			Expect(svc.NewWriter(tx).Create(ctx, &plot)).To(MatchError(
				ContainSubstring("project: must belong to a project"),
			))
		})
		It(
			"Should populate lines from channel and range bindings supplied at creation",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				plot := lineplot.LinePlot{
					Project:  &proj.Key,
					Name:     "test",
					Channels: lineplot.Channels{X1: ch10, Y1: []channel.Key{ch5, ch6}},
					Ranges:   refs(rk1),
				}
				Expect(svc.NewWriter(tx).Create(ctx, &plot)).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, tx),
				).
					To(Succeed())
				Expect(
					lineKeysOf(res.Lines),
				).To(ConsistOf(lk(plot.Key, ch10, ch5), lk(plot.Key, ch10, ch6)))
			},
		)

		It(
			"Should broadcast the create with the writer's dispatch key after commit",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				rec := &Recorder[lineplot.Key, lineplot.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				v := lineplot.LinePlot{Project: &proj.Key, Name: "broadcast"}
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
				Expect(seen[0].Actions[0].Type).To(Equal(lineplot.ActionTypeCreate))
			},
		)

		It(
			"Should not broadcast a create whose transaction closes uncommitted",
			func(ctx SpecContext) {
				rec := &Recorder[lineplot.Key, lineplot.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				uncommitted := db.OpenTx()
				v := lineplot.LinePlot{Project: &proj.Key, Name: "discarded"}
				Expect(
					svc.NewWriter(uncommitted).Create(ctx, &v),
				).To(Succeed())
				Expect(uncommitted.Close()).To(Succeed())
				Expect(rec.Snapshot()).To(BeEmpty())
			},
		)
	})

	Describe("CreateMany", func() {
		It("Should create multiple line plots", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			plots := []lineplot.LinePlot{
				{Name: "plot-1", Project: &proj.Key},
				{Name: "plot-2", Project: &proj.Key},
			}
			Expect(svc.NewWriter(tx).CreateMany(ctx, &plots)).To(Succeed())

			var retrieved []lineplot.LinePlot
			Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(
				plots[0].Key,
				plots[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})

	Describe("eager line creation", func() {
		It(
			"Should materialize one line per channel across every range",
			func(ctx SpecContext) {
				plot := lineplot.LinePlot{
					Project:  &proj.Key,
					Name:     "test",
					Channels: lineplot.Channels{X1: ch10},
					Ranges:   refs(rk1, rk2),
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				Expect(
					dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
							AxisKey: lineplot.YAxisKeyY1, Channel: ch5,
						}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(lineKeysOf(res.Lines)).
					To(ConsistOf(lk(plot.Key, ch10, ch5)))
			},
		)

		It(
			"Should give materialized lines the schema default styling",
			func(ctx SpecContext) {
				plot := lineplot.LinePlot{
					Project:  &proj.Key,
					Name:     "test",
					Channels: lineplot.Channels{X1: ch10},
					Ranges:   refs(rk1),
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				Expect(
					dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
							AxisKey: lineplot.YAxisKeyY1, Channel: ch5,
						}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(res.Lines).To(HaveLen(1))
				Expect(res.Lines[0].StrokeWidth).To(Equal(float64(2)))
				Expect(res.Lines[0].Aggregation).To(Equal(lineplot.AggregationMinMax))
				Expect(res.Lines[0].Detail).To(Equal(lineplot.DetailMedium))
			},
		)

		It(
			"Should remove a channel's lines when the channel is removed",
			func(ctx SpecContext) {
				plot := lineplot.LinePlot{
					Project:  &proj.Key,
					Name:     "test",
					Channels: lineplot.Channels{X1: ch10, Y1: []channel.Key{ch5, ch6}},
					Ranges:   refs(rk1),
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				Expect(
					dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewRemoveChannelAction(lineplot.RemoveChannelPayload{
							AxisKey: lineplot.YAxisKeyY1, Channel: ch5,
						}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(lineKeysOf(res.Lines)).To(ConsistOf(lk(plot.Key, ch10, ch6)))
			},
		)

		It(
			"Should rekey a y-axis's lines when the x-channel changes",
			func(ctx SpecContext) {
				plot := lineplot.LinePlot{
					Project:  &proj.Key,
					Name:     "test",
					Channels: lineplot.Channels{X1: ch10, Y1: []channel.Key{ch5}},
					Ranges:   refs(rk1),
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				Expect(
					dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewSetXChannelAction(lineplot.SetXChannelPayload{
							AxisKey: lineplot.XAxisKeyX1, Channel: ch20,
						}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(lineKeysOf(res.Lines)).To(ConsistOf(lk(plot.Key, ch20, ch5)))
			},
		)
	})

	Describe("SetProject", func() {
		It("Should move the line plots into the project", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			dest := project.Project{Name: "destination"}
			Expect(projectSvc.NewWriter(tx).Create(ctx, &dest)).To(Succeed())
			v := lineplot.LinePlot{Project: &proj.Key, Name: "moved"}
			Expect(svc.NewWriter(tx).Create(ctx, &v)).To(Succeed())
			Expect(svc.NewWriter(tx).SetProject(ctx, dest.Key, v.Key)).To(Succeed())
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().
				Where(lineplot.MatchKeys(v.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Project).To(HaveValue(Equal(dest.Key)))
		})

		It("Should reject a project that does not exist", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			v := lineplot.LinePlot{Project: &proj.Key, Name: "stuck"}
			Expect(svc.NewWriter(tx).Create(ctx, &v)).To(Succeed())
			Expect(svc.NewWriter(tx).SetProject(ctx, uuid.New(), v.Key)).
				To(MatchError(ContainSubstring("does not exist")))
		})
	})

	Describe("Dispatch", func() {
		It(
			"Should reject a create without changing the line plot it names",
			func(ctx SpecContext) {
				target := lineplot.LinePlot{Project: &proj.Key, Name: "target"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &target)
				})).To(Succeed())
				other := lineplot.LinePlot{Project: &proj.Key, Name: "other"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &other)
				})).To(Succeed())
				Expect(dispatch(ctx, target.Key, "d1", []lineplot.Action{
					lineplot.NewCreateAction(lineplot.CreatePayload{
						LinePlot: lineplot.LinePlot{
							Key:  other.Key,
							Name: "overwritten",
						},
					}),
				})).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring("only the Core dispatches a line plot create"),
					),
				))
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().
					Where(lineplot.MatchKeys(other.Key)).
					Entry(&res).
					Exec(ctx, db)).To(Succeed())
				Expect(res.Name).To(Equal("other"))
			},
		)

		Describe("per-action semantics", func() {
			It(
				"Should reject an action that leaves the line plot invalid",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).
						To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetAction(
								lineplot.SetPayload{Path: []string{"name"}, Value: ""},
							),
						}),
					).To(MatchError(ContainSubstring("name: required")))
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					Expect(res.Name).To(Equal("test"))
				},
			)

			DescribeTable(
				"Should reject a path action that does not set the name",
				func(ctx SpecContext, action lineplot.Action, msg string) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						action,
					})).To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring(msg)),
					))
				},
				Entry(
					"a set of another field",
					lineplot.NewSetAction(lineplot.SetPayload{
						Path:  []string{"title", "visible"},
						Value: true,
					}),
					`unknown object member name "title"`,
				),
				Entry(
					"a remove of the name",
					lineplot.NewRemoveAction(lineplot.RemovePayload{Key: "name"}),
					"[name]: the document requires a value",
				),
				Entry(
					"a set of the name to a number",
					lineplot.NewSetAction(lineplot.SetPayload{
						Path:  []string{"name"},
						Value: 1,
					}),
					"decoding document",
				),
				Entry(
					"an insert into the name",
					lineplot.NewInsertAction(lineplot.InsertPayload{
						Path:  []string{"name"},
						Value: map[string]any{"key": "a"},
					}),
					"[name]: the value is not a list",
				),
				Entry(
					"a move in the name",
					lineplot.NewMoveAction(
						lineplot.MovePayload{Path: []string{"name"}, Key: "a"},
					),
					"[name]: the value is not a list",
				),
			)

			It("Should apply a set of the name", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Project: &proj.Key, Name: "original"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				Expect(
					dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewSetAction(
							lineplot.SetPayload{
								Path:  []string{"name"},
								Value: "renamed",
							},
						),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(res.Name).To(Equal("renamed"))
			})

			It("Should apply SetTitleLevel", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				Expect(
					dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewSetTitleLevelAction(lineplot.SetTitleLevelPayload{
							Level: text.LevelH2,
						}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(res.Title.Level).To(Equal(text.LevelH2))
			})

			It("Should apply SetTitleVisible", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				Expect(
					dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewSetTitleVisibleAction(
							lineplot.SetTitleVisiblePayload{
								Visible: true,
							},
						),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(res.Title.Visible).To(BeTrue())
			})

			It("Should apply SetLegendPosition", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				pos := spatial.StickyXY{X: 10, Y: 20}
				pos.ApplyDefaults()
				Expect(
					dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewSetLegendPositionAction(
							lineplot.SetLegendPositionPayload{Position: pos},
						),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(res.Legend.Position.X).To(Equal(10.0))
			})

			It("Should apply SetLegendHidden", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				Expect(
					dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewSetLegendHiddenAction(
							lineplot.SetLegendHiddenPayload{
								Hidden: true,
							},
						),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(res.Legend.Hidden).To(BeTrue())
			})

			It(
				"Should append a channel to a y-axis via AddChannel",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
								AxisKey: lineplot.YAxisKeyY1, Channel: ch42,
							}),
							lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
								AxisKey: lineplot.YAxisKeyY1, Channel: ch43,
							}),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					Expect(
						res.Channels.Y1,
					).To(ConsistOf(ch42, ch43))
				},
			)

			It("Should treat a duplicate AddChannel as a no-op", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				Expect(
					dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
							AxisKey: lineplot.YAxisKeyY2, Channel: ch7,
						}),
						lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
							AxisKey: lineplot.YAxisKeyY2, Channel: ch7,
						}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(res.Channels.Y2).To(ConsistOf(ch7))
			})

			It(
				"Should reject AddChannel targeting an x-axis with validate.ErrValidation",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
								AxisKey: lineplot.YAxisKey("x1"), Channel: ch1,
							}),
						}),
					).Error().
						To(MatchError(validate.ErrValidation))
				},
			)

			It("Should drop a channel via RemoveChannel", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				Expect(
					dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
							AxisKey: lineplot.YAxisKeyY3, Channel: ch1,
						}),
						lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
							AxisKey: lineplot.YAxisKeyY3, Channel: ch2,
						}),
						lineplot.NewRemoveChannelAction(lineplot.RemoveChannelPayload{
							AxisKey: lineplot.YAxisKeyY3, Channel: ch1,
						}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(res.Channels.Y3).To(ConsistOf(ch2))
			})

			It(
				"Should replace a y-axis's whole channel set via SetChannels",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{
						Project:  &proj.Key,
						Name:     "test",
						Channels: lineplot.Channels{Y1: []channel.Key{ch1, ch2, ch3}},
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetChannelsAction(
								lineplot.SetChannelsPayload{
									AxisKey:  lineplot.YAxisKeyY1,
									Channels: []channel.Key{ch2, ch4},
								},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					Expect(res.Channels.Y1).To(Equal([]channel.Key{ch2, ch4}))
				},
			)

			It(
				"Should reconcile lines when SetChannels replaces the channel set",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{
						Project:  &proj.Key,
						Name:     "test",
						Channels: lineplot.Channels{X1: ch10, Y1: []channel.Key{ch5}},
						Ranges:   refs(rk1),
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetChannelsAction(
								lineplot.SetChannelsPayload{
									AxisKey:  lineplot.YAxisKeyY1,
									Channels: []channel.Key{ch6},
								},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					Expect(lineKeysOf(res.Lines)).To(ConsistOf(lk(plot.Key, ch10, ch6)))
				},
			)

			It(
				"Should preserve styling of surviving channels under SetChannels",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{
						Project: &proj.Key,
						Name:    "test",
						Channels: lineplot.Channels{
							X1: ch10,
							Y1: []channel.Key{ch5, ch6},
						},
						Ranges: refs(rk1),
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					keep := lineplot.NewSetLineColorAction(lineplot.SetLineColorPayload{
						Key:   lk(plot.Key, ch10, ch5),
						Color: new(MustSucceed(color.FromHex("#ff0000"))),
					})
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{keep}),
					).
						To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetChannelsAction(
								lineplot.SetChannelsPayload{
									AxisKey:  lineplot.YAxisKeyY1,
									Channels: []channel.Key{ch5},
								},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					Expect(res.Lines).To(HaveLen(1))
					Expect(
						res.Lines[0].Color,
					).To(Equal(new(MustSucceed(color.FromHex("#ff0000")))))
				},
			)

			It(
				"Should reject SetChannels targeting an x-axis with validate.ErrValidation",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetChannelsAction(
								lineplot.SetChannelsPayload{
									AxisKey:  lineplot.YAxisKey("x1"),
									Channels: []channel.Key{ch1},
								},
							),
						}),
					).Error().
						To(MatchError(validate.ErrValidation))
				},
			)

			It(
				"Should replace the single channel bound to an x-axis via SetXChannel",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetXChannelAction(
								lineplot.SetXChannelPayload{
									AxisKey: lineplot.XAxisKeyX1, Channel: ch99,
								},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					Expect(res.Channels.X1).To(Equal(ch99))
				},
			)

			It(
				"Should reject SetXChannel targeting a y-axis with validate.ErrValidation",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetXChannelAction(
								lineplot.SetXChannelPayload{
									AxisKey: lineplot.XAxisKey("y1"), Channel: ch1,
								},
							),
						}),
					).Error().
						To(MatchError(validate.ErrValidation))
				},
			)

			It("Should append and remove ranges on an x-axis", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				r1 := uuid.New().String()
				r2 := uuid.New().String()
				Expect(
					dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewAddRangeAction(
							lineplot.AddRangePayload{Range: keyRef(r1)},
						),
						lineplot.NewAddRangeAction(
							lineplot.AddRangePayload{Range: keyRef(r2)},
						),
						lineplot.NewRemoveRangeAction(
							lineplot.RemoveRangePayload{Range: keyRef(r1)},
						),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(res.Ranges).To(Equal(refs(r2)))
			})

			It(
				"Should replace an x-axis's whole range set via SetRanges",
				func(ctx SpecContext) {
					r1, r2, r3 := uuid.New().String(),
						uuid.New().String(),
						uuid.New().String()
					plot := lineplot.LinePlot{
						Project: &proj.Key,
						Name:    "test",
						Ranges:  refs(r1, r2),
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetRangesAction(
								lineplot.SetRangesPayload{Ranges: refs(r2, r3)},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					Expect(res.Ranges).To(Equal(refs(r2, r3)))
				},
			)

			It(
				"Should keep a channel's line when SetRanges replaces the range set",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{
						Project:  &proj.Key,
						Name:     "test",
						Channels: lineplot.Channels{X1: ch10, Y1: []channel.Key{ch5}},
						Ranges:   refs(rk1),
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetRangesAction(
								lineplot.SetRangesPayload{Ranges: refs(rk2)},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					Expect(lineKeysOf(res.Lines)).To(ConsistOf(lk(plot.Key, ch10, ch5)))
				},
			)

			It(
				"Should edit a window in place via SetRanges and keep the plot's lines",
				func(ctx SpecContext) {
					window := func(span telem.TimeSpan) ranger.Ref {
						return ranger.Ref{Variant: ranger.WindowRef{
							Window: telem.Window{
								Variant: telem.DynamicWindow{TimeSpan: span},
							},
						}}
					}
					plot := lineplot.LinePlot{
						Project:  &proj.Key,
						Name:     "test",
						Channels: lineplot.Channels{X1: ch10, Y1: []channel.Key{ch5}},
						Ranges:   []ranger.Ref{window(telem.Minute)},
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetRangesAction(lineplot.SetRangesPayload{
								Ranges: []ranger.Ref{window(telem.Hour)},
							}),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).To(Succeed())
					Expect(res.Ranges).To(Equal([]ranger.Ref{window(telem.Hour)}))
					Expect(res.Lines).To(Equal(plot.Lines))
				},
			)

			It(
				"Should ignore AddRange for a reference the plot already holds",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{
						Project: &proj.Key,
						Name:    "test",
						Ranges:  refs(rk1),
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewAddRangeAction(
								lineplot.AddRangePayload{Range: keyRef(rk1)},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).To(Succeed())
					Expect(res.Ranges).To(Equal(refs(rk1)))
				},
			)

			It(
				"Should reject a RemoveRange action with no payload",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							{Type: lineplot.ActionTypeRemoveRange},
						}),
					).To(MatchError(union.ErrMissingPayload))
				},
			)

			It(
				"Should set axis fields via the fine-grained actions",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					priorDirection := plot.Axes.Y1.LabelDirection
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetAxisLabelAction(
								lineplot.SetAxisLabelPayload{
									Key: lineplot.AxisKeyY1, Label: "pressure",
								},
							),
							lineplot.NewSetAxisTickSpacingAction(
								lineplot.SetAxisTickSpacingPayload{
									Key: lineplot.AxisKeyY1, TickSpacing: 50.0,
								},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					Expect(res.Axes.Y1.Label).To(Equal("pressure"))
					Expect(res.Axes.Y1.TickSpacing).To(Equal(50.0))
					Expect(res.Axes.Y1.LabelDirection).To(Equal(priorDirection))
				},
			)

			It(
				"Should reject a fine-grained axis action with an unknown key",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetAxisLabelAction(
								lineplot.SetAxisLabelPayload{
									Key: lineplot.AxisKey("nope"), Label: "x",
								},
							),
						}),
					).Error().
						To(MatchError(validate.ErrValidation))
				},
			)

			It(
				"Should set and clear line fields via the fine-grained actions",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d0", []lineplot.Action{
							lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
								AxisKey: lineplot.YAxisKeyY1, Channel: ch1,
							}),
							lineplot.NewAddRangeAction(
								lineplot.AddRangePayload{Range: keyRef(rk1)},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					Expect(res.Lines).To(HaveLen(1))
					Expect(res.Lines[0].StrokeWidth).To(Equal(2.0))
					key := res.Lines[0].Key

					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetLineStrokeWidthAction(
								lineplot.SetLineStrokeWidthPayload{
									Key: key, StrokeWidth: 5.0,
								},
							),
							lineplot.NewSetLineColorAction(
								lineplot.SetLineColorPayload{
									Key:   key,
									Color: new(MustSucceed(color.FromHex("#ff0000"))),
								},
							),
							lineplot.NewSetLineAggregationAction(
								lineplot.SetLineAggregationPayload{
									Key:         key,
									Aggregation: lineplot.AggregationAverage,
								},
							),
							lineplot.NewSetLineDetailAction(
								lineplot.SetLineDetailPayload{
									Key:    key,
									Detail: lineplot.DetailLow,
								},
							),
						}),
					).To(Succeed())
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					Expect(res.Lines[0].StrokeWidth).To(Equal(5.0))
					Expect(
						res.Lines[0].Aggregation,
					).To(Equal(lineplot.AggregationAverage))
					Expect(res.Lines[0].Detail).To(Equal(lineplot.DetailLow))
					Expect(
						res.Lines[0].Color,
					).To(Equal(new(MustSucceed(color.FromHex("#ff0000")))))

					Expect(
						dispatch(ctx, plot.Key, "d2", []lineplot.Action{
							lineplot.NewSetLineColorAction(
								lineplot.SetLineColorPayload{Key: key},
							),
						}),
					).To(Succeed())
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					Expect(res.Lines[0].Color).To(BeNil())
				},
			)

			It(
				"Should replace a line's full configuration via SetLine",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d0", []lineplot.Action{
							lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
								AxisKey: lineplot.YAxisKeyY1, Channel: ch1,
							}),
							lineplot.NewAddRangeAction(
								lineplot.AddRangePayload{Range: keyRef(rk1)},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					stored := res.Lines[0]
					key := stored.Key
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetLineAction(
								lineplot.SetLinePayload{Line: lineplot.Line{
									Key:         key,
									YAxis:       stored.YAxis,
									YChannel:    stored.YChannel,
									XChannel:    stored.XChannel,
									StrokeWidth: 4,
									Aggregation: lineplot.AggregationAverage,
									Detail:      lineplot.DetailHigh,
									Color: new(
										MustSucceed(color.FromHex("#0000ff")),
									),
								}},
							),
						}),
					).To(Succeed())
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					Expect(res.Lines).To(HaveLen(1))
					Expect(res.Lines[0].StrokeWidth).To(Equal(4.0))
					Expect(
						res.Lines[0].Aggregation,
					).To(Equal(lineplot.AggregationAverage))
					Expect(res.Lines[0].Detail).To(Equal(lineplot.DetailHigh))
					Expect(
						res.Lines[0].Color,
					).To(Equal(new(MustSucceed(color.FromHex("#0000ff")))))
				},
			)

			It("Should insert, update, and remove rules", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				rule := lineplot.Rule{
					Key:       rk1,
					Label:     "ceiling",
					Color:     new(MustSucceed(color.FromHex("#888888"))),
					Axis:      lineplot.AxisKeyY1,
					LineWidth: 1,
					LineDash:  4,
					Units:     "psi",
					Position:  100,
				}
				Expect(
					dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewSetRuleAction(lineplot.SetRulePayload{Rule: rule}),
					}),
				).To(Succeed())

				rule.Label = "max"
				Expect(
					dispatch(ctx, plot.Key, "d2", []lineplot.Action{
						lineplot.NewSetRuleAction(lineplot.SetRulePayload{Rule: rule}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(res.Rules[0].Label).To(Equal("max"))

				Expect(
					dispatch(ctx, plot.Key, "d3", []lineplot.Action{
						lineplot.NewRemoveRuleAction(
							lineplot.RemoveRulePayload{Key: rk1},
						),
					}),
				).To(Succeed())
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(res.Rules).To(BeEmpty())
			})

			It(
				"Should set rule fields via the fine-grained actions",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetRuleAction(
								lineplot.SetRulePayload{Rule: lineplot.Rule{
									Key:   rk1,
									Label: "ceiling",
									Axis:  lineplot.AxisKeyY1,
								}},
							),
						}),
					).To(Succeed())
					Expect(
						dispatch(ctx, plot.Key, "d2", []lineplot.Action{
							lineplot.NewSetRuleLabelAction(
								lineplot.SetRuleLabelPayload{
									Key: rk1, Label: "max",
								},
							),
							lineplot.NewSetRulePositionAction(
								lineplot.SetRulePositionPayload{
									Key: rk1, Position: 42,
								},
							),
							lineplot.NewSetRuleAxisAction(
								lineplot.SetRuleAxisPayload{
									Key: rk1, Axis: lineplot.AxisKeyY2,
								},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, db),
					).
						To(Succeed())
					Expect(res.Rules).To(HaveLen(1))
					Expect(res.Rules[0].Label).To(Equal("max"))
					Expect(res.Rules[0].Position).To(Equal(42.0))
					Expect(res.Rules[0].Axis).To(Equal(lineplot.AxisKeyY2))
				},
			)
		})

		Describe("atomicity and broadcast", func() {
			It("Should apply a multi-action batch atomically", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Project: &proj.Key, Name: "test"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				r := uuid.New().String()
				Expect(
					dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewSetXChannelAction(lineplot.SetXChannelPayload{
							AxisKey: lineplot.XAxisKeyX1, Channel: ch1,
						}),
						lineplot.NewAddRangeAction(
							lineplot.AddRangePayload{Range: keyRef(r)},
						),
						lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
							AxisKey: lineplot.YAxisKeyY1, Channel: ch10,
						}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, db),
				).
					To(Succeed())
				Expect(res.Channels.X1).To(Equal(ch1))
				Expect(res.Ranges).To(Equal(refs(r)))
				Expect(res.Channels.Y1).To(ConsistOf(ch10))
			})

			It(
				"Should notify subscribers with the dispatched ScopedAction on success",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "observed"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					rec := &Recorder[lineplot.Key, lineplot.Action]{}
					DeferCleanup(svc.OnAction(rec.Record))
					actions := []lineplot.Action{
						lineplot.NewSetAction(
							lineplot.SetPayload{
								Path:  []string{"name"},
								Value: "broadcast",
							},
						),
						lineplot.NewSetXChannelAction(lineplot.SetXChannelPayload{
							AxisKey: lineplot.XAxisKeyX1, Channel: ch5,
						}),
					}
					Expect(
						dispatch(ctx, plot.Key, "client-xyz", actions),
					).
						To(Succeed())
					seen := rec.Snapshot()
					Expect(seen).To(HaveLen(1))
					Expect(seen[0].Key).To(Equal(plot.Key))
					Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
					Expect(seen[0].Actions).To(HaveLen(2))
					Expect(seen[0].Actions[0].Type).To(Equal(lineplot.ActionTypeSet))
					Expect(
						seen[0].Actions[1].Type,
					).To(Equal(lineplot.ActionTypeSetXChannel))
				},
			)

			It(
				"Should assign strictly monotonic Seq across successive dispatches",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "seq"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					rec := &Recorder[lineplot.Key, lineplot.Action]{}
					DeferCleanup(svc.OnAction(rec.Record))
					for _, name := range []string{"a", "b", "c"} {
						Expect(
							dispatch(ctx, plot.Key, "d", []lineplot.Action{
								lineplot.NewSetAction(
									lineplot.SetPayload{
										Path:  []string{"name"},
										Value: name,
									},
								),
							}),
						).To(Succeed())
					}
					seen := rec.Snapshot()
					Expect(seen).To(HaveLen(3))
					Expect(seen[1].Seq).To(BeNumerically(">", seen[0].Seq))
					Expect(seen[2].Seq).To(BeNumerically(">", seen[1].Seq))
				},
			)

			It(
				"Should not notify subscribers when Reduce rejects the action",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Project: &proj.Key, Name: "rejected"}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					rec := &Recorder[lineplot.Key, lineplot.Action]{}
					DeferCleanup(svc.OnAction(rec.Record))
					Expect(
						dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
								AxisKey: lineplot.YAxisKey("x1"), Channel: ch1,
							}),
						}),
					).Error().
						To(MatchError(validate.ErrValidation))
					Expect(rec.Snapshot()).To(BeEmpty())
				},
			)
		})
	})
})
