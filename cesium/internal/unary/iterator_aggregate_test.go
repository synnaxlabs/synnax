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
	"math"
	"runtime"
	"strconv"

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

var _ = Describe("Aggregated Iteration", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, func() {
			var (
				indexDB  *unary.DB
				dataDB   *unary.DB
				stringDB *unary.DB
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
				stringDB = MustOpen(unary.Open(ctx, unary.Config{
					FS:        MustSucceed(fs.Sub("string")),
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "string",
						DataType: telem.StringT,
						Index:    indexKey,
					},
				}))
				stringDB.SetIndex(indexDB.Index())
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
				cfg.Reduction.Variant = telem.LimitReduction{
					Aggregation: agg,
					PointLimit:  limit,
				}
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
				return iter.Value().SeriesSlice()
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
					chunks = append(chunks, iter.Value().Len())
					series = append(series, iter.Value().SeriesSlice()...)
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
					"a limit without a point limit",
					unary.IteratorConfig{Reduction: telem.Reduction{
						Variant: telem.LimitReduction{
							Aggregation: telem.AggregationMinMax,
						},
					}},
					"point_limit: must be greater than or equal to 1",
				),
				Entry(
					"an unknown aggregation",
					unary.IteratorConfig{Reduction: telem.Reduction{
						Variant: telem.LimitReduction{
							Aggregation: "median",
							PointLimit:  10,
						},
					}},
					"aggregation: invalid aggregation: median",
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

			DescribeTable("Should decimate to the first present sample of each group",
				func(
					ctx SpecContext,
					starts []telem.TimeStamp,
					count int,
					bounds telem.TimeRange,
					multiple uint32,
					groupStarts []uint32,
					expected [][]int64,
				) {
					for _, start := range starts {
						write(ctx, start, count)
					}
					series := readSpan(ctx, dataDB, aggregated(
						bounds,
						telem.AggregationDecimate,
						10,
					))
					Expect(series).To(HaveLen(len(expected)))
					for i, s := range series {
						Expect(s.Unmarshal[int64]()).To(Equal(expected[i]))
						Expect(s.Alignment.SampleIndex()).To(Equal(groupStarts[i]))
						Expect(s.AlignmentMultiple).To(Equal(multiple))
					}
				},
				Entry(
					"a partial first group",
					[]telem.TimeStamp{telem.SecondTS},
					100,
					(6*telem.SecondTS).Range(101*telem.SecondTS),
					uint32(10),
					[]uint32{0},
					[][]int64{{6, 11, 21, 31, 41, 51, 61, 71, 81, 91}},
				),
				Entry(
					"a partial last group",
					[]telem.TimeStamp{telem.SecondTS},
					100,
					telem.SecondTS.Range(96*telem.SecondTS),
					uint32(10),
					[]uint32{0},
					[][]int64{{1, 11, 21, 31, 41, 51, 61, 71, 81, 91}},
				),
				Entry(
					"two index domains",
					[]telem.TimeStamp{telem.SecondTS, 100 * telem.SecondTS},
					25,
					(7*telem.SecondTS).Range(123*telem.SecondTS),
					uint32(5),
					[]uint32{5, 0},
					[][]int64{{7, 11, 16, 21}, {1, 6, 11, 16, 21}},
				),
			)

			It("Should decimate reads larger than one read buffer", func(
				ctx SpecContext,
			) {
				const count = 300_000
				write(ctx, telem.SecondTS, count)
				bounds := (6 * telem.SecondTS).Range((count + 1) * telem.SecondTS)
				series := readSpan(
					ctx,
					dataDB,
					aggregated(bounds, telem.AggregationDecimate, 1000),
				)
				Expect(series).To(HaveLen(1))
				values := series[0].Unmarshal[int64]()
				Expect(values).To(HaveLen(1000))
				Expect(values[:3]).To(Equal([]int64{6, 301, 601}))
				Expect(values[len(values)-1]).To(Equal(int64(299_701)))
			})

			It("Should resize groups when the bounds change", func(ctx SpecContext) {
				write(ctx, telem.SecondTS, 100)
				cfg := aggregated(
					telem.SecondTS.Range(101*telem.SecondTS),
					telem.AggregationMinMax,
					10,
				)
				iter := MustOpen(dataDB.OpenIterator(cfg))
				Expect(iter.SeekFirst(ctx)).To(BeTrue())
				Expect(iter.Next(ctx, cfg.Bounds.Span())).To(BeTrue())
				Expect(iter.Value().SeriesAt(0).AlignmentMultiple).To(Equal(uint32(10)))
				half := telem.SecondTS.Range(51 * telem.SecondTS)
				iter.SetBounds(half)
				Expect(iter.SeekFirst(ctx)).To(BeTrue())
				Expect(iter.Next(ctx, half.Span())).To(BeTrue())
				Expect(iter.Value().SeriesAt(0).Unmarshal[int64]()).
					To(Equal([]int64{1, 10, 11, 20, 21, 30, 31, 40, 41, 50}))
				Expect(iter.Value().SeriesAt(0).AlignmentMultiple).To(Equal(uint32(5)))
			})

			It("Should leave a non-numeric channel raw", func(ctx SpecContext) {
				write(ctx, telem.SecondTS, 100)
				values := make([]string, 100)
				for i := range values {
					values[i] = strconv.Itoa(i + 1)
				}
				Expect(unary.Write(
					ctx,
					stringDB,
					telem.SecondTS,
					telem.NewSeriesV(values...),
				)).To(Succeed())
				series := readSpan(ctx, stringDB, aggregated(
					telem.SecondTS.Range(101*telem.SecondTS),
					telem.AggregationMinMax,
					10,
				))
				Expect(series).To(HaveLen(1))
				Expect(series[0].Unmarshal[string]()).To(Equal(values))
				Expect(series[0].AlignmentMultiple).To(BeZero())
			})

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

			Describe("A data domain that spans two index domains", func() {
				BeforeEach(func(ctx SpecContext) {
					stamps := make([]telem.TimeStamp, 20)
					values := make([]int64, 20)
					for i := range stamps {
						stamps[i] = telem.TimeStamp(i+1) * telem.SecondTS
						values[i] = int64(i) + 1
					}
					Expect(unary.Write(
						ctx,
						indexDB,
						stamps[0],
						telem.NewSeries(stamps[:10]),
					)).To(Succeed())
					Expect(unary.Write(
						ctx,
						indexDB,
						stamps[9]+1,
						telem.NewSeries(stamps[10:]),
					)).To(Succeed())
					Expect(unary.Write(
						ctx,
						dataDB,
						stamps[0],
						telem.NewSeries(values),
					)).To(Succeed())
				})

				config := func() unary.IteratorConfig {
					cfg := aggregated(
						telem.SecondTS.Range(21*telem.SecondTS),
						telem.AggregationMinMax,
						8,
					)
					cfg.AutoChunkSize = 2
					return cfg
				}

				It("Should end forward auto spans on the groups of each domain", func(
					ctx SpecContext,
				) {
					series, chunks := readAuto(ctx, dataDB, config())
					Expect(chunks).To(Equal([]int64{2, 2, 2, 2}))
					var values []int64
					for _, s := range series {
						values = append(values, s.Unmarshal[int64]()...)
						Expect(s.Alignment.SampleIndex() % 6).To(BeZero())
					}
					Expect(values).To(Equal([]int64{1, 6, 7, 10, 11, 16, 17, 20}))
					Expect(series[0].Alignment.DomainIndex()).
						ToNot(Equal(series[2].Alignment.DomainIndex()))
				})

				It("Should fill backward auto spans across both domains", func(
					ctx SpecContext,
				) {
					cfg := config()
					cfg.AutoChunkSize = 6
					iter := MustOpen(dataDB.OpenIterator(cfg))
					Expect(iter.SeekLast(ctx)).To(BeTrue())
					var chunks [][]int64
					for iter.Prev(ctx, unary.AutoSpan) {
						var values []int64
						for _, s := range iter.Value().SeriesSlice() {
							values = append(values, s.Unmarshal[int64]()...)
						}
						chunks = append(chunks, values)
					}
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(chunks).To(Equal([][]int64{
						{7, 10, 11, 16, 17, 20},
						{1, 6},
					}))
				})
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

			Describe("A group cut by a data domain edge", func() {
				const count = 128
				var wholeDB, cutDB *unary.DB
				BeforeEach(func(ctx SpecContext) {
					fs := openFS()
					open := func(name string) *unary.DB {
						db := MustOpen(unary.Open(ctx, unary.Config{
							FS:        MustSucceed(fs.Sub(name)),
							MetaCodec: json.Codec,
							Channel: channel.Channel{
								Key:      GenerateChannelKey(),
								Name:     name,
								DataType: telem.Float64T,
								Index:    indexDB.Channel().Key,
							},
						}))
						db.SetIndex(indexDB.Index())
						return db
					}
					wholeDB, cutDB = open("whole"), open("cut")
					stamps := make([]telem.TimeStamp, count)
					values := make([]float64, count)
					for i := range count {
						stamps[i] = telem.TimeStamp(i+1) * telem.SecondTS
						values[i] = float64((i*37)%61 - 30)
					}
					values[40] = math.NaN()
					Expect(
						unary.Write(ctx, indexDB, stamps[0], telem.NewSeries(stamps)),
					).To(Succeed())
					Expect(
						unary.Write(ctx, wholeDB, stamps[0], telem.NewSeries(values)),
					).To(Succeed())
					// Groups hold 16 samples, so the edges at 40 and 42 cut the group
					// at 32 three ways, and the edge at 64 falls between groups.
					edges := []int{0, 40, 42, 64, count}
					for k := 1; k < len(edges); k++ {
						lo, hi := edges[k-1], edges[k]
						part := telem.NewSeries(values[lo:hi])
						Expect(unary.Write(ctx, cutDB, stamps[lo], part)).To(Succeed())
					}
				})

				// read reads every series of the bounds in time order.
				read := func(
					ctx SpecContext,
					db *unary.DB,
					cfg unary.IteratorConfig,
					backward bool,
					span telem.TimeSpan,
				) []telem.Series {
					GinkgoHelper()
					iter := MustOpen(db.OpenIterator(cfg))
					var series []telem.Series
					if backward {
						Expect(iter.SeekLast(ctx)).To(BeTrue())
						for iter.Prev(ctx, span) {
							series = append(iter.Value().SeriesSlice(), series...)
						}
					} else {
						Expect(iter.SeekFirst(ctx)).To(BeTrue())
						for iter.Next(ctx, span) {
							series = append(series, iter.Value().SeriesSlice()...)
						}
					}
					Expect(iter.Error()).ToNot(HaveOccurred())
					return series
				}

				const (
					minMax   = telem.AggregationMinMax
					average  = telem.AggregationAverage
					decimate = telem.AggregationDecimate
					fixed    = telem.TimeSpanMax
					auto     = unary.AutoSpan
				)
				DescribeTable(
					"Should reduce the cut group as one group",
					func(
						ctx SpecContext,
						agg telem.Aggregation,
						backward bool,
						span telem.TimeSpan,
						chunk int,
					) {
						// Every aggregation gets groups of 16 samples.
						limit := 8 * agg.PointsPerGroup()
						cfg := aggregated(
							telem.SecondTS.Range((count+1)*telem.SecondTS),
							agg,
							uint32(limit),
						)
						cfg.AutoChunkSize = int64(chunk)
						whole := read(ctx, wholeDB, cfg, backward, span)
						cut := read(ctx, cutDB, cfg, backward, span)
						Expect(telem.NewMultiSeries(cut).Data()).
							To(Equal(telem.NewMultiSeries(whole).Data()))
						idx := read(ctx, indexDB, cfg, backward, span)
						Expect(telem.NewMultiSeries(cut).Len()).
							To(Equal(telem.NewMultiSeries(idx).Len()))
						for k := 1; k < len(cut); k++ {
							Expect(cut[k].Alignment).
								To(Equal(cut[k-1].AlignmentBounds().Upper))
							Expect(cut[k].TimeRange.Start).
								To(BeNumerically(">=", cut[k-1].TimeRange.End))
						}
					},
					func(
						agg telem.Aggregation,
						backward bool,
						span telem.TimeSpan,
						chunk int,
					) string {
						direction := "forward"
						if backward {
							direction = "backward"
						}
						mode := "a fixed span"
						if span == auto {
							mode = "auto spans of " + strconv.Itoa(chunk) + " points"
						}
						return string(agg) + ", " + direction + ", " + mode
					},
					Entry(nil, minMax, false, fixed, 100000),
					Entry(nil, minMax, true, fixed, 100000),
					Entry(nil, minMax, false, auto, 100000),
					Entry(nil, minMax, false, auto, 3),
					Entry(nil, minMax, true, auto, 3),
					Entry(nil, average, false, fixed, 100000),
					Entry(nil, average, true, fixed, 100000),
					Entry(nil, average, false, auto, 1),
					Entry(nil, average, true, auto, 1),
					Entry(nil, decimate, false, fixed, 100000),
					Entry(nil, decimate, true, fixed, 100000),
					Entry(nil, decimate, false, auto, 1),
					Entry(nil, decimate, true, auto, 1),
				)
			})

			DescribeTable(
				"Should reduce a group larger than the read buffer within it",
				func(
					ctx SpecContext,
					agg telem.Aggregation,
					limit uint32,
					cut int,
					expected []int64,
				) {
					const count = 400_000
					stamps := make([]telem.TimeStamp, count)
					values := make([]int64, count)
					for i := range count {
						stamps[i] = telem.TimeStamp(i+1) * telem.SecondTS
						values[i] = int64(i) + 1
					}
					Expect(
						unary.Write(ctx, indexDB, stamps[0], telem.NewSeries(stamps)),
					).To(Succeed())
					edges := []int{0, count}
					if cut > 0 {
						edges = []int{0, cut, count}
					}
					for k := 1; k < len(edges); k++ {
						lo, hi := edges[k-1], edges[k]
						part := telem.NewSeries(values[lo:hi])
						Expect(unary.Write(ctx, dataDB, stamps[lo], part)).To(Succeed())
					}
					bounds := telem.SecondTS.Range((count + 1) * telem.SecondTS)
					var before, after runtime.MemStats
					runtime.ReadMemStats(&before)
					series := readSpan(ctx, dataDB, aggregated(bounds, agg, limit))
					runtime.ReadMemStats(&after)
					Expect(telem.NewMultiSeries(series).Data()).
						To(Equal(telem.NewSeries(expected).Data))
					// The group holds 3.2 MB of samples, and each segment read holds
					// one
					// buffer of 1 MB.
					segments := uint64(len(edges) - 1)
					Expect(after.TotalAlloc - before.TotalAlloc).
						To(BeNumerically("<", segments*uint64(2*telem.Megabyte)))
				},
				Entry(
					"min_max",
					telem.AggregationMinMax,
					uint32(2),
					0,
					[]int64{1, 400_000},
				),
				Entry(
					"average",
					telem.AggregationAverage,
					uint32(1),
					0,
					[]int64{200_000},
				),
				Entry(
					"min_max cut by a write",
					telem.AggregationMinMax,
					uint32(2),
					200_000,
					[]int64{1, 400_000},
				),
				Entry(
					"average cut by a write",
					telem.AggregationAverage,
					uint32(1),
					200_000,
					[]int64{200_000},
				),
			)
		})
	}
})
