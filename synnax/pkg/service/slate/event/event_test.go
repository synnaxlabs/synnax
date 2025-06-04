package event_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/service/slate/event"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"

	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	. "github.com/synnaxlabs/x/testutil"
)

var ctx = context.Background()

var _ = Describe("Event", Ordered, func() {
	Describe("New", func() {
		var (
			c    *calculation.Service
			dist distribution.Distribution
		)

		BeforeAll(func() {
			distB := mock.NewBuilder()
			dist = distB.New(ctx)
			c = MustSucceed(calculation.Open(ctx, calculation.Config{
				Framer:            dist.Framer,
				Channel:           dist.Channel,
				ChannelObservable: dist.Channel.NewObservable(),
			}))
		})

		AfterAll(func() {
			Expect(c.Close()).To(Succeed())
		})

		It("Should run a flow", func() {
			sourceCh := &channel.Channel{
				Name:     "source",
				DataType: telem.Int64T,
				Virtual:  true,
			}
			w := dist.Channel.NewWriter(nil)
			Expect(w.Create(ctx, sourceCh)).To(Succeed())
			sinkCh := &channel.Channel{
				Name:     "sink",
				DataType: telem.Uint8T,
				Virtual:  true,
			}
			Expect(w.Create(ctx, sinkCh)).To(Succeed())
			g := spec.Graph{
				Nodes: []spec.Node{
					{
						Type: "cat",
						Key:  "source",
						Data: map[string]any{
							"channel": uint32(sourceCh.Key()),
						},
					},
					{
						Type: "constant",
						Key:  "constant",
						Data: map[string]any{
							"data_type": "int64",
							"value":     15,
						},
					},
					{
						Type: "operator.gte",
						Key:  "operator.gte",
					},
					{
						Type: "sink",
						Key:  "sink",
						Data: map[string]any{
							"channel": uint32(sinkCh.Key()),
						},
					},
				},
				Edges: []spec.Edge{
					{
						Source: spec.Handle{Node: "source", Key: "value"},
						Sink:   spec.Handle{Node: "operator.gte", Key: "x"},
					},
					{
						Source: spec.Handle{Node: "constant", Key: "value"},
						Sink:   spec.Handle{Node: "operator.gte", Key: "y"},
					},
					{
						Source: spec.Handle{Node: "operator.gte", Key: "value"},
						Sink:   spec.Handle{Node: "sink", Key: "value"},
					},
				},
			}
			cfg := spec.Config{
				Channel: dist.Channel,
				Framer:  dist.Framer,
			}
			g = MustSucceed(spec.Validate(ctx, cfg, g))
			slate := MustSucceed(event.Create(ctx, cfg, g))

			writer := MustSucceed(dist.Framer.OpenWriter(
				ctx,
				framer.WriterConfig{Keys: []channel.Key{sourceCh.Key()}},
			))
			streamer := MustSucceed(dist.Framer.NewStreamer(
				ctx,
				framer.StreamerConfig{
					Keys: []channel.Key{sinkCh.Key()},
				},
			))
			streamerIn, streamerOut := confluence.Attach[framer.StreamerRequest, framer.StreamerResponse](streamer)
			sCtx, cancel := signal.WithCancel(ctx)
			defer func() {
				cancel()
				streamerIn.Close()
			}()
			slate.Flow(sCtx, confluence.CancelOnFail())
			streamer.Flow(sCtx, confluence.CancelOnFail())
			time.Sleep(500 * time.Millisecond)
			MustSucceed(writer.Write(core.UnaryFrame(
				sourceCh.Key(),
				telem.NewSeriesV[int64](20),
			)))
			var res framer.StreamerResponse
			Eventually(streamerOut.Outlet(), "30s").Should(Receive(&res))
			Expect(res.Frame.Len()).To(Equal(int64(1)))
			Expect(telem.ValueAt[uint8](res.Frame.Get(sinkCh.Key()).Series[0], -1)).To(Equal(uint8(0)))
		})
	})
})
