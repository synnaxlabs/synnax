package cdc_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/cdc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Observable", func() {
	It("Should correctly propagate a change", func() {
		obs := observe.New[[]change.Change[[]byte, struct{}]]()
		cfg := cdc.ObservableConfig{
			Set: channel.Channel{
				Name:     "observable_set",
				DataType: telem.UUIDT,
			},
			Delete: channel.Channel{
				Name:     "observable_delete",
				DataType: telem.UUIDT,
			},
			Observable: obs,
		}

		closer := MustSucceed(dist.CDC.SubscribeToObservable(ctx, cfg))
		defer func() {
			Expect(closer.Close()).To(Succeed())
		}()

		var setCh channel.Channel
		Expect(dist.Channel.NewRetrieve().
			WhereNames("observable_set").
			Entry(&setCh).
			Exec(ctx, nil),
		).To(Succeed())
		streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys:  channel.Keys{setCh.Key()},
			Start: telem.Now(),
		}))
		req, res := confluence.Attach(streamer, 1)
		sCtx, cancel := signal.Isolated()
		defer cancel()
		defer req.Close()
		streamer.Flow(sCtx)

		uid := uuid.New()
		obs.Notify(ctx, []change.Change[[]byte, struct{}]{
			{
				Variant: change.Set,
				Key:     uid[:],
			},
		})

		streamRes := <-res.Outlet()
		Expect(streamRes.Frame.Keys).To(ConsistOf(setCh.Key()))
		Expect(streamRes.Frame.Series[0].Data).To(HaveLen(int(telem.Bit128)))
		Expect(streamRes.Frame.Series[0].Data).To(Equal(uid[:]))
	})
})
