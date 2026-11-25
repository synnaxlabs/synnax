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

var _ = Describe("Breaker 2 live join", Ordered, ContinueOnFailure, func() {
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

	// storedOn creates one index and a stored channel of each data type on it.
	storedOn := func(
		ctx SpecContext,
		dataTypes ...telem.DataType,
	) (channel.Channel, []channel.Channel) {
		GinkgoHelper()
		idx := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &idx)
		})).To(Succeed())
		chs := make([]channel.Channel, len(dataTypes))
		for i, dt := range dataTypes {
			chs[i] = channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   dt,
				LocalIndex: idx.LocalKey,
			}
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).CreateMany(ctx, &chs)
		})).To(Succeed())
		return idx, chs
	}

	// compile compiles the calculation of out for a live read when live is set.
	compile := func(ctx SpecContext, out channel.Channel, live bool) compiler.Module {
		GinkgoHelper()
		return MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        out,
			Calculation:    MustSucceed(creator.CalculationOf(ctx, out.Key())),
			Live:           live,
		}))
	}

	// A frame that holds two series of one input runs the join twice in one step: the
	// first run emits, and the second takes a backfill, which is a late input. Every
	// primary stamp must still get one output.
	It(
		"Should keep the output a step joined when a later series of the step backfills "+
			"another input, with no error",
		func(ctx SpecContext) {
			var now telem.TimeStamp
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
			c := MustSucceed(creator.CalculationOf(ctx, out.Key()))
			primary := bases[0].Key()
			c.Primary = &primary
			mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
				ChannelService: channelSvc,
				Channel:        out,
				Calculation:    c,
				Live:           true,
			}))
			Expect(mod.Joined).To(BeTrue())
			calc := MustOpen(calculator.Open(ctx, calculator.Config{
				Module: mod,
				Now:    func() telem.TimeStamp { return now },
			}))
			DeferCleanup(calc.Close)
			series := func(
				i int,
				domain, a uint32,
				values []float64,
				stamps ...telem.TimeStamp,
			) ([]channel.Key, []telem.Series) {
				t := telem.NewSeriesSecondsTSV(stamps...)
				d := telem.NewSeries(values)
				t.Alignment = telem.NewAlignment(domain, a)
				d.Alignment = t.Alignment
				return []channel.Key{
					indexes[i].Key(),
					bases[i].Key(),
				}, []telem.Series{
					t,
					d,
				}
			}
			var got []float64
			next := func(keys []channel.Key, sers []telem.Series) error {
				of, _, err := calc.Next(ctx, frame.NewMulti(keys, sers), frame.Frame{})
				if err != nil {
					return err
				}
				for _, s := range of.Get(out.Key()).Series {
					got = append(got, s.Unmarshal[float64]()...)
				}
				return nil
			}
			var (
				keys []channel.Key
				sers []telem.Series
			)
			add := func(k []channel.Key, s []telem.Series) {
				keys, sers = append(keys, k...), append(sers, s...)
			}
			// One frame: b at 10 s to 12 s, a at 10 s to 12 s, then a backfill of b in
			// a
			// new domain whose stamps do not rise.
			add(series(1, 0, 0, []float64{10, 20, 30}, 10, 11, 12))
			add(series(0, 0, 0, []float64{1, 2, 3}, 10, 11, 12))
			add(series(1, 1, 0, []float64{7, 8}, 1, 2))
			// The backfill is a late input: the step fails nothing and keeps its
			// output.
			Expect(next(keys, sers)).To(Succeed())
			Expect(got).To(Equal([]float64{11, 22, 33}))
			k, s := series(0, 0, 3, []float64{4}, 13)
			Expect(next(k, s)).To(Succeed())
			Expect(got).To(Equal([]float64{11, 22, 33, 34}))
		},
	)

	// The join's contract: each primary stamp gets an output or counts in Dropped. A
	// frame with three series of the primary, whose first the join drops before b's
	// first sample, must still pair the later two with b.
	It("Should join every series of the primary in a frame whose first series the "+
		"join drops", func(ctx SpecContext) {
		var now telem.TimeStamp
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
		c := MustSucceed(creator.CalculationOf(ctx, out.Key()))
		primary := bases[0].Key()
		c.Primary = &primary
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        out,
			Calculation:    c,
			Live:           true,
		}))
		Expect(mod.Joined).To(BeTrue())
		calc := MustOpen(calculator.Open(ctx, calculator.Config{
			Module: mod,
			Now:    func() telem.TimeStamp { return now },
		}))
		DeferCleanup(calc.Close)
		var (
			keys []channel.Key
			sers []telem.Series
			got  []float64
		)
		add := func(i int, a uint32, values []float64, stamps ...telem.TimeStamp) {
			t := telem.NewSeriesSecondsTSV(stamps...)
			d := telem.NewSeries(values)
			t.Alignment = telem.NewAlignment(0, a)
			d.Alignment = t.Alignment
			keys = append(keys, indexes[i].Key(), bases[i].Key())
			sers = append(sers, t, d)
		}
		next := func() {
			GinkgoHelper()
			of, _ := MustSucceed2(
				calc.Next(ctx, frame.NewMulti(keys, sers), frame.Frame{}),
			)
			for _, s := range of.Get(out.Key()).Series {
				got = append(got, s.Unmarshal[float64]()...)
			}
			keys, sers = nil, nil
		}
		add(1, 0, []float64{50, 60, 70, 80, 90}, 5, 6, 7, 8, 9)
		add(0, 0, []float64{1, 2}, 1, 2)
		add(0, 2, []float64{3, 4}, 6, 7)
		add(0, 4, []float64{5, 6}, 8, 9)
		next()
		add(0, 6, []float64{7}, 10)
		next()
		Expect(got).To(Equal([]float64{63, 74, 85, 96, 97}))
	})

	// B4, Staged: 15c-2 joins a live program only when every read has a fixed-density
	// type, and keeps today's alignment path otherwise, so it regresses nothing. A
	// stored UUID channel has a fixed density, but Arc reads it as a string.
	It("Should open a live calculation that reads a stored UUID channel, as the "+
		"same calculation opens with no join", func(ctx SpecContext) {
		_, chs := storedOn(ctx, telem.UUIDT)
		out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       UniqueChannelName(),
			DataType:   telem.Int64T,
			Expression: fmt.Sprintf("return len(%s)", chs[0].Name),
		}))
		history := MustOpen(calculator.Open(ctx, calculator.Config{
			Module: compile(ctx, out, false),
		}))
		DeferCleanup(history.Close)
		live := MustOpen(calculator.Open(ctx, calculator.Config{
			Module: compile(ctx, out, true),
		}))
		DeferCleanup(live.Close)
	})
})
