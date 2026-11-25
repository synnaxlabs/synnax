// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	calcsvc "github.com/synnaxlabs/synnax/pkg/service/calculation"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/calculation/graph"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/telem/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation check", Ordered, func() {
	var (
		c          *calculation.Service
		dist       testutil.Node
		statusSvc  *status.Service
		channelSvc *channel.Service
		creator    calctestutil.Creator
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		dist = testutil.NewNode(ctx)
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: dist.DB}))
		statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: dist.DB}))
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: dist.Channel,
			TS:      dist.Storage.TS,
			DB:      dist.DB,
			HostKey: dist.Identity.HostKey,
			Status:  statusSvc,
		}))
		calcSvc := MustOpen(calcsvc.OpenService(ctx, calcsvc.ServiceConfig{
			DB:      dist.DB,
			Channel: channelSvc,
			Framer:  dist.Framer,
		}))
		creator = calctestutil.Creator{
			DB:          dist.DB,
			Channel:     channelSvc,
			Calculation: calcSvc,
		}
		writerSvc := MustSucceed(writer.NewService(writer.ServiceConfig{
			DB:      dist.DB,
			Framer:  dist.Framer,
			Channel: channelSvc,
		}))
		channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
			DB:          dist.DB,
			Calculation: calcSvc,
			Channel:     channelSvc,
			Status:      statusSvc,
		}))
		c = MustOpen(calculation.OpenService(ctx, calculation.ServiceConfig{
			DB:           dist.DB,
			Framer:       dist.Framer,
			Writer:       writerSvc,
			Channel:      channelSvc,
			Calculation:  calcSvc,
			ChannelGraph: channelGraph,
		}))
	})

	// stored creates a stored channel of dataType on an index of its own, and returns
	// the index, then the channel.
	stored := func(
		ctx context.Context,
		dataType telem.DataType,
	) (channel.Channel, channel.Channel) {
		GinkgoHelper()
		idx := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(dist.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &idx)
		})).To(Succeed())
		ch := channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   dataType,
			LocalIndex: idx.LocalKey,
		}
		Expect(dist.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return idx, ch
	}

	It("Should run the calculations of other groups when one group fails to open, and "+
		"give the failed calculation an error status", func(ctx SpecContext) {
		_, text := stored(ctx, telem.StringT)
		idx, value := stored(ctx, telem.Float64T)
		calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
			{
				Name:       UniqueChannelName(),
				DataType:   telem.Int64T,
				Expression: fmt.Sprintf("return len(%s)", text.Name),
			},
			{
				Name:       UniqueChannelName(),
				DataType:   telem.Float64T,
				Expression: fmt.Sprintf("return %s * 2", value.Name),
			},
		}))
		failed, running := calcs[0], calcs[1]
		indexes := MustSucceed(creator.Indexes(ctx, calcs))
		rm := c.OpenRequestManager()
		DeferCleanup(func() { Expect(rm.Close(ctx)).To(Succeed()) })
		sCtx, cancel := signal.Isolated()
		DeferCleanup(cancel)
		streamer := MustSucceed(dist.Framer.NewStreamer(framer.StreamerConfig{
			Keys:        channel.Keys{running.Key()},
			SendOpenAck: new(true),
		}))
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		Eventually(sOutlet.Outlet()).Should(Receive())
		Expect(rm.Set(ctx, channel.KeysFromChannels(append(calcs, indexes...)))).
			To(Succeed())
		w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
			Start: 1 * telem.SecondTS,
			Keys:  channel.Keys{idx.Key(), value.Key()},
		}))
		DeferCleanup(func() { Expect(w.Close()).To(Succeed()) })
		Expect(w.Write(frame.NewMulti(
			channel.Keys{idx.Key(), value.Key()},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(1, 2),
				telem.NewSeriesV[float64](1, 2),
			},
		))).To(BeTrue())
		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 2*time.Second).Should(Receive(&res))
		Expect(res.Frame.Get(running.Key()).Series[0]).
			To(MatchSeriesDataV[float64](2, 4))
		owner := calcsvc.ResourceID(
			MustSucceed(creator.CalculationOf(ctx, failed.Key())).Key,
		)
		Eventually(func() (status.Variant, error) {
			var st calculation.Status
			err := statusSvc.NewRetrieve[struct{}]().
				Where(status.MatchOwners[struct{}](owner)).
				Entry(&st).
				Exec(ctx, dist.DB)
			return st.Variant, err
		}).Should(Equal(status.VariantError))
	})
})
