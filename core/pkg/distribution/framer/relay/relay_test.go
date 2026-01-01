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
	"fmt"
	"io"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	core "github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type scenario struct {
	resCount int
	name     string
	channels []channel.Channel
	dist     mock.Node
	close    io.Closer
}

var _ = Describe("Relay", func() {
	Describe("Happy Path", Ordered, func() {

		scenarios := []func() scenario{
			gatewayOnlyScenario,
			peerOnlyScenario,
			mixedScenario,
			freeScenario,
		}
		for i, sF := range scenarios {
			_sF := sF
			var s scenario
			BeforeAll(func() { s = _sF() })
			AfterAll(func() { Expect(s.close.Close()).To(Succeed()) })
			Specify(fmt.Sprintf("Scenario: %v - Happy Path", i), func() {
				keys := channel.KeysFromChannels(s.channels)
				reader := MustSucceed(s.dist.Framer.Relay.NewStreamer(ctx, relay.StreamerConfig{
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
				writeF := core.NewMulti(
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
					f = core.Merge([]core.Frame{f, res.Frame})
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
	Describe("Errors", func() {
		It("Should raise an error if a channel is not found", func() {
			builder := mock.ProvisionCluster(ctx, 1)
			defer func() {
				Expect(builder.Close()).To(Succeed())
			}()
			svc := builder.Nodes[1]
			_, err := svc.Framer.Relay.NewStreamer(ctx, relay.StreamerConfig{
				Keys: []channel.Key{12345},
			})
			Expect(err).To(HaveOccurredAs(query.NotFound))
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

func gatewayOnlyScenario() scenario {
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

func peerOnlyScenario() scenario {
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
func mixedScenario() scenario {
	channels := newChannelSet()
	cluster_ := mock.ProvisionCluster(ctx, 3)
	node := cluster_.Nodes[1]
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
		close:    cluster_,
	}
}

func freeScenario() scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 1)
	dist := builder.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = cluster.Free
		ch.Virtual = true
		channels[i] = ch
	}
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		g.Expect(dist.Channel.NewRetrieve().Entries(&chs).WhereKeys(keys...).
			Exec(ctx, nil)).To(Succeed())
	})
	return scenario{
		name:     "Free Channel",
		resCount: 1,
		channels: channels,
		dist:     dist,
		close:    builder,
	}
}
