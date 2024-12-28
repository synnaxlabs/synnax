package calculated_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/synnaxlabs/computron"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculated"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"time"
)

var _ = Describe("Calculated", func() {
	It("Output a basic calculation", func() {
		distB := mock.NewBuilder()
		dist := distB.New(ctx)
		computer := MustSucceed(computron.New())
		c := MustSucceed(calculated.Open(calculated.Config{
			Instrumentation: Instrumentation("calculated", InstrumentationConfig{Log: config.True()}),
			Computron:       computer,
			Framer:          dist.Framer,
			Channel:         dist.Channel,
		}))

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
			Expression:  "result = base * 2",
		}
		logrus.Info(calculatedCH, baseCH)
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
		time.Sleep(10 * time.Millisecond)
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
	FIt("Divide by zero", func() {
		distB := mock.NewBuilder()
		dist := distB.New(ctx)
		computer := MustSucceed(computron.New())
		c := MustSucceed(calculated.Open(calculated.Config{
			Instrumentation: Instrumentation("calculated", InstrumentationConfig{Log: config.True()}),
			Computron:       computer,
			Framer:          dist.Framer,
			Channel:         dist.Channel,
		}))
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
		logrus.Info(calculatedCH, baseCH)
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
		time.Sleep(100 * time.Millisecond)
		wInlet.Inlet() <- framer.WriterRequest{
			Command: writer.Data,
			Frame: framer.Frame{
				Keys:   channel.Keys{baseCH.Key()},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2)},
			},
		}
		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet()).Should(Receive(&res))
		Expect(res.Frame.Keys).To(Equal(channel.Keys{calculatedCH.Key()}))
	})

	It("Handle undefined symbols", func() {
		distB := mock.NewBuilder()
		dist := distB.New(ctx)
		computer := MustSucceed(computron.New())
		c := MustSucceed(calculated.Open(calculated.Config{
			Instrumentation: Instrumentation("calculated", InstrumentationConfig{Log: config.True()}),
			Computron:       computer,
			Framer:          dist.Framer,
			Channel:         dist.Channel,
		}))
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
			Expression:  "result = base * fake",
		}
		logrus.Info(calculatedCH, baseCH)
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
		wInlet.Inlet() <- framer.WriterRequest{
			Command: writer.Data,
			Frame: framer.Frame{
				Keys:   channel.Keys{baseCH.Key()},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2)},
			},
		}
		var res framer.StreamerResponse
		Eventually(sOutlet.Outlet()).Should(Receive(&res))
		Expect(res.Frame.Keys).To(Equal(channel.Keys{calculatedCH.Key()}))
	})
})
