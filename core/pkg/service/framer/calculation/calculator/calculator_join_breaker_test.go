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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Breaker live join", Ordered, ContinueOnFailure, func() {
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
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: node.DB,
		}))
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

	// openSum opens a live calculation of the sum of two stored channels on their own
	// indexes, with bases[primary] as the primary, on the clock *now. bases[0] has the
	// lower key.
	openSum := func(
		ctx SpecContext,
		now *telem.TimeStamp,
		primary int,
	) (*calculator.Calculator, []channel.Channel, []channel.Channel) {
		GinkgoHelper()
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
			Name:       UniqueChannelName(),
			DataType:   telem.Float64T,
			Expression: fmt.Sprintf("return %s + %s", bases[0].Name, bases[1].Name),
		}))
		Expect(bases[0].Key()).To(BeNumerically("<", bases[1].Key()))
		c := MustSucceed(creator.CalculationOf(ctx, out.Key()))
		key := bases[primary].Key()
		c.Primary = &key
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        out,
			Calculation:    c,
			Live:           true,
		}))
		Expect(mod.Joined).To(BeTrue())
		calc := MustOpen(calculator.Open(ctx, calculator.Config{
			Module: mod,
			Now:    func() telem.TimeStamp { return *now },
		}))
		DeferCleanup(calc.Close)
		return calc, indexes, bases
	}

	// series returns a series of input key and its index series idx, at stamps in
	// seconds, from alignment a of domain 0.
	series := func(
		idx, key channel.Key,
		a uint32,
		values []float64,
		stamps ...telem.TimeStamp,
	) ([]channel.Key, []telem.Series) {
		t := telem.NewSeriesSecondsTSV(stamps...)
		d := telem.NewSeries(values)
		t.Alignment = telem.NewAlignment(0, a)
		d.Alignment = t.Alignment
		return []channel.Key{idx, key}, []telem.Series{t, d}
	}

	// values returns every output sample of calc in of, in frame order.
	values := func(of frame.Frame, calc *calculator.Calculator) []float64 {
		ser := of.Get(calc.Channel().Key()).Series
		v := make([]float64, 0, len(ser))
		for _, s := range ser {
			v = append(v, s.Unmarshal[float64]()...)
		}
		return v
	}

	// A calculation is as fresh as its primary. A stamp past the hold limit gets no
	// output, and every other primary stamp gets one.
	It(
		"Should give an output for a primary series that arrives while runs remain",
		func(
			ctx SpecContext,
		) {
			var now telem.TimeStamp
			calc, indexes, bases := openSum(ctx, &now, 0)
			var (
				keys []channel.Key
				sers []telem.Series
			)
			add := func(k []channel.Key, s []telem.Series) {
				keys, sers = append(keys, k...), append(sers, s...)
			}
			// b steps 1 s to 2 s, so its hold limit of 3 s drops a at 10 s and 11 s,
			// and
			// the join gives two runs for a's first series.
			add(series(indexes[1].Key(), bases[1].Key(), 0,
				[]float64{10, 11, 12, 20}, 0, 1, 2, 20))
			add(series(indexes[0].Key(), bases[0].Key(), 0,
				[]float64{1, 2, 3, 4, 5}, 1, 2, 10, 11, 21))
			// A second series of a in the same frame.
			add(series(indexes[0].Key(), bases[0].Key(), 5, []float64{6, 7}, 22, 23))
			of, _ := MustSucceed2(
				calc.Next(ctx, frame.NewMulti(keys, sers), frame.Frame{}),
			)
			got := values(of, calc)
			k, s := series(indexes[0].Key(), bases[0].Key(), 7, []float64{8}, 24)
			of, _ = MustSucceed2(calc.Next(ctx, frame.NewMulti(k, s), frame.Frame{}))
			got = append(got, values(of, calc)...)
			Expect(got).To(Equal([]float64{12, 14, 25, 26, 27, 28}))
		},
	)

	// Group.Next reports a stale warning on each step that drops stamps while no stale
	// warning is saved.
	It("Should warn of drops after an error replaced the stale warning", func(
		ctx SpecContext,
	) {
		var now telem.TimeStamp
		calc, indexes, bases := openSum(ctx, &now, 0)
		g := calculator.Group{calc}
		step := func(stamp telem.TimeStamp, a uint32) []calculator.Status {
			GinkgoHelper()
			now += telem.SecondTS
			k, s := series(indexes[0].Key(), bases[0].Key(), a, []float64{1}, stamp)
			var reported []calculator.Status
			g.Next(ctx, frame.NewMulti(k, s), func(
				_ context.Context,
				_ channel.Key,
				_ string,
				st calculator.Status,
			) bool {
				reported = append(reported, st)
				return true
			})
			return reported
		}
		for i := range 3 {
			Expect(step(telem.TimeStamp(i+1), uint32(i))).To(BeEmpty())
		}
		Expect(step(4, 3)).To(ConsistOf(calculation.StaleStatus(calc.Channel().Name)))
		// A stamp that does not rise fails the step, and the error replaces the saved
		// stale warning.
		errored := step(3, 4)
		Expect(errored).To(HaveLen(1))
		Expect(errored[0].Message).To(Equal("Failed to run calculation"))
		// b still has no sample, so the join drops the next primary stamp.
		Expect(step(6, 5)).To(ConsistOf(calculation.StaleStatus(calc.Channel().Name)))
	})

	// A step with no drops reports a success, which replaces a stale warning.
	It("Should not warn of stale inputs on a step with no drops after an error", func(
		ctx SpecContext,
	) {
		var now telem.TimeStamp
		calc, indexes, bases := openSum(ctx, &now, 0)
		g := calculator.Group{calc}
		var keys []channel.Key
		var sers []telem.Series
		add := func(k []channel.Key, s []telem.Series) {
			keys, sers = append(keys, k...), append(sers, s...)
		}
		step := func() []calculator.Status {
			GinkgoHelper()
			now += telem.SecondTS
			var reported []calculator.Status
			g.Next(ctx, frame.NewMulti(keys, sers), func(
				_ context.Context,
				_ channel.Key,
				_ string,
				st calculator.Status,
			) bool {
				reported = append(reported, st)
				return true
			})
			keys, sers = nil, nil
			return reported
		}
		// a and b at 1 Hz settle the join on the fourth step with no drops.
		for i := range 4 {
			stamp := telem.TimeStamp(i + 1)
			add(
				series(
					indexes[1].Key(),
					bases[1].Key(),
					uint32(i),
					[]float64{10},
					stamp,
				),
			)
			add(
				series(
					indexes[0].Key(),
					bases[0].Key(),
					uint32(i),
					[]float64{1},
					stamp,
				),
			)
			Expect(step()).To(BeEmpty())
		}
		// b skips from 5 s to 21 s, so a at 20 s is past b's hold limit, and a's second
		// series in the frame does not rise.
		add(series(indexes[1].Key(), bases[1].Key(), 4, []float64{10, 10}, 5, 21))
		add(series(indexes[0].Key(), bases[0].Key(), 4, []float64{1, 1}, 5, 20))
		add(series(indexes[0].Key(), bases[0].Key(), 6, []float64{1}, 19))
		errored := step()
		Expect(errored).To(HaveLen(1))
		Expect(errored[0].Message).To(Equal("Failed to run calculation"))
		// A clean step with output and no drops.
		add(series(indexes[1].Key(), bases[1].Key(), 6, []float64{10}, 22))
		add(series(indexes[0].Key(), bases[0].Key(), 7, []float64{1}, 22))
		Expect(step()).To(ConsistOf(calculator.Status{
			Name:    calc.Channel().Name,
			Variant: status.VariantSuccess,
			Message: "Calculation running",
		}))
	})

	// A live calculation of one stored channel opens and runs, as it does today. The
	// same module compiled with Live false opens and gives [2, 3].
	It("Should run a live calculation that reads a stored string channel", func(
		ctx SpecContext,
	) {
		indexes := []channel.Channel{
			{Name: UniqueChannelName(), DataType: telem.TimestampT, IsIndex: true},
		}
		bases := []channel.Channel{{Name: UniqueChannelName(), DataType: telem.StringT}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := channelSvc.NewWriter(tx)
			if err := w.CreateMany(ctx, &indexes); err != nil {
				return err
			}
			bases[0].LocalIndex = indexes[0].LocalKey
			return w.CreateMany(ctx, &bases)
		})).To(Succeed())
		out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       UniqueChannelName(),
			DataType:   telem.Int64T,
			Expression: fmt.Sprintf("return len(%s)", bases[0].Name),
		}))
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        out,
			Calculation:    MustSucceed(creator.CalculationOf(ctx, out.Key())),
			Live:           true,
		}))
		calc := MustOpen(calculator.Open(ctx, calculator.Config{Module: mod}))
		DeferCleanup(calc.Close)
		t := telem.NewSeriesSecondsTSV(1, 2)
		d := telem.NewSeriesV("ab", "abc")
		of, changed := MustSucceed2(calc.Next(ctx, frame.NewMulti(
			[]channel.Key{indexes[0].Key(), bases[0].Key()},
			[]telem.Series{t, d},
		), frame.Frame{}))
		Expect(changed).To(BeTrue())
		Expect(of.Get(out.Key()).Series[0].Unmarshal[int64]()).To(Equal([]int64{2, 3}))
	})

	// A primary stamp either gets an output or counts in Dropped, so the order of the
	// keys must not decide whether a primary sample survives a refused push of another
	// input.
	DescribeTable(
		"Should keep the primary samples of a step whose other input fails",
		func(ctx SpecContext, primary int) {
			var now telem.TimeStamp
			calc, indexes, bases := openSum(ctx, &now, primary)
			other := 1 - primary
			pIdx, pKey := indexes[primary].Key(), bases[primary].Key()
			oIdx, oKey := indexes[other].Key(), bases[other].Key()
			next := func(k []channel.Key, s []telem.Series) ([]float64, error) {
				of, _, err := calc.Next(ctx, frame.NewMulti(k, s), frame.Frame{})
				if err != nil {
					return nil, err
				}
				return values(of, calc), nil
			}
			k1, s1 := series(oIdx, oKey, 0, []float64{10, 20, 30}, 10, 11, 12)
			k2, s2 := series(pIdx, pKey, 0, []float64{1, 2, 3}, 10, 11, 12)
			Expect(next(append(k1, k2...), append(s1, s2...))).
				To(Equal([]float64{11, 22, 33}))
			// The other input gets a backfill in a new domain, whose stamps do not
			// rise, beside two new primary samples.
			k1, s1 = series(oIdx, oKey, 0, []float64{7, 8}, 1, 2)
			for i := range s1 {
				s1[i].Alignment = telem.NewAlignment(1, 0)
			}
			k2, s2 = series(pIdx, pKey, 3, []float64{4, 5}, 13, 14)
			Expect(next(append(k1, k2...), append(s1, s2...))).Error().
				To(MatchError(ContainSubstring("does not follow")))
			k2, s2 = series(pIdx, pKey, 5, []float64{6}, 15)
			Expect(next(k2, s2)).To(Equal([]float64{34, 35, 36}))
		},
		Entry("with the primary at the lower key", 0),
		Entry("with the primary at the higher key", 1),
	)
})
