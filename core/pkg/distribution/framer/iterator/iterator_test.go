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
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator", func() {
	Describe("Happy Path", Ordered, func() {
		scenarios := []func() scenario{
			gatewayOnlyScenario,
			peerOnlyScenario,
		}
		for i, sF := range scenarios {
			_sF := sF
			var s scenario
			Describe(fmt.Sprintf("Scenario: %v - Iteration", i), func() {
				BeforeAll(func() {
					s = _sF()
					writer := MustSucceed(s.dist.Framer.OpenWriter(context.TODO(), writer.Config{
						Keys:  s.keys,
						Start: 10 * telem.SecondTS,
						Sync:  config.True(),
					}))
					MustSucceed(writer.Write(core.MultiFrame(
						s.keys,
						[]telem.Series{
							telem.NewSeriesSecondsTSV(10, 11, 12),
						},
					)))
					MustSucceed(writer.Write(core.MultiFrame(
						s.keys,
						[]telem.Series{
							telem.NewSeriesSecondsTSV(13, 14, 15, 16, 17),
						},
					)))
					MustSucceed(writer.Write(core.MultiFrame(
						s.keys,
						[]telem.Series{
							telem.NewSeriesSecondsTSV(18, 19, 20, 21, 22),
						},
					)))
					MustSucceed(writer.Commit())
					Expect(writer.Close()).To(Succeed())
				})
				AfterAll(func() { Expect(s.close.Close()).To(Succeed()) })
				Specify(fmt.Sprintf("Scenario: %v - Iteration", i), func() {
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

				Specify("Auto chunk", func() {
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

				Specify("Reverse Auto Chunk", func() {
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
	name  string
	keys  channel.Keys
	dist  mock.Node
	close io.Closer
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

func gatewayOnlyScenario() scenario {
	channels := newChannelSet()
	builder := mock.ProvisionCluster(ctx, 1)
	dist := builder.Nodes[1]
	Expect(dist.Channel.NewWriter(nil).CreateMany(ctx, &channels)).To(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Gateway Only", keys: keys, dist: dist, close: builder}
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
	Eventually(func(g Gomega) {
		var chs []channel.Channel
		err := dist.Channel.NewRetrieve().Entries(&chs).WhereKeys(channel.KeysFromChannels(channels)...).Exec(ctx, nil)
		g.Expect(err).To(Succeed())
		g.Expect(chs).To(HaveLen(len(channels)))
	}).Should(Succeed())
	keys := channel.KeysFromChannels(channels)
	return scenario{name: "Peer Only", keys: keys, dist: dist, close: builder}
}
