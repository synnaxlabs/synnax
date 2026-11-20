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
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xcolor "github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Channel references", func() {
	retrieve := func(ctx SpecContext, key lineplot.Key) lineplot.LinePlot {
		GinkgoHelper()
		var res lineplot.LinePlot
		Expect(svc.NewRetrieve().
			Where(lineplot.MatchKeys(key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}

	It(
		"Should attach the line plot as written to the delete that detaches a channel",
		func(ctx SpecContext) {
			keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 2))
			gone, kept := keys[0], keys[1]
			plot := defaults.Fill(lineplot.LinePlot{
				Project:  &proj.Key,
				Name:     "edited",
				Channels: lineplot.Channels{Y1: []channel.Key{gone, kept}},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &plot)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, plot.Key)
				})).To(Succeed())
			})
			rec := &Recorder[lineplot.Key, lineplot.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, gone, false)
			})).To(Succeed())
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Key).To(Equal(plot.Key))
			Expect(seen[0].Actions).To(HaveLen(1))
			Expect(seen[0].Actions[0].Create).ToNot(BeNil())
			Expect(seen[0].Actions[0].Create.LinePlot.Channels.Y1).
				To(Equal([]channel.Key{kept}))
		},
	)

	It("Should refuse a channel the Core does not hold", func(ctx SpecContext) {
		plot := defaults.Fill(lineplot.LinePlot{
			Project:  &proj.Key,
			Name:     "missing",
			Channels: lineplot.Channels{Y1: []channel.Key{ch5, 404}},
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &plot)
		})).
			To(MatchError(ContainSubstring("channels: channel 404 does not exist")))
	})

	It(
		"Should remove a deleted channel and keep the other channels and styles",
		func(ctx SpecContext) {
			keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 4))
			x, gone, kept, y2 := keys[0], keys[1], keys[2], keys[3]
			plot := defaults.Fill(lineplot.LinePlot{
				Project: &proj.Key,
				Name:    "detach",
				Channels: lineplot.Channels{
					X1: x,
					Y1: []channel.Key{gone, kept},
					Y2: []channel.Key{y2},
				},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &plot)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, plot.Key)
				})).To(Succeed())
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := svc.NewWriter(tx).Apply(
					ctx,
					plot.Key,
					lineplot.NewSetLineStrokeWidthAction(
						lineplot.SetLineStrokeWidthPayload{
							Key:         plot.Lines[1].Key,
							StrokeWidth: 7,
						},
					),
				)
				return err
			})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).DeleteMany(
					ctx,
					[]channel.Key{gone, x},
					false,
				)
			})).To(Succeed())
			res := retrieve(ctx, plot.Key)
			Expect(res.Channels).To(Equal(lineplot.Channels{
				Y1: []channel.Key{kept},
				Y2: []channel.Key{y2},
			}))
			Expect(lineKeysOf(res.Lines)).To(Equal([]string{
				lk(plot.Key, 0, kept),
				v8.LineKey(plot.Key, lineplot.YAxisKeyY2, y2, 0),
			}))
			Expect(res.Lines[0].StrokeWidth).To(Equal(7.0))
		},
	)

	It(
		"Should drop channels the Core does not hold from an imported plot",
		func(ctx SpecContext) {
			keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 2))
			kept, gone := keys[0], keys[1]
			plot := defaults.Fill(lineplot.LinePlot{
				Project:  &proj.Key,
				Name:     "export",
				Channels: lineplot.Channels{Y1: []channel.Key{kept, gone}},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &plot)
			})).To(Succeed())
			env := MustSucceed(svc.Export(ctx, db, lineplot.ResourceID(plot.Key)))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, gone, false)
			})).To(Succeed())
			var id resource.ID
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				id, err = imexSvc.Import(ctx, tx, WireRoundTrip(env),
					imex.ImportOptions{Parent: proj.ResourceID()})
				return err
			})).To(Succeed())
			res := retrieve(ctx, MustSucceed(uuid.Parse(id.Key)))
			Expect(res.Channels.Y1).To(Equal([]channel.Key{kept}))
			Expect(lineKeysOf(res.Lines)).To(Equal([]string{lk(res.Key, 0, kept)}))
		},
	)
})

var _ = Describe("Color references", func() {
	It(
		"Should inline a deleted named color into the lines and rules that use it",
		func(ctx SpecContext) {
			gone := color.Named{
				Name:  "Fault",
				Light: MustSucceed(xcolor.Parse("#ff0000")),
			}
			kept := color.Named{
				Name:  "Nominal",
				Light: MustSucceed(xcolor.Parse("#00ff00")),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := colorSvc.NewWriter(tx)
				if err := w.Create(ctx, &gone); err != nil {
					return err
				}
				return w.Create(ctx, &kept)
			})).To(Succeed())
			named := func(n color.Named) *color.Ref {
				return &color.Ref{Variant: color.NamedRef{Key: n.Key}}
			}
			plot := defaults.Fill(lineplot.LinePlot{
				Project:  &proj.Key,
				Name:     "colored",
				Channels: lineplot.Channels{Y1: []channel.Key{ch5, ch6}},
				Lines: []lineplot.Line{
					{YAxis: lineplot.YAxisKeyY1, YChannel: ch5, Color: named(gone)},
					{YAxis: lineplot.YAxisKeyY1, YChannel: ch6, Color: named(kept)},
				},
				Rules: []lineplot.Rule{
					{Key: "r", Axis: lineplot.AxisKeyY1, Color: named(gone)},
				},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &plot)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					if err := svc.NewWriter(tx).Delete(ctx, plot.Key); err != nil {
						return err
					}
					return colorSvc.NewWriter(tx).Delete(ctx, kept.Key)
				})).To(Succeed())
			})
			rec := &Recorder[lineplot.Key, lineplot.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return colorSvc.NewWriter(tx).Delete(ctx, gone.Key)
			})).To(Succeed())
			var res lineplot.LinePlot
			Expect(svc.NewRetrieve().
				Where(lineplot.MatchKeys(plot.Key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(res.Lines).To(HaveLen(2))
			Expect(res.Lines[0].Color).To(Equal(fixed("#ff0000")))
			Expect(res.Lines[1].Color).To(Equal(named(kept)))
			Expect(res.Rules[0].Color).To(Equal(fixed("#ff0000")))
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Key).To(Equal(plot.Key))
			Expect(seen[0].Actions).To(HaveLen(1))
			Expect(seen[0].Actions[0].Create).ToNot(BeNil())
			Expect(seen[0].Actions[0].Create.LinePlot).To(Equal(res))
		},
	)
})

var _ = Describe("Channel retargets", func() {
	create := func(ctx SpecContext, channels lineplot.Channels) lineplot.LinePlot {
		GinkgoHelper()
		plot := defaults.Fill(lineplot.LinePlot{
			Project:  &proj.Key,
			Name:     "retarget",
			Channels: channels,
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &plot)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, plot.Key)
			})).To(Succeed())
		})
		return plot
	}
	style := func(ctx SpecContext, plot lineplot.Key, line string, width float64) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := svc.NewWriter(tx).Apply(
				ctx,
				plot,
				lineplot.NewSetLineStrokeWidthAction(lineplot.SetLineStrokeWidthPayload{
					Key:         line,
					StrokeWidth: width,
				}),
			)
			return err
		})).To(Succeed())
	}
	retarget := func(
		ctx SpecContext,
		plot lineplot.Key,
		to map[channel.Key]channel.Key,
	) {
		GinkgoHelper()
		id := lineplot.ResourceID(plot)
		holders := []gorp.Dependent{{
			Holder:    gorp.Target{Table: string(id.Type), Key: id.Key},
			Reference: "channels",
		}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Retarget(ctx, to, holders)
		})).To(Succeed())
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
	widthsOf := func(lines []lineplot.Line) []float64 {
		widths := make([]float64, len(lines))
		for i, l := range lines {
			widths[i] = l.StrokeWidth
		}
		return widths
	}

	It("Should swap two y channels and keep the style of each line", func(
		ctx SpecContext,
	) {
		keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 2))
		a, b := keys[0], keys[1]
		plot := create(ctx, lineplot.Channels{Y1: []channel.Key{a, b}})
		style(ctx, plot.Key, lk(plot.Key, 0, a), 7)
		style(ctx, plot.Key, lk(plot.Key, 0, b), 9)
		retarget(ctx, plot.Key, map[channel.Key]channel.Key{a: b, b: a})
		res := retrieve(ctx, plot.Key)
		Expect(res.Channels.Y1).To(Equal([]channel.Key{b, a}))
		Expect(lineKeysOf(res.Lines)).
			To(Equal([]string{lk(plot.Key, 0, b), lk(plot.Key, 0, a)}))
		Expect(widthsOf(res.Lines)).To(Equal([]float64{7, 9}))
	})

	It(
		"Should keep the first line when a retarget moves a channel onto one the plot "+
			"shows",
		func(ctx SpecContext) {
			keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 3))
			a, b, c := keys[0], keys[1], keys[2]
			plot := create(ctx, lineplot.Channels{Y1: []channel.Key{a, c, b}})
			style(ctx, plot.Key, lk(plot.Key, 0, a), 7)
			style(ctx, plot.Key, lk(plot.Key, 0, b), 9)
			retarget(ctx, plot.Key, map[channel.Key]channel.Key{a: b})
			res := retrieve(ctx, plot.Key)
			Expect(res.Channels.Y1).To(Equal([]channel.Key{b, c}))
			Expect(lineKeysOf(res.Lines)).
				To(Equal([]string{lk(plot.Key, 0, b), lk(plot.Key, 0, c)}))
			Expect(widthsOf(res.Lines)).To(Equal([]float64{7, 2}))
		},
	)

	It("Should rekey every line when the x channel moves and keep the styles", func(
		ctx SpecContext,
	) {
		keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 3))
		x, moved, y := keys[0], keys[1], keys[2]
		plot := create(ctx, lineplot.Channels{X1: x, Y1: []channel.Key{y}})
		style(ctx, plot.Key, lk(plot.Key, x, y), 7)
		retarget(ctx, plot.Key, map[channel.Key]channel.Key{x: moved})
		res := retrieve(ctx, plot.Key)
		Expect(res.Channels.X1).To(Equal(moved))
		Expect(lineKeysOf(res.Lines)).To(Equal([]string{lk(plot.Key, moved, y)}))
		Expect(res.Lines[0].XChannel).To(Equal(moved))
		Expect(widthsOf(res.Lines)).To(Equal([]float64{7}))
	})

	It("Should send the line plot as written to open Consoles", func(
		ctx SpecContext,
	) {
		keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 2))
		a, b := keys[0], keys[1]
		plot := create(ctx, lineplot.Channels{Y1: []channel.Key{a}})
		rec := &Recorder[lineplot.Key, lineplot.Action]{}
		DeferCleanup(svc.OnAction(rec.Record))
		retarget(ctx, plot.Key, map[channel.Key]channel.Key{a: b})
		seen := rec.Snapshot()
		Expect(seen).To(HaveLen(1))
		Expect(seen[0].Key).To(Equal(plot.Key))
		Expect(seen[0].Actions).To(HaveLen(1))
		Expect(seen[0].Actions[0].Create).ToNot(BeNil())
		Expect(seen[0].Actions[0].Create.LinePlot.Channels.Y1).
			To(Equal([]channel.Key{b}))
	})
})
