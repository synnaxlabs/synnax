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
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type autoIndexScenario struct {
	closer io.Closer
	dist   mock.Node
	idx    channel.Channel
	data   channel.Channel
}

func openAutoIndexScenario(ctx context.Context) autoIndexScenario {
	builder := mock.ProvisionCluster(ctx, 1)
	dist := builder.Nodes[1]
	idx := channel.Channel{
		Name:     channel.NewRandomName(),
		IsIndex:  true,
		DataType: telem.TimeStampT,
	}
	Expect(dist.Channel.Create(ctx, &idx)).To(Succeed())
	data := channel.Channel{
		Name:       channel.NewRandomName(),
		DataType:   telem.Float64T,
		LocalIndex: idx.LocalKey,
	}
	Expect(dist.Channel.Create(ctx, &data)).To(Succeed())
	return autoIndexScenario{closer: builder, dist: dist, idx: idx, data: data}
}

var _ = Describe("AutoIndexing", func() {
	Describe("Pure data-only keys", func() {
		It("Should generate timestamps for a writer opened with only a data channel", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			before := telem.Now()
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:         []channel.Key{s.data.Key()},
				Sync:         new(true),
				AutoIndexing: new(true),
			}))
			MustSucceed(w.Write(frame.NewUnary(
				s.data.Key(),
				telem.NewSeriesV(1.1, 2.2, 3.3),
			)))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())
			after := telem.Now()

			iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   []channel.Key{s.idx.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
			ts := telem.UnmarshalSeries[telem.TimeStamp](iter.Value().SeriesAt(0))
			Expect(iter.Close()).To(Succeed())

			Expect(ts).To(HaveLen(3))
			Expect(ts[0]).To(BeNumerically(">=", before))
			Expect(ts[2]).To(BeNumerically("<=", after+telem.TimeStamp(3)))
			Expect(ts[1]).To(Equal(ts[0] + 1))
			Expect(ts[2]).To(Equal(ts[0] + 2))
		})

		It("Should be strictly monotonic across multiple Write calls", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:         []channel.Key{s.data.Key()},
				Sync:         new(true),
				AutoIndexing: new(true),
			}))
			MustSucceed(w.Write(frame.NewUnary(
				s.data.Key(),
				telem.NewSeriesV[float64](10, 11, 12),
			)))
			MustSucceed(w.Write(frame.NewUnary(
				s.data.Key(),
				telem.NewSeriesV[float64](13, 14, 15),
			)))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   []channel.Key{s.idx.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			var all []telem.TimeStamp
			for iter.Next(telem.TimeSpanMax) {
				for _, ser := range iter.Value().RawSeries() {
					all = append(all, telem.UnmarshalSeries[telem.TimeStamp](ser)...)
				}
			}
			Expect(iter.Close()).To(Succeed())
			Expect(len(all)).To(BeNumerically(">=", 6))
			for i := 1; i < len(all); i++ {
				Expect(all[i]).To(BeNumerically(">", all[i-1]))
			}
		})
	})

	Describe("Multiple data channels", func() {
		It("Should share a single auto-generated index across data channels with the same LocalIndex", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			other := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float64T,
				LocalIndex: s.idx.LocalKey,
			}
			Expect(s.dist.Channel.Create(ctx, &other)).To(Succeed())

			keys := []channel.Key{s.data.Key(), other.Key()}
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:         keys,
				Sync:         new(true),
				AutoIndexing: new(true),
			}))
			MustSucceed(w.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesV[float64](1, 2, 3),
				telem.NewSeriesV[float64](4, 5, 6),
			})))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   []channel.Key{s.idx.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
			ts := telem.UnmarshalSeries[telem.TimeStamp](iter.Value().SeriesAt(0))
			Expect(iter.Close()).To(Succeed())
			Expect(ts).To(HaveLen(3))
		})

		It("Should size each generated index to its own data channel when lengths differ", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			idx2 := channel.Channel{
				Name:     channel.NewRandomName(),
				IsIndex:  true,
				DataType: telem.TimeStampT,
			}
			Expect(s.dist.Channel.Create(ctx, &idx2)).To(Succeed())
			data2 := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float64T,
				LocalIndex: idx2.LocalKey,
			}
			Expect(s.dist.Channel.Create(ctx, &data2)).To(Succeed())

			keys := []channel.Key{s.data.Key(), data2.Key()}
			before := telem.Now()
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:         keys,
				Sync:         new(true),
				AutoIndexing: new(true),
			}))
			MustSucceed(w.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesV[float64](1, 2, 3),
				telem.NewSeriesV[float64](10, 20, 30, 40),
			})))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			read := func(k channel.Key) []telem.TimeStamp {
				iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
					Keys:   []channel.Key{k},
					Bounds: telem.TimeRangeMax,
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				var ts []telem.TimeStamp
				for iter.Next(telem.TimeSpanMax) {
					for _, ser := range iter.Value().RawSeries() {
						ts = append(ts, telem.UnmarshalSeries[telem.TimeStamp](ser)...)
					}
				}
				Expect(iter.Close()).To(Succeed())
				return ts
			}

			ts1 := read(s.idx.Key())
			Expect(ts1).To(HaveLen(3))
			Expect(ts1[0]).To(BeNumerically(">=", before))
			Expect(ts1[1]).To(Equal(ts1[0] + 1))
			Expect(ts1[2]).To(Equal(ts1[0] + 2))

			ts2 := read(idx2.Key())
			Expect(ts2).To(HaveLen(4))
			Expect(ts2[0]).To(Equal(ts1[0]))
			Expect(ts2[1]).To(Equal(ts1[0] + 1))
			Expect(ts2[2]).To(Equal(ts1[0] + 2))
			Expect(ts2[3]).To(Equal(ts1[0] + 3))
		})

		It("Should generate co-aligned timestamps for data channels with different indexes", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			idx2 := channel.Channel{
				Name:     channel.NewRandomName(),
				IsIndex:  true,
				DataType: telem.TimeStampT,
			}
			Expect(s.dist.Channel.Create(ctx, &idx2)).To(Succeed())
			data2 := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float64T,
				LocalIndex: idx2.LocalKey,
			}
			Expect(s.dist.Channel.Create(ctx, &data2)).To(Succeed())

			keys := []channel.Key{s.data.Key(), data2.Key()}
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:         keys,
				Sync:         new(true),
				AutoIndexing: new(true),
			}))
			MustSucceed(w.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesV[float64](1, 2, 3),
				telem.NewSeriesV[float64](4, 5, 6),
			})))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			read := func(k channel.Key) []telem.TimeStamp {
				iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
					Keys:   []channel.Key{k},
					Bounds: telem.TimeRangeMax,
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
				ts := telem.UnmarshalSeries[telem.TimeStamp](iter.Value().SeriesAt(0))
				Expect(iter.Close()).To(Succeed())
				return ts
			}
			Expect(read(s.idx.Key())).To(Equal(read(idx2.Key())))
		})
	})

	Describe("Mixed mode", func() {
		It("Should accept user-provided index data and only auto-stamp omitted index channels", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			keys := []channel.Key{s.idx.Key(), s.data.Key()}
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:         keys,
				Start:        100 * telem.SecondTS,
				Sync:         new(true),
				AutoIndexing: new(true),
			}))
			MustSucceed(w.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(100, 101, 102),
				telem.NewSeriesV[float64](1, 2, 3),
			})))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   []channel.Key{s.idx.Key()},
				Bounds: (100 * telem.SecondTS).Range(103 * telem.SecondTS),
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
			ts := telem.UnmarshalSeries[telem.TimeStamp](iter.Value().SeriesAt(0))
			Expect(iter.Close()).To(Succeed())
			Expect(ts).To(Equal([]telem.TimeStamp{
				100 * telem.SecondTS,
				101 * telem.SecondTS,
				102 * telem.SecondTS,
			}))
		})

		It("Should auto-stamp the index when the caller omits it from a Write frame", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			keys := []channel.Key{s.idx.Key(), s.data.Key()}
			before := telem.Now()
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:         keys,
				Sync:         new(true),
				AutoIndexing: new(true),
			}))
			MustSucceed(w.Write(frame.NewUnary(
				s.data.Key(),
				telem.NewSeriesV[float64](1, 2),
			)))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   []channel.Key{s.idx.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
			ts := telem.UnmarshalSeries[telem.TimeStamp](iter.Value().SeriesAt(0))
			Expect(iter.Close()).To(Succeed())
			Expect(ts).To(HaveLen(2))
			Expect(ts[0]).To(BeNumerically(">=", before))
			Expect(ts[1]).To(Equal(ts[0] + 1))
		})
	})

	Describe("Multi-node leaseholder stamping", func() {
		It("Should auto-stamp from the leaseholder when the data channel lives on a peer", func(ctx SpecContext) {
			builder := mock.ProvisionCluster(ctx, 2)
			defer func() { Expect(builder.Close()).To(Succeed()) }()
			gw := builder.Nodes[1]
			peer := node.Key(2)

			idx := channel.Channel{
				Name:        channel.NewRandomName(),
				IsIndex:     true,
				DataType:    telem.TimeStampT,
				Leaseholder: peer,
			}
			Expect(gw.Channel.Create(ctx, &idx)).To(Succeed())
			data := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.Float64T,
				LocalIndex:  idx.LocalKey,
				Leaseholder: peer,
			}
			Expect(gw.Channel.Create(ctx, &data)).To(Succeed())
			Eventually(func(g Gomega) {
				var chs []channel.Channel
				g.Expect(gw.Channel.NewRetrieve().
					Entries(&chs).
					Where(channel.MatchKeys(idx.Key(), data.Key())).
					Exec(ctx, nil)).To(Succeed())
				g.Expect(chs).To(HaveLen(2))
			}).Should(Succeed())

			before := telem.Now()
			w := MustSucceed(gw.Framer.OpenWriter(ctx, writer.Config{
				Keys:         []channel.Key{data.Key()},
				Sync:         new(true),
				AutoIndexing: new(true),
			}))
			MustSucceed(w.Write(frame.NewUnary(
				data.Key(),
				telem.NewSeriesV(1.1, 2.2, 3.3),
			)))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())
			after := telem.Now()

			iter := MustSucceed(gw.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   []channel.Key{idx.Key()},
				Bounds: telem.TimeRangeMax,
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
			ts := telem.UnmarshalSeries[telem.TimeStamp](iter.Value().SeriesAt(0))
			Expect(iter.Close()).To(Succeed())

			Expect(ts).To(HaveLen(3))
			Expect(ts[0]).To(BeNumerically(">=", before))
			Expect(ts[2]).To(BeNumerically("<=", after+telem.TimeStamp(3)))
			Expect(ts[1]).To(Equal(ts[0] + 1))
			Expect(ts[2]).To(Equal(ts[0] + 2))
		})
	})

	Describe("Disabled", func() {
		It("Should be a no-op when AutoIndexing is false (default)", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			keys := []channel.Key{s.idx.Key(), s.data.Key()}
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:  keys,
				Start: 200 * telem.SecondTS,
				Sync:  new(true),
			}))
			MustSucceed(w.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(200, 201),
				telem.NewSeriesV[float64](1, 2),
			})))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
				Keys:   []channel.Key{s.idx.Key()},
				Bounds: (200 * telem.SecondTS).Range(202 * telem.SecondTS),
			}))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
			ts := telem.UnmarshalSeries[telem.TimeStamp](iter.Value().SeriesAt(0))
			Expect(iter.Close()).To(Succeed())
			Expect(ts).To(Equal([]telem.TimeStamp{
				200 * telem.SecondTS,
				201 * telem.SecondTS,
			}))
		})
	})

	Describe("Conflict between user-provided and auto-generated index timestamps", func() {
		It("Should surface a storage conflict when a user-provided index timestamp is ahead of a subsequent auto-stamp", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			// A user-provided index timestamp far enough in the future that telem.Now()
			// at the next write is guaranteed to be less. The autoIndexer intentionally
			// does not absorb the future user-provided value — mixing future user-TS
			// with auto-stamping is not a supported pattern, so cesium rejects the
			// conflicting write rather than silently coercing.
			future := telem.Now() + telem.HourTS
			keys := []channel.Key{s.idx.Key(), s.data.Key()}
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:         keys,
				Sync:         new(true),
				AutoIndexing: new(true),
			}))
			MustSucceed(w.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesV(future),
				telem.NewSeriesV[float64](1),
			})))
			Expect(w.Write(frame.NewUnary(
				s.data.Key(),
				telem.NewSeriesV[float64](2),
			))).Error().To(MatchError(
				ContainSubstring("must not be less than the previous commit timestamp"),
			))
		})
	})

	Describe("Watermark and mixed-frame handling", func() {
		It("Should leave one index untouched while auto-stamping another in the same frame", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			idx2 := channel.Channel{
				Name:     channel.NewRandomName(),
				IsIndex:  true,
				DataType: telem.TimeStampT,
			}
			Expect(s.dist.Channel.Create(ctx, &idx2)).To(Succeed())
			data2 := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float64T,
				LocalIndex: idx2.LocalKey,
			}
			Expect(s.dist.Channel.Create(ctx, &data2)).To(Succeed())

			keys := []channel.Key{s.idx.Key(), s.data.Key(), data2.Key()}
			before := telem.Now()
			w := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:         keys,
				Start:        100 * telem.SecondTS,
				Sync:         new(true),
				AutoIndexing: new(true),
			}))
			MustSucceed(w.Write(frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(100, 101),
				telem.NewSeriesV[float64](1, 2),
				telem.NewSeriesV[float64](3, 4),
			})))
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())

			read := func(k channel.Key, bounds telem.TimeRange) []telem.TimeStamp {
				iter := MustSucceed(s.dist.Framer.OpenIterator(ctx, iterator.Config{
					Keys:   []channel.Key{k},
					Bounds: bounds,
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(telem.TimeSpanMax)).To(BeTrue())
				ts := telem.UnmarshalSeries[telem.TimeStamp](iter.Value().SeriesAt(0))
				Expect(iter.Close()).To(Succeed())
				return ts
			}
			Expect(read(
				s.idx.Key(),
				(100 * telem.SecondTS).Range(102*telem.SecondTS),
			)).To(Equal([]telem.TimeStamp{
				100 * telem.SecondTS,
				101 * telem.SecondTS,
			}))
			autoTS := read(idx2.Key(), telem.TimeRangeMax)
			Expect(autoTS).To(HaveLen(2))
			Expect(autoTS[0]).To(BeNumerically(">=", before))
			Expect(autoTS[1]).To(Equal(autoTS[0] + 1))
		})
	})

	Describe("SetAuthority propagation to implicit index channels", func() {
		It("Raises the implicit index when a referencing data channel is raised", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			wA := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:         []channel.Key{s.data.Key()},
				Authorities:  []control.Authority{control.Authority(50)},
				AutoIndexing: new(true),
				Sync:         new(true),
			}))
			defer func() { Expect(wA.Close()).To(Succeed()) }()

			wB := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:        []channel.Key{s.idx.Key()},
				Authorities: []control.Authority{control.Authority(100)},
				Sync:        new(true),
			}))
			defer func() { Expect(wB.Close()).To(Succeed()) }()
			Expect(MustSucceed(wB.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(10),
			)))).To(BeTrue())

			Expect(wA.SetAuthority(writer.Config{
				Keys:        []channel.Key{s.data.Key()},
				Authorities: []control.Authority{control.Authority(200)},
			})).To(Succeed())

			Expect(MustSucceed(wB.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(11),
			)))).To(BeFalse())
		})

		It("Tracks the max across data channels sharing an implicit index", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			other := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float64T,
				LocalIndex: s.idx.LocalKey,
			}
			Expect(s.dist.Channel.Create(ctx, &other)).To(Succeed())

			wA := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys: []channel.Key{s.data.Key(), other.Key()},
				Authorities: []control.Authority{
					control.Authority(50),
					control.Authority(50),
				},
				AutoIndexing: new(true),
				Sync:         new(true),
			}))
			defer func() { Expect(wA.Close()).To(Succeed()) }()

			wB := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:        []channel.Key{s.idx.Key()},
				Authorities: []control.Authority{control.Authority(150)},
				Sync:        new(true),
			}))
			defer func() { Expect(wB.Close()).To(Succeed()) }()
			Expect(MustSucceed(wB.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(10),
			)))).To(BeTrue())

			Expect(wA.SetAuthority(writer.Config{
				Keys:        []channel.Key{s.data.Key()},
				Authorities: []control.Authority{control.Authority(100)},
			})).To(Succeed())
			Expect(MustSucceed(wB.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(11),
			)))).To(BeTrue())

			Expect(wA.SetAuthority(writer.Config{
				Keys:        []channel.Key{other.Key()},
				Authorities: []control.Authority{control.Authority(200)},
			})).To(Succeed())
			Expect(MustSucceed(wB.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(12),
			)))).To(BeFalse())
		})

		It("Lowers the implicit index when both referencing data channels are lowered", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			other := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float64T,
				LocalIndex: s.idx.LocalKey,
			}
			Expect(s.dist.Channel.Create(ctx, &other)).To(Succeed())

			wA := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys: []channel.Key{s.data.Key(), other.Key()},
				Authorities: []control.Authority{
					control.Authority(200),
					control.Authority(200),
				},
				AutoIndexing: new(true),
				Sync:         new(true),
			}))
			defer func() { Expect(wA.Close()).To(Succeed()) }()

			wB := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:        []channel.Key{s.idx.Key()},
				Authorities: []control.Authority{control.Authority(150)},
				Sync:        new(true),
			}))
			defer func() { Expect(wB.Close()).To(Succeed()) }()
			Expect(MustSucceed(wB.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(10),
			)))).To(BeFalse())

			Expect(wA.SetAuthority(writer.Config{
				Keys: []channel.Key{s.data.Key(), other.Key()},
				Authorities: []control.Authority{
					control.Authority(50),
					control.Authority(50),
				},
			})).To(Succeed())

			Expect(MustSucceed(wB.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(11),
			)))).To(BeTrue())
		})

		It("Respects an explicit authority supplied for the implicit index", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			wA := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:         []channel.Key{s.data.Key()},
				Authorities:  []control.Authority{control.Authority(50)},
				AutoIndexing: new(true),
				Sync:         new(true),
			}))
			defer func() { Expect(wA.Close()).To(Succeed()) }()

			wB := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:        []channel.Key{s.idx.Key()},
				Authorities: []control.Authority{control.Authority(100)},
				Sync:        new(true),
			}))
			defer func() { Expect(wB.Close()).To(Succeed()) }()
			Expect(MustSucceed(wB.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(10),
			)))).To(BeTrue())

			Expect(wA.SetAuthority(writer.Config{
				Keys: []channel.Key{s.data.Key(), s.idx.Key()},
				Authorities: []control.Authority{
					control.Authority(250),
					control.Authority(30),
				},
			})).To(Succeed())

			Expect(MustSucceed(wB.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(11),
			)))).To(BeTrue())
		})

		It("Propagates from a broadcast SetAuthority through subsequent per-channel calls", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			other := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float64T,
				LocalIndex: s.idx.LocalKey,
			}
			Expect(s.dist.Channel.Create(ctx, &other)).To(Succeed())

			wA := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:         []channel.Key{s.data.Key(), other.Key()},
				Authorities:  []control.Authority{control.Authority(50)},
				AutoIndexing: new(true),
				Sync:         new(true),
			}))
			defer func() { Expect(wA.Close()).To(Succeed()) }()

			wB := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:        []channel.Key{s.idx.Key()},
				Authorities: []control.Authority{control.Authority(150)},
				Sync:        new(true),
			}))
			defer func() { Expect(wB.Close()).To(Succeed()) }()
			Expect(MustSucceed(wB.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(10),
			)))).To(BeTrue())

			Expect(wA.SetAuthority(writer.Config{
				Authorities: []control.Authority{control.Authority(100)},
			})).To(Succeed())

			Expect(wA.SetAuthority(writer.Config{
				Keys:        []channel.Key{s.data.Key()},
				Authorities: []control.Authority{control.Authority(200)},
			})).To(Succeed())

			Expect(MustSucceed(wB.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(11),
			)))).To(BeFalse())
		})
	})

	Describe("Authority resolution for implicit index channels", func() {
		// These tests verify the authority assigned to an implicit (auto-added) index
		// channel by opening a second writer on the same index with a known authority
		// and observing which writer wins control. The reported `authorized` flag is
		// the public signal of who holds the index.
		It("Inherits a broadcast authority", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			wA := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:         []channel.Key{s.data.Key()},
				Authorities:  []control.Authority{control.Authority(150)},
				AutoIndexing: new(true),
				Sync:         new(true),
			}))
			defer func() { Expect(wA.Close()).To(Succeed()) }()

			wLower := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:        []channel.Key{s.idx.Key()},
				Authorities: []control.Authority{control.Authority(100)},
				Sync:        new(true),
			}))
			defer func() { Expect(wLower.Close()).To(Succeed()) }()
			authorized := MustSucceed(wLower.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(10),
			)))
			Expect(authorized).To(BeFalse())

			wHigher := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:        []channel.Key{s.idx.Key()},
				Authorities: []control.Authority{control.Authority(200)},
				Sync:        new(true),
			}))
			defer func() { Expect(wHigher.Close()).To(Succeed()) }()
			authorized = MustSucceed(wHigher.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(11),
			)))
			Expect(authorized).To(BeTrue())
		})

		It("Takes the max authority across data channels sharing an index", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			other := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float64T,
				LocalIndex: s.idx.LocalKey,
			}
			Expect(s.dist.Channel.Create(ctx, &other)).To(Succeed())

			wA := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys: []channel.Key{s.data.Key(), other.Key()},
				Authorities: []control.Authority{
					control.Authority(50),
					control.Authority(200),
				},
				AutoIndexing: new(true),
				Sync:         new(true),
			}))
			defer func() { Expect(wA.Close()).To(Succeed()) }()

			// If the implicit index had inherited min(50, 200) = 50, this writer
			// at authority 100 would win the index. With max it loses.
			wBetween := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:        []channel.Key{s.idx.Key()},
				Authorities: []control.Authority{control.Authority(100)},
				Sync:        new(true),
			}))
			defer func() { Expect(wBetween.Close()).To(Succeed()) }()
			authorized := MustSucceed(wBetween.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(10),
			)))
			Expect(authorized).To(BeFalse())

			// If the implicit index had defaulted to AuthorityAbsolute (255), this
			// writer at authority 220 would lose. With max=200 it wins.
			wAbove := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:        []channel.Key{s.idx.Key()},
				Authorities: []control.Authority{control.Authority(220)},
				Sync:        new(true),
			}))
			defer func() { Expect(wAbove.Close()).To(Succeed()) }()
			authorized = MustSucceed(wAbove.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(11),
			)))
			Expect(authorized).To(BeTrue())
		})

		It("Resolves authority per-index when distinct indexes are implicit", func(ctx SpecContext) {
			s := openAutoIndexScenario(ctx)
			defer func() { Expect(s.closer.Close()).To(Succeed()) }()

			idx2 := channel.Channel{
				Name:     channel.NewRandomName(),
				IsIndex:  true,
				DataType: telem.TimeStampT,
			}
			Expect(s.dist.Channel.Create(ctx, &idx2)).To(Succeed())
			data2 := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float64T,
				LocalIndex: idx2.LocalKey,
			}
			Expect(s.dist.Channel.Create(ctx, &data2)).To(Succeed())

			wA := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys: []channel.Key{s.data.Key(), data2.Key()},
				Authorities: []control.Authority{
					control.Authority(80),
					control.Authority(180),
				},
				AutoIndexing: new(true),
				Sync:         new(true),
			}))
			defer func() { Expect(wA.Close()).To(Succeed()) }()

			// idx1 should have inherited 80; a writer at 100 outranks it.
			wIdx1 := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:        []channel.Key{s.idx.Key()},
				Authorities: []control.Authority{control.Authority(100)},
				Sync:        new(true),
			}))
			defer func() { Expect(wIdx1.Close()).To(Succeed()) }()
			authorized := MustSucceed(wIdx1.Write(frame.NewUnary(
				s.idx.Key(),
				telem.NewSeriesSecondsTSV(10),
			)))
			Expect(authorized).To(BeTrue())

			// idx2 should have inherited 180; a writer at 100 is outranked.
			wIdx2 := MustSucceed(s.dist.Framer.OpenWriter(ctx, writer.Config{
				Keys:        []channel.Key{idx2.Key()},
				Authorities: []control.Authority{control.Authority(100)},
				Sync:        new(true),
			}))
			defer func() { Expect(wIdx2.Close()).To(Succeed()) }()
			authorized = MustSucceed(wIdx2.Write(frame.NewUnary(
				idx2.Key(),
				telem.NewSeriesSecondsTSV(11),
			)))
			Expect(authorized).To(BeFalse())
		})
	})
})
