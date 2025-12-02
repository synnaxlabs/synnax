// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	calculation "github.com/synnaxlabs/synnax/pkg/service/framer/calculation/legacy"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/status"
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
			Framer:  dist.Framer,
			Channel: dist.Channel,
		}))
	})

	AfterAll(func() {
		Expect(c.Close()).To(Succeed())
		Expect(dist.Close()).To(Succeed())
	})

	It("Output a basic calculation", func() {
		baseCh := channel.Channel{
			Name:     channel.NewRandomName(),
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(dist.Channel.Create(ctx, &baseCh)).To(Succeed())
		calculatedCh := channel.Channel{
			Name:        channel.NewRandomName(),
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: cluster.Free,
			Requires:    []channel.Key{baseCh.Key()},
			Expression:  fmt.Sprintf("return %s * 2", baseCh.Name),
		}
		Expect(dist.Channel.Create(ctx, &calculatedCh)).To(Succeed())
		Expect(c.Add(ctx, calculatedCh.Key())).To(Succeed())
		defer func() { Expect(c.Remove(ctx, calculatedCh.Key())).To(Succeed()) }()
		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()
		w := MustSucceed(dist.Framer.OpenWriter(
			ctx,
			framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{baseCh.Key()},
			},
		))
		streamer := MustSucceed(
			dist.Framer.NewStreamer(
				ctx,
				framer.StreamerConfig{
					Keys: []channel.Key{calculatedCh.Key()},
				},
			),
		)
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		time.Sleep(sleepInterval)
		MustSucceed(w.Write(core.UnaryFrame(baseCh.Key(), telem.NewSeriesV[int64](1, 2))))
		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calculatedCh.Key()}))
		Expect(w.Close()).To(Succeed())
	})

	It("Handle undefined symbols", func() {
		baseCh := channel.Channel{
			Name:     channel.NewRandomName(),
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(dist.Channel.Create(ctx, &baseCh)).To(Succeed())
		calculatedCh := channel.Channel{
			Name:        channel.NewRandomName(),
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: cluster.Free,
			Requires:    []channel.Key{baseCh.Key()},
			Expression:  fmt.Sprintf("return %s * fake", baseCh.Name),
		}
		Expect(dist.Channel.Create(ctx, &calculatedCh)).To(Succeed())
		Expect(c.Add(ctx, calculatedCh.Key())).To(Succeed())
		defer func() { Expect(c.Remove(ctx, calculatedCh.Key())).To(Succeed()) }()
		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()
		w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
			Start: telem.Now(),
			Keys:  []channel.Key{baseCh.Key()},
		}))
		streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys:        []channel.Key{calculatedCh.Key()},
			SendOpenAck: config.True(),
		}))
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive())
		MustSucceed(w.Write(core.UnaryFrame(baseCh.Key(), telem.NewSeriesV[int64](1, 2))))
		Consistently(sOutlet.Outlet(), 500*time.Millisecond).ShouldNot(Receive())
		Expect(w.Close()).To(Succeed())
	})

	It("Return a warning for dividing by zero", func() {
		baseCh := channel.Channel{
			Name:     channel.NewRandomName(),
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(dist.Channel.Create(ctx, &baseCh)).To(Succeed())
		calculatedCh := channel.Channel{
			Name:        channel.NewRandomName(),
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: cluster.Free,
			Requires:    []channel.Key{baseCh.Key()},
			Expression:  fmt.Sprintf("return %s / 0", baseCh.Name),
		}
		Expect(dist.Channel.Create(ctx, &calculatedCh)).To(Succeed())
		Expect(c.Add(ctx, calculatedCh.Key())).To(Succeed())
		defer func() { Expect(c.Remove(ctx, calculatedCh.Key())).To(Succeed()) }()
		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()
		w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
			Start: telem.Now(),
			Keys:  []channel.Key{baseCh.Key()},
		}))
		streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys:        []channel.Key{calculatedCh.Key()},
			SendOpenAck: config.True(),
		}))
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive())
		MustSucceed(w.Write(core.UnaryFrame(
			baseCh.Key(),
			telem.NewSeriesV[int64](1, 2),
		)))
		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calculatedCh.Key()}))
	})

	It("Should handle nested calculations", func() {
		baseCh := channel.Channel{
			Name:     channel.NewRandomName(),
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(dist.Channel.Create(ctx, &baseCh)).To(Succeed())

		// First calculated channel that doubles the base value
		calc1Ch := channel.Channel{
			Name:        channel.NewRandomName(),
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: cluster.Free,
			Requires:    []channel.Key{baseCh.Key()},
			Expression:  fmt.Sprintf("return %s * 2", baseCh.Name),
		}
		Expect(dist.Channel.Create(ctx, &calc1Ch)).To(Succeed())

		// Second calculated channel that adds 1 to the first calculated channel
		calc2Ch := channel.Channel{
			Name:        channel.NewRandomName(),
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: cluster.Free,
			Requires:    []channel.Key{calc1Ch.Key()},
			Expression:  fmt.Sprintf("return %s + 1", calc1Ch.Name),
		}
		Expect(dist.Channel.Create(ctx, &calc2Ch)).To(Succeed())

		Expect(c.Add(ctx, calc1Ch.Key())).To(Succeed())
		defer func() { Expect(c.Remove(ctx, calc1Ch.Key())).To(Succeed()) }()
		Expect(c.Add(ctx, calc2Ch.Key())).To(Succeed())
		defer func() { Expect(c.Remove(ctx, calc2Ch.Key())).To(Succeed()) }()

		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()

		w := MustSucceed(dist.Framer.OpenWriter(ctx,
			framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{baseCh.Key()},
			},
		))
		streamer := MustSucceed(
			dist.Framer.NewStreamer(
				ctx,
				framer.StreamerConfig{
					Keys:        []channel.Key{calc2Ch.Key()},
					SendOpenAck: config.True(),
				},
			),
		)
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive())

		MustSucceed(w.Write(core.UnaryFrame(baseCh.Key(), telem.NewSeriesV[int64](1, 2))))
		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calc2Ch.Key()}))

		// For base values [1, 2]:
		// calc1 should be [2, 4] (base * 2)
		// calc2 should be [3, 5] (calc1 + 1)
		series := res.Frame.SeriesAt(0)
		Expect(series.Len()).To(Equal(int64(2)))
		Expect(telem.ValueAt[int64](series, 0)).To(Equal(int64(3)))
		Expect(telem.ValueAt[int64](series, 1)).To(Equal(int64(5)))
	})

	It("Should error when calculating with undefined channels in expression", func() {
		baseCh := channel.Channel{
			Name:     channel.NewRandomName(),
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(dist.Channel.Create(ctx, &baseCh)).To(Succeed())

		calculatedCh := channel.Channel{
			Name:        channel.NewRandomName(),
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: cluster.Free,
			Requires:    []channel.Key{baseCh.Key()},
			Expression:  "return fake1 + fake2",
		}
		Expect(dist.Channel.Create(ctx, &calculatedCh)).To(Succeed())

		var statusCh channel.Channel
		statusCh.Name = calculation.StatusChannelName
		Expect(
			dist.Channel.Create(
				ctx,
				&statusCh,
				channel.RetrieveIfNameExists(),
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
					Keys:  []channel.Key{baseCh.Key()},
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
		Expect(c.Add(ctx, calculatedCh.Key())).To(Succeed())
		defer func() { Expect(c.Remove(ctx, calculatedCh.Key())).To(Succeed()) }()
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive())

		time.Sleep(5 * time.Millisecond)
		MustSucceed(w.Write(core.UnaryFrame(
			baseCh.Key(),
			telem.NewSeriesV[int64](1, 2),
		)))

		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.SeriesAt(0).DataType).To(Equal(telem.JSONT))

		var s calculation.Status
		data := res.Frame.SeriesAt(0).Data
		Expect(json.Unmarshal(data[:len(data)-1], &s)).To(Succeed()) // -1 to remove newline

		Expect(s.Key).To(Equal(calculatedCh.Key().String()))
		Expect(s.Variant).To(Equal(status.ErrorVariant))
		Expect(s.Message).To(ContainSubstring("Failed to start calculation for"))
		Expect(s.Description).To(ContainSubstring("cannot perform add operation between nil and nil"))
	})

	It("Should allow the caller to update a calculation", func() {
		baseCh := channel.Channel{
			Name:     channel.NewRandomName(),
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(dist.Channel.Create(ctx, &baseCh)).To(Succeed())
		calculatedCh := channel.Channel{
			Name:        channel.NewRandomName(),
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: cluster.Free,
			Requires:    []channel.Key{baseCh.Key()},
			Expression:  fmt.Sprintf("return %s * 2", baseCh.Name),
		}
		Expect(dist.Channel.Create(ctx, &calculatedCh)).To(Succeed())
		Expect(c.Add(ctx, calculatedCh.Key())).To(Succeed())
		defer func() { Expect(c.Remove(ctx, calculatedCh.Key())).To(Succeed()) }()
		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()
		w := MustSucceed(dist.Framer.OpenWriter(
			ctx,
			framer.WriterConfig{
				Start: telem.Now(),
				Keys:  []channel.Key{baseCh.Key()},
			},
		))
		streamer := MustSucceed(
			dist.Framer.NewStreamer(
				ctx,
				framer.StreamerConfig{
					Keys: []channel.Key{calculatedCh.Key()},
				},
			),
		)
		_, sOutlet := confluence.Attach(streamer, 1, 1)
		streamer.Flow(sCtx)
		time.Sleep(sleepInterval)
		MustSucceed(w.Write(core.UnaryFrame(baseCh.Key(), telem.NewSeriesV[int64](1, 2))))
		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calculatedCh.Key()}))
		series := res.Frame.SeriesAt(0)
		Expect(telem.ValueAt[int64](series, 0)).To(Equal(int64(2)))
		Expect(telem.ValueAt[int64](series, 1)).To(Equal(int64(4)))

		calculatedCh.Expression = fmt.Sprintf("return %s * 3", baseCh.Name)
		Expect(dist.Channel.Create(ctx, &calculatedCh)).To(Succeed())
		c.Update(ctx, calculatedCh)
		time.Sleep(sleepInterval)
		MustSucceed(w.Write(core.UnaryFrame(baseCh.Key(), telem.NewSeriesV[int64](1, 2))))
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(Equal([]channel.Key{calculatedCh.Key()}))
		series = res.Frame.SeriesAt(0)
		Expect(telem.ValueAt[int64](series, 0)).To(Equal(int64(3)))
		Expect(telem.ValueAt[int64](series, 1)).To(Equal(int64(6)))
		Expect(w.Close()).To(Succeed())
	})
})
