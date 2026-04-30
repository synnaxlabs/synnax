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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
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
					s = sF(ctx)
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
				AfterAll(func() { Expect(s.close.Close()).To(Succeed()) })
				Specify(fmt.Sprintf("Scenario: %v - Iteration", i), func(ctx SpecContext) {
					iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
						Keys:   s.keys,
						Bounds: telem.TimeRangeMax,
					}))
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(4 * telem.Second)).To(BeTrue())
					Expect(iter.Value().SeriesAt(0)).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(10, 11, 12, 13)))
					Expect(iter.SeekLast()).To(BeTrue())
					Expect(iter.Prev(6 * telem.Second)).To(BeTrue())
					Expect(iter.Value().SeriesAt(0)).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(17, 18, 19, 20, 21, 22)))

					Expect(iter.SeekGE(100 * telem.SecondTS)).To(BeFalse())
					Expect(iter.Valid()).To(BeFalse())
					Expect(iter.SeekLE(22*telem.SecondTS + 1)).To(BeTrue())
					Expect(iter.Prev(2 * telem.Second)).To(BeTrue())
					Expect(iter.Value().SeriesAt(0)).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(21, 22)))

					Expect(iter.SeekLE(0 * telem.SecondTS)).To(BeFalse())
					Expect(iter.Valid()).To(BeFalse())
					Expect(iter.SeekGE(13 * telem.SecondTS)).To(BeTrue())
					Expect(iter.Next(20 * telem.Second)).To(BeTrue())
					Expect(iter.Value().SeriesAt(0)).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(13, 14, 15, 16, 17, 18, 19, 20, 21, 22)))

					Expect(iter.Close()).To(Succeed())
				})

				Specify("Auto chunk", func(ctx SpecContext) {
					iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
						Keys:      s.keys,
						Bounds:    telem.TimeRangeMax,
						ChunkSize: 3,
					}))
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
					Expect(iter.Value().SeriesAt(0)).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(10, 11, 12)))
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
					Expect(iter.Value().SeriesAt(0)).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(13, 14, 15)))
					Expect(iter.Next(iterator.AutoSpan)).To(BeTrue())
					Expect(iter.Value().SeriesAt(0)).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(16, 17, 18)))

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
					Expect(iter.Value().SeriesAt(0)).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(20, 21, 22)))
					Expect(iter.Prev(iterator.AutoSpan)).To(BeTrue())
					Expect(iter.Value().SeriesAt(0)).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(17, 18, 19)))
					Expect(iter.Prev(iterator.AutoSpan)).To(BeTrue())
					Expect(iter.Value().SeriesAt(0)).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(14, 15, 16)))
					Expect(iter.Prev(iterator.AutoSpan)).To(BeTrue())
					Expect(iter.Value().SeriesAt(0)).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(11, 12, 13)))
					Expect(iter.Prev(iterator.AutoSpan)).To(BeTrue())
					Expect(iter.Value().SeriesAt(0)).To(telem.MatchWrittenSeries(telem.NewSeriesSecondsTSV(10)))
					Expect(iter.Prev(iterator.AutoSpan)).To(BeFalse())
					Expect(iter.Close()).To(Succeed())
				})
			})
		}
	})
})

type scenario struct {
	dist  mock.Node
	close io.Closer
	name  string
	keys  channel.Keys
}

func newChannelSet() []channel.Channel {
	return []channel.Channel{
		{
			Name:     "test1",
			IsIndex:  true,
			DataType: telem.TimeStampT,
		},
	}
}

func gatewayOnlyScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 1)
	dist := builder.Nodes[1]
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Gateway Only", keys: keys, dist: dist, close: builder}
}

func peerOnlyScenario(ctx context.Context) scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 4)
	dist := builder.Nodes[1]
	for i, ch := range channels {
		ch.Leaseholder = node.Key(i + 2)
		channels[i] = ch
	}
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		err := dist.Channel.NewRetrieve().Entries(&chs).WhereKeys(channel.KeysFromChannels(channels)...).Exec(ctx, nil)
		g.Expect(err).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Peer Only", keys: keys, dist: dist, close: builder}
}

func mixedScenario(ctx context.Context) scenario {
	channels := []channel.Channel{
		{Name: "mixed_gateway", IsIndex: true, DataType: telem.TimeStampT, Leaseholder: 1},
		{Name: "mixed_peer", IsIndex: true, DataType: telem.TimeStampT, Leaseholder: 2},
	}
	builder := mock.ProvisionCluster(ctx, 2)
	dist := builder.Nodes[1]
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		g.Expect(dist.Channel.NewRetrieve().Entries(&chs).WhereKeys(keys...).
			Exec(ctx, nil)).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	return scenario{name: "Mixed Gateway and Peer", keys: keys, dist: dist, close: builder}
}
