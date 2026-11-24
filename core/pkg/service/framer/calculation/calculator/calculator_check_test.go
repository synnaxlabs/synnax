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
		var (
			keys   []channel.Key
			series []telem.Series
		)
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
			To(telem.MatchSeriesData(telem.NewSeries(stamps[lo])))
	})
})
