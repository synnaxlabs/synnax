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
	"slices"

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
		bases := make([]channel.Channel, 3)
		for i := range bases {
			bases[i] = channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Int64T,
				Virtual:  true,
			}
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
		})).To(Succeed())
		// The expression names the inputs from the highest key down.
		byKey := slices.Clone(bases)
		slices.SortFunc(byKey, func(a, b channel.Channel) int {
			return int(b.Key()) - int(a.Key())
		})
		out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:     UniqueChannelName(),
			DataType: telem.Int64T,
			Expression: fmt.Sprintf(
				"return %s + %s + %s",
				byKey[0].Name,
				byKey[1].Name,
				byKey[2].Name,
			),
		}))
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        out,
			Calculation:    MustSucceed(creator.CalculationOf(ctx, out.Key())),
		}))
		c := MustOpen(calculator.Open(ctx, calculator.Config{Module: mod}))
		keys := make([]channel.Key, len(bases))
		series := make([]telem.Series, len(bases))
		for i, b := range byKey {
			keys[i] = b.Key()
			series[i] = telem.NewSeriesV[int64](int64(i + 1))
			series[i].Alignment = telem.NewAlignment(uint32(10*(i+1)), uint32(i))
		}
		fr := frame.NewMulti(keys, series)
		o, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
		Expect(changed).To(BeTrue())
		lowest := series[len(series)-1].Alignment
		Expect(o.Get(out.Key()).Series[0].Alignment).To(Equal(lowest))
	})
})
