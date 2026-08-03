// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"context"
	"io"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/alignment"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// openStreamer opens a streamer on db with the given config and starts it in a signal
// context derived from ctx, returning the streamer's request inlet, its response
// outlet, and a closer that shuts the streamer down.
func openStreamer(ctx context.Context, db *cesium.DB, cfg cesium.StreamerConfig) (
	confluence.Inlet[cesium.StreamerRequest],
	confluence.Outlet[cesium.StreamerResponse],
	io.Closer,
) {
	streamer := MustSucceed(db.NewStreamer(ctx, cfg))
	requests := confluence.NewStream[cesium.StreamerRequest](1)
	responses := confluence.NewStream[cesium.StreamerResponse](2)
	streamer.InFrom(requests)
	streamer.OutTo(responses)
	sCtx, cancel := signal.WithCancel(ctx)
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	return requests, responses, signal.NewHardShutdown(sCtx, cancel)
}

var _ = Describe("Streamer Behavior", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db         *cesium.DB
				fs         fs.FS
				controlKey cesium.ChannelKey = 5
			)
			BeforeAll(func(ctx SpecContext) {
				ShouldNotLeakGoroutines()
				fs = openFS()
				db = mustOpenDBOnFS(ctx, fs)
				Expect(db.ConfigureControlUpdateChannel(ctx, controlKey, "cesium_control")).To(Succeed())
			})

			Describe("Happy Path", func() {
				It("Should subscribe to written frames for the given channels", func(ctx SpecContext) {
					var basic1 cesium.ChannelKey = 1
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: basic1, Name: "Planck", DataType: telem.TimeStampT, IsIndex: true},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{basic1},
						Start:    10 * telem.SecondTS,
					}))
					_, o, closer := openStreamer(ctx, db, cesium.StreamerConfig{
						Channels: []cesium.ChannelKey{basic1},
					})

					d := telem.NewSeriesSecondsTSV(10, 11, 12)
					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{basic1},
						[]telem.Series{d},
					)))

					var f cesium.StreamerResponse
					Eventually(o.Outlet()).Should(Receive(&f))
					Expect(f.Frame.Count()).To(Equal(1))
					d.Alignment = alignment.Leading(1, 0)
					Expect(f.Frame.SeriesAt(0)).To(Equal(d))
					Expect(closer.Close()).To(Succeed())
					Expect(w.Close()).To(Succeed())
				})

				It("Should deliver writes issued after a subscription update is sent", func(ctx SpecContext) {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: key, Name: "Curie", DataType: telem.TimeStampT, IsIndex: true},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{key},
						Start:    10 * telem.SecondTS,
					}))
					r, o, closer := openStreamer(ctx, db, cesium.StreamerConfig{})

					r.Inlet() <- cesium.StreamerRequest{Channels: []cesium.ChannelKey{key}}

					// The subscription update is applied asynchronously, so writes
					// racing ahead of it are dropped. Retry with increasing timestamps
					// until a frame comes through.
					var res cesium.StreamerResponse
					ts := telem.TimeStamp(10)
					Eventually(func(g Gomega) {
						g.Expect(w.Write(telem.MultiFrame(
							[]cesium.ChannelKey{key},
							[]telem.Series{telem.NewSeriesSecondsTSV(ts)},
						))).To(BeTrue())
						ts++
						g.Expect(o.Outlet()).To(Receive(&res))
					}).Should(Succeed())
					Expect(res.Frame.KeysSlice()).To(ContainElement(key))
					Expect(closer.Close()).To(Succeed())
					Expect(w.Close()).To(Succeed())
				})
			})

			Describe("Writer is in PersistOnly mode", func() {
				It("Should not receive any frames", func(ctx SpecContext) {
					var basic2 cesium.ChannelKey = 3
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: basic2, Name: "Bohr", DataType: telem.TimeStampT, IsIndex: true},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{basic2},
						Start:    10 * telem.SecondTS,
						Mode:     cesium.WriterModePersistOnly,
					}))
					_, o, closer := openStreamer(ctx, db, cesium.StreamerConfig{
						Channels: []cesium.ChannelKey{basic2},
					})

					d := telem.NewSeriesSecondsTSV(10, 11, 12)
					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{basic2},
						[]telem.Series{d},
					)))

					Consistently(o.Outlet()).ShouldNot(Receive())
					Expect(closer.Close()).To(Succeed())
					Expect(w.Close()).To(Succeed())
				})
			})

			Describe("Virtual Channels", func() {
				It("Should subscribe to written frames for virtual channels", func(ctx SpecContext) {
					var basic2 cesium.ChannelKey = 4
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: basic2, Name: "Heisenberg", DataType: telem.Int64T, Virtual: true},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{basic2},
						Start:    10 * telem.SecondTS,
					}))
					_, o, closer := openStreamer(ctx, db, cesium.StreamerConfig{
						Channels: []cesium.ChannelKey{basic2},
					})

					written := telem.NewSeriesV[int64](1, 2, 3)
					MustSucceed(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{basic2},
						[]telem.Series{written},
					)))
					var res cesium.StreamerResponse
					Eventually(o.Outlet()).Should(Receive(&res))
					Expect(res.Frame.Count()).To(Equal(1))
					written.Alignment = alignment.Leading(1, 0)
					Expect(res.Frame.SeriesAt(0)).To(Equal(written))
					Expect(closer.Close()).To(Succeed())
					Expect(w.Close()).To(Succeed())
				})
			})

			Describe("Control Updates", func() {
				It("Should forward control updates to the streamer", func(ctx SpecContext) {
					var basic3 cesium.ChannelKey = 6
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: basic3, Name: "Schrodinger", DataType: telem.TimeStampT, IsIndex: true},
					)).To(Succeed())
					_, o, closer := openStreamer(ctx, db, cesium.StreamerConfig{
						Channels:    []cesium.ChannelKey{controlKey},
						SendOpenAck: true,
					})
					// Do a best effort schedule for the streamer to boot up
					Eventually(o.Outlet()).Should(Receive())
					runtime.Gosched()
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:       []cesium.ChannelKey{basic3},
						ControlSubject: control.Subject{Name: "Writer"},
						Start:          10 * telem.SecondTS,
					}))
					var r cesium.StreamerResponse
					// Move this into an eventual closure, as we may be getting latent
					// control updates from other tests, so we just assert on updates
					// until we get one that matches.
					Eventually(func(g Gomega) {
						g.Eventually(o.Outlet()).Should(Receive(&r))
						g.Expect(r.Frame.Count()).To(Equal(1))
						u := MustSucceed(cesium.DecodeControlUpdate(r.Frame.SeriesAt(0)))
						g.Expect(u.Transfers).To(HaveLen(1))
						first := u.Transfers[0]
						g.Expect(first.Occurred()).To(BeTrue())
						g.Expect(first.IsAcquire()).To(BeTrue())
					}).Should(Succeed())

					Expect(w.Close()).To(Succeed())
					Eventually(o.Outlet()).Should(Receive(&r))
					Expect(r.Frame.Count()).To(Equal(1))
					Expect(closer.Close()).To(Succeed())
				})
			})

			Describe("Group Propagation", func() {
				It("Should propagate the writer's group to the streamer response", func(ctx SpecContext) {
					var groupCh cesium.ChannelKey = 7
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: groupCh, Name: "GroupTest", DataType: telem.Int64T, Virtual: true},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:       []cesium.ChannelKey{groupCh},
						Start:          10 * telem.SecondTS,
						ControlSubject: control.Subject{Name: "GroupWriter", Group: 42},
					}))
					_, o, closer := openStreamer(ctx, db, cesium.StreamerConfig{
						Channels: []cesium.ChannelKey{groupCh},
					})

					Expect(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{groupCh},
						[]telem.Series{telem.NewSeriesV[int64](1, 2, 3)},
					))).To(BeTrue())
					var res cesium.StreamerResponse
					Eventually(o.Outlet()).Should(Receive(&res))
					Expect(res.Group).To(Equal(uint32(42)))
					Expect(res.Frame.Count()).To(Equal(1))
					Expect(closer.Close()).To(Succeed())
					Expect(w.Close()).To(Succeed())
				})
				It("Should set group to zero when the writer has no group", func(ctx SpecContext) {
					var noGroupCh cesium.ChannelKey = 8
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: noGroupCh, Name: "NoGroupTest", DataType: telem.Int64T, Virtual: true},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels:       []cesium.ChannelKey{noGroupCh},
						Start:          10 * telem.SecondTS,
						ControlSubject: control.Subject{Name: "NoGroupWriter"},
					}))
					_, o, closer := openStreamer(ctx, db, cesium.StreamerConfig{
						Channels: []cesium.ChannelKey{noGroupCh},
					})

					Expect(w.Write(telem.MultiFrame(
						[]cesium.ChannelKey{noGroupCh},
						[]telem.Series{telem.NewSeriesV[int64](4, 5, 6)},
					))).To(BeTrue())
					var res cesium.StreamerResponse
					Eventually(o.Outlet()).Should(Receive(&res))
					Expect(res.Group).To(Equal(uint32(0)))
					Expect(closer.Close()).To(Succeed())
					Expect(w.Close()).To(Succeed())
				})
			})

			Describe("Channel Validation", func() {
				It("Should allow subscription updates for channels that do not exist", func(ctx SpecContext) {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: key, Name: "Noether", DataType: telem.TimeStampT, IsIndex: true},
					)).To(Succeed())
					w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{key},
						Start:    10 * telem.SecondTS,
					}))
					r, o, closer := openStreamer(ctx, db, cesium.StreamerConfig{})

					r.Inlet() <- cesium.StreamerRequest{
						Channels: []cesium.ChannelKey{key, GenerateChannelKey()},
					}

					// The subscription update is applied asynchronously, so writes
					// racing ahead of it are dropped. Retry with increasing timestamps
					// until a frame comes through.
					var res cesium.StreamerResponse
					ts := telem.TimeStamp(10)
					Eventually(func(g Gomega) {
						g.Expect(w.Write(
							telem.UnaryFrame(key, telem.NewSeriesSecondsTSV(ts)),
						)).To(BeTrue())
						ts++
						g.Expect(o.Outlet()).To(Receive(&res))
					}).Should(Succeed())
					Expect(res.Frame.KeysSlice()).To(ConsistOf(key))
					Expect(closer.Close()).To(Succeed())
					Expect(w.Close()).To(Succeed())
				})
			})

			Describe("Slow Consumers", func() {
				It("Should buffer frames for a consumer that stalls during writes", func(ctx SpecContext) {
					const bufferSize = 10
					// Writing past the relay's total buffered capacity would force the
					// writer to block until the relay times out and drops frames for
					// the stalled consumer, so undersized buffering surfaces as missing
					// frames in the drain below.
					const frameCount int64 = 2 * bufferSize
					subFS := MustSucceed(fs.Sub("slow-consumer"))
					subDB := mustOpenDBOnFS(ctx, subFS, cesium.WithStreamBufferSize(bufferSize))
					key := GenerateChannelKey()
					Expect(subDB.CreateChannel(
						ctx,
						cesium.Channel{
							Key:      key,
							Name:     "Feynman",
							DataType: telem.Int64T,
							Virtual:  true,
						},
					)).To(Succeed())
					w := MustOpen(subDB.OpenWriter(ctx, cesium.WriterConfig{
						Channels: []cesium.ChannelKey{key},
						Start:    10 * telem.SecondTS,
					}))
					_, o, closer := openStreamer(ctx, subDB, cesium.StreamerConfig{
						Channels: []cesium.ChannelKey{key},
					})
					DeferClose(closer)

					for v := range frameCount {
						Expect(
							w.Write(telem.UnaryFrame(key, telem.NewSeriesV(v))),
						).To(BeTrue())
					}
					for v := range frameCount {
						var res cesium.StreamerResponse
						Eventually(o.Outlet()).Should(Receive(&res))
						Expect(res.Frame.Count()).To(Equal(1))
						Expect(res.Frame.SeriesAt(0)).To(telem.MatchSeriesDataV(v))
					}
				})
			})

			Describe("Closed", func() {
				It("Should not allow opening a streamer on a closed db", func(ctx SpecContext) {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(ctx, sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{
						Key:      key,
						Name:     "Einstein",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					Expect(subDB.NewStreamer(ctx, cesium.StreamerConfig{
						Channels: []cesium.ChannelKey{key},
					})).Error().To(MatchError(cesium.ErrDBClosed))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})
			})
		})
	}
})

var _ = Describe("Virtual Channel Streaming", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var db *cesium.DB
			BeforeAll(func(ctx SpecContext) {
				ShouldNotLeakGoroutines()
				db = mustOpenDBOnFS(ctx, openFS())
			})

			It("Should deliver written frames to streamers", func(ctx SpecContext) {
				key := GenerateChannelKey()
				Expect(db.CreateChannel(ctx, virtualChannel(key, "streamed"))).To(Succeed())
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: []cesium.ChannelKey{key},
					Start:    10 * telem.SecondTS,
				}))
				_, o, closer := openStreamer(ctx, db, cesium.StreamerConfig{
					Channels: []cesium.ChannelKey{key},
				})

				d := telem.NewSeriesV[int64](1, 2, 3)
				Expect(w.Write(telem.UnaryFrame(key, d))).To(BeTrue())

				var res cesium.StreamerResponse
				Eventually(o.Outlet()).Should(Receive(&res))
				Expect(res.Frame.Count()).To(Equal(1))
				d.Alignment = alignment.Leading(1, 0)
				Expect(res.Frame.SeriesAt(0)).To(Equal(d))
				Expect(closer.Close()).To(Succeed())
				Expect(w.Close()).To(Succeed())
			})
		})
	}
})
