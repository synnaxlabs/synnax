// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Streamed series", func() {
	var host testutil.Node
	BeforeEach(func(ctx SpecContext) { host = testutil.NewNode(ctx) })
	// stream opens a streamer on keys and returns its outlet after the open ack.
	stream := func(
		ctx SpecContext,
		keys channel.Keys,
	) confluence.Outlet[framer.StreamerResponse] {
		GinkgoHelper()
		streamer := MustSucceed(host.Framer.NewStreamer(framer.StreamerConfig{
			Keys:        keys,
			SendOpenAck: new(true),
		}))
		_, out := confluence.Attach(streamer, 10)
		sCtx, cancel := signal.WithCancel(ctx)
		DeferCleanup(func() {
			cancel()
			Expect(sCtx.Wait()).To(Or(Succeed(), MatchError(context.Canceled)))
		})
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		Eventually(out.Outlet()).Should(Receive())
		return out
	}
	It(
		"Should stamp streamed series with the index series' time range",
		func(ctx SpecContext) {
			idx := MustSucceed(host.CreateChannels(ctx, []channel.Channel{
				{Name: "noether", IsIndex: true, DataType: telem.TimestampT},
			}))[0]
			data := MustSucceed(host.CreateChannels(ctx, []channel.Channel{
				{
					Name:       "germain",
					DataType:   telem.Float32T,
					LocalIndex: idx.LocalKey,
				},
			}))[0]
			keys := channel.Keys{idx.Key(), data.Key()}
			out := stream(ctx, keys)
			w := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
				Keys:  keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			Expect(w.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(10, 11, 12),
				telem.NewSeriesV[float32](1, 2, 3),
			}))).To(BeTrue())
			var res framer.StreamerResponse
			Eventually(out.Outlet()).Should(Receive(&res))
			expected := (10 * telem.SecondTS).Range(12*telem.SecondTS + 1)
			Expect(res.Frame.Get(idx.Key()).Series[0].TimeRange).To(Equal(expected))
			Expect(res.Frame.Get(data.Key()).Series[0].TimeRange).To(Equal(expected))
			Expect(w.Close()).To(Succeed())
		},
	)
	It(
		"Should leave series unstamped when the frame does not carry the index",
		func(ctx SpecContext) {
			idx := MustSucceed(host.CreateChannels(ctx, []channel.Channel{
				{Name: "franklin", IsIndex: true, DataType: telem.TimestampT},
			}))[0]
			data := MustSucceed(host.CreateChannels(ctx, []channel.Channel{
				{
					Name:       "meitner",
					DataType:   telem.Float32T,
					LocalIndex: idx.LocalKey,
				},
			}))[0]
			idxW := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
				Keys:  channel.Keys{idx.Key()},
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			Expect(idxW.Write(frame.NewUnary(
				idx.Key(),
				telem.NewSeriesSecondsTSV(10, 11, 12),
			))).To(BeTrue())
			MustCommit(idxW)
			Expect(idxW.Close()).To(Succeed())
			out := stream(ctx, channel.Keys{data.Key()})
			w := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
				Keys:  channel.Keys{data.Key()},
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			Expect(w.Write(frame.NewUnary(
				data.Key(),
				telem.NewSeriesV[float32](1, 2, 3),
			))).To(BeTrue())
			var res framer.StreamerResponse
			Eventually(out.Outlet()).Should(Receive(&res))
			Expect(res.Frame.Get(data.Key()).Series[0].TimeRange).To(
				Equal(telem.TimeRangeZero),
			)
			Expect(w.Close()).To(Succeed())
		},
	)
	It(
		"Should deliver writes of a virtual channel with its leading alignment",
		func(ctx SpecContext) {
			virtual := MustSucceed(host.CreateChannels(ctx, []channel.Channel{
				{Name: "heisenberg", DataType: telem.Int64T, Virtual: true},
			}))[0]
			Expect(virtual.Key().Free()).To(BeFalse())
			keys := channel.Keys{virtual.Key()}
			out := stream(ctx, keys)
			w := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
				Keys:  keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			for i := range int64(2) {
				Expect(w.Write(frame.NewUnary(
					virtual.Key(),
					telem.NewSeriesV(3*i+1, 3*i+2, 3*i+3),
				))).To(BeTrue())
			}
			for i := range int64(2) {
				var res framer.StreamerResponse
				Eventually(out.Outlet()).Should(Receive(&res))
				Expect(res.Frame.Count()).To(Equal(1))
				s := res.Frame.SeriesAt(0)
				Expect(s).To(telem.MatchSeriesDataV(3*i+1, 3*i+2, 3*i+3))
				Expect(s.Alignment).To(Equal(
					telem.LeadingAlignment(1, uint32(3*i)),
				))
			}
			Consistently(out.Outlet()).ShouldNot(Receive())
			Expect(w.Close()).To(Succeed())
		},
	)
	It(
		"Should deliver every stored frame once and in order",
		func(ctx SpecContext) {
			idx := MustSucceed(host.CreateChannels(ctx, []channel.Channel{
				{Name: "mixed_time", IsIndex: true, DataType: telem.TimestampT},
			}))[0]
			data := MustSucceed(host.CreateChannels(ctx, []channel.Channel{
				{
					Name:       "mixed_data",
					DataType:   telem.Int64T,
					LocalIndex: idx.LocalKey,
				},
			}))[0]
			keys := channel.Keys{idx.Key(), data.Key()}
			out := stream(ctx, keys)
			w := MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
				Keys:  keys,
				Start: 10 * telem.SecondTS,
			}))
			const n = 50
			for i := range telem.TimeStamp(n) {
				Expect(w.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesSecondsTSV(10 + i),
					telem.NewSeriesV(int64(i)),
				}))).To(BeTrue())
			}
			for i := range int64(n) {
				var res framer.StreamerResponse
				Eventually(out.Outlet()).Should(Receive(&res))
				Expect(res.Frame.Get(data.Key()).Series[0]).To(
					telem.MatchSeriesDataV(i),
				)
				Expect(res.Frame.Get(data.Key()).Series[0].Alignment).To(Equal(
					res.Frame.Get(idx.Key()).Series[0].Alignment,
				))
				Expect(
					res.Frame.Get(data.Key()).Series[0].Alignment.SampleIndex(),
				).To(Equal(uint32(i)))
			}
			Consistently(out.Outlet()).ShouldNot(Receive())
			Expect(w.Close()).To(Succeed())
		},
	)
})
