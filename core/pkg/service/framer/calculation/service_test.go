// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	"time"

	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/status"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var sleepInterval = 25 * time.Millisecond

var _ = Describe("Calculation", Ordered, func() {
	var (
		c    *calculation.Service
		dist mock.Node
	)

	BeforeAll(func() {
		distB := mock.NewCluster()
		dist = distB.Provision(ctx)
		c = MustSucceed(calculation.OpenService(ctx, calculation.ServiceConfig{
			Framer:            dist.Framer,
			Channel:           dist.Channel,
			ChannelObservable: dist.Channel.NewObservable(),
		}))
	})

	AfterAll(func() {
		Expect(c.Close()).To(Succeed())
		Expect(dist.Close()).To(Succeed())
	})

	It("Output a basic calculation", func() {
		baseCH := channel.Channel{
			Name:     "base",
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(dist.Channel.Create(ctx, &baseCH)).To(Succeed())
		calculatedCH := channel.Channel{
			Name:        "calculated",
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: cluster.Free,
			Requires:    []channel.Key{baseCH.Key()},
			Expression:  "return base * 2",
		}
		Expect(dist.Channel.Create(ctx, &calculatedCH)).To(Succeed())
		MustSucceed(c.Request(ctx, calculatedCH.Key()))
		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()
		w := MustSucceed(dist.Framer.OpenWriter(
			ctx,
			framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{baseCH.Key()},
			},
		))
		streamer := MustSucceed(
			dist.Framer.NewStreamer(
				ctx,
				framer.StreamerConfig{
					Keys: []channel.Key{calculatedCH.Key()},
				},
			),
		)
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		time.Sleep(sleepInterval)
		MustSucceed(w.Write(core.UnaryFrame(baseCH.Key(), telem.NewSeriesV[int64](1, 2))))
		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calculatedCH.Key()}))
		Expect(w.Close()).To(Succeed())
	})

	It("Handle undefined symbols", func() {
		baseCH := channel.Channel{
			Name:     "base",
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(dist.Channel.Create(ctx, &baseCH)).To(Succeed())
		calculatedCH := channel.Channel{
			Name:        "calculated",
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: cluster.Free,
			Requires:    []channel.Key{baseCH.Key()},
			Expression:  "return base * fake",
		}
		Expect(dist.Channel.Create(ctx, &calculatedCH)).To(Succeed())
		MustSucceed(c.Request(ctx, calculatedCH.Key()))
		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()
		w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
			Start: telem.Now(),
			Keys:  []channel.Key{baseCH.Key()},
		}))
		streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys:        []channel.Key{calculatedCH.Key()},
			SendOpenAck: config.True(),
		}))
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive())
		MustSucceed(w.Write(core.UnaryFrame(baseCH.Key(), telem.NewSeriesV[int64](1, 2))))
		Consistently(sOutlet.Outlet(), 500*time.Millisecond).ShouldNot(Receive())
		Expect(w.Close()).To(Succeed())
	})

	It("Return a warning for dividing by zero", func() {
		baseCH := channel.Channel{
			Name:     "base",
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(dist.Channel.Create(ctx, &baseCH)).To(Succeed())
		calculatedCH := channel.Channel{
			Name:        "calculated",
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: cluster.Free,
			Requires:    []channel.Key{baseCH.Key()},
			Expression:  "return base / 0",
		}
		Expect(dist.Channel.Create(ctx, &calculatedCH)).To(Succeed())
		MustSucceed(c.Request(ctx, calculatedCH.Key()))
		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()
		w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
			Start: telem.Now(),
			Keys:  []channel.Key{baseCH.Key()},
		}))
		streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys:        []channel.Key{calculatedCH.Key()},
			SendOpenAck: config.True(),
		}))
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive())
		MustSucceed(w.Write(core.UnaryFrame(
			baseCH.Key(),
			telem.NewSeriesV[int64](1, 2),
		)))
		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calculatedCH.Key()}))
	})

	It("Should handle nested calculations", func() {
		baseCH := channel.Channel{
			Name:     "base",
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(dist.Channel.Create(ctx, &baseCH)).To(Succeed())

		// First calculated channel that doubles the base value
		calc1CH := channel.Channel{
			Name:        "calc1",
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: cluster.Free,
			Requires:    []channel.Key{baseCH.Key()},
			Expression:  "return base * 2",
		}
		Expect(dist.Channel.Create(ctx, &calc1CH)).To(Succeed())

		// Second calculated channel that adds 1 to the first calculated channel
		calc2CH := channel.Channel{
			Name:        "calc2",
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: cluster.Free,
			Requires:    []channel.Key{calc1CH.Key()},
			Expression:  "return calc1 + 1",
		}
		Expect(dist.Channel.Create(ctx, &calc2CH)).To(Succeed())

		MustSucceed(c.Request(ctx, calc1CH.Key()))
		MustSucceed(c.Request(ctx, calc2CH.Key()))

		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()

		w := MustSucceed(dist.Framer.OpenWriter(ctx,
			framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{baseCH.Key()},
			},
		))
		streamer := MustSucceed(
			dist.Framer.NewStreamer(
				ctx,
				framer.StreamerConfig{
					Keys:        []channel.Key{calc2CH.Key()},
					SendOpenAck: config.True(),
				},
			),
		)
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive())

		MustSucceed(w.Write(core.UnaryFrame(baseCH.Key(), telem.NewSeriesV[int64](1, 2))))
		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calc2CH.Key()}))

		// For base values [1, 2]:
		// calc1 should be [2, 4] (base * 2)
		// calc2 should be [3, 5] (calc1 + 1)
		series := res.Frame.SeriesAt(0)
		Expect(series.Len()).To(Equal(int64(2)))
		Expect(telem.ValueAt[int64](series, 0)).To(Equal(int64(3)))
		Expect(telem.ValueAt[int64](series, 1)).To(Equal(int64(5)))
	})

	It("Should error when calculating with undefined channels in expression", func() {
		baseCH := channel.Channel{
			Name:     "base",
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(dist.Channel.Create(ctx, &baseCH)).To(Succeed())

		calculatedCH := channel.Channel{
			Name:        "calculated",
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: cluster.Free,
			Requires:    []channel.Key{baseCH.Key()},
			Expression:  "return fake1 + fake2",
		}
		Expect(dist.Channel.Create(ctx, &calculatedCH)).To(Succeed())

		var statusCh channel.Channel
		statusCh.Name = calculation.StatusChannelName
		Expect(
			dist.Channel.Create(
				ctx,
				&statusCh,
				channel.RetrieveIfNameExists(true),
			),
		).To(Succeed())

		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()

		// Set up writer for base channel
		w := MustSucceed(
			dist.Framer.OpenWriter(
				ctx,
				framer.WriterConfig{
					Start: telem.Now(),
					Keys:  []channel.Key{baseCH.Key()},
				},
			),
		)
		// Set up a streamer to watch for status changes
		streamer := MustSucceed(
			dist.Framer.NewStreamer(
				ctx,
				framer.StreamerConfig{
					Keys:        []channel.Key{statusCh.Key()},
					SendOpenAck: config.True(),
				},
			),
		)
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		MustSucceed(c.Request(ctx, calculatedCH.Key()))
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive())

		time.Sleep(5 * time.Millisecond)
		MustSucceed(w.Write(core.UnaryFrame(
			baseCH.Key(),
			telem.NewSeriesV[int64](1, 2),
		)))

		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.SeriesAt(0).DataType).To(Equal(telem.JSONT))

		var s calculation.Status
		data := res.Frame.SeriesAt(0).Data
		Expect(json.Unmarshal(data[:len(data)-1], &s)).To(Succeed()) // -1 to remove newline

		Expect(s.Key).To(Equal(calculatedCH.Key().String()))
		Expect(s.Variant).To(Equal(status.ErrorVariant))
		Expect(s.Message).To(ContainSubstring("Failed to start calculation for"))
		Expect(s.Description).To(ContainSubstring("cannot perform add operation between nil and nil"))
	})

	It("Should allow the caller to update a calculation", func() {
		baseCH := channel.Channel{
			Name:     "base",
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(dist.Channel.Create(ctx, &baseCH)).To(Succeed())
		calculatedCH := channel.Channel{
			Name:        "calculated",
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: cluster.Free,
			Requires:    []channel.Key{baseCH.Key()},
			Expression:  "return base * 2",
		}
		Expect(dist.Channel.Create(ctx, &calculatedCH)).To(Succeed())
		MustSucceed(c.Request(ctx, calculatedCH.Key()))
		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()
		w := MustSucceed(dist.Framer.OpenWriter(
			ctx,
			framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{baseCH.Key()},
			},
		))
		streamer := MustSucceed(
			dist.Framer.NewStreamer(
				ctx,
				framer.StreamerConfig{
					Keys: []channel.Key{calculatedCH.Key()},
				},
			),
		)
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		time.Sleep(sleepInterval)
		MustSucceed(w.Write(core.UnaryFrame(baseCH.Key(), telem.NewSeriesV[int64](1, 2))))
		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calculatedCH.Key()}))
		series := res.Frame.SeriesAt(0)
		Expect(telem.ValueAt[int64](series, 0)).To(Equal(int64(2)))
		Expect(telem.ValueAt[int64](series, 1)).To(Equal(int64(4)))

		calculatedCH.Expression = "return base * 3"
		Expect(dist.Channel.Create(ctx, &calculatedCH)).To(Succeed())
		time.Sleep(sleepInterval)
		MustSucceed(w.Write(core.UnaryFrame(baseCH.Key(), telem.NewSeriesV[int64](1, 2))))
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calculatedCH.Key()}))
		series = res.Frame.SeriesAt(0)
		Expect(telem.ValueAt[int64](series, 0)).To(Equal(int64(3)))
		Expect(telem.ValueAt[int64](series, 1)).To(Equal(int64(6)))
		Expect(w.Close()).To(Succeed())
	})
})
