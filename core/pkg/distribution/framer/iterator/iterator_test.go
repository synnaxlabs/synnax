// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator_test

import (
	"context"
	"fmt"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator", func() {
	Describe("Happy Path", Ordered, func() {
		scenarios := []func(context.Context) scenario{
			gatewayOnlyScenario,
			peerOnlyScenario,
			mixedScenario,
		}
		for i, sF := range scenarios {
			var s scenario
			Describe(fmt.Sprintf("Scenario: %v - Iteration", i), func() {
				BeforeAll(func(ctx SpecContext) {
					ShouldNotLeakGoroutines()
					s = DeferClose(sF(ctx))
					writer := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
						Keys:  s.keys,
						Start: 10 * telem.SecondTS,
						Sync:  new(true),
					}))
					writeBatch := func(ts ...telem.TimeStamp) {
						series := make([]telem.Series, len(s.keys))
						for i := range s.keys {
							cp := make([]telem.TimeStamp, len(ts))
							copy(cp, ts)
							series[i] = telem.NewSeriesSecondsTSV(cp...)
						}
						Expect(writer.Write(frame.NewMulti(s.keys, series))).
							To(BeTrue())
					}
					writeBatch(10, 11, 12)
					writeBatch(13, 14, 15, 16, 17)
					writeBatch(18, 19, 20, 21, 22)
					Expect(writer.Commit()).To(BeNumerically("==", telem.SecondTS*22+1))
					Expect(writer.Close()).To(Succeed())
				})
				Specify(
					fmt.Sprintf("Scenario: %v - Iteration", i),
					func(ctx SpecContext) {
						iter := MustSucceed(
							s.dist.Framer.OpenIterator(ctx, iterator.Config{
								Keys:   s.keys,
								Bounds: telem.TimeRangeMax,
							}),
						)
						Expect(iter.SeekFirst()).To(BeTrue())
						Expect(iter.Next(4 * telem.Second)).To(BeTrue())
						Expect(
							iter.Value().SeriesAt(0),
						).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(10, 11, 12, 13)))
						Expect(iter.SeekLast()).To(BeTrue())
						Expect(iter.Prev(6 * telem.Second)).To(BeTrue())
						Expect(
							iter.Value().SeriesAt(0),
						).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(17, 18, 19, 20, 21, 22)))

						Expect(iter.SeekGE(100 * telem.SecondTS)).To(BeFalse())
						Expect(iter.Valid()).To(BeFalse())
						Expect(iter.SeekLE(22*telem.SecondTS + 1)).To(BeTrue())
						Expect(iter.Prev(2 * telem.Second)).To(BeTrue())
						Expect(
							iter.Value().SeriesAt(0),
						).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(21, 22)))

						Expect(iter.SeekLE(0 * telem.SecondTS)).To(BeFalse())
						Expect(iter.Valid()).To(BeFalse())
						Expect(iter.SeekGE(13 * telem.SecondTS)).To(BeTrue())
						Expect(iter.Next(20 * telem.Second)).To(BeTrue())
						Expect(
							iter.Value().SeriesAt(0),
						).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(13, 14, 15, 16, 17, 18, 19, 20, 21, 22)))

						Expect(iter.Close()).To(Succeed())
					},
				)

				Specify("Auto chunk", func(ctx SpecContext) {
					iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
						Keys:      s.keys,
						Bounds:    telem.TimeRangeMax,
						ChunkSize: 3,
					}))
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
					Expect(
						iter.Value().SeriesAt(0),
					).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(10, 11, 12)))
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
					Expect(
						iter.Value().SeriesAt(0),
					).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(13, 14, 15)))
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
					Expect(
						iter.Value().SeriesAt(0),
					).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(16, 17, 18)))

					Expect(iter.Close()).To(Succeed())
				})

				Specify("Reverse Auto Chunk", func(ctx SpecContext) {
					iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
						Keys:      s.keys,
						Bounds:    telem.TimeRangeMax,
						ChunkSize: 3,
					}))
					Expect(iter.SeekLast()).To(BeTrue())
					Expect(iter.Prev(iterator.AutoSpan)).To(BeTrue())
					Expect(
						iter.Value().SeriesAt(0),
					).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(20, 21, 22)))
					Expect(iter.Prev(iterator.AutoSpan)).To(BeTrue())
					Expect(
						iter.Value().SeriesAt(0),
					).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(17, 18, 19)))
					Expect(iter.Prev(iterator.AutoSpan)).To(BeTrue())
					Expect(
						iter.Value().SeriesAt(0),
					).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(14, 15, 16)))
					Expect(iter.Prev(iterator.AutoSpan)).To(BeTrue())
					Expect(
						iter.Value().SeriesAt(0),
					).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(11, 12, 13)))
					Expect(iter.Prev(iterator.AutoSpan)).To(BeTrue())
					Expect(
						iter.Value().SeriesAt(0),
					).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(10)))
					Expect(iter.Prev(iterator.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})
			})
		}
	})
})

// openFaultyCluster opens a two-node cluster whose nodes each store telemetry on their
// own fault-injecting file system, keyed by node.
func openFaultyCluster(ctx context.Context) (*mock.Cluster, map[node.Key]*FaultyFS) {
	GinkgoHelper()
	var (
		cluster = mock.OpenCluster(ctx, 0)
		faulty  = make(map[node.Key]*FaultyFS)
	)
	for range 2 {
		var (
			nodeFS = WrapFaultyFS(xfs.NewMem())
			store  = &storage.Layer{
				KV: DeferClose(memkv.New()),
				TS: DeferClose(MustSucceed(ts.Open(ctx, ts.Config{FS: nodeFS}))),
			}
			n = cluster.Provision(ctx, distribution.LayerConfig{Storage: store})
		)
		faulty[n.Cluster.HostKey()] = nodeFS
	}
	return cluster, faulty
}

var _ = Describe("Close", func() {
	It("Should reject every call after Close", func(ctx SpecContext) {
		s := DeferClose(gatewayOnlyScenario(ctx))
		iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
			Keys:   s.keys,
			Bounds: telem.TimeRangeMax,
		}))
		Expect(iter.Close()).To(Succeed())
		Expect(iter.SeekFirst()).To(BeFalse())
		Expect(iter.Next(iterator.AutoSpan)).To(BeFalse())
		Expect(iter.Valid()).To(BeFalse())
		Expect(iter.Error()).To(MatchError(iterator.ErrClosed))
		Expect(iter.Close()).To(Succeed())
	})
})

var _ = Describe("Read failure", func() {
	DescribeTable(
		"Should return the read error from Close",
		func(ctx SpecContext, faultyNode node.Key, leaseholders ...node.Key) {
			cluster, faulty := openFaultyCluster(ctx)
			defer func() { Expect(cluster.Close()).To(Succeed()) }()
			channels := make([]channel.Channel, len(leaseholders))
			for i, leaseholder := range leaseholders {
				channels[i] = channel.Channel{
					Name:        fmt.Sprintf("failure_%d", i),
					IsIndex:     true,
					DataType:    telem.TimestampT,
					Leaseholder: leaseholder,
				}
			}
			gateway := cluster.Nodes[1]
			channels = MustSucceed(gateway.Channel.Create(ctx, channels))
			keys := channel.KeysFromChannels(channels)
			w := MustSucceed(gateway.Framer.OpenWriter(ctx, writer.Config{
				Keys:  keys,
				Start: 10 * telem.SecondTS,
				Sync:  new(true),
			}))
			series := make([]telem.Series, len(keys))
			for i := range keys {
				series[i] = telem.NewSeriesSecondsTSV(10, 11, 12)
			}
			Expect(w.Write(frame.NewMulti(keys, series))).To(BeTrue())
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			iter := MustSucceed(gateway.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   keys,
				Bounds: telem.TimeRangeMax,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			faulty[faultyNode].SetOptions(WithFailReadAt())
			for iter.Next(iterator.AutoSpan) {
			}
			Expect(iter.Close()).To(MatchError(ContainSubstring(ErrFault.Error())))
			Expect(iter.Close()).To(Succeed())
		},
		Entry("on the gateway", node.Key(1), node.Key(1)),
		Entry("on a peer", node.Key(2), node.Key(2)),
		Entry(
			"on a peer beside a healthy gateway",
			node.Key(2),
			node.Key(1),
			node.Key(2),
		),
	)
})

type scenario struct {
	dist     mock.Node
	close    io.Closer
	name     string
	keys     channel.Keys
	channels []channel.Channel
}

func (s scenario) Close() error { return s.close.Close() }

func newChannelSet() []channel.Channel {
	return []channel.Channel{
		{
			Name:     "test1",
			IsIndex:  true,
			DataType: telem.TimestampT,
		},
	}
}

func gatewayOnlyScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.OpenCluster(ctx, 1)
	dist := builder.Nodes[1]
	channels = MustSucceed(dist.Channel.Create(ctx, channels))
	keys := channel.KeysFromChannels(channels)
	return scenario{
		name:     "Gateway Only",
		keys:     keys,
		dist:     dist,
		close:    builder,
		channels: channels,
	}
}

func peerOnlyScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.OpenCluster(ctx, 4)
	dist := builder.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = node.Key(i + 2)
		channels[i] = ch
	}
	channels = MustSucceed(dist.Channel.Create(ctx, channels))
	keys := channel.KeysFromChannels(channels)
	return scenario{
		name:     "Peer Only",
		keys:     keys,
		dist:     dist,
		close:    builder,
		channels: channels,
	}
}

func mixedScenario(ctx context.Context) scenario {
	channels := []channel.Channel{
		{
			Name:        "mixed_gateway",
			IsIndex:     true,
			DataType:    telem.TimestampT,
			Leaseholder: 1,
		},
		{Name: "mixed_peer", IsIndex: true, DataType: telem.TimestampT, Leaseholder: 2},
	}
	builder := mock.OpenCluster(ctx, 2)
	dist := builder.Nodes[1]
	channels = MustSucceed(dist.Channel.Create(ctx, channels))
	keys := channel.KeysFromChannels(channels)
	return scenario{
		name:     "Mixed Gateway and Peer",
		keys:     keys,
		dist:     dist,
		close:    builder,
		channels: channels,
	}
}
