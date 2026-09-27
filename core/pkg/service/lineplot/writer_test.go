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
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/text"
	"github.com/synnaxlabs/x/union"
	"github.com/synnaxlabs/x/validate"
)

var (
	keyR1 = uuid.MustParse("5f6c1d3e-2a4b-4c8d-9e0f-1a2b3c4d5e61")
	keyR2 = uuid.MustParse("5f6c1d3e-2a4b-4c8d-9e0f-1a2b3c4d5e62")
)

func persisted(key uuid.UUID) lineplot.Range {
	return lineplot.Range{
		Variant: lineplot.PersistedRange{BaseRange: lineplot.BaseRange{Key: key}},
	}
}

func persistedAxis(keys ...uuid.UUID) lineplot.XAxisRanges {
	ranges := make([]lineplot.Range, len(keys))
	for i, k := range keys {
		ranges[i] = persisted(k)
	}
	return lineplot.XAxisRanges{Ranges: ranges}
}

func rangeKeysOf(axis lineplot.XAxisRanges) []uuid.UUID {
	keys := make([]uuid.UUID, len(axis.Ranges))
	for i, r := range axis.Ranges {
		switch v := r.Variant.(type) {
		case lineplot.PersistedRange:
			keys[i] = v.Key
		case lineplot.StaticRange:
			keys[i] = v.Key
		}
	}
	return keys
}

func staticRange(key uuid.UUID, start, end telem.TimeStamp) lineplot.Range {
	return lineplot.Range{Variant: lineplot.StaticRange{
		BaseRange: lineplot.BaseRange{Key: key},
		Start:     start,
		End:       end,
	}}
}

func retrievePlot(ctx SpecContext, key lineplot.Key) lineplot.LinePlot {
	GinkgoHelper()
	var res lineplot.LinePlot
	Expect(
		svc.NewRetrieve().Where(lineplot.MatchKeys(key)).Entry(&res).Exec(ctx, tx),
	).To(Succeed())
	return res
}

// x1Line returns the key of the y1 line plotted against x1 over the range part rng.
func x1Line(rng string, xChannel, yChannel channel.Key) string {
	return "y1---x1---" + rng + "---" + xChannel.String() + "---" + yChannel.String()
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
			plot := lineplot.LinePlot{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, proj.Key, &plot)).To(Succeed())
			Expect(plot.Key).ToNot(Equal(uuid.Nil()))
		})
		It(
			"Should return a validation error when the name is empty",
			func(ctx SpecContext) {
				plot := lineplot.LinePlot{}
				Expect(svc.NewWriter(tx).Create(ctx, proj.Key, &plot)).
					To(MatchError(ContainSubstring("name: required")))
			},
		)
		It(
			"Should populate lines from channel and range bindings supplied at creation",
			func(ctx SpecContext) {
				plot := lineplot.LinePlot{
					Name:     "test",
					Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5, 6}},
					Ranges:   lineplot.Ranges{X1: persistedAxis(keyR1)},
				}
				Expect(svc.NewWriter(tx).Create(ctx, proj.Key, &plot)).To(Succeed())
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
				).To(ConsistOf(x1Line(keyR1.String(), 10, 5), x1Line(keyR1.String(), 10, 6)))
			},
		)
	})

	Describe("CreateMany", func() {
		It("Should create multiple line plots", func(ctx SpecContext) {
			plots := []lineplot.LinePlot{
				{Name: "plot-1"},
				{Name: "plot-2"},
			}
			Expect(svc.NewWriter(tx).CreateMany(ctx, proj.Key, &plots)).To(Succeed())

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
			"Should materialize a line per range when a channel is added",
			func(ctx SpecContext) {
				plot := lineplot.LinePlot{
					Name:     "test",
					Channels: lineplot.Channels{X1: 10},
					Ranges:   lineplot.Ranges{X1: persistedAxis(keyR1, keyR2)},
				}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
							AxisKey: lineplot.YAxisKeyY1, Channel: 5,
						}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, tx),
				).
					To(Succeed())
				Expect(lineKeysOf(res.Lines)).
					To(ConsistOf(x1Line(keyR1.String(), 10, 5), x1Line(keyR2.String(), 10, 5)))
			},
		)

		It(
			"Should give materialized lines the schema default styling",
			func(ctx SpecContext) {
				plot := lineplot.LinePlot{
					Name:     "test",
					Channels: lineplot.Channels{X1: 10},
					Ranges:   lineplot.Ranges{X1: persistedAxis(keyR1)},
				}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
							AxisKey: lineplot.YAxisKeyY1, Channel: 5,
						}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, tx),
				).
					To(Succeed())
				Expect(res.Lines).To(HaveLen(1))
				Expect(res.Lines[0].StrokeWidth).To(Equal(float64(2)))
				Expect(res.Lines[0].Downsample).To(Equal(uint32(1)))
				Expect(
					res.Lines[0].DownsampleMode,
				).To(Equal(lineplot.DownsampleModeDecimate))
			},
		)

		It(
			"Should remove a channel's lines when the channel is removed",
			func(ctx SpecContext) {
				plot := lineplot.LinePlot{
					Name:     "test",
					Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5, 6}},
					Ranges:   lineplot.Ranges{X1: persistedAxis(keyR1)},
				}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewRemoveChannelAction(lineplot.RemoveChannelPayload{
							AxisKey: lineplot.YAxisKeyY1, Channel: 5,
						}),
					}),
				).To(Succeed())
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
				).To(ConsistOf(x1Line(keyR1.String(), 10, 6)))
			},
		)

		It(
			"Should rekey a y-axis's lines when the x-channel changes",
			func(ctx SpecContext) {
				plot := lineplot.LinePlot{
					Name:     "test",
					Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5}},
					Ranges:   lineplot.Ranges{X1: persistedAxis(keyR1)},
				}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewSetXChannelAction(lineplot.SetXChannelPayload{
							AxisKey: lineplot.XAxisKeyX1, Channel: 20,
						}),
					}),
				).To(Succeed())
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
				).To(ConsistOf(x1Line(keyR1.String(), 20, 5)))
			},
		)
	})

	Describe("Dispatch", func() {
		Describe("per-action semantics", func() {
			It("Should apply Rename", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "original"}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewRenameAction(
							lineplot.RenamePayload{Name: "renamed"},
						),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, tx),
				).
					To(Succeed())
				Expect(res.Name).To(Equal("renamed"))
			})

			It("Should apply SetTitleLevel", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
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
						Exec(ctx, tx),
				).
					To(Succeed())
				Expect(res.Title.Level).To(Equal(text.LevelH2))
			})

			It("Should apply SetTitleVisible", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
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
						Exec(ctx, tx),
				).
					To(Succeed())
				Expect(res.Title.Visible).To(BeTrue())
			})

			It("Should apply SetLegendPosition", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewSetLegendPositionAction(
							lineplot.SetLegendPositionPayload{
								Position: spatial.StickyXY{X: 10, Y: 20},
							},
						),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, tx),
				).
					To(Succeed())
				Expect(res.Legend.Position.X).To(Equal(10.0))
			})

			It("Should apply SetLegendHidden", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
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
						Exec(ctx, tx),
				).
					To(Succeed())
				Expect(res.Legend.Hidden).To(BeTrue())
			})

			It(
				"Should append a channel to a y-axis via AddChannel",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
								AxisKey: lineplot.YAxisKeyY1, Channel: 42,
							}),
							lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
								AxisKey: lineplot.YAxisKeyY1, Channel: 43,
							}),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, tx),
					).
						To(Succeed())
					Expect(
						res.Channels.Y1,
					).To(ConsistOf(channel.Key(42), channel.Key(43)))
				},
			)

			It("Should treat a duplicate AddChannel as a no-op", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
							AxisKey: lineplot.YAxisKeyY2, Channel: 7,
						}),
						lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
							AxisKey: lineplot.YAxisKeyY2, Channel: 7,
						}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, tx),
				).
					To(Succeed())
				Expect(res.Channels.Y2).To(ConsistOf(channel.Key(7)))
			})

			It(
				"Should reject AddChannel targeting an x-axis with validate.ErrValidation",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
								AxisKey: lineplot.YAxisKey("x1"), Channel: 1,
							}),
						}),
					).Error().
						To(MatchError(validate.ErrValidation))
				},
			)

			It("Should drop a channel via RemoveChannel", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
							AxisKey: lineplot.YAxisKeyY3, Channel: 1,
						}),
						lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
							AxisKey: lineplot.YAxisKeyY3, Channel: 2,
						}),
						lineplot.NewRemoveChannelAction(lineplot.RemoveChannelPayload{
							AxisKey: lineplot.YAxisKeyY3, Channel: 1,
						}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, tx),
				).
					To(Succeed())
				Expect(res.Channels.Y3).To(ConsistOf(channel.Key(2)))
			})

			It(
				"Should replace a y-axis's whole channel set via SetChannels",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{
						Name:     "test",
						Channels: lineplot.Channels{Y1: []channel.Key{1, 2, 3}},
					}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetChannelsAction(
								lineplot.SetChannelsPayload{
									AxisKey:  lineplot.YAxisKeyY1,
									Channels: []channel.Key{2, 4},
								},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, tx),
					).
						To(Succeed())
					Expect(res.Channels.Y1).To(Equal([]channel.Key{2, 4}))
				},
			)

			It(
				"Should reconcile lines when SetChannels replaces the channel set",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{
						Name:     "test",
						Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5}},
						Ranges:   lineplot.Ranges{X1: persistedAxis(keyR1)},
					}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetChannelsAction(
								lineplot.SetChannelsPayload{
									AxisKey:  lineplot.YAxisKeyY1,
									Channels: []channel.Key{6},
								},
							),
						}),
					).To(Succeed())
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
					).To(ConsistOf(x1Line(keyR1.String(), 10, 6)))
				},
			)

			It(
				"Should preserve styling of surviving channels under SetChannels",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{
						Name:     "test",
						Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5, 6}},
						Ranges:   lineplot.Ranges{X1: persistedAxis(keyR1)},
					}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					keep := lineplot.NewSetLineColorAction(lineplot.SetLineColorPayload{
						Key:   x1Line(keyR1.String(), 10, 5),
						Color: new(color.MustFromHex("#ff0000")),
					})
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{keep}),
					).
						To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetChannelsAction(
								lineplot.SetChannelsPayload{
									AxisKey:  lineplot.YAxisKeyY1,
									Channels: []channel.Key{5},
								},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, tx),
					).
						To(Succeed())
					Expect(res.Lines).To(HaveLen(1))
					Expect(
						res.Lines[0].Color,
					).To(Equal(new(color.MustFromHex("#ff0000"))))
				},
			)

			It(
				"Should reject SetChannels targeting an x-axis with validate.ErrValidation",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetChannelsAction(
								lineplot.SetChannelsPayload{
									AxisKey:  lineplot.YAxisKey("x1"),
									Channels: []channel.Key{1},
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
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetXChannelAction(
								lineplot.SetXChannelPayload{
									AxisKey: lineplot.XAxisKeyX1, Channel: 99,
								},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, tx),
					).
						To(Succeed())
					Expect(res.Channels.X1).To(BeEquivalentTo(99))
				},
			)

			It(
				"Should reject SetXChannel targeting a y-axis with validate.ErrValidation",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetXChannelAction(
								lineplot.SetXChannelPayload{
									AxisKey: lineplot.XAxisKey("y1"), Channel: 1,
								},
							),
						}),
					).Error().
						To(MatchError(validate.ErrValidation))
				},
			)

			It("Should append and remove ranges on an x-axis", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				r1, r2 := uuid.New(), uuid.New()
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewAddRangeAction(lineplot.AddRangePayload{
							AxisKey: lineplot.XAxisKeyX1, Range: persisted(r1),
						}),
						lineplot.NewAddRangeAction(lineplot.AddRangePayload{
							AxisKey: lineplot.XAxisKeyX1, Range: persisted(r2),
						}),
						lineplot.NewRemoveRangeAction(lineplot.RemoveRangePayload{
							AxisKey: lineplot.XAxisKeyX1, Key: r1,
						}),
					}),
				).To(Succeed())
				Expect(rangeKeysOf(retrievePlot(ctx, plot.Key).Ranges.X1)).
					To(Equal([]uuid.UUID{r2}))
			})

			It(
				"Should not add a range whose key the axis already holds",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{
						Name:   "test",
						Ranges: lineplot.Ranges{X1: persistedAxis(keyR1)},
					}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewAddRangeAction(lineplot.AddRangePayload{
								AxisKey: lineplot.XAxisKeyX1,
								Range:   staticRange(keyR1, 0, 10),
							}),
						}),
					).To(Succeed())
					Expect(retrievePlot(ctx, plot.Key).Ranges.X1).
						To(Equal(persistedAxis(keyR1)))
				},
			)

			It("Should reject an AddRange with no variant", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewAddRangeAction(lineplot.AddRangePayload{
							AxisKey: lineplot.XAxisKeyX1,
						}),
					}),
				).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("range has no variant")),
				))
			})

			DescribeTable(
				"Should reject range actions targeting a y-axis",
				func(ctx SpecContext, action lineplot.Action) {
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{action}),
					).To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring(`unknown x-axis "y1"`)),
					))
				},
				Entry("SetRolling", lineplot.NewSetRollingAction(
					lineplot.SetRollingPayload{
						AxisKey: lineplot.XAxisKey("y1"),
						Span:    new(telem.Minute),
					},
				)),
				Entry("AddRange", lineplot.NewAddRangeAction(lineplot.AddRangePayload{
					AxisKey: lineplot.XAxisKey("y1"), Range: persisted(keyR1),
				})),
				Entry("RemoveRange", lineplot.NewRemoveRangeAction(
					lineplot.RemoveRangePayload{
						AxisKey: lineplot.XAxisKey("y1"), Key: keyR1,
					},
				)),
				Entry("SetRange", lineplot.NewSetRangeAction(lineplot.SetRangePayload{
					AxisKey: lineplot.XAxisKey("y1"), Range: persisted(keyR1),
				})),
				Entry("SetRanges", lineplot.NewSetRangesAction(
					lineplot.SetRangesPayload{
						AxisKey: lineplot.XAxisKey("y1"),
						Ranges:  []lineplot.Range{persisted(keyR1)},
					},
				)),
			)

			It(
				"Should replace an x-axis's whole range set via SetRanges",
				func(ctx SpecContext) {
					r1, r2, r3 := uuid.New(), uuid.New(), uuid.New()
					plot := lineplot.LinePlot{
						Name:   "test",
						Ranges: lineplot.Ranges{X1: persistedAxis(r1, r2)},
					}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetRangesAction(lineplot.SetRangesPayload{
								AxisKey: lineplot.XAxisKeyX1,
								Ranges:  []lineplot.Range{persisted(r2), persisted(r3)},
							}),
						}),
					).To(Succeed())
					Expect(rangeKeysOf(retrievePlot(ctx, plot.Key).Ranges.X1)).
						To(Equal([]uuid.UUID{r2, r3}))
				},
			)

			It(
				"Should reconcile lines when SetRanges replaces the range set",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{
						Name:     "test",
						Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5}},
						Ranges:   lineplot.Ranges{X1: persistedAxis(keyR1)},
					}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetRangesAction(lineplot.SetRangesPayload{
								AxisKey: lineplot.XAxisKeyX1,
								Ranges:  []lineplot.Range{persisted(keyR2)},
							}),
						}),
					).To(Succeed())
					Expect(lineKeysOf(retrievePlot(ctx, plot.Key).Lines)).
						To(ConsistOf(x1Line(keyR2.String(), 10, 5)))
				},
			)

			It(
				"Should keep the styling of ranges that survive SetRanges",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{
						Name:     "test",
						Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5}},
						Ranges:   lineplot.Ranges{X1: persistedAxis(keyR1, keyR2)},
					}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					red := new(color.MustFromHex("#ff0000"))
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetLineColorAction(lineplot.SetLineColorPayload{
								Key: x1Line(keyR1.String(), 10, 5), Color: red,
							}),
							lineplot.NewSetRangesAction(lineplot.SetRangesPayload{
								AxisKey: lineplot.XAxisKeyX1,
								Ranges:  []lineplot.Range{persisted(keyR1)},
							}),
						}),
					).To(Succeed())
					res := retrievePlot(ctx, plot.Key)
					Expect(res.Lines).To(HaveLen(1))
					Expect(res.Lines[0].Key).To(Equal(x1Line(keyR1.String(), 10, 5)))
					Expect(res.Lines[0].Color).To(Equal(red))
				},
			)

			It(
				"Should reject SetRanges with two ranges sharing a key",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetRangesAction(lineplot.SetRangesPayload{
								AxisKey: lineplot.XAxisKeyX1,
								Ranges: []lineplot.Range{
									persisted(keyR1),
									staticRange(keyR1, 0, 10),
								},
							}),
						}),
					).To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring(
							`duplicate range `+keyR1.String()+` on x-axis "x1"`,
						)),
					))
				},
			)

			It(
				"Should replace a range in place via SetRange, keeping its line styling",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{
						Name:     "test",
						Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5}},
						Ranges: lineplot.Ranges{X1: lineplot.XAxisRanges{
							Ranges: []lineplot.Range{
								staticRange(keyR1, 0, 10),
								persisted(keyR2),
							},
						}},
					}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					red := new(color.MustFromHex("#ff0000"))
					edited := staticRange(keyR1, 5, 20)
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetLineColorAction(lineplot.SetLineColorPayload{
								Key: x1Line(keyR1.String(), 10, 5), Color: red,
							}),
							lineplot.NewSetRangeAction(lineplot.SetRangePayload{
								AxisKey: lineplot.XAxisKeyX1, Range: edited,
							}),
						}),
					).To(Succeed())
					res := retrievePlot(ctx, plot.Key)
					Expect(res.Ranges.X1.Ranges).
						To(Equal([]lineplot.Range{edited, persisted(keyR2)}))
					Expect(lineKeysOf(res.Lines)).To(Equal([]string{
						x1Line(keyR1.String(), 10, 5),
						x1Line(keyR2.String(), 10, 5),
					}))
					Expect(res.Lines[0].Color).To(Equal(red))
				},
			)

			It(
				"Should reject a SetRange naming no range on the axis",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetRangeAction(lineplot.SetRangePayload{
								AxisKey: lineplot.XAxisKeyX1, Range: persisted(keyR1),
							}),
						}),
					).To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring(
							`no range `+keyR1.String()+` on x-axis "x1"`,
						)),
					))
				},
			)

			It(
				"Should set, replace, and clear the rolling window via SetRolling",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{
						Name:     "test",
						Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5}},
					}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					setRolling := func(span *telem.TimeSpan) {
						GinkgoHelper()
						Expect(
							svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
								lineplot.NewSetRollingAction(lineplot.SetRollingPayload{
									AxisKey: lineplot.XAxisKeyX1, Span: span,
								}),
							}),
						).To(Succeed())
					}
					rollingLine := x1Line("rolling", 10, 5)
					setRolling(new(telem.Minute))
					res := retrievePlot(ctx, plot.Key)
					Expect(res.Ranges.X1.Rolling).To(Equal(new(telem.Minute)))
					Expect(lineKeysOf(res.Lines)).To(Equal([]string{rollingLine}))
					red := new(color.MustFromHex("#ff0000"))
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetLineColorAction(lineplot.SetLineColorPayload{
								Key: rollingLine, Color: red,
							}),
						}),
					).To(Succeed())
					setRolling(new(5 * telem.Minute))
					res = retrievePlot(ctx, plot.Key)
					Expect(res.Ranges.X1.Rolling).To(Equal(new(5 * telem.Minute)))
					Expect(lineKeysOf(res.Lines)).To(Equal([]string{rollingLine}))
					Expect(res.Lines[0].Color).To(Equal(red))
					setRolling(nil)
					res = retrievePlot(ctx, plot.Key)
					Expect(res.Ranges.X1.Rolling).To(BeNil())
					Expect(res.Lines).To(BeEmpty())
				},
			)

			It(
				"Should plot the rolling window ahead of the axis's ranges",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{
						Name:     "test",
						Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5}},
						Ranges: lineplot.Ranges{X1: lineplot.XAxisRanges{
							Rolling: new(telem.Minute),
							Ranges:  []lineplot.Range{persisted(keyR1)},
						}},
					}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(lineKeysOf(retrievePlot(ctx, plot.Key).Lines)).To(Equal(
						[]string{
							x1Line("rolling", 10, 5),
							x1Line(keyR1.String(), 10, 5),
						},
					))
				},
			)

			DescribeTable(
				"Should reject a SetRolling span that is not positive",
				func(ctx SpecContext, span telem.TimeSpan) {
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetRollingAction(lineplot.SetRollingPayload{
								AxisKey: lineplot.XAxisKeyX1, Span: &span,
							}),
						}),
					).To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring("rolling span must be positive")),
					))
				},
				Entry("zero", telem.TimeSpan(0)),
				Entry("negative", -telem.Minute),
			)

			It(
				"Should reject a SetRolling action with no payload",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							{Type: lineplot.ActionTypeSetRolling},
						}),
					).To(MatchError(union.ErrMissingPayload))
				},
			)

			It(
				"Should set axis fields via the fine-grained actions",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					priorDirection := plot.Axes.Y1.LabelDirection
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
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
							Exec(ctx, tx),
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
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
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
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d0", []lineplot.Action{
							lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
								AxisKey: lineplot.YAxisKeyY1, Channel: 1,
							}),
							lineplot.NewAddRangeAction(lineplot.AddRangePayload{
								AxisKey: lineplot.XAxisKeyX1, Range: persisted(keyR1),
							}),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, tx),
					).
						To(Succeed())
					Expect(res.Lines).To(HaveLen(1))
					Expect(res.Lines[0].StrokeWidth).To(Equal(2.0))
					key := res.Lines[0].Key

					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetLineStrokeWidthAction(
								lineplot.SetLineStrokeWidthPayload{
									Key: key, StrokeWidth: 5.0,
								},
							),
							lineplot.NewSetLineColorAction(
								lineplot.SetLineColorPayload{
									Key:   key,
									Color: new(color.MustFromHex("#ff0000")),
								},
							),
						}),
					).To(Succeed())
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, tx),
					).
						To(Succeed())
					Expect(res.Lines[0].StrokeWidth).To(Equal(5.0))
					Expect(
						res.Lines[0].Color,
					).To(Equal(new(color.MustFromHex("#ff0000"))))

					Expect(
						svc.Dispatch(ctx, plot.Key, "d2", []lineplot.Action{
							lineplot.NewSetLineColorAction(
								lineplot.SetLineColorPayload{Key: key},
							),
						}),
					).To(Succeed())
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, tx),
					).
						To(Succeed())
					Expect(res.Lines[0].Color).To(BeNil())
				},
			)

			It(
				"Should replace a line's full configuration via SetLine",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d0", []lineplot.Action{
							lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
								AxisKey: lineplot.YAxisKeyY1, Channel: 1,
							}),
							lineplot.NewAddRangeAction(lineplot.AddRangePayload{
								AxisKey: lineplot.XAxisKeyX1, Range: persisted(keyR1),
							}),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, tx),
					).
						To(Succeed())
					key := res.Lines[0].Key
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetLineAction(
								lineplot.SetLinePayload{Line: lineplot.Line{
									Key:            key,
									StrokeWidth:    4,
									Downsample:     2,
									DownsampleMode: lineplot.DownsampleModeAverage,
									Color: new(
										color.MustFromHex("#0000ff"),
									),
								}},
							),
						}),
					).To(Succeed())
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, tx),
					).
						To(Succeed())
					Expect(res.Lines).To(HaveLen(1))
					Expect(res.Lines[0].StrokeWidth).To(Equal(4.0))
					Expect(
						res.Lines[0].Color,
					).To(Equal(new(color.MustFromHex("#0000ff"))))
				},
			)

			It("Should insert, update, and remove rules", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				rule := lineplot.Rule{
					Key:       "r1",
					Label:     "ceiling",
					Color:     new(color.MustFromHex("#888888")),
					Axis:      lineplot.AxisKeyY1,
					LineWidth: 1,
					LineDash:  4,
					Units:     "psi",
					Position:  100,
				}
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewSetRuleAction(lineplot.SetRulePayload{Rule: rule}),
					}),
				).To(Succeed())

				rule.Label = "max"
				Expect(
					svc.Dispatch(ctx, plot.Key, "d2", []lineplot.Action{
						lineplot.NewSetRuleAction(lineplot.SetRulePayload{Rule: rule}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, tx),
				).
					To(Succeed())
				Expect(res.Rules[0].Label).To(Equal("max"))

				Expect(
					svc.Dispatch(ctx, plot.Key, "d3", []lineplot.Action{
						lineplot.NewRemoveRuleAction(
							lineplot.RemoveRulePayload{Key: "r1"},
						),
					}),
				).To(Succeed())
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, tx),
				).
					To(Succeed())
				Expect(res.Rules).To(BeEmpty())
			})

			It(
				"Should set rule fields via the fine-grained actions",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Name: "test"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewSetRuleAction(
								lineplot.SetRulePayload{Rule: lineplot.Rule{
									Key:   "r1",
									Label: "ceiling",
									Axis:  lineplot.AxisKeyY1,
								}},
							),
						}),
					).To(Succeed())
					Expect(
						svc.Dispatch(ctx, plot.Key, "d2", []lineplot.Action{
							lineplot.NewSetRuleLabelAction(
								lineplot.SetRuleLabelPayload{
									Key: "r1", Label: "max",
								},
							),
							lineplot.NewSetRulePositionAction(
								lineplot.SetRulePositionPayload{
									Key: "r1", Position: 42,
								},
							),
							lineplot.NewSetRuleAxisAction(
								lineplot.SetRuleAxisPayload{
									Key: "r1", Axis: lineplot.AxisKeyY2,
								},
							),
						}),
					).To(Succeed())
					var res lineplot.LinePlot
					Expect(
						svc.NewRetrieve().
							Where(lineplot.MatchKeys(plot.Key)).
							Entry(&res).
							Exec(ctx, tx),
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
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &plot)).To(Succeed())
				r := uuid.New()
				Expect(
					svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
						lineplot.NewSetXChannelAction(lineplot.SetXChannelPayload{
							AxisKey: lineplot.XAxisKeyX1, Channel: 1,
						}),
						lineplot.NewAddRangeAction(lineplot.AddRangePayload{
							AxisKey: lineplot.XAxisKeyX1, Range: persisted(r),
						}),
						lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
							AxisKey: lineplot.YAxisKeyY1, Channel: 10,
						}),
					}),
				).To(Succeed())
				var res lineplot.LinePlot
				Expect(
					svc.NewRetrieve().
						Where(lineplot.MatchKeys(plot.Key)).
						Entry(&res).
						Exec(ctx, tx),
				).
					To(Succeed())
				Expect(res.Channels.X1).To(BeEquivalentTo(1))
				Expect(rangeKeysOf(res.Ranges.X1)).To(Equal([]uuid.UUID{r}))
				Expect(res.Channels.Y1).To(ConsistOf(channel.Key(10)))
			})

			It(
				"Should notify subscribers with the dispatched ScopedAction on success",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Name: "observed"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					rec := &Recorder[lineplot.Key, lineplot.Action]{}
					DeferCleanup(svc.OnAction(rec.Record))
					actions := []lineplot.Action{
						lineplot.NewRenameAction(
							lineplot.RenamePayload{Name: "broadcast"},
						),
						lineplot.NewSetXChannelAction(lineplot.SetXChannelPayload{
							AxisKey: lineplot.XAxisKeyX1, Channel: 5,
						}),
					}
					Expect(
						svc.Dispatch(ctx, plot.Key, "client-xyz", actions),
					).
						To(Succeed())
					seen := rec.Snapshot()
					Expect(seen).To(HaveLen(1))
					Expect(seen[0].Key).To(Equal(plot.Key))
					Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
					Expect(seen[0].Actions).To(HaveLen(2))
					Expect(seen[0].Actions[0].Type).To(Equal(lineplot.ActionTypeRename))
					Expect(
						seen[0].Actions[1].Type,
					).To(Equal(lineplot.ActionTypeSetXChannel))
				},
			)

			It(
				"Should assign strictly monotonic Seq across successive dispatches",
				func(ctx SpecContext) {
					plot := lineplot.LinePlot{Name: "seq"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					rec := &Recorder[lineplot.Key, lineplot.Action]{}
					DeferCleanup(svc.OnAction(rec.Record))
					for _, name := range []string{"a", "b", "c"} {
						Expect(
							svc.Dispatch(ctx, plot.Key, "d", []lineplot.Action{
								lineplot.NewRenameAction(
									lineplot.RenamePayload{Name: name},
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
					plot := lineplot.LinePlot{Name: "rejected"}
					Expect(
						svc.NewWriter(nil).Create(ctx, proj.Key, &plot),
					).To(Succeed())
					rec := &Recorder[lineplot.Key, lineplot.Action]{}
					DeferCleanup(svc.OnAction(rec.Record))
					Expect(
						svc.Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
							lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
								AxisKey: lineplot.YAxisKey("x1"), Channel: 1,
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
