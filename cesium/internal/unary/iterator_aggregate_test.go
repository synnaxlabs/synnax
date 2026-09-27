// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/encoding/json"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// collect returns the series of fr in order.
func collect(fr channel.Frame) []telem.Series {
	series := make([]telem.Series, 0, fr.Count())
	for _, s := range fr.SeriesI() {
		series = append(series, s)
	}
	return series
}

var _ = Describe("Aggregated Iteration", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, func() {
			var (
				indexDB *unary.DB
				dataDB  *unary.DB
			)
			BeforeEach(func(ctx SpecContext) {
				fs := openFS()
				indexKey := GenerateChannelKey()
				indexDB = MustOpen(unary.Open(ctx, unary.Config{
					FS:        MustSucceed(fs.Sub("index")),
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      indexKey,
						Name:     "index",
						DataType: telem.TimestampT,
						IsIndex:  true,
						Index:    indexKey,
					},
				}))
				dataDB = MustOpen(unary.Open(ctx, unary.Config{
					FS:        MustSucceed(fs.Sub("data")),
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "data",
						DataType: telem.Int64T,
						Index:    indexKey,
					},
				}))
				dataDB.SetIndex(indexDB.Index())
			})

			// write writes count samples to the index and data channels starting at
			// start seconds, one per second. The value of each data sample is its
			// one-based position in the write.
			write := func(ctx SpecContext, start telem.TimeStamp, count int) {
				GinkgoHelper()
				stamps := make([]telem.TimeStamp, count)
				values := make([]int64, count)
				for i := range count {
					stamps[i] = start + telem.TimeStamp(i)*telem.SecondTS
					values[i] = int64(i) + 1
				}
				Expect(
					unary.Write(ctx, indexDB, stamps[0], telem.NewSeries(stamps)),
				).To(Succeed())
				Expect(
					unary.Write(ctx, dataDB, stamps[0], telem.NewSeries(values)),
				).To(Succeed())
			}

			aggregated := func(
				bounds telem.TimeRange,
				agg telem.Aggregation,
				limit uint32,
			) unary.IteratorConfig {
				cfg := unary.IterRange(bounds)
				cfg.Aggregation = agg
				cfg.PointLimit = limit
				return cfg
			}

			// readSpan reads bounds in a single fixed-span Next.
			readSpan := func(
				ctx SpecContext,
				db *unary.DB,
				cfg unary.IteratorConfig,
			) []telem.Series {
				GinkgoHelper()
				iter := MustOpen(db.OpenIterator(cfg))
				Expect(iter.SeekFirst(ctx)).To(BeTrue())
				Expect(iter.Next(ctx, cfg.Bounds.Span())).To(BeTrue())
				return collect(iter.Value())
			}

			// readAuto reads bounds with AutoSpan, returning every series of every
			// chunk in order and the number of points each chunk held.
			readAuto := func(
				ctx SpecContext,
				db *unary.DB,
				cfg unary.IteratorConfig,
			) ([]telem.Series, []int64) {
				GinkgoHelper()
				iter := MustOpen(db.OpenIterator(cfg))
				Expect(iter.SeekFirst(ctx)).To(BeTrue())
				var (
					series []telem.Series
					chunks []int64
				)
				for iter.Next(ctx, unary.AutoSpan) {
					chunks = append(chunks, iter.Len())
					series = append(series, collect(iter.Value())...)
				}
				Expect(iter.Error()).ToNot(HaveOccurred())
				return series, chunks
			}

			DescribeTable("Config validation",
				func(cfg unary.IteratorConfig, message string) {
					Expect(dataDB.OpenIterator(cfg)).Error().
						To(MatchError(ContainSubstring(message)))
				},
				Entry(
					"aggregation without a point limit",
					unary.IteratorConfig{Aggregation: telem.AggregationMinMax},
					"point_limit: must be set together with aggregation",
				),
				Entry(
					"a point limit without aggregation",
					unary.IteratorConfig{PointLimit: 10},
					"point_limit: must be set together with aggregation",
				),
				Entry(
					"aggregation with a downsample factor",
					unary.IteratorConfig{
						Aggregation:      telem.AggregationMinMax,
						PointLimit:       10,
						DownsampleFactor: 2,
					},
					"downsample_factor: cannot be combined with aggregation",
				),
				Entry(
					"an unknown aggregation",
					unary.IteratorConfig{
						Aggregation: telem.AggregationDecimate + 1,
						PointLimit:  10,
					},
					"aggregation: unknown aggregation",
				),
			)

			It("Should keep the min and max of each group", func(ctx SpecContext) {
				write(ctx, telem.SecondTS, 100)
				bounds := telem.SecondTS.Range(101 * telem.SecondTS)
				series := readSpan(
					ctx,
					dataDB,
					aggregated(bounds, telem.AggregationMinMax, 10),
				)
				Expect(series).To(HaveLen(1))
				Expect(series[0].Unmarshal[int64]()).To(Equal([]int64{
					1, 20, 21, 40, 41, 60, 61, 80, 81, 100,
				}))
				Expect(series[0].AlignmentMultiple).To(Equal(uint32(10)))
				Expect(series[0].Alignment.SampleIndex()).To(BeZero())
			})

			It("Should reduce the index to the alignments of its data", func(
				ctx SpecContext,
			) {
				write(ctx, telem.SecondTS, 100)
				cfg := aggregated(
					telem.SecondTS.Range(101*telem.SecondTS),
					telem.AggregationMinMax,
					10,
				)
				idx := readSpan(ctx, indexDB, cfg)
				data := readSpan(ctx, dataDB, cfg)
				Expect(idx).To(HaveLen(1))
				Expect(idx[0].AlignmentBounds()).To(Equal(data[0].AlignmentBounds()))
				Expect(idx[0].Unmarshal[telem.TimeStamp]()).To(Equal([]telem.TimeStamp{
					telem.SecondTS * 1, telem.SecondTS * 20,
					telem.SecondTS * 21, telem.SecondTS * 40,
					telem.SecondTS * 41, telem.SecondTS * 60,
					telem.SecondTS * 61, telem.SecondTS * 80,
					telem.SecondTS * 81, telem.SecondTS * 100,
				}))
			})

			It("Should read every sample when the count fits under the limit", func(
				ctx SpecContext,
			) {
				write(ctx, telem.SecondTS, 8)
				series := readSpan(ctx, dataDB, aggregated(
					telem.SecondTS.Range(9*telem.SecondTS),
					telem.AggregationMinMax,
					8,
				))
				Expect(series[0].Unmarshal[int64]()).To(Equal([]int64{
					1, 2, 3, 4, 5, 6, 7, 8,
				}))
				Expect(series[0].AlignmentMultiple).To(BeZero())
			})

			DescribeTable("Should count only the samples inside inexact bounds",
				func(ctx SpecContext, bounds telem.TimeRange, expected []int64) {
					write(ctx, telem.SecondTS, 8)
					series := readSpan(ctx, dataDB, aggregated(
						bounds,
						telem.AggregationMinMax,
						uint32(len(expected)),
					))
					Expect(series[0].Unmarshal[int64]()).To(Equal(expected))
					Expect(series[0].AlignmentMultiple).To(BeZero())
				},
				Entry(
					"start before the first sample",
					telem.TimeStamp(0).Range(9*telem.SecondTS),
					[]int64{1, 2, 3, 4, 5, 6, 7, 8},
				),
				Entry(
					"start between samples",
					(1500*telem.MillisecondTS).Range(20*telem.SecondTS),
					[]int64{2, 3, 4, 5, 6, 7, 8},
				),
				Entry(
					"start and end between samples",
					(1500*telem.MillisecondTS).Range(7500*telem.MillisecondTS),
					[]int64{2, 3, 4, 5, 6, 7},
				),
			)

			DescribeTable("Aggregations",
				func(ctx SpecContext, agg telem.Aggregation, expected []int64) {
					write(ctx, telem.SecondTS, 12)
					series := readSpan(ctx, dataDB, aggregated(
						telem.SecondTS.Range(13*telem.SecondTS),
						agg,
						4,
					))
					Expect(series[0].Unmarshal[int64]()).To(Equal(expected))
				},
				Entry("average", telem.AggregationAverage, []int64{2, 5, 8, 11}),
				Entry("decimate", telem.AggregationDecimate, []int64{1, 4, 7, 10}),
				Entry("min/max", telem.AggregationMinMax, []int64{1, 6, 7, 12}),
			)

			It("Should anchor groups at the start of each index domain", func(
				ctx SpecContext,
			) {
				write(ctx, telem.SecondTS, 30)
				write(ctx, 100*telem.SecondTS, 30)
				series := readSpan(ctx, dataDB, aggregated(
					telem.SecondTS.Range(130*telem.SecondTS),
					telem.AggregationMinMax,
					6,
				))
				Expect(series).To(HaveLen(2))
				for _, s := range series {
					Expect(s.Unmarshal[int64]()).To(Equal([]int64{1, 20, 21, 30}))
					Expect(s.Alignment.SampleIndex()).To(BeZero())
				}
				Expect(series[0].Alignment.DomainIndex()).
					ToNot(Equal(series[1].Alignment.DomainIndex()))
			})

			It("Should label a group that starts before the read at its start", func(
				ctx SpecContext,
			) {
				write(ctx, telem.SecondTS, 100)
				series := readSpan(ctx, dataDB, aggregated(
					(6*telem.SecondTS).Range(101*telem.SecondTS),
					telem.AggregationMinMax,
					10,
				))
				Expect(series[0].Alignment.SampleIndex()).To(BeZero())
				Expect(series[0].Unmarshal[int64]()[:2]).To(Equal([]int64{6, 20}))
			})

			It("Should align a data domain that starts inside its index domain", func(
				ctx SpecContext,
			) {
				stamps := make([]telem.TimeStamp, 100)
				for i := range stamps {
					stamps[i] = telem.TimeStamp(i+1) * telem.SecondTS
				}
				Expect(
					unary.Write(ctx, indexDB, stamps[0], telem.NewSeries(stamps)),
				).To(Succeed())
				values := make([]int64, 50)
				for i := range values {
					values[i] = int64(i) + 51
				}
				Expect(
					unary.Write(ctx, dataDB, stamps[50], telem.NewSeries(values)),
				).To(Succeed())
				cfg := aggregated(
					telem.SecondTS.Range(101*telem.SecondTS),
					telem.AggregationMinMax,
					10,
				)
				data := readSpan(ctx, dataDB, cfg)
				Expect(data).To(HaveLen(1))
				Expect(data[0].Alignment.SampleIndex()).To(Equal(uint32(40)))
				Expect(data[0].Unmarshal[int64]()).To(Equal([]int64{
					51, 60, 61, 80, 81, 100,
				}))
				idx := readSpan(ctx, indexDB, cfg)
				Expect(idx[0].AlignmentBounds().Upper).
					To(Equal(data[0].AlignmentBounds().Upper))
			})

			It("Should count auto spans in points and end them on groups", func(
				ctx SpecContext,
			) {
				write(ctx, telem.SecondTS, 100)
				cfg := aggregated(
					telem.SecondTS.Range(101*telem.SecondTS),
					telem.AggregationMinMax,
					10,
				)
				cfg.AutoChunkSize = 4
				series, chunks := readAuto(ctx, dataDB, cfg)
				Expect(chunks).To(Equal([]int64{4, 4, 2}))
				var values []int64
				for _, s := range series {
					values = append(values, s.Unmarshal[int64]()...)
				}
				Expect(values).To(Equal([]int64{
					1, 20, 21, 40, 41, 60, 61, 80, 81, 100,
				}))
				for _, s := range series {
					Expect(s.Alignment.SampleIndex() % 20).To(BeZero())
				}
			})

			It("Should read backward in whole groups", func(ctx SpecContext) {
				write(ctx, telem.SecondTS, 100)
				cfg := aggregated(
					telem.SecondTS.Range(101*telem.SecondTS),
					telem.AggregationMinMax,
					10,
				)
				cfg.AutoChunkSize = 4
				iter := MustOpen(dataDB.OpenIterator(cfg))
				Expect(iter.SeekLast(ctx)).To(BeTrue())
				Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeTrue())
				Expect(iter.Value().SeriesAt(0).Unmarshal[int64]()).
					To(Equal([]int64{61, 80, 81, 100}))
				Expect(iter.Prev(ctx, unary.AutoSpan)).To(BeTrue())
				Expect(iter.Value().SeriesAt(0).Unmarshal[int64]()).
					To(Equal([]int64{21, 40, 41, 60}))
			})

			It(
				"Should reduce reads larger than one read buffer",
				func(ctx SpecContext) {
					const count = 300_000
					write(ctx, telem.SecondTS, count)
					bounds := telem.SecondTS.Range((count + 1) * telem.SecondTS)
					series := readSpan(
						ctx,
						dataDB,
						aggregated(bounds, telem.AggregationMinMax, 1000),
					)
					Expect(series).To(HaveLen(1))
					values := series[0].Unmarshal[int64]()
					Expect(values).To(HaveLen(1000))
					Expect(values[:4]).To(Equal([]int64{1, 600, 601, 1200}))
					Expect(values[len(values)-1]).To(Equal(int64(count)))
				},
			)

			It("Should reduce groups larger than one read buffer", func(
				ctx SpecContext,
			) {
				const count = 400_000
				write(ctx, telem.SecondTS, count)
				bounds := telem.SecondTS.Range((count + 1) * telem.SecondTS)
				series := readSpan(
					ctx,
					dataDB,
					aggregated(bounds, telem.AggregationMinMax, 2),
				)
				Expect(series[0].Unmarshal[int64]()).To(Equal([]int64{1, count}))
			})
		})
	}
})
