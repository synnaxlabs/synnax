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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation breaker", Ordered, func() {
	var (
		c          *calculation.Service
		dist       testutil.Node
		channelSvc *channel.Service
		calcSvc    *calcsvc.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		dist = testutil.NewNode(ctx)
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: dist.DB}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: dist.DB,
		}))
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: dist.Channel,
			TS:      dist.Storage.TS,
			DB:      dist.DB,
			HostKey: dist.Identity.HostKey,
			Status:  statusSvc,
		}))
		calcSvc = MustOpen(calcsvc.OpenService(ctx, calcsvc.ServiceConfig{
			DB:      dist.DB,
			Channel: channelSvc,
		}))
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

	It("Should stop computing an output whose calculation is deleted", func(
		ctx SpecContext,
	) {
		base := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Int64T,
			Virtual:  true,
		}
		output := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Int64T,
			Virtual:  true,
		}
		record := calcsvc.Calculation{
			Expression: fmt.Sprintf("return %s * 2", base.Name),
		}
		Expect(dist.DB.WithTx(ctx, func(tx gorp.Tx) error {
			w := channelSvc.NewWriter(tx)
			if err := w.Create(ctx, &base); err != nil {
				return err
			}
			if err := w.Create(ctx, &output); err != nil {
				return err
			}
			record.Output = output.Key()
			return calcSvc.NewWriter(tx).Create(ctx, &record)
		})).To(Succeed())
		rm := c.OpenRequestManager()
		Expect(rm.Set(ctx, channel.Keys{output.Key()})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) { Expect(rm.Close(ctx)).To(Succeed()) })
		sCtx, cancel := signal.Isolated()
		DeferCleanup(cancel)
		w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
			Start: 1 * telem.SecondTS,
			Keys:  channel.Keys{base.Key()},
		}))
		DeferCleanup(func() { Expect(w.Close()).To(Succeed()) })
		streamer := MustSucceed(dist.Framer.NewStreamer(framer.StreamerConfig{
			Keys:        channel.Keys{output.Key()},
			SendOpenAck: new(true),
		}))
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		Eventually(sOutlet.Outlet()).Should(Receive())
		Expect(
			w.Write(frame.NewUnary(base.Key(), telem.NewSeriesV[int64](1, 2))),
		).To(BeTrue())
		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 1*time.Second).Should(Receive(&res))
		Expect(res.Frame.Get(output.Key()).Series[0]).
			To(telem.MatchSeriesDataV[int64](2, 4))

		By("Deleting the calculation and keeping its output")
		Expect(dist.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return calcSvc.NewWriter(tx).Delete(ctx, record.Key)
		})).To(Succeed())
		// Give the graph and the runtime time to see the delete.
		time.Sleep(500 * time.Millisecond)
		for range 5 {
			Expect(
				w.Write(frame.NewUnary(base.Key(), telem.NewSeriesV[int64](3))),
			).To(BeTrue())
		}
		var computed []int64
		timeout := time.After(500 * time.Millisecond)
	collect:
		for {
			select {
			case r := <-sOutlet.Outlet():
				for _, s := range r.Frame.Get(output.Key()).Series {
					computed = append(computed, s.Unmarshal[int64]()...)
				}
			case <-timeout:
				break collect
			}
		}
		Expect(computed).To(BeEmpty())
	})
})
