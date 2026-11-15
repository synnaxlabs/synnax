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
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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
