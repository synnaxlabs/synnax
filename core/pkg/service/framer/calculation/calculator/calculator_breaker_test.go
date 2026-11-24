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

var _ = Describe("Breaker calculation bounds", Ordered, ContinueOnFailure, func() {
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

	create := func(ctx context.Context, chs *[]channel.Channel) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).CreateMany(ctx, chs)
		})).To(Succeed())
	}

	openCalc := func(
		ctx context.Context,
		calc calctestutil.Calculation,
	) (*calculator.Calculator, channel.Channel) {
		GinkgoHelper()
		out := MustSucceed(creator.Create(ctx, calc))
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        out,
			Calculation:    MustSucceed(creator.CalculationOf(ctx, out.Key())),
		}))
		return MustOpen(calculator.Open(ctx, calculator.Config{Module: mod})), out
	}

	type sample struct {
		stamp telem.TimeStamp
		value float64
	}

	// record adds each output sample of calc in of under its alignment, and fails when
	// two different samples land on one alignment.
	record := func(seen map[telem.Alignment]sample, of frame.Frame, calc channel.Channel) {
		GinkgoHelper()
		data := of.Get(calc.Key())
		stamps := of.Get(calc.Index())
		Expect(data.Series).To(HaveLen(1))
		Expect(stamps.Series).To(HaveLen(1))
		d, t := data.Series[0], stamps.Series[0]
		for i := range d.Len() {
			a := d.Alignment.AddSamples(uint32(i))
			s := sample{
				stamp: t.ValueAt[telem.TimeStamp](int(i)),
				value: d.ValueAt[float64](int(i)),
			}
			if prev, ok := seen[a]; ok {
				Expect(s).To(Equal(prev), fmt.Sprintf(
					"two output samples at alignment %v", a,
				))
			}
			seen[a] = s
		}
	}

	It(
		"Should give each output sample of a two-index calculation its own alignment",
		func(
			ctx SpecContext,
		) {
			indexes := []channel.Channel{
				{Name: UniqueChannelName(), DataType: telem.TimestampT, IsIndex: true},
				{Name: UniqueChannelName(), DataType: telem.TimestampT, IsIndex: true},
			}
			create(ctx, &indexes)
			bases := []channel.Channel{
				{
					Name:       UniqueChannelName(),
					DataType:   telem.Float64T,
					LocalIndex: indexes[0].LocalKey,
				},
				{
					Name:       UniqueChannelName(),
					DataType:   telem.Float64T,
					LocalIndex: indexes[1].LocalKey,
				},
			}
			create(ctx, &bases)
			c, calc := openCalc(ctx, calctestutil.Calculation{
				Name:       UniqueChannelName(),
				DataType:   telem.Float64T,
				Expression: fmt.Sprintf("return %s + %s", bases[0].Name, bases[1].Name),
			})
			// Two writers, one sample per frame each. Each index starts its own first
			// domain, so both write alignments (0, 0), (0, 1), ...
			push := func(w int, k uint32, sec, v float64) (frame.Frame, bool) {
				idx := telem.NewSeriesV(telem.TimeStamp(sec * float64(telem.Second)))
				idx.Alignment = telem.NewAlignment(0, k)
				d := telem.NewSeriesV(v)
				d.Alignment = idx.Alignment
				fr := frame.NewMulti(
					[]channel.Key{indexes[w].Key(), bases[w].Key()},
					[]telem.Series{idx, d},
				)
				return MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			}
			seen := map[telem.Alignment]sample{}
			_, changed := push(0, 0, 1, 10)
			Expect(changed).To(BeFalse())
			for k := range uint32(4) {
				of, changed := push(1, k, float64(k)+1.5, float64(k)+1)
				Expect(changed).To(BeTrue())
				record(seen, of, calc)
				of, changed = push(0, k+1, float64(k)+2, float64(k+2)*10)
				Expect(changed).To(BeTrue())
				record(seen, of, calc)
			}
		},
	)

	It("Should take the stamp input in key order whatever the expression order", func(
		ctx SpecContext,
	) {
		bases := []channel.Channel{
			{Name: UniqueChannelName(), DataType: telem.Int64T, Virtual: true},
			{Name: UniqueChannelName(), DataType: telem.Int64T, Virtual: true},
		}
		create(ctx, &bases)
		Expect(bases[0].Key()).To(BeNumerically("<", bases[1].Key()))
		c, calc := openCalc(ctx, calctestutil.Calculation{
			Name:       UniqueChannelName(),
			DataType:   telem.Int64T,
			Expression: fmt.Sprintf("return %s - %s", bases[1].Name, bases[0].Name),
		})
		lo := telem.NewSeriesV[int64](1, 2)
		lo.Alignment = telem.NewAlignment(7, 3)
		hi := telem.NewSeriesV[int64](30, 40)
		hi.Alignment = telem.NewAlignment(9, 1)
		of, changed := MustSucceed2(c.Next(ctx, frame.NewMulti(
			[]channel.Key{bases[1].Key(), bases[0].Key()},
			[]telem.Series{hi, lo},
		), frame.Frame{}))
		Expect(changed).To(BeTrue())
		od := of.Get(calc.Key()).Series[0]
		Expect(od).To(MatchSeriesDataV[int64](29, 38))
	})

	DescribeTable("epoch windows of a calculation", func(
		ctx SpecContext,
		op calculation.Operation,
		first, second []int64,
	) {
		idx := []channel.Channel{
			{Name: UniqueChannelName(), DataType: telem.TimestampT, IsIndex: true},
		}
		create(ctx, &idx)
		base := []channel.Channel{{
			Name:       UniqueChannelName(),
			DataType:   telem.Int64T,
			LocalIndex: idx[0].LocalKey,
		}}
		create(ctx, &base)
		c, calc := openCalc(ctx, calctestutil.Calculation{
			Name:       UniqueChannelName(),
			DataType:   telem.Int64T,
			Expression: "return " + base[0].Name,
			Operations: []calculation.Operation{op},
		})
		send := func(a telem.Alignment, stamps []telem.TimeStamp, v ...int64) frame.Frame {
			i := telem.NewSeries(stamps)
			i.Alignment = a
			d := telem.NewSeriesV(v...)
			d.Alignment = a
			of, changed := MustSucceed2(c.Next(ctx, frame.NewMulti(
				[]channel.Key{idx[0].Key(), base[0].Key()},
				[]telem.Series{i, d},
			), frame.Frame{}))
			Expect(changed).To(BeTrue())
			return of
		}
		s := func(v ...int64) []telem.TimeStamp {
			out := make([]telem.TimeStamp, len(v))
			for i, x := range v {
				out[i] = telem.TimeStamp(x) * telem.SecondTS
			}
			return out
		}
		of := send(telem.NewAlignment(2, 0), s(3, 4, 5, 6, 7, 11), 10, 20, 100, 5, 7, 1)
		Expect(of.Get(calc.Key()).Series[0]).To(MatchSeriesDataV(first...))
		Expect(of.Get(calc.Index()).Series[0]).To(MatchSeriesDataV(
			4*telem.SecondTS, 7*telem.SecondTS, 11*telem.SecondTS,
		))
		Expect(
			of.Get(calc.Key()).Series[0].Alignment,
		).To(Equal(telem.NewAlignment(2, 0)))
		of = send(telem.NewAlignment(2, 6), s(12, 16), 3, 2)
		Expect(of.Get(calc.Key()).Series[0]).To(MatchSeriesDataV(second...))
		Expect(
			of.Get(calc.Key()).Series[0].Alignment,
		).To(Equal(telem.NewAlignment(2, 6)))
	},
		Entry(
			"max",
			calculation.Operation{Variant: calculation.MaxOperation{
				Window: calculation.Window{Duration: 5 * telem.Second},
			}},
			[]int64{20, 100, 1},
			[]int64{3, 2},
		),
		Entry(
			"min",
			calculation.Operation{Variant: calculation.MinOperation{
				Window: calculation.Window{Duration: 5 * telem.Second},
			}},
			[]int64{10, 5, 1},
			[]int64{1, 2},
		),
		Entry(
			"avg",
			calculation.Operation{Variant: calculation.AvgOperation{
				Window: calculation.Window{Duration: 5 * telem.Second},
			}},
			[]int64{15, 37, 1},
			[]int64{2, 2},
		),
	)

	It("Should not give a reset's output the alignment of an earlier output", func(
		ctx SpecContext,
	) {
		reset := []channel.Channel{{
			Name:     UniqueChannelName(),
			DataType: telem.BooleanT,
			Virtual:  true,
		}}
		create(ctx, &reset)
		idx := []channel.Channel{
			{Name: UniqueChannelName(), DataType: telem.TimestampT, IsIndex: true},
		}
		create(ctx, &idx)
		base := []channel.Channel{{
			Name:       UniqueChannelName(),
			DataType:   telem.Float64T,
			LocalIndex: idx[0].LocalKey,
		}}
		create(ctx, &base)
		c, calc := openCalc(ctx, calctestutil.Calculation{
			Name:       UniqueChannelName(),
			DataType:   telem.Float64T,
			Expression: "return " + base[0].Name,
			Operations: []calculation.Operation{{
				Variant: calculation.AvgOperation{Window: calculation.Window{
					Duration: 10 * telem.Second,
					Reset:    new(reset[0].Key()),
				}},
			}},
		})
		seen := map[telem.Alignment]sample{}
		for k, v := range []float64{10, 30} {
			i := telem.NewSeriesV(telem.TimeStamp(k+1) * telem.SecondTS)
			i.Alignment = telem.NewAlignment(1, uint32(k))
			d := telem.NewSeriesV(v)
			d.Alignment = i.Alignment
			of, changed := MustSucceed2(c.Next(ctx, frame.NewMulti(
				[]channel.Key{idx[0].Key(), base[0].Key()},
				[]telem.Series{i, d},
			), frame.Frame{}))
			Expect(changed).To(BeTrue())
			record(seen, of, calc)
		}
		// A free write counter gives the reset channel its own alignment.
		r := telem.NewSeriesV(true)
		r.Alignment = telem.NewAlignment(3, 0)
		of, changed := MustSucceed2(c.Next(
			ctx,
			frame.NewUnary(reset[0].Key(), r),
			frame.Frame{},
		))
		if changed {
			record(seen, of, calc)
		}
	})
})
