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
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/query"
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
			BeforeAll(func(ctx SpecContext) { s = _sF(ctx) })
			AfterAll(func() { Expect(s.close.Close()).To(Succeed()) })
			Specify(fmt.Sprintf("Scenario: %v - Happy Path", i), func(ctx SpecContext) {
				keys := channel.KeysFromChannels(s.channels)
				reader := MustSucceed(s.dist.Framer.NewStreamer(ctx, relay.StreamerConfig{
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
			builder := mock.ProvisionCluster(ctx, 1)
			defer func() {
				Expect(builder.Close()).To(Succeed())
			}()
			svc := builder.Nodes[1]
			Expect(svc.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)

			reader := MustSucceed(svc.Framer.NewStreamer(ctx, relay.StreamerConfig{
				Keys:          keys,
				ExcludeGroups: []uint32{99},
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			streamerReq, readerRes := confluence.Attach(reader, 10)
			reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(10 * time.Millisecond)

			w := MustSucceed(svc.Framer.OpenWriter(ctx, writer.Config{
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
			builder := mock.ProvisionCluster(ctx, 1)
			defer func() {
				Expect(builder.Close()).To(Succeed())
			}()
			svc := builder.Nodes[1]
			Expect(svc.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)

			reader := MustSucceed(svc.Framer.NewStreamer(ctx, relay.StreamerConfig{
				Keys:          keys,
				ExcludeGroups: []uint32{99},
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			streamerReq, readerRes := confluence.Attach(reader, 10)
			reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(10 * time.Millisecond)

			w := MustSucceed(svc.Framer.OpenWriter(ctx, writer.Config{
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
			builder := mock.ProvisionCluster(ctx, 1)
			defer func() {
				Expect(builder.Close()).To(Succeed())
			}()
			svc := builder.Nodes[1]
			Expect(svc.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)

			reader := MustSucceed(svc.Framer.NewStreamer(ctx, relay.StreamerConfig{
				Keys:          keys,
				ExcludeGroups: []uint32{99},
			}))
			sCtx, cancel := signal.Isolated()
			defer cancel()
			streamerReq, readerRes := confluence.Attach(reader, 10)
			reader.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			time.Sleep(10 * time.Millisecond)

			w := MustSucceed(svc.Framer.OpenWriter(ctx, writer.Config{
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
			builder := mock.ProvisionCluster(ctx, 1)
			defer func() {
				Expect(builder.Close()).To(Succeed())
			}()
			svc := builder.Nodes[1]
			for i, ch := range channels {
				ch.Leaseholder = cluster.NodeKeyFree
				ch.Virtual = true
				channels[i] = ch
			}
			Expect(svc.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
			keys := channel.KeysFromChannels(channels)

			reader := MustSucceed(svc.Framer.NewStreamer(ctx, relay.StreamerConfig{
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
	Describe("Errors", func() {
		It("Should raise an error if a channel is not found", func(ctx SpecContext) {
			builder := mock.ProvisionCluster(ctx, 1)
			defer func() {
				Expect(builder.Close()).To(Succeed())
			}()
			svc := builder.Nodes[1]
			_, err := svc.Framer.NewStreamer(ctx, relay.StreamerConfig{
				Keys: []channel.Key{12345},
			})
			Expect(err).To(MatchError(query.ErrNotFound))
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
	builder := mock.ProvisionCluster(ctx, 1)
	svc := builder.Nodes[1]
	Expect(svc.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	return scenario{
		resCount: 1,
		name:     "Gateway Only",
		channels: channels,
		dist:     svc,
		close:    svc,
	}
}

func peerOnlyScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 4)
	dist := builder.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = cluster.NodeKey(i + 2)
		channels[i] = ch
	}
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		g.Expect(dist.Channel.NewRetrieve().Entries(&chs).WhereKeys(keys...).Exec(ctx, nil)).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	return scenario{
		resCount: 3,
		name:     "Peer Only",
		channels: channels,
		dist:     dist,
		close:    dist,
	}
}
func mixedScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	clstr := mock.ProvisionCluster(ctx, 3)
	node := clstr.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = cluster.NodeKey(i + 1)
		channels[i] = ch
	}
	Expect(node.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		g.Expect(node.Channel.NewRetrieve().Entries(&chs).WhereKeys(keys...).Exec(ctx, nil)).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	return scenario{
		resCount: 3,
		name:     "Mixed Gateway and Peer",
		channels: channels,
		dist:     node,
		close:    clstr,
	}
}

func freeScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 1)
	dist := builder.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = cluster.NodeKeyFree
		ch.Virtual = true
		channels[i] = ch
	}
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		g.Expect(dist.Channel.NewRetrieve().Entries(&chs).WhereKeys(keys...).
			Exec(ctx, nil)).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	return scenario{
		name:     "Free Channel",
		resCount: 1,
		channels: channels,
		dist:     dist,
		close:    builder,
	}
}
