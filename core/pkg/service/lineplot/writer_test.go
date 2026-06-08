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
	"context"
	"sync"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/text"
	"github.com/synnaxlabs/x/validate"
)

// scopedAction is a short local alias for the line plot's action envelope so
// the test files don't have to spell out the generic parameters at every use.
type scopedAction = actions.Scoped[lineplot.Key, lineplot.Action]

// actionRecorder collects ScopedAction notifications emitted by the service's
// action observer for assertions in tests.
type actionRecorder struct {
	mu      sync.Mutex
	scopeds []scopedAction
}

func (r *actionRecorder) record(_ context.Context, sa scopedAction) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scopeds = append(r.scopeds, sa)
}

func (r *actionRecorder) snapshot() []scopedAction {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]scopedAction, len(r.scopeds))
	copy(out, r.scopeds)
	return out
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
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
			Expect(plot.Key).ToNot(Equal(uuid.Nil))
		})
		It("Should populate lines from channel and range bindings supplied at creation", func(ctx SpecContext) {
			plot := lineplot.LinePlot{
				Name:     "test",
				Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5, 6}},
				Ranges:   lineplot.Ranges{X1: []string{"r1"}},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(lineKeysOf(res.Lines)).To(ConsistOf("y1---x1---r1---10---5", "y1---x1---r1---10---6"))
		})
	})

	Describe("eager line creation", func() {
		It("Should materialize a line per range when a channel is added", func(ctx SpecContext) {
			plot := lineplot.LinePlot{
				Name:     "test",
				Channels: lineplot.Channels{X1: 10},
				Ranges:   lineplot.Ranges{X1: []string{"r1", "r2"}},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
				lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
					AxisKey: lineplot.YAxisKeyY1, Channel: 5,
				}),
			})).To(Succeed())
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(lineKeysOf(res.Lines)).
				To(ConsistOf("y1---x1---r1---10---5", "y1---x1---r2---10---5"))
		})

		It("Should give materialized lines the schema default styling", func(ctx SpecContext) {
			plot := lineplot.LinePlot{
				Name:     "test",
				Channels: lineplot.Channels{X1: 10},
				Ranges:   lineplot.Ranges{X1: []string{"r1"}},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
				lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
					AxisKey: lineplot.YAxisKeyY1, Channel: 5,
				}),
			})).To(Succeed())
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(res.Lines).To(HaveLen(1))
			Expect(res.Lines[0].StrokeWidth).To(Equal(float64(2)))
			Expect(res.Lines[0].Downsample).To(Equal(uint32(1)))
			Expect(res.Lines[0].DownsampleMode).To(Equal(lineplot.DownsampleModeDecimate))
		})

		It("Should remove a channel's lines when the channel is removed", func(ctx SpecContext) {
			plot := lineplot.LinePlot{
				Name:     "test",
				Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5, 6}},
				Ranges:   lineplot.Ranges{X1: []string{"r1"}},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
				lineplot.NewRemoveChannelAction(lineplot.RemoveChannelPayload{
					AxisKey: lineplot.YAxisKeyY1, Channel: 5,
				}),
			})).To(Succeed())
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(lineKeysOf(res.Lines)).To(ConsistOf("y1---x1---r1---10---6"))
		})

		It("Should rekey a y-axis's lines when the x-channel changes", func(ctx SpecContext) {
			plot := lineplot.LinePlot{
				Name:     "test",
				Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5}},
				Ranges:   lineplot.Ranges{X1: []string{"r1"}},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
				lineplot.NewSetXChannelAction(lineplot.SetXChannelPayload{
					AxisKey: lineplot.XAxisKeyX1, Channel: 20,
				}),
			})).To(Succeed())
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(lineKeysOf(res.Lines)).To(ConsistOf("y1---x1---r1---20---5"))
		})
	})

	Describe("Update", func() {
		It("Should rename a LinePlot", func(ctx SpecContext) {
			plot := lineplot.LinePlot{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, plot.Key, "test2")).To(Succeed())
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})

	Describe("SetData", func() {
		It("Should replace the body of a LinePlot while preserving key and name", func(ctx SpecContext) {
			plot := lineplot.LinePlot{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
			body := lineplot.LinePlot{
				Title:  lineplot.Title{Level: text.LevelH4, Visible: true},
				Legend: lineplot.Legend{Visible: true},
				Lines: []lineplot.Line{{
					Key:            "l1",
					Color:          new(color.MustFromHex("#abcdef")),
					StrokeWidth:    2,
					Downsample:     1,
					DownsampleMode: lineplot.DownsampleModeDecimate,
				}},
			}
			Expect(svc.NewWriter(tx).SetData(ctx, plot.Key, body)).To(Succeed())
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
				To(Succeed())
			Expect(res.Key).To(Equal(plot.Key))
			Expect(res.Name).To(Equal("test"))
			Expect(res.Title.Level).To(Equal(text.LevelH4))
			Expect(res.Lines).To(HaveLen(1))
			Expect(res.Lines[0].Color).To(Equal(new(color.MustFromHex("#abcdef"))))
		})
	})

	Describe("Dispatch", func() {
		Describe("per-action semantics", func() {
			It("Should apply Rename", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "original"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewRenameAction(lineplot.RenamePayload{Name: "renamed"}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Name).To(Equal("renamed"))
			})

			It("Should apply SetTitle", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetTitleAction(lineplot.SetTitlePayload{
						Title: lineplot.Title{Level: text.LevelH2, Visible: true},
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Title.Level).To(Equal(text.LevelH2))
				Expect(res.Title.Visible).To(BeTrue())
			})

			It("Should apply SetLegendPosition", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetLegendPositionAction(lineplot.SetLegendPositionPayload{
						Position: spatial.StickyXY{X: 10, Y: 20},
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Legend.Position.X).To(Equal(10.0))
			})

			It("Should apply SetLegendVisible", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetLegendVisibleAction(lineplot.SetLegendVisiblePayload{
						Visible: false,
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Legend.Visible).To(BeFalse())
			})

			It("Should append a channel to a y-axis via AddChannel", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
						AxisKey: lineplot.YAxisKeyY1, Channel: 42,
					}),
					lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
						AxisKey: lineplot.YAxisKeyY1, Channel: 43,
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Channels.Y1).To(ConsistOf(channel.Key(42), channel.Key(43)))
			})

			It("Should treat a duplicate AddChannel as a no-op", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
						AxisKey: lineplot.YAxisKeyY2, Channel: 7,
					}),
					lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
						AxisKey: lineplot.YAxisKeyY2, Channel: 7,
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Channels.Y2).To(ConsistOf(channel.Key(7)))
			})

			It("Should reject AddChannel targeting an x-axis with validate.ErrValidation", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
						AxisKey: lineplot.YAxisKey("x1"), Channel: 1,
					}),
				})).Error().To(MatchError(validate.ErrValidation))
			})

			It("Should drop a channel via RemoveChannel", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
						AxisKey: lineplot.YAxisKeyY3, Channel: 1,
					}),
					lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
						AxisKey: lineplot.YAxisKeyY3, Channel: 2,
					}),
					lineplot.NewRemoveChannelAction(lineplot.RemoveChannelPayload{
						AxisKey: lineplot.YAxisKeyY3, Channel: 1,
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Channels.Y3).To(ConsistOf(channel.Key(2)))
			})

			It("Should replace a y-axis's whole channel set via SetChannels", func(ctx SpecContext) {
				plot := lineplot.LinePlot{
					Name:     "test",
					Channels: lineplot.Channels{Y1: []channel.Key{1, 2, 3}},
				}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetChannelsAction(lineplot.SetChannelsPayload{
						AxisKey: lineplot.YAxisKeyY1, Channels: []channel.Key{2, 4},
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Channels.Y1).To(Equal([]channel.Key{2, 4}))
			})

			It("Should reconcile lines when SetChannels replaces the channel set", func(ctx SpecContext) {
				plot := lineplot.LinePlot{
					Name:     "test",
					Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5}},
					Ranges:   lineplot.Ranges{X1: []string{"r1"}},
				}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetChannelsAction(lineplot.SetChannelsPayload{
						AxisKey: lineplot.YAxisKeyY1, Channels: []channel.Key{6},
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(lineKeysOf(res.Lines)).To(ConsistOf("y1---x1---r1---10---6"))
			})

			It("Should preserve styling of surviving channels under SetChannels", func(ctx SpecContext) {
				plot := lineplot.LinePlot{
					Name:     "test",
					Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5, 6}},
					Ranges:   lineplot.Ranges{X1: []string{"r1"}},
				}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				keep := lineplot.NewSetLineColorAction(lineplot.SetLineColorPayload{
					Key:   "y1---x1---r1---10---5",
					Color: new(color.MustFromHex("#ff0000")),
				})
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{keep})).
					To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetChannelsAction(lineplot.SetChannelsPayload{
						AxisKey: lineplot.YAxisKeyY1, Channels: []channel.Key{5},
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Lines).To(HaveLen(1))
				Expect(res.Lines[0].Color).To(Equal(new(color.MustFromHex("#ff0000"))))
			})

			It("Should reject SetChannels targeting an x-axis with validate.ErrValidation", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetChannelsAction(lineplot.SetChannelsPayload{
						AxisKey: lineplot.YAxisKey("x1"), Channels: []channel.Key{1},
					}),
				})).Error().To(MatchError(validate.ErrValidation))
			})

			It("Should replace the single channel bound to an x-axis via SetXChannel", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetXChannelAction(lineplot.SetXChannelPayload{
						AxisKey: lineplot.XAxisKeyX1, Channel: 99,
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Channels.X1).To(BeEquivalentTo(99))
			})

			It("Should reject SetXChannel targeting a y-axis with validate.ErrValidation", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetXChannelAction(lineplot.SetXChannelPayload{
						AxisKey: lineplot.XAxisKey("y1"), Channel: 1,
					}),
				})).Error().To(MatchError(validate.ErrValidation))
			})

			It("Should append and remove ranges on an x-axis", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				r1 := uuid.NewString()
				r2 := uuid.NewString()
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewAddRangeAction(lineplot.AddRangePayload{
						AxisKey: lineplot.XAxisKeyX1, Range: r1,
					}),
					lineplot.NewAddRangeAction(lineplot.AddRangePayload{
						AxisKey: lineplot.XAxisKeyX1, Range: r2,
					}),
					lineplot.NewRemoveRangeAction(lineplot.RemoveRangePayload{
						AxisKey: lineplot.XAxisKeyX1, Range: r1,
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Ranges.X1).To(Equal([]string{r2}))
			})

			It("Should reject range actions targeting a y-axis with validate.ErrValidation", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewAddRangeAction(lineplot.AddRangePayload{
						AxisKey: lineplot.XAxisKey("y1"), Range: uuid.NewString(),
					}),
				})).Error().To(MatchError(validate.ErrValidation))
			})

			It("Should replace an x-axis's whole range set via SetRanges", func(ctx SpecContext) {
				r1, r2, r3 := uuid.NewString(), uuid.NewString(), uuid.NewString()
				plot := lineplot.LinePlot{
					Name:   "test",
					Ranges: lineplot.Ranges{X1: []string{r1, r2}},
				}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetRangesAction(lineplot.SetRangesPayload{
						AxisKey: lineplot.XAxisKeyX1, Ranges: []string{r2, r3},
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Ranges.X1).To(Equal([]string{r2, r3}))
			})

			It("Should reconcile lines when SetRanges replaces the range set", func(ctx SpecContext) {
				plot := lineplot.LinePlot{
					Name:     "test",
					Channels: lineplot.Channels{X1: 10, Y1: []channel.Key{5}},
					Ranges:   lineplot.Ranges{X1: []string{"r1"}},
				}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetRangesAction(lineplot.SetRangesPayload{
						AxisKey: lineplot.XAxisKeyX1, Ranges: []string{"r2"},
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(lineKeysOf(res.Lines)).To(ConsistOf("y1---x1---r2---10---5"))
			})

			It("Should reject SetRanges targeting a y-axis with validate.ErrValidation", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetRangesAction(lineplot.SetRangesPayload{
						AxisKey: lineplot.XAxisKey("y1"), Ranges: []string{uuid.NewString()},
					}),
				})).Error().To(MatchError(validate.ErrValidation))
			})

			It("Should set axis fields via the fine-grained actions", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				priorDirection := plot.Axes.Y1.LabelDirection
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetAxisLabelAction(lineplot.SetAxisLabelPayload{
						Key: lineplot.AxisKeyY1, Label: "pressure",
					}),
					lineplot.NewSetAxisTickSpacingAction(lineplot.SetAxisTickSpacingPayload{
						Key: lineplot.AxisKeyY1, TickSpacing: 50.0,
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Axes.Y1.Label).To(Equal("pressure"))
				Expect(res.Axes.Y1.TickSpacing).To(Equal(50.0))
				Expect(res.Axes.Y1.LabelDirection).To(Equal(priorDirection))
			})

			It("Should reject a fine-grained axis action with an unknown key", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetAxisLabelAction(lineplot.SetAxisLabelPayload{
						Key: lineplot.AxisKey("nope"), Label: "x",
					}),
				})).Error().To(MatchError(validate.ErrValidation))
			})

			It("Should set and clear line fields via the fine-grained actions", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d0", []lineplot.Action{
					lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
						AxisKey: lineplot.YAxisKeyY1, Channel: 1,
					}),
					lineplot.NewAddRangeAction(lineplot.AddRangePayload{
						AxisKey: lineplot.XAxisKeyX1, Range: "r1",
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Lines).To(HaveLen(1))
				Expect(res.Lines[0].StrokeWidth).To(Equal(2.0))
				key := res.Lines[0].Key

				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetLineStrokeWidthAction(lineplot.SetLineStrokeWidthPayload{
						Key: key, StrokeWidth: 5.0,
					}),
					lineplot.NewSetLineColorAction(lineplot.SetLineColorPayload{
						Key: key, Color: new(color.MustFromHex("#ff0000")),
					}),
				})).To(Succeed())
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Lines[0].StrokeWidth).To(Equal(5.0))
				Expect(res.Lines[0].Color).To(Equal(new(color.MustFromHex("#ff0000"))))

				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d2", []lineplot.Action{
					lineplot.NewSetLineColorAction(lineplot.SetLineColorPayload{Key: key}),
				})).To(Succeed())
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Lines[0].Color).To(BeNil())
			})

			It("Should replace a line's full configuration via SetLine", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d0", []lineplot.Action{
					lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
						AxisKey: lineplot.YAxisKeyY1, Channel: 1,
					}),
					lineplot.NewAddRangeAction(lineplot.AddRangePayload{
						AxisKey: lineplot.XAxisKeyX1, Range: "r1",
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				key := res.Lines[0].Key
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetLineAction(lineplot.SetLinePayload{Line: lineplot.Line{
						Key:            key,
						StrokeWidth:    4,
						Downsample:     2,
						DownsampleMode: lineplot.DownsampleModeAverage,
						Color:          new(color.MustFromHex("#0000ff")),
					}}),
				})).To(Succeed())
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Lines).To(HaveLen(1))
				Expect(res.Lines[0].StrokeWidth).To(Equal(4.0))
				Expect(res.Lines[0].Color).To(Equal(new(color.MustFromHex("#0000ff"))))
			})

			It("Should insert, update, and remove rules", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
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
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetRuleAction(lineplot.SetRulePayload{Rule: rule}),
				})).To(Succeed())

				rule.Label = "max"
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d2", []lineplot.Action{
					lineplot.NewSetRuleAction(lineplot.SetRulePayload{Rule: rule}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Rules[0].Label).To(Equal("max"))

				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d3", []lineplot.Action{
					lineplot.NewRemoveRuleAction(lineplot.RemoveRulePayload{Key: "r1"}),
				})).To(Succeed())
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Rules).To(BeEmpty())
			})

			It("Should set rule fields via the fine-grained actions", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetRuleAction(lineplot.SetRulePayload{Rule: lineplot.Rule{
						Key: "r1", Label: "ceiling", Axis: lineplot.AxisKeyY1,
					}}),
				})).To(Succeed())
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d2", []lineplot.Action{
					lineplot.NewSetRuleLabelAction(lineplot.SetRuleLabelPayload{
						Key: "r1", Label: "max",
					}),
					lineplot.NewSetRulePositionAction(lineplot.SetRulePositionPayload{
						Key: "r1", Position: 42,
					}),
					lineplot.NewSetRuleAxisAction(lineplot.SetRuleAxisPayload{
						Key: "r1", Axis: lineplot.AxisKeyY2,
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Rules).To(HaveLen(1))
				Expect(res.Rules[0].Label).To(Equal("max"))
				Expect(res.Rules[0].Position).To(Equal(42.0))
				Expect(res.Rules[0].Axis).To(Equal(lineplot.AxisKeyY2))
			})
		})

		Describe("atomicity and broadcast", func() {
			It("Should apply a multi-action batch atomically", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "test"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				r := uuid.NewString()
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewSetXChannelAction(lineplot.SetXChannelPayload{
						AxisKey: lineplot.XAxisKeyX1, Channel: 1,
					}),
					lineplot.NewAddRangeAction(lineplot.AddRangePayload{
						AxisKey: lineplot.XAxisKeyX1, Range: r,
					}),
					lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
						AxisKey: lineplot.YAxisKeyY1, Channel: 10,
					}),
				})).To(Succeed())
				var res lineplot.LinePlot
				Expect(svc.NewRetrieve().Where(lineplot.MatchKeys(plot.Key)).Entry(&res).Exec(ctx, tx)).
					To(Succeed())
				Expect(res.Channels.X1).To(BeEquivalentTo(1))
				Expect(res.Ranges.X1).To(Equal([]string{r}))
				Expect(res.Channels.Y1).To(ConsistOf(channel.Key(10)))
			})

			It("Should notify subscribers with the dispatched ScopedAction on success", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "observed"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				rec := &actionRecorder{}
				DeferCleanup(svc.OnAction(rec.record))
				actions := []lineplot.Action{
					lineplot.NewRenameAction(lineplot.RenamePayload{Name: "broadcast"}),
					lineplot.NewSetXChannelAction(lineplot.SetXChannelPayload{
						AxisKey: lineplot.XAxisKeyX1, Channel: 5,
					}),
				}
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "client-xyz", actions)).
					To(Succeed())
				seen := rec.snapshot()
				Expect(seen).To(HaveLen(1))
				Expect(seen[0].Key).To(Equal(plot.Key))
				Expect(seen[0].DispatchKey).To(Equal("client-xyz"))
				Expect(seen[0].Actions).To(HaveLen(2))
				Expect(seen[0].Actions[0].Type).To(Equal(lineplot.ActionTypeRename))
				Expect(seen[0].Actions[1].Type).To(Equal(lineplot.ActionTypeSetXChannel))
			})

			It("Should assign strictly monotonic Seq across successive dispatches", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "seq"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				rec := &actionRecorder{}
				DeferCleanup(svc.OnAction(rec.record))
				for _, name := range []string{"a", "b", "c"} {
					Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d", []lineplot.Action{
						lineplot.NewRenameAction(lineplot.RenamePayload{Name: name}),
					})).To(Succeed())
				}
				seen := rec.snapshot()
				Expect(seen).To(HaveLen(3))
				Expect(seen[1].Seq).To(BeNumerically(">", seen[0].Seq))
				Expect(seen[2].Seq).To(BeNumerically(">", seen[1].Seq))
			})

			It("Should not notify subscribers when Reduce rejects the action", func(ctx SpecContext) {
				plot := lineplot.LinePlot{Name: "rejected"}
				Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &plot)).To(Succeed())
				rec := &actionRecorder{}
				DeferCleanup(svc.OnAction(rec.record))
				Expect(svc.NewWriter(tx).Dispatch(ctx, plot.Key, "d1", []lineplot.Action{
					lineplot.NewAddChannelAction(lineplot.AddChannelPayload{
						AxisKey: lineplot.YAxisKey("x1"), Channel: 1,
					}),
				})).Error().To(MatchError(validate.ErrValidation))
				Expect(rec.snapshot()).To(BeEmpty())
			})
		})
	})
})
