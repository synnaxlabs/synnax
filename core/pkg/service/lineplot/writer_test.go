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
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	xcolor "github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/defaults"
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

// uncolored returns rs with the color of each reference cleared, so a spec can compare
// references without the series colors that writes give them.
func uncolored(rs []ranger.Ref) []ranger.Ref {
	out := make([]ranger.Ref, len(rs))
	for i, r := range rs {
		switch v := r.Variant.(type) {
		case ranger.KeyRef:
			v.Color = nil
			out[i] = ranger.Ref{Variant: v}
		case ranger.WindowRef:
			v.Color = nil
			out[i] = ranger.Ref{Variant: v}
		}
	}
	return out
}

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

func fixed(hex string) *color.Ref {
	return &color.Ref{Variant: color.FixedRef{Value: MustSucceed(xcolor.Parse(hex))}}
}

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a LinePlot", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			plot := defaults.Fill(lineplot.LinePlot{Project: &proj.Key, Name: "test"})
			Expect(svc.NewWriter(tx).Create(ctx, &plot)).To(Succeed())
			Expect(plot.Key).ToNot(Equal(uuid.Nil()))
		})
		It("Should keep an explicit zero in a defaulted field", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			plot := defaults.Fill(lineplot.LinePlot{Project: &proj.Key, Name: "zero"})
			plot.Legend.Position.X = 0
			Expect(svc.NewWriter(tx).Create(ctx, &plot)).To(Succeed())
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().
				Where(lineplot.MatchKeys(plot.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Legend.Position.X).To(BeZero())
			Expect(res.Legend.Position.Y).To(Equal(50.0))
		})
		It(
			"Should return a validation error when the name is empty",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				plot := defaults.Fill(lineplot.LinePlot{Project: &proj.Key})
				Expect(svc.NewWriter(tx).Create(ctx, &plot)).
					To(MatchError(ContainSubstring("name: required")))
			},
		)
		It("Should refuse a line plot whose key exists", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			stored := defaults.Fill(
				lineplot.LinePlot{Project: &proj.Key, Name: "stored"},
			)
			Expect(svc.NewWriter(tx).Create(ctx, &stored)).To(Succeed())
			again := defaults.Fill(lineplot.LinePlot{
				Key:      stored.Key,
				Project:  &proj.Key,
				Name:     "refused",
				Channels: lineplot.Channels{Y1: []channel.Key{ch5}},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &again)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("lineplot %s already exists", stored.Key)),
			))
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().
				Where(lineplot.MatchKeys(stored.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(stored))
		})
		It("Should reject a line plot with no project", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			plot := defaults.Fill(lineplot.LinePlot{Name: "owner"})
			Expect(svc.NewWriter(tx).Create(ctx, &plot)).To(MatchError(
				ContainSubstring("project: must belong to a project"),
			))
		})
		It(
			"Should populate lines from channel and range bindings supplied at creation",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				plot := defaults.Fill(lineplot.LinePlot{
					Project:  &proj.Key,
					Name:     "test",
					Channels: lineplot.Channels{X1: ch10, Y1: []channel.Key{ch5, ch6}},
					Ranges:   refs(rk1),
				})
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
				v := defaults.Fill(
					lineplot.LinePlot{Project: &proj.Key, Name: "broadcast"},
				)
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
				v := defaults.Fill(
					lineplot.LinePlot{Project: &proj.Key, Name: "discarded"},
				)
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
			plots := defaults.Fill([]lineplot.LinePlot{
				{Name: "plot-1", Project: &proj.Key},
				{Name: "plot-2", Project: &proj.Key},
			})
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
				plot := defaults.Fill(lineplot.LinePlot{
					Project:  &proj.Key,
					Name:     "test",
					Channels: lineplot.Channels{X1: ch10},
					Ranges:   refs(rk1, rk2),
				})
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
				plot := defaults.Fill(lineplot.LinePlot{
					Project:  &proj.Key,
					Name:     "test",
					Channels: lineplot.Channels{X1: ch10},
					Ranges:   refs(rk1),
				})
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
				plot := defaults.Fill(lineplot.LinePlot{
					Project:  &proj.Key,
					Name:     "test",
					Channels: lineplot.Channels{X1: ch10, Y1: []channel.Key{ch5, ch6}},
					Ranges:   refs(rk1),
				})
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
				plot := defaults.Fill(lineplot.LinePlot{
					Project:  &proj.Key,
					Name:     "test",
					Channels: lineplot.Channels{X1: ch10, Y1: []channel.Key{ch5}},
					Ranges:   refs(rk1),
				})
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

	Describe("Apply of a project move", func() {
		It("Should move the line plot into the project", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			dest := project.Project{Name: "destination"}
			Expect(projectSvc.NewWriter(tx).Create(ctx, &dest)).To(Succeed())
			v := defaults.Fill(lineplot.LinePlot{Project: &proj.Key, Name: "moved"})
			Expect(svc.NewWriter(tx).Create(ctx, &v)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(ctx, v.Key, setProject(dest.Key))).
				To(BeEmpty())
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().
				Where(lineplot.MatchKeys(v.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Project).To(HaveValue(Equal(dest.Key)))
		})

		It("Should reject a project that does not exist", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			v := defaults.Fill(lineplot.LinePlot{Project: &proj.Key, Name: "stuck"})
			Expect(svc.NewWriter(tx).Create(ctx, &v)).To(Succeed())
			Expect(svc.NewWriter(tx).Apply(ctx, v.Key, setProject(uuid.New()))).
				Error().To(MatchError(ContainSubstring("does not exist")))
		})
	})

	Describe("Dispatch", func() {
		It(
			"Should reject a create without changing the line plot it names",
			func(ctx SpecContext) {
				target := defaults.Fill(
					lineplot.LinePlot{Project: &proj.Key, Name: "target"},
				)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &target)
				})).To(Succeed())
				other := defaults.Fill(
					lineplot.LinePlot{Project: &proj.Key, Name: "other"},
				)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &other)
				})).To(Succeed())
				Expect(dispatch(ctx, target.Key, "d1", []lineplot.Action{
					lineplot.NewCreateAction(lineplot.CreatePayload{
						LinePlot: defaults.Fill(lineplot.LinePlot{
							Key:  other.Key,
							Name: "overwritten",
						}),
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
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "test"},
					)
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
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "test"},
					)
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
				plot := defaults.Fill(
					lineplot.LinePlot{Project: &proj.Key, Name: "original"},
				)
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
				plot := defaults.Fill(
					lineplot.LinePlot{Project: &proj.Key, Name: "test"},
				)
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
				plot := defaults.Fill(
					lineplot.LinePlot{Project: &proj.Key, Name: "test"},
				)
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
				plot := defaults.Fill(
					lineplot.LinePlot{Project: &proj.Key, Name: "test"},
				)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				pos := spatial.StickyXY{X: 10, Y: 20}
				pos = defaults.Fill(pos)
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
				plot := defaults.Fill(
					lineplot.LinePlot{Project: &proj.Key, Name: "test"},
				)
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
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "test"},
					)
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
				plot := defaults.Fill(
					lineplot.LinePlot{Project: &proj.Key, Name: "test"},
				)
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
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "test"},
					)
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
				plot := defaults.Fill(
					lineplot.LinePlot{Project: &proj.Key, Name: "test"},
				)
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
					plot := defaults.Fill(lineplot.LinePlot{
						Project:  &proj.Key,
						Name:     "test",
						Channels: lineplot.Channels{Y1: []channel.Key{ch1, ch2, ch3}},
					})
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
					plot := defaults.Fill(lineplot.LinePlot{
						Project:  &proj.Key,
						Name:     "test",
						Channels: lineplot.Channels{X1: ch10, Y1: []channel.Key{ch5}},
						Ranges:   refs(rk1),
					})
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
					plot := defaults.Fill(lineplot.LinePlot{
						Project: &proj.Key,
						Name:    "test",
						Channels: lineplot.Channels{
							X1: ch10,
							Y1: []channel.Key{ch5, ch6},
						},
						Ranges: refs(rk1),
					})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Create(ctx, &plot)
					})).To(Succeed())
					keep := lineplot.NewSetLineColorAction(lineplot.SetLineColorPayload{
						Key:   lk(plot.Key, ch10, ch5),
						Color: fixed("#ff0000"),
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
					).To(Equal(fixed("#ff0000")))
				},
			)

			It(
				"Should reject SetChannels targeting an x-axis with validate.ErrValidation",
				func(ctx SpecContext) {
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "test"},
					)
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
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "test"},
					)
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
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "test"},
					)
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
				plot := defaults.Fill(
					lineplot.LinePlot{Project: &proj.Key, Name: "test"},
				)
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
				Expect(uncolored(res.Ranges)).To(Equal(refs(r2)))
			})

			It(
				"Should replace an x-axis's whole range set via SetRanges",
				func(ctx SpecContext) {
					r1, r2, r3 := uuid.New().String(),
						uuid.New().String(),
						uuid.New().String()
					plot := defaults.Fill(lineplot.LinePlot{
						Project: &proj.Key,
						Name:    "test",
						Ranges:  refs(r1, r2),
					})
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
					Expect(uncolored(res.Ranges)).To(Equal(refs(r2, r3)))
				},
			)

			It(
				"Should keep a channel's line when SetRanges replaces the range set",
				func(ctx SpecContext) {
					plot := defaults.Fill(lineplot.LinePlot{
						Project:  &proj.Key,
						Name:     "test",
						Channels: lineplot.Channels{X1: ch10, Y1: []channel.Key{ch5}},
						Ranges:   refs(rk1),
					})
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
					plot := defaults.Fill(lineplot.LinePlot{
						Project:  &proj.Key,
						Name:     "test",
						Channels: lineplot.Channels{X1: ch10, Y1: []channel.Key{ch5}},
						Ranges:   []ranger.Ref{window(telem.Minute)},
					})
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
					Expect(
						uncolored(res.Ranges),
					).To(Equal([]ranger.Ref{window(telem.Hour)}))
					Expect(res.Lines).To(Equal(plot.Lines))
				},
			)

			It(
				"Should ignore AddRange for a reference the plot already holds",
				func(ctx SpecContext) {
					plot := defaults.Fill(lineplot.LinePlot{
						Project: &proj.Key,
						Name:    "test",
						Ranges:  refs(rk1),
					})
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
					Expect(uncolored(res.Ranges)).To(Equal(refs(rk1)))
				},
			)

			It(
				"Should reject a RemoveRange action with no payload",
				func(ctx SpecContext) {
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "test"},
					)
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
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "test"},
					)
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
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "test"},
					)
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
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "test"},
					)
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
									Color: fixed("#ff0000"),
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
					).To(Equal(fixed("#ff0000")))

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
					series := MustSucceed(colorSvc.Series(ctx, db))
					Expect(res.Lines[0].Color).To(Equal(
						&color.Ref{Variant: color.NamedRef{Key: series[0]}},
					))
				},
			)

			It(
				"Should replace a line's full configuration via SetLine",
				func(ctx SpecContext) {
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "test"},
					)
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
									Color:       fixed("#0000ff"),
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
					).To(Equal(fixed("#0000ff")))
				},
			)

			It("Should insert, update, and remove rules", func(ctx SpecContext) {
				plot := defaults.Fill(
					lineplot.LinePlot{Project: &proj.Key, Name: "test"},
				)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &plot)
				})).To(Succeed())
				rule := lineplot.Rule{
					Key:       rk1,
					Label:     "ceiling",
					Color:     fixed("#888888"),
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
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "test"},
					)
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
				plot := defaults.Fill(
					lineplot.LinePlot{Project: &proj.Key, Name: "test"},
				)
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
				Expect(uncolored(res.Ranges)).To(Equal(refs(r)))
				Expect(res.Channels.Y1).To(ConsistOf(ch10))
			})

			It(
				"Should notify subscribers with the dispatched ScopedAction on success",
				func(ctx SpecContext) {
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "observed"},
					)
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
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "seq"},
					)
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
					plot := defaults.Fill(
						lineplot.LinePlot{Project: &proj.Key, Name: "rejected"},
					)
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

var _ = Describe("Line colors", func() {
	var series []color.Key
	BeforeEach(func(ctx SpecContext) {
		series = MustSucceed(colorSvc.Series(ctx, db))
	})
	named := func(k color.Key) *color.Ref {
		return &color.Ref{Variant: color.NamedRef{Key: k}}
	}
	colors := func(p lineplot.LinePlot) []*color.Ref {
		return lo.Map(
			p.Lines,
			func(l lineplot.Line, _ int) *color.Ref { return l.Color },
		)
	}
	retrieve := func(ctx SpecContext, key lineplot.Key) lineplot.LinePlot {
		GinkgoHelper()
		var res lineplot.LinePlot
		Expect(svc.NewRetrieve().
			Where(lineplot.MatchKeys(key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}
	create := func(ctx SpecContext, p lineplot.LinePlot) lineplot.LinePlot {
		GinkgoHelper()
		p = defaults.Fill(p)
		p.Project, p.Name = &proj.Key, "colored"
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &p)
		})).To(Succeed())
		return p
	}
	addChannel := func(ch channel.Key) lineplot.Action {
		return lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
			AxisKey: lineplot.YAxisKeyY1, Channel: ch,
		})
	}

	It(
		"Should give each line of a new plot the next series color",
		func(ctx SpecContext) {
			p := create(ctx, lineplot.LinePlot{
				Channels: lineplot.Channels{Y1: []channel.Key{ch1, ch2}},
			})
			Expect(
				colors(p),
			).To(Equal([]*color.Ref{named(series[0]), named(series[1])}))
		},
	)

	It(
		"Should give an added line the first series color that no line uses",
		func(ctx SpecContext) {
			p := create(ctx, lineplot.LinePlot{
				Channels: lineplot.Channels{Y1: []channel.Key{ch1, ch2}},
				Lines: []lineplot.Line{
					{
						YAxis:    lineplot.YAxisKeyY1,
						YChannel: ch1,
						Color:    named(series[0]),
					},
					{
						YAxis:    lineplot.YAxisKeyY1,
						YChannel: ch2,
						Color:    named(series[2]),
					},
				},
			})
			rec := &Recorder[lineplot.Key, lineplot.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{addChannel(ch3)})).
				To(Succeed())
			res := retrieve(ctx, p.Key)
			Expect(res.Lines[2].Color).To(Equal(named(series[1])))
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Actions).To(Equal([]lineplot.Action{
				addChannel(ch3),
				lineplot.NewSetLineColorAction(lineplot.SetLineColorPayload{
					Key: res.Lines[2].Key, Color: named(series[1]),
				}),
			}))
		},
	)

	It(
		"Should give a line past the last series color the one used least",
		func(ctx SpecContext) {
			chs := MustSucceed(
				channeltestutil.CreateVirtual(ctx, db, channelSvc, len(series)+2),
			)
			p := create(ctx, lineplot.LinePlot{
				Channels: lineplot.Channels{Y1: chs[:len(chs)-1]},
				Lines: []lineplot.Line{
					{
						YAxis:    lineplot.YAxisKeyY1,
						YChannel: chs[0],
						Color:    named(series[0]),
					},
					{
						YAxis:    lineplot.YAxisKeyY1,
						YChannel: chs[1],
						Color:    named(series[0]),
					},
				},
			})
			Expect(
				colors(p)[2:],
			).To(Equal(lo.Map(series[1:], func(k color.Key, _ int) *color.Ref {
				return named(k)
			})))
			Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
				addChannel(chs[len(chs)-1]),
			})).To(Succeed())
			Expect(
				retrieve(ctx, p.Key).Lines[len(chs)-1].Color,
			).To(Equal(named(series[1])))
		},
	)

	It(
		"Should give a cleared line the first series color that no other line uses",
		func(ctx SpecContext) {
			p := create(ctx, lineplot.LinePlot{
				Channels: lineplot.Channels{Y1: []channel.Key{ch1, ch2}},
				Lines: []lineplot.Line{
					{
						YAxis:    lineplot.YAxisKeyY1,
						YChannel: ch1,
						Color:    named(series[0]),
					},
					{
						YAxis:    lineplot.YAxisKeyY1,
						YChannel: ch2,
						Color:    named(series[0]),
					},
				},
			})
			rec := &Recorder[lineplot.Key, lineplot.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			cleared := lineplot.NewSetLineColorAction(
				lineplot.SetLineColorPayload{Key: p.Lines[0].Key},
			)
			Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{cleared})).To(Succeed())
			Expect(
				colors(retrieve(ctx, p.Key)),
			).To(Equal([]*color.Ref{named(series[1]), named(series[0])}))
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Actions).To(Equal([]lineplot.Action{
				cleared,
				lineplot.NewSetLineColorAction(lineplot.SetLineColorPayload{
					Key: p.Lines[0].Key, Color: named(series[1]),
				}),
			}))
		},
	)

	It(
		"Should give each rule with no color the next series color",
		func(ctx SpecContext) {
			p := create(ctx, lineplot.LinePlot{
				Rules: []lineplot.Rule{
					{Key: "r1", Axis: lineplot.AxisKeyY1, Color: named(series[0])},
					{Key: "r2", Axis: lineplot.AxisKeyY1},
					{Key: "r3", Axis: lineplot.AxisKeyY1},
				},
			})
			Expect(lo.Map(p.Rules, func(r lineplot.Rule, _ int) *color.Ref {
				return r.Color
			})).To(Equal([]*color.Ref{
				named(series[0]), named(series[1]), named(series[2]),
			}))
		},
	)
})

var _ = Describe("Range compare", func() {
	var series []color.Key
	BeforeEach(func(ctx SpecContext) {
		series = MustSucceed(colorSvc.Series(ctx, db))
	})
	named := func(k color.Key) *color.Ref {
		return &color.Ref{Variant: color.NamedRef{Key: k}}
	}
	fixed := func(c xcolor.Color) *color.Ref {
		return &color.Ref{Variant: color.FixedRef{Value: c}}
	}
	// orange is apart from the first two series colors in every kind of vision.
	orange := xcolor.Color{R: 255, G: 128, A: 255}
	retrieve := func(ctx SpecContext, key lineplot.Key) lineplot.LinePlot {
		GinkgoHelper()
		var res lineplot.LinePlot
		Expect(svc.NewRetrieve().
			Where(lineplot.MatchKeys(key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}
	create := func(ctx SpecContext, p lineplot.LinePlot) lineplot.LinePlot {
		GinkgoHelper()
		p = defaults.Fill(p)
		p.Project, p.Name = &proj.Key, "compare"
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &p)
		})).To(Succeed())
		return p
	}
	createRange := func(ctx SpecContext, c *xcolor.Color) ranger.Ref {
		GinkgoHelper()
		r := ranger.Range{
			Key:       uuid.New(),
			Name:      "range",
			TimeRange: telem.TimeRange{Start: 1, End: 2},
			Color:     c,
		}
		Expect(gorp.NewCreate[ranger.Key, ranger.Range]().
			Entry(&r).Exec(ctx, db)).To(Succeed())
		return ranger.Ref{Variant: ranger.KeyRef{Key: r.Key}}
	}
	colored := func(r ranger.Ref, c *color.Ref) ranger.Ref {
		GinkgoHelper()
		k, ok := r.Variant.(ranger.KeyRef)
		Expect(ok).To(BeTrue())
		k.Color = c
		return ranger.Ref{Variant: k}
	}
	t0Of := func(r ranger.Ref) *telem.TimeStamp {
		GinkgoHelper()
		k, ok := r.Variant.(ranger.KeyRef)
		Expect(ok).To(BeTrue())
		return k.T0
	}
	refColors := func(p lineplot.LinePlot) []*color.Ref {
		return lo.Map(p.Ranges, func(r ranger.Ref, _ int) *color.Ref {
			switch v := r.Variant.(type) {
			case ranger.KeyRef:
				return v.Color
			case ranger.WindowRef:
				return v.Color
			}
			return nil
		})
	}

	Describe("Colors", func() {
		It("Should give each range with no color of its own a series color",
			func(ctx SpecContext) {
				p := create(ctx, lineplot.LinePlot{
					Ranges: []ranger.Ref{createRange(ctx, nil), recentRef},
				})
				Expect(refColors(p)).To(Equal([]*color.Ref{
					named(series[0]), named(series[1]),
				}))
			})

		It("Should give a range its own color when it is apart from the other colors",
			func(ctx SpecContext) {
				p := create(ctx, lineplot.LinePlot{
					Ranges: []ranger.Ref{
						createRange(ctx, nil),
						createRange(ctx, &orange),
					},
				})
				Expect(refColors(p)).To(Equal([]*color.Ref{
					named(series[0]), fixed(orange),
				}))
			})

		It("Should give a series color to a range whose own color another range uses",
			func(ctx SpecContext) {
				p := create(ctx, lineplot.LinePlot{
					Ranges: []ranger.Ref{
						createRange(ctx, &orange),
						createRange(ctx, &orange),
					},
				})
				Expect(refColors(p)).To(Equal([]*color.Ref{
					fixed(orange), named(series[0]),
				}))
			})

		It("Should give a series color to an added range near the color of another",
			func(ctx SpecContext) {
				p := create(
					ctx,
					lineplot.LinePlot{Ranges: []ranger.Ref{createRange(ctx, nil)}},
				)
				// The first series color is 0xDC136C.
				near := xcolor.Color{R: 0xDC, G: 0x15, B: 0x6C, A: 255}
				added := createRange(ctx, &near)
				Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
					lineplot.NewAddRangeAction(lineplot.AddRangePayload{Range: added}),
				})).To(Succeed())
				Expect(refColors(retrieve(ctx, p.Key))).To(Equal([]*color.Ref{
					named(series[0]), named(series[1]),
				}))
			})

		It("Should skip a series color close to the own color of another range",
			func(ctx SpecContext) {
				near := xcolor.Color{R: 0xDC, G: 0x15, B: 0x6C, A: 255}
				p := create(ctx, lineplot.LinePlot{
					Ranges: []ranger.Ref{
						createRange(ctx, &near),
						createRange(ctx, nil),
					},
				})
				Expect(refColors(p)).To(Equal([]*color.Ref{
					fixed(near), named(series[1]),
				}))
			})

		It("Should give a series color to a range that no longer exists",
			func(ctx SpecContext) {
				p := create(ctx, lineplot.LinePlot{Ranges: refs(rk1)})
				Expect(refColors(p)).To(Equal([]*color.Ref{named(series[0])}))
			})

		It("Should give an added range the series color that no other range uses",
			func(ctx SpecContext) {
				p := create(ctx, lineplot.LinePlot{Ranges: refs(rk1)})
				rec := &Recorder[lineplot.Key, lineplot.Action]{}
				DeferCleanup(svc.OnAction(rec.Record))
				add := lineplot.NewAddRangeAction(
					lineplot.AddRangePayload{Range: keyRef(rk2)},
				)
				Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{add})).
					To(Succeed())
				Expect(refColors(retrieve(ctx, p.Key))).To(Equal([]*color.Ref{
					named(series[0]), named(series[1]),
				}))
				Expect(rec.Snapshot()[0].Actions).To(Equal([]lineplot.Action{
					add,
					lineplot.NewSetRangeColorAction(lineplot.SetRangeColorPayload{
						Range: keyRef(rk2), Color: *named(series[1]),
					}),
				}))
			})

		It("Should give a series color to a range near the dark value of another",
			func(ctx SpecContext) {
				blue := xcolor.Color{B: 255, A: 255}
				dark := color.Named{Name: "Signal", Light: blue, Dark: &orange}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return colorSvc.NewWriter(tx).Create(ctx, &dark)
				})).To(Succeed())
				p := create(ctx, lineplot.LinePlot{
					Ranges: []ranger.Ref{
						colored(createRange(ctx, nil), named(dark.Key)),
						createRange(ctx, &orange),
					},
				})
				Expect(refColors(p)).To(Equal([]*color.Ref{
					named(dark.Key), named(series[0]),
				}))
			})

		It("Should set the color of a window", func(ctx SpecContext) {
			p := create(ctx, lineplot.LinePlot{Ranges: []ranger.Ref{recentRef}})
			Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
				lineplot.NewSetRangeColorAction(lineplot.SetRangeColorPayload{
					Range: recentRef, Color: *fixed(orange),
				}),
			})).To(Succeed())
			Expect(
				refColors(retrieve(ctx, p.Key)),
			).To(Equal([]*color.Ref{fixed(orange)}))
		})

		It("Should set the color of a range by its key", func(ctx SpecContext) {
			p := create(ctx, lineplot.LinePlot{Ranges: refs(rk1)})
			Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
				lineplot.NewSetRangeColorAction(lineplot.SetRangeColorPayload{
					Range: keyRef(rk1), Color: *fixed(orange),
				}),
			})).To(Succeed())
			Expect(
				refColors(retrieve(ctx, p.Key)),
			).To(Equal([]*color.Ref{fixed(orange)}))
		})
	})

	Describe("Identity", func() {
		It("Should ignore an AddRange for a range the plot holds with another T0",
			func(ctx SpecContext) {
				p := create(ctx, lineplot.LinePlot{Ranges: refs(rk1)})
				t0 := telem.TimeStamp(5)
				Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
					lineplot.NewAddRangeAction(lineplot.AddRangePayload{
						Range: ranger.Ref{Variant: ranger.KeyRef{
							Key: uuid.MustParse(rk1), T0: &t0,
						}},
					}),
				})).To(Succeed())
				Expect(uncolored(retrieve(ctx, p.Key).Ranges)).To(Equal(refs(rk1)))
			})

		It("Should drop a repeated range reference from SetRanges",
			func(ctx SpecContext) {
				p := create(ctx, lineplot.LinePlot{})
				Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
					lineplot.NewSetRangesAction(lineplot.SetRangesPayload{
						Ranges: []ranger.Ref{
							keyRef(rk1), recentRef, keyRef(rk1), recentRef,
						},
					}),
				})).To(Succeed())
				Expect(uncolored(retrieve(ctx, p.Key).Ranges)).To(Equal(
					[]ranger.Ref{keyRef(rk1), recentRef},
				))
			})

		It("Should drop a repeated range reference when it creates a plot",
			func(ctx SpecContext) {
				p := create(ctx, lineplot.LinePlot{Ranges: refs(rk1, rk2, rk1)})
				Expect(uncolored(retrieve(ctx, p.Key).Ranges)).To(Equal(refs(rk1, rk2)))
			})

		It("Should keep the T0 and color of a range that SetRanges keeps",
			func(ctx SpecContext) {
				p := create(ctx, lineplot.LinePlot{Ranges: refs(rk1, rk2)})
				t0 := telem.TimeStamp(5)
				Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
					lineplot.NewSetRangeT0Action(lineplot.SetRangeT0Payload{
						Range: uuid.MustParse(rk1), T0: &t0,
					}),
					lineplot.NewSetRangesAction(lineplot.SetRangesPayload{
						Ranges: []ranger.Ref{
							keyRef(rk1), colored(keyRef(rk2), fixed(orange)),
						},
					}),
				})).To(Succeed())
				res := retrieve(ctx, p.Key)
				Expect(t0Of(res.Ranges[0])).To(HaveValue(Equal(t0)))
				Expect(refColors(res)).To(Equal([]*color.Ref{
					named(series[0]), fixed(orange),
				}))
			})

		It("Should remove a range by its key whatever its color",
			func(ctx SpecContext) {
				p := create(ctx, lineplot.LinePlot{Ranges: refs(rk1, rk2)})
				Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
					lineplot.NewRemoveRangeAction(
						lineplot.RemoveRangePayload{Range: keyRef(rk1)},
					),
				})).To(Succeed())
				Expect(uncolored(retrieve(ctx, p.Key).Ranges)).To(Equal(refs(rk2)))
			})
	})

	Describe("T0", func() {
		keyRef := func(r ranger.Ref) ranger.KeyRef {
			k, ok := r.Variant.(ranger.KeyRef)
			Expect(ok).To(BeTrue())
			return k
		}

		It("Should leave the plot alone on a T0 for a range it does not show",
			func(ctx SpecContext) {
				p := create(ctx, lineplot.LinePlot{Ranges: refs(rk1)})
				t0 := telem.TimeStamp(5)
				Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
					lineplot.NewSetRangeT0Action(lineplot.SetRangeT0Payload{
						Range: uuid.MustParse(rk2), T0: &t0,
					}),
				})).To(Succeed())
				Expect(retrieve(ctx, p.Key).Ranges).To(Equal(p.Ranges))
			})

		It("Should set and reset the T0 of a saved range", func(ctx SpecContext) {
			p := create(ctx, lineplot.LinePlot{Ranges: refs(rk1, rk2)})
			t0 := telem.TimeStamp(5)
			Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
				lineplot.NewSetRangeT0Action(lineplot.SetRangeT0Payload{
					Range: uuid.MustParse(rk1), T0: &t0,
				}),
			})).To(Succeed())
			res := retrieve(ctx, p.Key)
			Expect(
				keyRef(res.Ranges[0]).T0,
			).To(HaveValue(Equal(t0)))
			Expect(keyRef(res.Ranges[1]).T0).To(BeNil())
			Expect(keyRef(res.Ranges[0]).Color).ToNot(BeNil())
			Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
				lineplot.NewSetRangeT0Action(
					lineplot.SetRangeT0Payload{Range: uuid.MustParse(rk1)},
				),
			})).To(Succeed())
			res = retrieve(ctx, p.Key)
			Expect(keyRef(res.Ranges[0]).T0).To(BeNil())
		})
	})

	Describe("Time mode", func() {
		It("Should set the time mode of an x axis", func(ctx SpecContext) {
			p := create(ctx, lineplot.LinePlot{})
			mode := lineplot.TimeModeRelative
			Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
				lineplot.NewSetAxisTimeModeAction(lineplot.SetAxisTimeModePayload{
					Key: lineplot.XAxisKeyX1, TimeMode: &mode,
				}),
			})).To(Succeed())
			res := retrieve(ctx, p.Key)
			Expect(res.Axes.X1.TimeMode).To(HaveValue(Equal(mode)))
			Expect(res.Axes.X2.TimeMode).To(BeNil())
		})

		It(
			"Should return the bounds to auto on a change of mode",
			func(ctx SpecContext) {
				p := create(ctx, lineplot.LinePlot{})
				relative := lineplot.TimeModeRelative
				absolute := lineplot.TimeModeAbsolute
				manual := lineplot.ManualBounds{Lower: true, Upper: true}
				setBounds := lineplot.NewSetAxisBoundsAction(
					lineplot.SetAxisBoundsPayload{
						Key:          lineplot.AxisKeyX1,
						Bounds:       spatial.Bounds{Lower: 10, Upper: 20},
						ManualBounds: manual,
					},
				)
				setMode := func(m *lineplot.TimeMode) lineplot.Action {
					return lineplot.NewSetAxisTimeModeAction(
						lineplot.SetAxisTimeModePayload{
							Key: lineplot.XAxisKeyX1, TimeMode: m,
						},
					)
				}
				Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
					setBounds,
					setMode(&absolute),
				})).To(Succeed())
				Expect(retrieve(ctx, p.Key).Axes.X1.ManualBounds).To(Equal(manual))
				Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
					setMode(&relative),
				})).To(Succeed())
				x1 := retrieve(ctx, p.Key).Axes.X1
				Expect(x1.Bounds).To(Equal(spatial.Bounds{}))
				Expect(x1.ManualBounds).To(Equal(lineplot.ManualBounds{}))
			},
		)

		It("Should reject a time mode for an unknown x axis", func(ctx SpecContext) {
			p := create(ctx, lineplot.LinePlot{})
			Expect(dispatch(ctx, p.Key, "d1", []lineplot.Action{
				lineplot.NewSetAxisTimeModeAction(lineplot.SetAxisTimeModePayload{
					Key: lineplot.XAxisKey("y1"),
				}),
			})).Error().To(MatchError(ContainSubstring(`unknown x-axis "y1"`)))
		})
	})
})
