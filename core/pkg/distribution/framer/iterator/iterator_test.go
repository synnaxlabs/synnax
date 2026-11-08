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
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Iterator", func() {
	Describe("Happy Path", Ordered, func() {
		scenarios := []func(context.Context) scenario{
			gatewayOnlyScenario,
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
						GinkgoHelper()
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

				Specify("Aggregation", func(ctx SpecContext) {
					iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
						Keys:   s.keys,
						Bounds: telem.TimeRangeMax,
						Reduction: telem.Reduction{Variant: telem.LimitReduction{
							Aggregation: telem.AggregationMinMax,
							PointLimit:  4,
						}},
					}))
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
					for _, series := range iter.Value().SeriesI() {
						Expect(series).To(telem.MatchSeriesData(
							telem.NewSeriesSecondsTSV(10, 17, 18, 22),
						))
						Expect(series.AlignmentMultiple).To(Equal(uint32(4)))
					}
					Expect(iter.Value().Count()).To(Equal(len(s.keys)))
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

				Specify("Downsample", func(ctx SpecContext) {
					iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
						Keys:   s.keys,
						Bounds: telem.TimeRangeMax,
						Reduction: telem.Reduction{
							Variant: telem.StrideReduction{Factor: 3},
						},
					}))
					Expect(iter.SeekFirst()).To(BeTrue())
					Expect(iter.Next(20 * telem.Second)).To(BeTrue())
					Expect(
						iter.Value().SeriesAt(0),
					).To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(10, 13, 16, 19, 22)))
					Expect(
						iter.Value().SeriesAt(0).AlignmentMultiple,
					).To(Equal(uint32(3)))
					Expect(iter.Close()).To(Succeed())
				})
			})
		}
	})
})

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
	It("Should return the read error from Close", func(ctx SpecContext) {
		faulty := WrapFaultyFS(xfs.NewMem())
		store := &storage.Layer{
			KV: DeferClose(memkv.New()),
			TS: DeferClose(MustSucceed(ts.Open(ctx, ts.Config{FS: faulty}))),
		}
		gateway := testutil.NewNode(ctx, distribution.LayerConfig{Storage: store})
		channels := MustSucceed(gateway.Channel.Create(ctx, []channel.Channel{{
			Name:     "failure",
			IsIndex:  true,
			DataType: telem.TimestampT,
		}}))
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
		faulty.SetOptions(WithFailReadAt())
		for iter.Next(iterator.AutoSpan) {
		}
		Expect(iter.Close()).To(MatchError(ContainSubstring(ErrFault.Error())))
		Expect(iter.Close()).To(Succeed())
	})
})

type scenario struct {
	dist     testutil.Node
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
	GinkgoHelper()
	channels := newChannelSet()
	dist := testutil.OpenNode(ctx)
	channels = MustSucceed(dist.Channel.Create(ctx, channels))
	keys := channel.KeysFromChannels(channels)
	return scenario{
		name:     "Gateway Only",
		keys:     keys,
		dist:     dist,
		close:    dist,
		channels: channels,
	}
}
