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
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type scenario struct {
	dist     mock.Node
	close    io.Closer
	name     string
	channels []channel.Channel
	resCount int
}

func (s scenario) Close() error { return s.close.Close() }

var _ = Describe("Relay", func() {
	Describe("Happy Path", Ordered, func() {

		scenarios := []func(context.Context) scenario{
			gatewayOnlyScenario,
			peerOnlyScenario,
			mixedScenario,
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
					Keys: keys,
				}))
				sCtx, _ := signal.Isolated()
				streamerReq, readerRes := confluence.Attach(reader, 10)
				reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				// We need to give a few milliseconds for the reader to boot up.
				time.Sleep(10 * time.Millisecond)
				w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
					Keys:  keys,
					Start: 10 * telem.SecondTS,
				}))
				defer func() {
					defer GinkgoRecover()
					Expect(w.Close()).To(Succeed())
				}()
				writeF := frame.NewMulti(
					keys,
					[]telem.Series{
						telem.NewSeriesV[int64](1, 2, 3),
						telem.NewSeriesV[int64](3, 4, 5),
						telem.NewSeriesV[int64](5, 6, 7),
					},
				)
				Expect(w.Write(writeF)).To(BeTrue())
				var f framer.Frame
				for range s.resCount {
					var res relay.Response
					Eventually(readerRes.Outlet()).Should(Receive(&res))
					f = frame.Merge([]frame.Frame{f, res.Frame})
				}
				Expect(f.Count()).To(Equal(3))
				for i, k := range f.KeysSlice() {
					wi := lo.IndexOf(keys, k)
					ch := s.channels[wi]
					s := f.SeriesAt(i)
					ws := writeF.SeriesAt(wi)
					Expect(s.Data).To(Equal(ws.Data))
					Expect(s.DataType).To(Equal(ws.DataType))
					if !ch.Free() {
						Expect(s.Alignment).To(BeNumerically(">", telem.Alignment(0)))
					}
				}
				streamerReq.Close()
				confluence.Drain(readerRes)
			})
		}
	})
	Describe("ExcludeGroups", Ordered, func() {
		It("Should filter out frames from a matching group on gateway writes", func(ctx SpecContext) {
			channels := newChannelSet()
			node := mock.NewNode(ctx)
			channels = MustSucceed(node.Channel.Create(ctx, channels))
			keys := channel.KeysFromChannels(channels)

			reader := MustSucceed(node.Framer.NewStreamer(relay.StreamerConfig{
				Keys:          keys,
				ExcludeGroups: []uint32{99},
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			streamerReq, readerRes := confluence.Attach(reader, 10)
			reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(10 * time.Millisecond)

			w := MustSucceed(node.Framer.OpenWriter(ctx, writer.Config{
				Keys:           keys,
				Start:          10 * telem.SecondTS,
				ControlSubject: control.Subject{Name: "grouped", Key: "grouped", Group: 99},
			}))
			Expect(w.Write(frame.NewMulti(
				keys,
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
					telem.NewSeriesV[int64](7, 8, 9),
				},
			))).To(BeTrue())
			Consistently(readerRes.Outlet(), 100*time.Millisecond).ShouldNot(Receive())

			Expect(w.Close()).To(Succeed())
			streamerReq.Close()
			confluence.Drain(readerRes)
		})
		It("Should deliver frames from a non-matching group", func(ctx SpecContext) {
			channels := newChannelSet()
			node := mock.NewNode(ctx)
			channels = MustSucceed(node.Channel.Create(ctx, channels))
			keys := channel.KeysFromChannels(channels)

			reader := MustSucceed(node.Framer.NewStreamer(relay.StreamerConfig{
				Keys:          keys,
				ExcludeGroups: []uint32{99},
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			streamerReq, readerRes := confluence.Attach(reader, 10)
			reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(10 * time.Millisecond)

			w := MustSucceed(node.Framer.OpenWriter(ctx, writer.Config{
				Keys:           keys,
				Start:          10 * telem.SecondTS,
				ControlSubject: control.Subject{Name: "other", Key: "other", Group: 200},
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
		It("Should deliver frames with no group even when ExcludeGroups is set", func(ctx SpecContext) {
			channels := newChannelSet()
			node := mock.NewNode(ctx)
			channels = MustSucceed(node.Channel.Create(ctx, channels))
			keys := channel.KeysFromChannels(channels)

			reader := MustSucceed(node.Framer.NewStreamer(relay.StreamerConfig{
				Keys:          keys,
				ExcludeGroups: []uint32{99},
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			streamerReq, readerRes := confluence.Attach(reader, 10)
			reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(10 * time.Millisecond)

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
		})
		It("Should filter out free channel frames from a matching group", func(ctx SpecContext) {
			channels := newChannelSet()
			svc := mock.NewNode(ctx)
			for i, ch := range channels {
				ch.Leaseholder = node.KeyFree
				ch.Virtual = true
				channels[i] = ch
			}
			channels = MustSucceed(svc.Channel.Create(ctx, channels))
			keys := channel.KeysFromChannels(channels)

			reader := MustSucceed(svc.Framer.NewStreamer(relay.StreamerConfig{
				Keys:          keys,
				ExcludeGroups: []uint32{55},
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			streamerReq, readerRes := confluence.Attach(reader, 10)
			reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(10 * time.Millisecond)

			w := MustSucceed(svc.Framer.OpenWriter(ctx, writer.Config{
				Keys:           keys,
				Start:          10 * telem.SecondTS,
				ControlSubject: control.Subject{Name: "free-grouped", Key: "free-grouped", Group: 55},
			}))
			Expect(w.Write(frame.NewMulti(
				keys,
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
					telem.NewSeriesV[int64](7, 8, 9),
				},
			))).To(BeTrue())
			Consistently(readerRes.Outlet(), 100*time.Millisecond).ShouldNot(Receive())

			Expect(w.Close()).To(Succeed())
			streamerReq.Close()
			confluence.Drain(readerRes)
		})
	})
	// These tests cover the SendOpenAck happens-before guarantee: once a caller sees
	// the open ack, a subsequent write must be observable on the streamer. The relay
	// implements this by waiting for the tapper to install the streamer's keys into the
	// gateway tap before sending the ack. The cases below also exercise the edge cases
	// that could deadlock the synchronous wait — empty initial keys, non-free leases,
	// context cancellation mid-wait, and concurrent streamers.
	Describe("SendOpenAck", Ordered, func() {
		var svc mock.Node
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			svc = mock.NewNode(ctx)
		})

		newFreeChannels := func(ctx context.Context, n int) []channel.Channel {
			chs := make([]channel.Channel, n)
			ts := time.Now().UnixNano()
			for i := range chs {
				chs[i] = channel.Channel{
					Name:        fmt.Sprintf("free_%d_%d", ts, i),
					Virtual:     true,
					DataType:    telem.Int64T,
					Leaseholder: node.KeyFree,
				}
			}
			return MustSucceed(svc.Channel.Create(ctx, chs))
		}

		It("Should deliver a write issued immediately after the open ack on a free channel", func(ctx SpecContext) {
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
		})

		It("Should deliver the open ack when the streamer is opened with no keys", func(ctx SpecContext) {
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
		})

		It("Should deliver the open ack when the streamer subscribes only to gateway-leased channels", func(ctx SpecContext) {
			// Channels with no explicit leaseholder are assigned to the host node and
			// are served by the gateway tap alongside free channels. This exercises the
			// ack path for a demand whose keys are all host-leased.
			chs := MustSucceed(svc.Channel.Create(ctx, newChannelSet()))
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
		})

		It("Should not leak when the streamer's context is canceled before the open ack is delivered", func(ctx SpecContext) {
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
		})

		It("Should serve concurrent streamers on overlapping free channels without deadlocking", func(ctx SpecContext) {
			chs := newFreeChannels(ctx, 2)
			keys := channel.KeysFromChannels(chs)

			const streamerCount = 4
			readers := make([]relay.Streamer, streamerCount)
			outlets := make([]confluence.Outlet[relay.Response], streamerCount)
			inlets := make([]confluence.Inlet[relay.Request], streamerCount)
			sCtx, cancel := signal.Isolated()
			defer cancel()
			for i := range streamerCount {
				readers[i] = MustSucceed(svc.Framer.NewStreamer(relay.StreamerConfig{
					Keys:        keys,
					SendOpenAck: new(true),
				}))
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
		})

		It("Should apply a dynamic key update before subsequent writes are dropped", func(ctx SpecContext) {
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
		})
	})
	// The relay does not validate streamer keys against storage: the service layer
	// validates at open time, but a channel can be deleted between that check and the
	// demand reaching the tapper, and mid-stream key updates are never validated. The
	// gateway tap must drop keys that don't resolve in local storage instead of
	// failing outright, since a failed tap starves every streamer demanding host
	// channels on the node.
	Describe("Unknown Keys", Ordered, func() {
		var svc mock.Node
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			svc = mock.NewNode(ctx)
		})

		newUniqueChannels := func(ctx context.Context, n int) []channel.Channel {
			chs := make([]channel.Channel, n)
			ts := time.Now().UnixNano()
			for i := range chs {
				chs[i] = channel.Channel{
					Name:     fmt.Sprintf("unknown_keys_%d_%d", ts, i),
					Virtual:  true,
					DataType: telem.Int64T,
				}
			}
			return MustSucceed(svc.Channel.Create(ctx, chs))
		}

		It("Should deliver frames for valid keys when the streamer also demands an unknown host-leased key", func(ctx SpecContext) {
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

			w := MustSucceed(svc.Framer.OpenWriter(ctx, writer.Config{
				Keys:  keys,
				Start: 10 * telem.SecondTS,
			}))
			defer func() { Expect(w.Close()).To(Succeed()) }()
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
		})

		It("Should not starve a concurrent streamer with valid keys when another streamer demands an unknown key", func(ctx SpecContext) {
			chs := newUniqueChannels(ctx, 2)
			keys := channel.KeysFromChannels(chs)
			unknown := channel.NewKey(node.KeyBootstrapper, 0xFFFFE)

			// The streamer with the unknown key opens first, so its demand is the one
			// that opens the shared gateway tap.
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

			w := MustSucceed(svc.Framer.OpenWriter(ctx, writer.Config{
				Keys:  keys,
				Start: 10 * telem.SecondTS,
			}))
			defer func() { Expect(w.Close()).To(Succeed()) }()
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
		})
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

func gatewayOnlyScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	node := mock.OpenNode(ctx)
	channels = MustSucceed(node.Channel.Create(ctx, channels))
	return scenario{
		resCount: 1,
		name:     "Gateway Only",
		channels: channels,
		dist:     node,
		close:    node,
	}
}

func peerOnlyScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	cluster := mock.OpenCluster(ctx, 4)
	dist := cluster.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = node.Key(i + 2)
		channels[i] = ch
	}
	channels = MustSucceed(dist.Channel.Create(ctx, channels))
	return scenario{
		resCount: 3,
		name:     "Peer Only",
		channels: channels,
		dist:     dist,
		close:    cluster,
	}
}
func mixedScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	cluster := mock.OpenCluster(ctx, 3)
	gateway := cluster.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = node.Key(i + 1)
		channels[i] = ch
	}
	channels = MustSucceed(gateway.Channel.Create(ctx, channels))
	return scenario{
		resCount: 3,
		name:     "Mixed Gateway and Peer",
		channels: channels,
		dist:     gateway,
		close:    cluster,
	}
}

func freeScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	dist := mock.OpenNode(ctx)
	for i, ch := range channels {
		ch.Leaseholder = node.KeyFree
		ch.Virtual = true
		channels[i] = ch
	}
	channels = MustSucceed(dist.Channel.Create(ctx, channels))
	return scenario{
		name:     "Free Channel",
		resCount: 1,
		channels: channels,
		dist:     dist,
		close:    dist,
	}
}
