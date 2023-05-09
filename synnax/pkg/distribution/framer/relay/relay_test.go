package relay_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"time"
)

var _ = Describe("Relay", func() {
	Describe("Relay", func() {
		It("Should work", func() {
			builder, services := provision(1)

			ch := &channel.Channel{
				Leaseholder: 1,
				DataType:    telem.Int64T,
				Rate:        1 * telem.Hz,
			}
			Expect(services[1].channel.NewWriter(nil).Create(ctx, ch)).To(Succeed())

			r := services[1].relay
			reader := r.NewReader(ch.Key())
			sCtx, cancel := signal.Isolated(signal.WithInstrumentation(ins))
			defer cancel()
			reader.Flow(sCtx)

			requests := confluence.NewStream[relay.Request](1)
			reader.InFrom(requests)
			frames := confluence.NewStream[framer.Frame](1)
			reader.OutTo(frames)

			d := telem.NewArrayV[int64](1)

			r.Writes().Inlet() <- framer.NewFrame(channel.Keys{ch.Key()}, []telem.Array{d})

			f := <-frames.Outlet()
			Expect(f.Len()).To(Equal(int64(1)))

			time.Sleep(5 * time.Second)

			Expect(builder.Close()).To(Succeed())
		})
	})
})
