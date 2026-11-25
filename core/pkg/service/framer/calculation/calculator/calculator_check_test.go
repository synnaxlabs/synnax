// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculator_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/calculation/compiler"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/telem/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculator check", Ordered, func() {
	var (
		db         *gorp.DB
		channelSvc *channel.Service
		creator    calctestutil.Creator
	)

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: node.DB}))
		statusSvc := MustOpen(
			status.OpenService(ctx, status.ServiceConfig{DB: node.DB}),
		)
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  statusSvc,
		}))
		creator = calctestutil.Creator{
			DB:      db,
			Channel: channelSvc,
			Calculation: MustOpen(calculation.OpenService(
				ctx,
				calculation.ServiceConfig{
					DB:      db,
					Channel: channelSvc,
					Framer:  node.Framer,
				},
			)),
		}
	})

	It("Should take the stamps of the lowest key among equal inputs, whatever order "+
		"the expression names them in", func(ctx SpecContext) {
		indexes := make([]channel.Channel, 2)
		for i := range indexes {
			indexes[i] = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).CreateMany(ctx, &indexes)
		})).To(Succeed())
		bases := make([]channel.Channel, 2)
		for i := range bases {
			bases[i] = channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Int64T,
				LocalIndex: indexes[i].LocalKey,
			}
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
		})).To(Succeed())
		lo, hi := 0, 1
		if bases[1].Key() < bases[0].Key() {
			lo, hi = 1, 0
		}
		// The expression names the higher key first.
		out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:     UniqueChannelName(),
			DataType: telem.Int64T,
			Expression: fmt.Sprintf(
				"return %s + %s",
				bases[hi].Name,
				bases[lo].Name,
			),
		}))
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        out,
			Calculation:    MustSucceed(creator.CalculationOf(ctx, out.Key())),
		}))
		c := MustOpen(calculator.Open(ctx, calculator.Config{Module: mod}))
		stamps := make([][]telem.TimeStamp, 2)
		stamps[lo] = []telem.TimeStamp{1 * telem.SecondTS, 2 * telem.SecondTS}
		stamps[hi] = []telem.TimeStamp{5 * telem.SecondTS, 6 * telem.SecondTS}
		keys := make([]channel.Key, 0, 2*len(bases))
		series := make([]telem.Series, 0, 2*len(bases))
		for _, i := range []int{hi, lo} {
			keys = append(keys, indexes[i].Key(), bases[i].Key())
			series = append(
				series,
				telem.NewSeries(stamps[i]),
				telem.NewSeriesV[int64](int64(10*(i+1)), int64(10*(i+1))),
			)
		}
		fr := frame.NewMulti(keys, series)
		o, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
		Expect(changed).To(BeTrue())
		Expect(o.Get(out.Index()).Series[0]).
			To(MatchSeriesData(telem.NewSeries(stamps[lo])))
	})

	It("Should give a joined calculation's outputs the bounds of its primary when the "+
		"primary is not the lowest key", func(ctx SpecContext) {
		indexes := make([]channel.Channel, 2)
		for i := range indexes {
			indexes[i] = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).CreateMany(ctx, &indexes)
		})).To(Succeed())
		bases := make([]channel.Channel, 2)
		for i := range bases {
			bases[i] = channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float64T,
				LocalIndex: indexes[i].LocalKey,
			}
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
		})).To(Succeed())
		lo, hi := 0, 1
		if bases[1].Key() < bases[0].Key() {
			lo, hi = 1, 0
		}
		out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:     UniqueChannelName(),
			DataType: telem.Float64T,
			Expression: fmt.Sprintf(
				"return %s + %s",
				bases[lo].Name,
				bases[hi].Name,
			),
		}))
		Expect(creator.Apply(ctx, out.Key(), calculation.NewSetAction(
			calculation.SetPayload{Path: []string{"primary"}, Value: bases[hi].Key()},
		))).To(Succeed())
		calc := MustSucceed(creator.CalculationOf(ctx, out.Key()))
		Expect(calc.Primary).To(HaveValue(Equal(bases[hi].Key())))
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        out,
			Calculation:    calc,
			Live:           true,
		}))
		Expect(mod.Joined).To(BeTrue())
		c := MustOpen(calculator.Open(ctx, calculator.Config{Module: mod}))
		DeferCleanup(c.Close)
		send := func(i int, a telem.Alignment, v ...float64) frame.Frame {
			stamps := telem.NewSeriesSecondsTSV(1, 2)
			values := telem.NewSeriesV(v...)
			stamps.Alignment, values.Alignment = a, a
			return frame.NewMulti(
				[]channel.Key{indexes[i].Key(), bases[i].Key()},
				[]telem.Series{stamps, values},
			)
		}
		MustSucceed2(
			c.Next(ctx, send(lo, telem.NewAlignment(0, 5), 1, 2), frame.Frame{}),
		)
		of, changed := MustSucceed2(
			c.Next(ctx, send(hi, telem.NewAlignment(0, 9), 10, 20), frame.Frame{}),
		)
		Expect(changed).To(BeTrue())
		od := of.Get(out.Key()).Series[0]
		Expect(od).To(MatchSeriesDataV[float64](11, 22))
		Expect(od.Alignment).To(Equal(telem.NewAlignment(0, 9)),
			"the outputs take the alignment of the stored primary, not the lowest key")
		Expect(od.TimeRange).To(Equal(telem.TimeRange{
			Start: telem.SecondTS,
			End:   2*telem.SecondTS + 1,
		}))
	})

	It("Should warn of stale inputs again after an error replaced the warning",
		func(ctx SpecContext) {
			indexes := []channel.Channel{
				{Name: UniqueChannelName(), DataType: telem.TimestampT, IsIndex: true},
				{Name: UniqueChannelName(), DataType: telem.TimestampT, IsIndex: true},
			}
			bases := []channel.Channel{
				{Name: UniqueChannelName(), DataType: telem.Float64T},
				{Name: UniqueChannelName(), DataType: telem.Float64T},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := channelSvc.NewWriter(tx)
				if err := w.CreateMany(ctx, &indexes); err != nil {
					return err
				}
				for i := range bases {
					bases[i].LocalIndex = indexes[i].LocalKey
				}
				return w.CreateMany(ctx, &bases)
			})).To(Succeed())
			out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:     UniqueChannelName(),
				DataType: telem.Float64T,
				Expression: fmt.Sprintf(
					"return %s + %s",
					bases[0].Name,
					bases[1].Name,
				),
			}))
			calc := MustSucceed(creator.CalculationOf(ctx, out.Key()))
			Expect(calc.Primary).To(HaveValue(Equal(bases[0].Key())))
			mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
				ChannelService: channelSvc,
				Channel:        out,
				Calculation:    calc,
				Live:           true,
			}))
			var now telem.TimeStamp
			c := MustOpen(calculator.Open(ctx, calculator.Config{
				Module: mod,
				Now:    func() telem.TimeStamp { return now },
			}))
			DeferCleanup(c.Close)
			g := calculator.Group{c}
			// step writes the primary at stamp, in seconds, with alignment a, a second
			// later on the clock, and returns the variant and message of each report.
			step := func(stamp telem.TimeStamp, a uint32) []string {
				GinkgoHelper()
				now += telem.SecondTS
				idx := telem.NewSeriesSecondsTSV(stamp)
				data := telem.NewSeriesV[float64](1)
				idx.Alignment = telem.NewAlignment(0, a)
				data.Alignment = idx.Alignment
				var reported []string
				g.Next(ctx, frame.NewMulti(
					[]channel.Key{indexes[0].Key(), bases[0].Key()},
					[]telem.Series{idx, data},
				), func(
					_ context.Context,
					_ channel.Key,
					_ string,
					st calculator.Status,
				) bool {
					reported = append(reported, string(st.Variant)+": "+st.Message)
					return true
				})
				return reported
			}
			stale := "warning: " + calculation.StaleStatus("").Message
			// The join settles on the fourth frame of the primary, 1 s apart, and the
			// other input never writes.
			for i := range 3 {
				Expect(step(telem.TimeStamp(i+1), uint32(i))).To(BeEmpty())
			}
			Expect(step(4, 3)).To(ConsistOf(stale))
			// A primary stamp before one the join passed fails the step.
			Expect(step(2, 4)).To(ConsistOf(HavePrefix("error: ")))
			// The inputs are still stale, and the error is not the cause now.
			Expect(step(5, 5)).To(ConsistOf(stale))
		})
})
