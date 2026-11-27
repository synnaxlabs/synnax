// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Time", func() {
	var (
		node  testutil.Node
		index channel.Channel
		data  channel.Channel
		both  channel.Keys
	)
	BeforeEach(func(ctx SpecContext) {
		node = testutil.NewNode(ctx)
		index = MustSucceed(node.CreateChannels(ctx, []channel.Channel{{
			Name:     "time_index",
			IsIndex:  true,
			DataType: telem.TimestampT,
		}}))[0]
		data = MustSucceed(node.CreateChannels(ctx, []channel.Channel{{
			Name:       "time_data",
			DataType:   telem.Float32T,
			LocalIndex: index.LocalKey,
		}}))[0]
		both = channel.Keys{index.Key(), data.Key()}
	})

	// stream opens a streamer on cfg and waits for its open ack.
	stream := func(
		cfg relay.StreamerConfig,
	) (confluence.Inlet[relay.Request], confluence.Outlet[relay.Response]) {
		GinkgoHelper()
		cfg.SendOpenAck = new(true)
		s := MustSucceed(node.Framer.NewStreamer(cfg))
		sCtx, cancel := signal.Isolated()
		req, res := confluence.Attach(s, 10)
		s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		DeferCleanup(func() {
			req.Close()
			cancel()
			confluence.Drain(res)
		})
		var ack relay.Response
		Eventually(res.Outlet()).Should(Receive(&ack))
		Expect(ack.Frame.Empty()).To(BeTrue())
		return req, res
	}

	// write writes the stamps and the values to index and data through w.
	write := func(w *writer.Writer, stamps telem.Series) {
		GinkgoHelper()
		values := telem.NewSeriesV[float32](make([]float32, stamps.Len())...)
		Expect(w.Write(frame.NewMulti(both, []telem.Series{stamps, values}))).
			To(BeTrue())
	}

	// receive returns the next frame that res sends.
	receive := func(res confluence.Outlet[relay.Response]) frame.Frame {
		GinkgoHelper()
		var r relay.Response
		Eventually(res.Outlet()).Should(Receive(&r))
		return r.Frame
	}

	It("Should send the time of a data channel that the streamer requests alone", func(
		ctx SpecContext,
	) {
		_, res := stream(relay.StreamerConfig{Keys: channel.Keys{data.Key()}})
		w := MustOpen(node.Framer.OpenWriter(ctx, writer.Config{
			Keys:  both,
			Start: 10 * telem.SecondTS,
		}))
		write(w, telem.NewSeriesSecondsTSV(10, 11, 12))
		fr := receive(res)
		Expect(fr.KeysSlice()).To(ConsistOf(index.Key(), data.Key()))
		stamps := fr.Get(index.Key()).Series[0]
		values := fr.Get(data.Key()).Series[0]
		Expect(stamps.Data).To(Equal(telem.NewSeriesSecondsTSV(10, 11, 12).Data))
		Expect(stamps.Clock).To(Equal(uint32(index.Key())))
		Expect(values.Clock).To(Equal(uint32(index.Key())))
		Expect(stamps.AlignmentBounds()).To(Equal(values.AlignmentBounds()))
	})

	It("Should send no time for an untimed channel", func(ctx SpecContext) {
		_, res := stream(relay.StreamerConfig{
			Keys:        channel.Keys{data.Key()},
			UntimedKeys: channel.Keys{data.Key()},
		})
		w := MustOpen(node.Framer.OpenWriter(ctx, writer.Config{
			Keys:  both,
			Start: 10 * telem.SecondTS,
		}))
		write(w, telem.NewSeriesSecondsTSV(10, 11))
		Expect(receive(res).KeysSlice()).To(Equal([]channel.Key{data.Key()}))
	})

	It("Should send the time once when the streamer also requests the index", func(
		ctx SpecContext,
	) {
		_, res := stream(relay.StreamerConfig{Keys: both})
		w := MustOpen(node.Framer.OpenWriter(ctx, writer.Config{
			Keys:  both,
			Start: 10 * telem.SecondTS,
		}))
		write(w, telem.NewSeriesSecondsTSV(10, 11))
		Expect(receive(res).KeysSlice()).To(ConsistOf(index.Key(), data.Key()))
	})

	It("Should send time once a request clears the untimed keys", func(
		ctx SpecContext,
	) {
		req, res := stream(relay.StreamerConfig{
			Keys:        channel.Keys{data.Key()},
			UntimedKeys: channel.Keys{data.Key()},
		})
		req.Inlet() <- relay.Request{Keys: channel.Keys{data.Key()}, AckRequested: true}
		var ack relay.Response
		Eventually(res.Outlet()).Should(Receive(&ack))
		Expect(ack.Ack).To(BeTrue())
		w := MustOpen(node.Framer.OpenWriter(ctx, writer.Config{
			Keys:  both,
			Start: 10 * telem.SecondTS,
		}))
		write(w, telem.NewSeriesSecondsTSV(10, 11))
		Expect(receive(res).KeysSlice()).To(ConsistOf(index.Key(), data.Key()))
	})

	It("Should send no time once a request marks the channel untimed", func(
		ctx SpecContext,
	) {
		req, res := stream(relay.StreamerConfig{Keys: channel.Keys{data.Key()}})
		req.Inlet() <- relay.Request{
			Keys:         channel.Keys{data.Key()},
			UntimedKeys:  channel.Keys{data.Key()},
			AckRequested: true,
		}
		var ack relay.Response
		Eventually(res.Outlet()).Should(Receive(&ack))
		Expect(ack.Ack).To(BeTrue())
		w := MustOpen(node.Framer.OpenWriter(ctx, writer.Config{
			Keys:  both,
			Start: 10 * telem.SecondTS,
		}))
		write(w, telem.NewSeriesSecondsTSV(10, 11))
		Expect(receive(res).KeysSlice()).To(Equal([]channel.Key{data.Key()}))
	})

	It("Should never send time with no data beside it", func(ctx SpecContext) {
		_, res := stream(relay.StreamerConfig{Keys: channel.Keys{data.Key()}})
		indexOnly := MustOpen(node.Framer.OpenWriter(ctx, writer.Config{
			Keys:  channel.Keys{index.Key()},
			Start: 10 * telem.SecondTS,
		}))
		Expect(indexOnly.Write(frame.NewUnary(
			index.Key(),
			telem.NewSeriesSecondsTSV(10, 11),
		))).To(BeTrue())
		Expect(indexOnly.Close()).To(Succeed())
		w := MustOpen(node.Framer.OpenWriter(ctx, writer.Config{
			Keys:  both,
			Start: 20 * telem.SecondTS,
		}))
		write(w, telem.NewSeriesSecondsTSV(20, 21))
		fr := receive(res)
		Expect(fr.KeysSlice()).To(ConsistOf(index.Key(), data.Key()))
		Expect(fr.Get(index.Key()).Series[0].Data).
			To(Equal(telem.NewSeriesSecondsTSV(20, 21).Data))
	})

	It("Should send the time of a data channel after a control handoff", func(
		ctx SpecContext,
	) {
		_, res := stream(relay.StreamerConfig{Keys: channel.Keys{data.Key()}})
		first := MustOpen(node.Framer.OpenWriter(ctx, writer.Config{
			Keys:           both,
			Start:          10 * telem.SecondTS,
			Authorities:    []control.Authority{100},
			ControlSubject: control.Subject{Name: "first", Key: "first"},
		}))
		write(first, telem.NewSeriesSecondsTSV(10, 11))
		Expect(receive(res).KeysSlice()).To(ConsistOf(index.Key(), data.Key()))
		second := MustOpen(node.Framer.OpenWriter(ctx, writer.Config{
			Keys:           both,
			Start:          20 * telem.SecondTS,
			Authorities:    []control.Authority{200},
			ControlSubject: control.Subject{Name: "second", Key: "second"},
		}))
		write(second, telem.NewSeriesSecondsTSV(20, 21))
		fr := receive(res)
		Expect(fr.KeysSlice()).To(ConsistOf(index.Key(), data.Key()))
		Expect(fr.Get(index.Key()).Series[0].Data).
			To(Equal(telem.NewSeriesSecondsTSV(20, 21).Data))
	})
})
