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
	"context"
	"fmt"
	"io"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	alamostestutil "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap/zapcore"
)

type scenario struct {
	dist     testutil.Node
	close    io.Closer
	name     string
	channels []channel.Channel
}

func (s scenario) Close() error { return s.close.Close() }

var _ = Describe("Relay", func() {
	Describe("Happy Path", Ordered, func() {
		scenarios := []func(context.Context) scenario{
			storageScenario,
			freeScenario,
		}
		for i, sF := range scenarios {
			_sF := sF
			var s scenario
			BeforeAll(func(ctx SpecContext) {
				ShouldNotLeakGoroutines()
				s = DeferClose(_sF(ctx))
			})
			Specify(fmt.Sprintf("Scenario: %v - Happy Path", i), func(ctx SpecContext) {
				keys := channel.KeysFromChannels(s.channels)
				reader := MustSucceed(s.dist.Framer.NewStreamer(relay.StreamerConfig{
					Keys:        keys,
					SendOpenAck: new(true),
				}))
				sCtx, cancel := signal.Isolated()
				streamerReq, readerRes := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				DeferCleanup(func() {
					streamerReq.Close()
					// A streamer parked on the open ack never reads its request inlet,
					// so the drain below only returns once the context is cancelled.
					cancel()
					confluence.Drain(readerRes)
				})
				var ack relay.Response
				Eventually(readerRes.Outlet()).Should(Receive(&ack))
				Expect(ack.Frame.Empty()).To(BeTrue())
				w := MustOpen(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
				}))
				// The writer takes ownership of the frame it is handed and mutates it
				// asynchronously, so the expected values live in a frame the writer
				// never sees.
				newWriteFrame := func() frame.Frame {
					return frame.NewMulti(
						keys,
						[]telem.Series{
							telem.NewSeriesV[int64](1, 2, 3),
							telem.NewSeriesV[int64](3, 4, 5),
							telem.NewSeriesV[int64](5, 6, 7),
						},
					)
				}
				writeF := newWriteFrame()
				Expect(w.Write(newWriteFrame())).To(BeTrue())
				var res relay.Response
				Eventually(readerRes.Outlet()).Should(Receive(&res))
				resKeys := res.Frame.KeysSlice()
				Expect(set.New(resKeys...)).To(HaveLen(len(keys)))
				got := make(map[channel.Key]telem.Series, len(keys))
				for i, k := range resKeys {
					got[k] = res.Frame.SeriesAt(i)
				}
				for i, k := range keys {
					series, ws := got[k], writeF.SeriesAt(i)
					Expect(series.Data).To(Equal(ws.Data))
					Expect(series.DataType).To(Equal(ws.DataType))
					if !s.channels[i].Key().Free() {
						Expect(series.Alignment).
							To(BeNumerically(">", telem.Alignment(0)))
					}
				}
			})
		}
	})
	Describe("Time Range Stamping", func() {
		It(
			"Should deliver series stamped with the index's time range",
			func(ctx SpecContext) {
				node := testutil.NewNode(ctx)
				idx := channel.Channel{
					Name:     "stamp_time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				}
				idx = MustSucceed(node.CreateChannels(ctx, []channel.Channel{idx}))[0]
				data := channel.Channel{
					Name:       "stamp_data",
					DataType:   telem.Float32T,
					LocalIndex: idx.LocalKey,
				}
				data = MustSucceed(node.CreateChannels(ctx, []channel.Channel{data}))[0]
				keys := channel.KeysFromChannels([]channel.Channel{idx, data})

				reader := MustSucceed(node.Framer.NewStreamer(relay.StreamerConfig{
					Keys:        keys,
					SendOpenAck: new(true),
				}))
				sCtx, cancel := signal.Isolated()
				defer cancel()
				streamerReq, readerRes := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				var ack relay.Response
				Eventually(readerRes.Outlet()).Should(Receive(&ack))
				Expect(ack.Frame.Empty()).To(BeTrue())

				w := MustSucceed(node.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
				}))
				Expect(w.Write(frame.NewMulti(
					keys,
					[]telem.Series{
						telem.NewSeriesSecondsTSV(10, 11, 12),
						telem.NewSeriesV[float32](1, 2, 3),
					},
				))).To(BeTrue())
				var res relay.Response
				Eventually(readerRes.Outlet()).Should(Receive(&res))
				Expect(res.Frame.Count()).To(Equal(2))
				expected := (10 * telem.SecondTS).Range(12*telem.SecondTS + 1)
				for i := range res.Frame.Count() {
					Expect(res.Frame.SeriesAt(i).TimeRange).To(Equal(expected))
				}
				Expect(w.Close()).To(Succeed())
				streamerReq.Close()
				confluence.Drain(readerRes)
			},
		)
	})
	Describe("Slow streamers", func() {
		It("Should not delay other streamers", func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			channels := MustSucceed(node.CreateChannels(ctx, newChannelSet()))
			keys := channel.KeysFromChannels(channels)
			openStreamer := func() confluence.Outlet[relay.Response] {
				GinkgoHelper()
				reader := MustSucceed(node.Framer.NewStreamer(relay.StreamerConfig{
					Keys:        keys,
					SendOpenAck: new(true),
				}))
				sCtx, cancel := signal.Isolated()
				req, res := confluence.Attach(reader, 1)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				DeferCleanup(func() {
					req.Close()
					cancel()
					confluence.Drain(res)
				})
				Eventually(res.Outlet()).Should(Receive())
				return res
			}
			live := openStreamer()
			w := MustOpen(node.Framer.OpenWriter(ctx, writer.Config{
				Keys:  keys,
				Start: 10 * telem.SecondTS,
			}))
			openStreamer()
			// The frames fill the stalled streamer's buffer three times over.
			const batchSize = 50
			batches := 3 * relay.DefaultConfig.StreamerBufferSize / batchSize
			for range batches {
				for range batchSize {
					Expect(w.Write(frame.NewMulti(
						keys,
						[]telem.Series{
							telem.NewSeriesV[int64](1),
							telem.NewSeriesV[int64](2),
							telem.NewSeriesV[int64](3),
						},
					))).To(BeTrue())
				}
				received := 0
				Eventually(func(g Gomega) {
					for ; received < batchSize; received++ {
						g.Expect(live.Outlet()).To(Receive())
					}
				}).
					WithTimeout(500 * time.Millisecond).
					WithPolling(time.Millisecond).
					Should(Succeed())
			}
		})

		It(
			"Should not drop frames for a stalled streamer under unrelated traffic",
			func(ctx SpecContext) {
				node := testutil.NewNode(ctx)
				channels := MustSucceed(node.CreateChannels(ctx, newChannelSet()))
				keys := channel.KeysFromChannels(channels)
				quietKey, busyKeys := keys[0], keys[1:]
				busyFrames := 3 * relay.DefaultConfig.StreamerBufferSize
				openStreamer := func(
					keys channel.Keys,
					buffer int,
				) confluence.Outlet[relay.Response] {
					GinkgoHelper()
					reader := MustSucceed(node.Framer.NewStreamer(relay.StreamerConfig{
						Keys:        keys,
						SendOpenAck: new(true),
					}))
					sCtx, cancel := signal.Isolated()
					req, res := confluence.Attach(reader, buffer)
					reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
					DeferCleanup(func() {
						req.Close()
						cancel()
						confluence.Drain(res)
					})
					Eventually(res.Outlet()).Should(Receive())
					return res
				}
				quiet := openStreamer(channel.Keys{quietKey}, 1)
				busy := openStreamer(busyKeys, 1)
				w := MustOpen(node.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
				}))
				writeQuiet := func(v int64) {
					GinkgoHelper()
					Expect(w.Write(frame.NewUnary(
						quietKey,
						telem.NewSeriesV(v),
					))).To(BeTrue())
				}
				writeBusy := func(n int) {
					GinkgoHelper()
					for range n {
						Expect(w.Write(frame.NewMulti(
							busyKeys,
							[]telem.Series{
								telem.NewSeriesV[int64](1),
								telem.NewSeriesV[int64](2),
							},
						))).To(BeTrue())
					}
					for range n {
						Eventually(busy.Outlet()).Should(Receive())
					}
				}
				// Nobody reads the quiet streamer, so it stalls on the second frame.
				writeQuiet(1)
				writeQuiet(2)
				// The busy frames fill the stalled buffer three times over. Each batch
				// fits in the busy streamer's buffer, so the busy streamer drops none.
				const batchSize = 50
				for range busyFrames / batchSize {
					writeBusy(batchSize)
				}
				writeQuiet(3)
				// The relay handles frames in order, so this busy frame arrives only
				// after the relay handles the third quiet frame.
				writeBusy(1)
				for v := range int64(3) {
					var r relay.Response
					Eventually(quiet.Outlet()).Should(Receive(&r))
					Expect(r.Frame.Get(quietKey).Series[0].Data).
						To(Equal(telem.NewSeriesV(v + 1).Data))
				}
				Expect(w.Close()).To(Succeed())
			},
		)

		It(
			"Should warn once per drop episode, ending it when the buffer empties",
			func(ctx SpecContext) {
				ins, logs := alamostestutil.ObservedInstrumentation(zapcore.WarnLevel)
				store := storagetestutil.OpenLayer(ctx)
				DeferCleanup(store.Close)
				freeWrites := confluence.NewStream[relay.Response](10)
				r := MustSucceed(relay.Open(relay.Config{
					Instrumentation:    ins,
					TS:                 store.TS,
					FreeWrites:         freeWrites,
					StreamerBufferSize: 4,
				}))
				DeferCleanup(r.Close)
				key := channel.NewKey(node.KeyFree, 1)
				openStreamer := func(buffer int) confluence.Outlet[relay.Response] {
					GinkgoHelper()
					reader := MustSucceed(r.NewStreamer(relay.StreamerConfig{
						Keys:        channel.Keys{key},
						SendOpenAck: new(true),
					}))
					sCtx, cancel := signal.Isolated()
					req, res := confluence.Attach(reader, buffer)
					reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
					DeferCleanup(func() {
						req.Close()
						cancel()
						confluence.Drain(res)
					})
					Eventually(res.Outlet()).Should(Receive())
					return res
				}
				stalled := openStreamer(0)
				live := openStreamer(10)
				// The delta serves streamers in connection order, so a frame on live
				// means the delta is done with stalled.
				send := func(n int) {
					GinkgoHelper()
					for v := range int64(n) {
						freeWrites.Inlet() <- relay.Response{
							Frame: frame.NewUnary(key, telem.NewSeriesV(v)),
						}
						Eventually(live.Outlet()).Should(Receive())
					}
				}
				warnings := func() int {
					return logs.FilterMessageSnippet("dropping frames").Len()
				}
				// stalled holds at most one frame and buffers four.
				send(8)
				Expect(warnings()).To(Equal(1))
				// stalled takes the next buffered frame, so its buffer stays non-empty.
				Eventually(stalled.Outlet()).Should(Receive())
				send(4)
				Expect(warnings()).To(Equal(1))
				// Each failed attempt takes one frame, so this passes once stalled is
				// empty.
				Eventually(func(g Gomega) {
					g.Consistently(stalled.Outlet(), 200*time.Millisecond).
						ShouldNot(Receive())
				}).WithTimeout(5 * time.Second).Should(Succeed())
				send(6)
				Expect(warnings()).To(Equal(2))
			},
		)
	})
	Describe("ExcludeGroups", Ordered, func() {
		It(
			"Should filter out frames from a matching group on storage writes",
			func(ctx SpecContext) {
				channels := newChannelSet()
				node := testutil.NewNode(ctx)
				channels = MustSucceed(node.CreateChannels(ctx, channels))
				keys := channel.KeysFromChannels(channels)

				reader := MustSucceed(node.Framer.NewStreamer(relay.StreamerConfig{
					Keys:          keys,
					ExcludeGroups: []uint32{99},
					SendOpenAck:   new(true),
				}))
				sCtx, cancel := signal.Isolated()
				defer cancel()
				streamerReq, readerRes := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				var ack relay.Response
				Eventually(readerRes.Outlet()).Should(Receive(&ack))
				Expect(ack.Frame.Empty()).To(BeTrue())

				w := MustSucceed(node.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
					ControlSubject: control.Subject{
						Name:  "grouped",
						Key:   "grouped",
						Group: 99,
					},
				}))
				Expect(w.Write(frame.NewMulti(
					keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](4, 5, 6),
						telem.NewSeriesV[int64](7, 8, 9),
					},
				))).To(BeTrue())
				Consistently(
					readerRes.Outlet(),
					100*time.Millisecond,
				).ShouldNot(Receive())

				Expect(w.Close()).To(Succeed())
				streamerReq.Close()
				confluence.Drain(readerRes)
			},
		)
		It("Should deliver frames from a non-matching group", func(ctx SpecContext) {
			channels := newChannelSet()
			node := testutil.NewNode(ctx)
			channels = MustSucceed(node.CreateChannels(ctx, channels))
			keys := channel.KeysFromChannels(channels)

			reader := MustSucceed(node.Framer.NewStreamer(relay.StreamerConfig{
				Keys:          keys,
				ExcludeGroups: []uint32{99},
				SendOpenAck:   new(true),
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			streamerReq, readerRes := confluence.Attach(reader, 10)
			reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			var ack relay.Response
			Eventually(readerRes.Outlet()).Should(Receive(&ack))
			Expect(ack.Frame.Empty()).To(BeTrue())

			w := MustSucceed(node.Framer.OpenWriter(ctx, writer.Config{
				Keys:  keys,
				Start: 10 * telem.SecondTS,
				ControlSubject: control.Subject{
					Name:  "other",
					Key:   "other",
					Group: 200,
				},
			}))
			Expect(w.Write(frame.NewMulti(
				keys,
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
					telem.NewSeriesV[int64](7, 8, 9),
				},
			))).To(BeTrue())
			var res relay.Response
			Eventually(readerRes.Outlet()).Should(Receive(&res))
			Expect(res.Frame.Count()).To(Equal(3))
			Expect(res.Group).To(Equal(uint32(200)))

			Expect(w.Close()).To(Succeed())
			streamerReq.Close()
			confluence.Drain(readerRes)
		})
		It(
			"Should deliver frames with no group even when ExcludeGroups is set",
			func(ctx SpecContext) {
				channels := newChannelSet()
				node := testutil.NewNode(ctx)
				channels = MustSucceed(node.CreateChannels(ctx, channels))
				keys := channel.KeysFromChannels(channels)

				reader := MustSucceed(node.Framer.NewStreamer(relay.StreamerConfig{
					Keys:          keys,
					ExcludeGroups: []uint32{99},
					SendOpenAck:   new(true),
				}))
				sCtx, cancel := signal.Isolated()
				defer cancel()
				streamerReq, readerRes := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				var ack relay.Response
				Eventually(readerRes.Outlet()).Should(Receive(&ack))
				Expect(ack.Frame.Empty()).To(BeTrue())

				w := MustSucceed(node.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
				}))
				Expect(w.Write(frame.NewMulti(
					keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](4, 5, 6),
						telem.NewSeriesV[int64](7, 8, 9),
					},
				))).To(BeTrue())
				var res relay.Response
				Eventually(readerRes.Outlet()).Should(Receive(&res))
				Expect(res.Frame.Count()).To(Equal(3))

				Expect(w.Close()).To(Succeed())
				streamerReq.Close()
				confluence.Drain(readerRes)
			},
		)
		It(
			"Should filter out free channel frames from a matching group",
			func(ctx SpecContext) {
				channels := newChannelSet()
				svc := testutil.NewNode(ctx)
				for i, ch := range channels {
					ch.Leaseholder = node.KeyFree
					ch.Virtual = true
					channels[i] = ch
				}
				channels = MustSucceed(svc.CreateChannels(ctx, channels))
				keys := channel.KeysFromChannels(channels)

				reader := MustSucceed(svc.Framer.NewStreamer(relay.StreamerConfig{
					Keys:          keys,
					ExcludeGroups: []uint32{55},
					SendOpenAck:   new(true),
				}))
				sCtx, cancel := signal.Isolated()
				defer cancel()
				streamerReq, readerRes := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				var ack relay.Response
				Eventually(readerRes.Outlet()).Should(Receive(&ack))
				Expect(ack.Frame.Empty()).To(BeTrue())

				w := MustSucceed(svc.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
					ControlSubject: control.Subject{
						Name:  "free-grouped",
						Key:   "free-grouped",
						Group: 55,
					},
				}))
				Expect(w.Write(frame.NewMulti(
					keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](4, 5, 6),
						telem.NewSeriesV[int64](7, 8, 9),
					},
				))).To(BeTrue())
				Consistently(
					readerRes.Outlet(),
					100*time.Millisecond,
				).ShouldNot(Receive())

				Expect(w.Close()).To(Succeed())
				streamerReq.Close()
				confluence.Drain(readerRes)
			},
		)
	})
	// These tests cover the SendOpenAck happens-before guarantee: once a caller sees
	// the open ack, a subsequent write must be observable on the streamer. The relay
	// implements this by waiting for the tapper to install the streamer's keys into the
	// storage tap before sending the ack. The cases below also exercise the edge cases
	// that could deadlock the synchronous wait — empty initial keys, non-free leases,
	// context cancellation mid-wait, and concurrent streamers.
	Describe("SendOpenAck", Ordered, func() {
		var svc testutil.Node
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			svc = testutil.NewNode(ctx)
		})

		newChannels := func(
			ctx context.Context,
			n int,
			leaseholder node.Key,
		) []channel.Channel {
			GinkgoHelper()
			chs := make([]channel.Channel, n)
			ts := time.Now().UnixNano()
			for i := range chs {
				chs[i] = channel.Channel{
					Name:        fmt.Sprintf("ch_%d_%d", ts, i),
					Virtual:     true,
					DataType:    telem.Int64T,
					Leaseholder: leaseholder,
				}
			}
			return MustSucceed(svc.CreateChannels(ctx, chs))
		}
		newFreeChannels := func(ctx context.Context, n int) []channel.Channel {
			GinkgoHelper()
			return newChannels(ctx, n, node.KeyFree)
		}

		It(
			"Should deliver a write issued immediately after the open ack on a free channel",
			func(ctx SpecContext) {
				chs := newFreeChannels(ctx, 1)
				keys := channel.KeysFromChannels(chs)

				reader := MustSucceed(svc.Framer.NewStreamer(relay.StreamerConfig{
					Keys:        keys,
					SendOpenAck: new(true),
				}))
				sCtx, cancel := signal.Isolated()
				defer cancel()
				req, res := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())

				var ack relay.Response
				Eventually(res.Outlet()).Should(Receive(&ack))
				Expect(ack.Frame.Empty()).To(BeTrue())

				w := MustSucceed(svc.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
				}))
				defer func() { Expect(w.Close()).To(Succeed()) }()
				Expect(w.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
				}))).To(BeTrue())

				var got relay.Response
				Eventually(res.Outlet()).Should(Receive(&got))
				Expect(got.Frame.Count()).To(Equal(1))
				req.Close()
				confluence.Drain(res)
			},
		)

		It(
			"Should deliver the open ack when the streamer is opened with no keys",
			func(ctx SpecContext) {
				reader := MustSucceed(svc.Framer.NewStreamer(relay.StreamerConfig{
					SendOpenAck: new(true),
				}))
				sCtx, cancel := signal.Isolated()
				defer cancel()
				req, res := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())

				var ack relay.Response
				Eventually(res.Outlet()).Should(Receive(&ack))
				Expect(ack.Frame.Empty()).To(BeTrue())

				req.Close()
				confluence.Drain(res)
			},
		)

		It(
			"Should deliver the open ack when the streamer subscribes only to stored channels",
			func(ctx SpecContext) {
				// Channels with no explicit leaseholder are assigned to the host node
				// and are served by the storage tap. This exercises the ack path for a
				// demand whose keys are all stored.
				chs := MustSucceed(svc.CreateChannels(ctx, newChannelSet()))
				keys := channel.KeysFromChannels(chs)

				reader := MustSucceed(svc.Framer.NewStreamer(relay.StreamerConfig{
					Keys:        keys,
					SendOpenAck: new(true),
				}))
				sCtx, cancel := signal.Isolated()
				defer cancel()
				req, res := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())

				var ack relay.Response
				Eventually(res.Outlet()).Should(Receive(&ack))
				Expect(ack.Frame.Empty()).To(BeTrue())

				req.Close()
				confluence.Drain(res)
			},
		)

		It(
			"Should not leak when the streamer's context is canceled before the open ack is delivered",
			func(ctx SpecContext) {
				chs := newFreeChannels(ctx, 1)
				keys := channel.KeysFromChannels(chs)

				reader := MustSucceed(svc.Framer.NewStreamer(relay.StreamerConfig{
					Keys:        keys,
					SendOpenAck: new(true),
				}))
				sCtx, cancel := signal.Isolated()
				req, res := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				cancel()

				req.Close()
				Eventually(res.Outlet()).Should(BeClosed())
			},
		)

		It(
			"Should exit at once when the streamer flows in a canceled context",
			func(ctx SpecContext) {
				chs := newFreeChannels(ctx, 1)
				reader := MustSucceed(svc.Framer.NewStreamer(relay.StreamerConfig{
					Keys:        channel.KeysFromChannels(chs),
					SendOpenAck: new(true),
				}))
				sCtx, cancel := signal.Isolated()
				cancel()
				_, res := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				Expect(sCtx.Wait()).To(MatchError(context.Canceled))
				Eventually(res.Outlet()).Should(BeClosed())
			},
		)

		It(
			"Should serve concurrent streamers on overlapping free channels without deadlocking",
			func(ctx SpecContext) {
				chs := newFreeChannels(ctx, 2)
				keys := channel.KeysFromChannels(chs)

				const streamerCount = 4
				readers := make([]relay.Streamer, streamerCount)
				outlets := make([]confluence.Outlet[relay.Response], streamerCount)
				inlets := make([]confluence.Inlet[relay.Request], streamerCount)
				sCtx, cancel := signal.Isolated()
				defer cancel()
				for i := range streamerCount {
					readers[i] = MustSucceed(
						svc.Framer.NewStreamer(relay.StreamerConfig{
							Keys:        keys,
							SendOpenAck: new(true),
						}),
					)
					inlets[i], outlets[i] = confluence.Attach(readers[i], 10)
					readers[i].Flow(sCtx, confluence.CloseOutputInletsOnExit())
				}
				for _, o := range outlets {
					var ack relay.Response
					Eventually(o.Outlet()).Should(Receive(&ack))
					Expect(ack.Frame.Empty()).To(BeTrue())
				}

				w := MustSucceed(svc.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
				}))
				defer func() { Expect(w.Close()).To(Succeed()) }()
				Expect(w.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
				}))).To(BeTrue())

				for _, o := range outlets {
					var got relay.Response
					Eventually(o.Outlet()).Should(Receive(&got))
					Expect(got.Frame.Count()).To(Equal(2))
				}
				for _, i := range inlets {
					i.Close()
				}
				for _, o := range outlets {
					confluence.Drain(o)
				}
			},
		)

		It(
			"Should apply a dynamic key update before subsequent writes are dropped",
			func(ctx SpecContext) {
				chs := newFreeChannels(ctx, 1)
				keys := channel.KeysFromChannels(chs)

				reader := MustSucceed(svc.Framer.NewStreamer(relay.StreamerConfig{
					SendOpenAck: new(true),
				}))
				sCtx, cancel := signal.Isolated()
				defer cancel()
				req, res := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())

				var ack relay.Response
				Eventually(res.Outlet()).Should(Receive(&ack))
				Expect(ack.Frame.Empty()).To(BeTrue())

				req.Inlet() <- relay.Request{Keys: keys}

				w := MustSucceed(svc.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
				}))
				defer func() { Expect(w.Close()).To(Succeed()) }()

				// The dynamic key update propagates asynchronously, so retry the write
				// until the streamer has picked up the new keys and delivers a frame.
				var got relay.Response
				Eventually(func(g Gomega) {
					g.Expect(w.Write(frame.NewMulti(keys, []telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
					}))).To(BeTrue())
					g.Expect(res.Outlet()).To(Receive(&got))
				}).Should(Succeed())
				Expect(got.Frame.Count()).To(Equal(1))
				req.Close()
				confluence.Drain(res)
			},
		)

		Describe("Key update acks", func() {
			openStreamer := func() (
				confluence.Inlet[relay.Request],
				confluence.Outlet[relay.Response],
			) {
				GinkgoHelper()
				reader := MustSucceed(svc.Framer.NewStreamer(relay.StreamerConfig{
					SendOpenAck: new(true),
				}))
				sCtx, cancel := signal.Isolated()
				DeferCleanup(cancel)
				req, res := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				var ack relay.Response
				Eventually(res.Outlet()).Should(Receive(&ack))
				Expect(ack.Ack).To(BeFalse())
				DeferCleanup(func() {
					req.Close()
					confluence.Drain(res)
				})
				return req, res
			}

			DescribeTable(
				"Should deliver a write issued immediately after a key update ack",
				func(ctx SpecContext, leaseholder node.Key) {
					keys := channel.KeysFromChannels(newChannels(ctx, 1, leaseholder))
					req, res := openStreamer()
					req.Inlet() <- relay.Request{Keys: keys, AckRequested: true}
					var ack relay.Response
					Eventually(res.Outlet()).Should(Receive(&ack))
					Expect(ack.Ack).To(BeTrue())
					Expect(ack.Frame.Empty()).To(BeTrue())

					w := MustSucceed(svc.Framer.OpenWriter(ctx, writer.Config{
						Keys:  keys,
						Start: 10 * telem.SecondTS,
					}))
					defer func() { Expect(w.Close()).To(Succeed()) }()
					Expect(w.Write(frame.NewMulti(keys, []telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
					}))).To(BeTrue())
					var got relay.Response
					Eventually(res.Outlet()).Should(Receive(&got))
					Expect(got.Ack).To(BeFalse())
					Expect(got.Frame.Count()).To(Equal(1))
				},
				Entry("on a free channel", node.KeyFree),
				Entry("on a channel leased to this node", node.KeyBootstrapper),
			)

			It(
				"Should not ack a key update that does not request one",
				func(ctx SpecContext) {
					keys := channel.KeysFromChannels(newFreeChannels(ctx, 1))
					req, res := openStreamer()
					req.Inlet() <- relay.Request{Keys: keys}
					Consistently(
						res.Outlet(),
						100*time.Millisecond,
					).ShouldNot(Receive())
				},
			)

			It(
				"Should ack each update that requests one exactly once",
				func(ctx SpecContext) {
					keys := channel.KeysFromChannels(newFreeChannels(ctx, 2))
					req, res := openStreamer()
					req.Inlet() <- relay.Request{Keys: keys[:1], AckRequested: true}
					req.Inlet() <- relay.Request{Keys: keys}
					req.Inlet() <- relay.Request{Keys: keys[1:], AckRequested: true}
					for range 2 {
						var ack relay.Response
						Eventually(res.Outlet()).Should(Receive(&ack))
						Expect(ack.Ack).To(BeTrue())
					}
					Consistently(
						res.Outlet(),
						100*time.Millisecond,
					).ShouldNot(Receive())
				},
			)
		})
	})
	// The relay does not validate streamer keys against storage: the service layer
	// validates at open time, but a channel can be deleted between that check and the
	// demand reaching the tapper, and mid-stream key updates are never validated. The
	// storage tap must drop keys that don't resolve in storage instead of failing
	// outright, since a failed tap starves every streamer demanding stored channels.
	Describe("Unknown Keys", Ordered, func() {
		var svc testutil.Node
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			svc = testutil.NewNode(ctx)
		})

		newUniqueChannels := func(ctx context.Context, n int) []channel.Channel {
			GinkgoHelper()
			chs := make([]channel.Channel, n)
			ts := time.Now().UnixNano()
			for i := range chs {
				chs[i] = channel.Channel{
					Name:     fmt.Sprintf("unknown_keys_%d_%d", ts, i),
					Virtual:  true,
					DataType: telem.Int64T,
				}
			}
			return MustSucceed(svc.CreateChannels(ctx, chs))
		}

		It(
			"Should deliver frames for valid keys when the streamer also demands an unknown host-leased key",
			func(ctx SpecContext) {
				chs := newUniqueChannels(ctx, 3)
				keys := channel.KeysFromChannels(chs)
				unknown := channel.NewKey(node.KeyBootstrapper, 0xFFFFF)

				reader := MustSucceed(svc.Framer.NewStreamer(relay.StreamerConfig{
					Keys:        append(channel.Keys{unknown}, keys...),
					SendOpenAck: new(true),
				}))
				sCtx, cancel := signal.Isolated()
				defer cancel()
				req, res := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())

				var ack relay.Response
				Eventually(res.Outlet()).Should(Receive(&ack))
				Expect(ack.Frame.Empty()).To(BeTrue())

				w := MustOpen(svc.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
				}))
				Expect(w.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
					telem.NewSeriesV[int64](7, 8, 9),
				}))).To(BeTrue())

				var got relay.Response
				Eventually(res.Outlet()).Should(Receive(&got))
				Expect(got.Frame.Count()).To(Equal(3))
				Expect(got.Frame.KeysSlice()).To(ConsistOf(keys))
				req.Close()
				confluence.Drain(res)
			},
		)

		It(
			"Should not starve a concurrent streamer with valid keys when another streamer demands an unknown key",
			func(ctx SpecContext) {
				chs := newUniqueChannels(ctx, 2)
				keys := channel.KeysFromChannels(chs)
				unknown := channel.NewKey(node.KeyBootstrapper, 0xFFFFE)

				// The streamer with the unknown key opens first, so its demand is the
				// one that opens the shared storage tap.
				mixed := MustSucceed(svc.Framer.NewStreamer(relay.StreamerConfig{
					Keys:        append(channel.Keys{unknown}, keys...),
					SendOpenAck: new(true),
				}))
				sCtx, cancel := signal.Isolated()
				defer cancel()
				mixedReq, mixedRes := confluence.Attach(mixed, 10)
				mixed.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				var ack relay.Response
				Eventually(mixedRes.Outlet()).Should(Receive(&ack))
				Expect(ack.Frame.Empty()).To(BeTrue())

				valid := MustSucceed(svc.Framer.NewStreamer(relay.StreamerConfig{
					Keys:        keys,
					SendOpenAck: new(true),
				}))
				validReq, validRes := confluence.Attach(valid, 10)
				valid.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				Eventually(validRes.Outlet()).Should(Receive(&ack))
				Expect(ack.Frame.Empty()).To(BeTrue())

				w := MustOpen(svc.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
				}))
				Expect(w.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
				}))).To(BeTrue())

				var got relay.Response
				Eventually(mixedRes.Outlet()).Should(Receive(&got))
				Expect(got.Frame.Count()).To(Equal(2))
				Eventually(validRes.Outlet()).Should(Receive(&got))
				Expect(got.Frame.Count()).To(Equal(2))

				mixedReq.Close()
				validReq.Close()
				confluence.Drain(mixedRes)
				confluence.Drain(validRes)
			},
		)
	})
})

func newChannelSet() []channel.Channel {
	return []channel.Channel{
		{
			Name:     "test1",
			Virtual:  true,
			DataType: telem.Int64T,
		},
		{
			Name:     "test2",
			Virtual:  true,
			DataType: telem.Int64T,
		},
		{
			Name:     "test3",
			Virtual:  true,
			DataType: telem.Int64T,
		},
	}
}

func storageScenario(ctx context.Context) scenario {
	GinkgoHelper()
	channels := newChannelSet()
	node := testutil.OpenNode(ctx)
	channels = MustSucceed(node.CreateChannels(ctx, channels))
	return scenario{
		name:     "Storage",
		channels: channels,
		dist:     node,
		close:    node,
	}
}

func freeScenario(ctx context.Context) scenario {
	GinkgoHelper()
	channels := newChannelSet()
	dist := testutil.OpenNode(ctx)
	for i, ch := range channels {
		ch.Leaseholder = node.KeyFree
		ch.Virtual = true
		channels[i] = ch
	}
	channels = MustSucceed(dist.CreateChannels(ctx, channels))
	return scenario{
		name:     "Free Channel",
		channels: channels,
		dist:     dist,
		close:    dist,
	}
}
