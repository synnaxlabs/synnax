package cesium_test

import (
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Control", Ordered, func() {
	var db *cesium.DB
	BeforeAll(func() { db = openMemDB() })
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Single Channel, Two Writer Contention", func() {
		It("Should work", func() {
			var ch1 cesium.ChannelKey = 1
			Expect(db.CreateChannel(ctx, cesium.Channel{Key: ch1, DataType: telem.Int16T, Rate: 1 * telem.Hz})).To(Succeed())
			start := telem.SecondTS * 10
			w1 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{
				Name:               "Writer One",
				Start:              start,
				Channels:           []cesium.ChannelKey{ch1},
				Authorities:        []control.Authority{control.Absolute - 2},
				SendControlDigests: config.True(),
			}))
			w2 := MustSucceed(db.NewStreamWriter(ctx, cesium.WriterConfig{
				Start:              start,
				Name:               "Writer Two",
				Channels:           []cesium.ChannelKey{ch1},
				Authorities:        []control.Authority{control.Absolute - 2},
				SendControlDigests: config.True(),
			}))
			ctx, cancel := signal.Isolated()
			defer cancel()
			w1In, w1Out := confluence.Attach(w1, 2)
			w2In, w2Out := confluence.Attach(w2, 2)
			w1.Flow(ctx)
			w2.Flow(ctx)
			w1In.Inlet() <- cesium.WriterRequest{
				Command: cesium.WriterWrite,
				Frame: core.NewFrame(
					[]cesium.ChannelKey{ch1},
					[]telem.Series{telem.NewSeriesV[int16](1, 2, 3)},
				),
			}
			w2In.Inlet() <- cesium.WriterRequest{
				Command: cesium.WriterWrite,
				Frame: core.NewFrame(
					[]cesium.ChannelKey{ch1},
					[]telem.Series{telem.NewSeriesV[int16](4, 5, 6)},
				),
			}
			r := <-w2Out.Outlet()
			Expect(r.Variant).To(Equal(cesium.WriterResponseAck))
			Expect(r.Ack).To(BeFalse())
			w2In.Inlet() <- cesium.WriterRequest{
				Command: cesium.WriterError,
			}
			r = <-w2Out.Outlet()
			Expect(r.Variant).To(Equal(cesium.WriterResponseAck))
			Expect(errors.Is(r.Err, controller.Unauthorized("Writer Two", ch1))).To(BeTrue())
			w2In.Inlet() <- cesium.WriterRequest{
				Command: cesium.WriterSetAuthority,
				Config: cesium.WriterConfig{
					Authorities: []control.Authority{control.Absolute - 1},
				},
			}
			r = <-w1Out.Outlet()
			Expect(r.Variant).To(Equal(cesium.WriterResponseControl))
			Expect(r.Control.Name).To(Equal("Writer Two"))
			Expect(r.Control.Authority).To(Equal(control.Absolute - 1))
			w2In.Inlet() <- cesium.WriterRequest{
				Command: cesium.WriterWrite,
				Frame: core.NewFrame(
					[]cesium.ChannelKey{ch1},
					[]telem.Series{telem.NewSeriesV[int16](4, 5, 6)},
				),
			}
			w2In.Inlet() <- cesium.WriterRequest{
				Command: cesium.WriterCommit,
			}
			r = <-w2Out.Outlet()
			Expect(r.Variant).To(Equal(cesium.WriterResponseAck))
			Expect(r.Ack).To(BeTrue())
			w1In.Close()
			w2In.Close()
			Expect(ctx.Wait()).To(Succeed())

			f := MustSucceed(db.Read(
				ctx,
				start.SpanRange(10*telem.Second),
				ch1,
			))
			Expect(f.Series).To(HaveLen(1))
		})
	})
})
