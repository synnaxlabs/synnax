package calculated_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculated"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var sleepInterval = 25 * time.Millisecond

var _ = Describe("Calculated", Ordered, func() {
	var (
		c    *calculated.Service
		dist distribution.Distribution
	)

	BeforeAll(func() {
		distB := mock.NewBuilder()
		dist = distB.New(ctx)
		c = MustSucceed(calculated.Open(calculated.Config{
			Computron:         interpreter,
			Framer:            dist.Framer,
			Channel:           dist.Channel,
			ChannelObservable: dist.Channel.NewObservable(),
		}))
	})

	AfterAll(func() {
		Expect(c.Close()).To(Succeed())
	})

	It("Output a basic calculation", func() {
		baseCH := channel.Channel{Name: "base", DataType: telem.Int64T, Virtual: true}
		Expect(dist.Channel.Create(ctx, &baseCH)).To(Succeed())
		calculatedCH := channel.Channel{
			Name:        "calculated",
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: core.Free,
			Requires:    []channel.Key{baseCH.Key()},
			Expression:  "result = base * 2",
		}
		Expect(dist.Channel.Create(ctx, &calculatedCH)).To(Succeed())
		MustSucceed(c.Request(ctx, calculatedCH.Key()))
		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()
		w := MustSucceed(dist.Framer.NewStreamWriter(ctx, framer.WriterConfig{
			Start: telem.Now(),
			Keys:  []channel.Key{baseCH.Key()},
		}))
		wInlet, _ := confluence.Attach[framer.WriterRequest, framer.WriterResponse](w, 1, 1)
		w.Flow(sCtx)
		streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys: []channel.Key{calculatedCH.Key()},
		}))
		_, sOutlet := confluence.Attach[framer.StreamerRequest, framer.StreamerResponse](streamer, 1, 1)
		streamer.Flow(sCtx)
		time.Sleep(sleepInterval)
		wInlet.Inlet() <- framer.WriterRequest{
			Command: writer.Data,
			Frame: framer.Frame{
				Keys:   channel.Keys{baseCH.Key()},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2)},
			},
		}
		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.Keys).To(Equal(channel.Keys{calculatedCH.Key()}))
	})

	It("Handle undefined symbols", func() {
		baseCH := channel.Channel{Name: "base", DataType: telem.Int64T, Virtual: true}
		Expect(dist.Channel.Create(ctx, &baseCH)).To(Succeed())
		calculatedCH := channel.Channel{
			Name:        "calculated",
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: core.Free,
			Requires:    []channel.Key{baseCH.Key()},
			Expression:  "result = base * fake",
		}
		Expect(dist.Channel.Create(ctx, &calculatedCH)).To(Succeed())
		MustSucceed(c.Request(ctx, calculatedCH.Key()))
		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()
		w := MustSucceed(dist.Framer.NewStreamWriter(ctx, framer.WriterConfig{
			Start: telem.Now(),
			Keys:  []channel.Key{baseCH.Key()},
		}))
		wInlet, _ := confluence.Attach[framer.WriterRequest, framer.WriterResponse](w, 1, 1)
		w.Flow(sCtx)
		streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys: []channel.Key{calculatedCH.Key()},
		}))
		_, sOutlet := confluence.Attach[framer.StreamerRequest, framer.StreamerResponse](streamer, 1, 1)
		streamer.Flow(sCtx)
		time.Sleep(sleepInterval)
		wInlet.Inlet() <- framer.WriterRequest{
			Command: writer.Data,
			Frame: framer.Frame{
				Keys:   channel.Keys{baseCH.Key()},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2)},
			},
		}
		Consistently(sOutlet.Outlet(), 500*time.Millisecond).ShouldNot(Receive())
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
			Leaseholder: core.Free,
			Requires:    []channel.Key{baseCH.Key()},
			Expression:  "result = base / 0",
		}
		Expect(dist.Channel.Create(ctx, &calculatedCH)).To(Succeed())
		MustSucceed(c.Request(ctx, calculatedCH.Key()))
		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()
		w := MustSucceed(dist.Framer.NewStreamWriter(ctx, framer.WriterConfig{
			Start: telem.Now(),
			Keys:  []channel.Key{baseCH.Key()},
		}))
		wInlet, _ := confluence.Attach[framer.WriterRequest, framer.WriterResponse](w, 1, 1)
		w.Flow(sCtx)
		streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys: []channel.Key{calculatedCH.Key()},
		}))
		_, sOutlet := confluence.Attach[framer.StreamerRequest, framer.StreamerResponse](streamer, 1, 1)
		streamer.Flow(sCtx)
		time.Sleep(sleepInterval)
		wInlet.Inlet() <- framer.WriterRequest{
			Command: writer.Data,
			Frame: framer.Frame{
				Keys:   channel.Keys{baseCH.Key()},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2)},
			},
		}
		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.Keys).To(Equal(channel.Keys{calculatedCH.Key()}))
	})

	It("Should handle nested calculations", func() {
		baseCH := channel.Channel{Name: "base", DataType: telem.Int64T, Virtual: true}
		Expect(dist.Channel.Create(ctx, &baseCH)).To(Succeed())

		// First calculated channel that doubles the base value
		calc1CH := channel.Channel{
			Name:        "calc1",
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: core.Free,
			Requires:    []channel.Key{baseCH.Key()},
			Expression:  "result = base * 2",
		}
		Expect(dist.Channel.Create(ctx, &calc1CH)).To(Succeed())

		// Second calculated channel that adds 1 to the first calculated channel
		calc2CH := channel.Channel{
			Name:        "calc2",
			DataType:    telem.Int64T,
			Virtual:     true,
			Leaseholder: core.Free,
			Requires:    []channel.Key{calc1CH.Key()},
			Expression:  "result = calc1 + 1",
		}
		Expect(dist.Channel.Create(ctx, &calc2CH)).To(Succeed())

		MustSucceed(c.Request(ctx, calc1CH.Key()))
		MustSucceed(c.Request(ctx, calc2CH.Key()))

		sCtx, cancel := signal.WithCancel(ctx)
		defer cancel()

		w := MustSucceed(dist.Framer.NewStreamWriter(ctx, framer.WriterConfig{
			Start: telem.Now(),
			Keys:  []channel.Key{baseCH.Key()},
		}))
		wInlet, _ := confluence.Attach[framer.WriterRequest, framer.WriterResponse](w, 1, 1)
		w.Flow(sCtx)

		streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys: []channel.Key{calc2CH.Key()},
		}))
		_, sOutlet := confluence.Attach[framer.StreamerRequest, framer.StreamerResponse](streamer, 1, 1)
		streamer.Flow(sCtx)

		time.Sleep(sleepInterval)

		// Write base values [1, 2]
		wInlet.Inlet() <- framer.WriterRequest{
			Command: writer.Data,
			Frame: framer.Frame{
				Keys:   channel.Keys{baseCH.Key()},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2)},
			},
		}

		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.Keys).To(Equal(channel.Keys{calc2CH.Key()}))

		// For base values [1, 2]:
		// calc1 should be [2, 4] (base * 2)
		// calc2 should be [3, 5] (calc1 + 1)
		series := res.Frame.Series[0]
		Expect(series.Len()).To(Equal(int64(2)))
		Expect(telem.ValueAt[int64](series, 0)).To(Equal(int64(3)))
		Expect(telem.ValueAt[int64](series, 1)).To(Equal(int64(5)))
	})
})
