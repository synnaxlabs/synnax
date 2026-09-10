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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/encoding/json"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Downsampled Iteration", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, func() {
			var (
				fs       xfs.FS
				indexDB  *unary.DB
				dataDB   *unary.DB
				stringDB *unary.DB
			)
			BeforeEach(func(ctx SpecContext) {
				fs = openFS()
				indexKey := GenerateChannelKey()
				indexDB = MustSucceed(unary.Open(ctx, unary.Config{
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
				dataDB = MustSucceed(unary.Open(ctx, unary.Config{
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
				stringDB = MustSucceed(unary.Open(ctx, unary.Config{
					FS:        MustSucceed(fs.Sub("strings")),
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "strings",
						DataType: telem.StringT,
						Index:    indexKey,
					},
				}))
				stringDB.SetIndex(indexDB.Index())
			})
			AfterEach(func() {
				Expect(stringDB.Close()).To(Succeed())
				Expect(dataDB.Close()).To(Succeed())
				Expect(indexDB.Close()).To(Succeed())
			})

			// writeInt64 writes count samples starting at start seconds, one sample per
			// second, where the value of each sample is its one-based index.
			writeInt64 := func(ctx SpecContext, start telem.TimeStamp, count int) {
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

			readAll := func(
				ctx SpecContext,
				db *unary.DB,
				factor int,
			) []telem.Series {
				GinkgoHelper()
				cfg := unary.IterRange(telem.TimeRangeMax)
				cfg.DownsampleFactor = factor
				iter := MustSucceed(db.OpenIterator(cfg))
				defer func() { Expect(iter.Close()).To(Succeed()) }()
				Expect(iter.SeekFirst(ctx)).To(BeTrue())
				Expect(iter.Next(ctx, telem.TimeSpan(1e6)*telem.Second)).To(BeTrue())
				frame := iter.Value()
				series := make([]telem.Series, frame.Count())
				for i := range series {
					series[i] = frame.SeriesAt(i)
				}
				return series
			}

			DescribeTable("Fixed-density channels",
				func(ctx SpecContext, count, factor int, expected []int64) {
					writeInt64(ctx, telem.SecondTS, count)
					series := readAll(ctx, dataDB, factor)
					Expect(series).To(HaveLen(1))
					Expect(series[0].Unmarshal[int64]()).To(Equal(expected))
				},
				Entry(
					"Should keep every other sample",
					8,
					2,
					[]int64{1, 3, 5, 7},
				),
				Entry(
					"Should keep every third sample",
					9,
					3,
					[]int64{1, 4, 7},
				),
				Entry(
					"Should keep every sample when the factor is one",
					4,
					1,
					[]int64{1, 2, 3, 4},
				),
				Entry(
					"Should keep every sample when the factor is unset",
					4,
					0,
					[]int64{1, 2, 3, 4},
				),
				Entry(
					"Should keep the first sample when the factor exceeds the count",
					4,
					10,
					[]int64{1},
				),
				Entry(
					"Should keep the first sample when the factor is unbounded",
					4,
					math.MaxInt,
					[]int64{1},
				),
			)

			It("Should skip ahead when the stride exceeds the read buffer", func(
				ctx SpecContext,
			) {
				writeInt64(ctx, telem.SecondTS, 20000)
				series := readAll(ctx, dataDB, 9000)
				Expect(series).To(HaveLen(1))
				Expect(
					series[0].Unmarshal[int64](),
				).To(Equal([]int64{1, 9001, 18001}))
			})

			It("Should restart the stride at each domain", func(ctx SpecContext) {
				writeInt64(ctx, telem.SecondTS, 4)
				writeInt64(ctx, 100*telem.SecondTS, 4)
				series := readAll(ctx, dataDB, 3)
				Expect(series).To(HaveLen(2))
				Expect(series[0].Unmarshal[int64]()).To(Equal([]int64{1, 4}))
				Expect(series[1].Unmarshal[int64]()).To(Equal([]int64{1, 4}))
			})

			It("Should keep every other variable-length sample", func(
				ctx SpecContext,
			) {
				Expect(unary.Write(
					ctx,
					indexDB,
					telem.SecondTS,
					telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5),
				)).To(Succeed())
				Expect(unary.Write(
					ctx,
					stringDB,
					telem.SecondTS,
					telem.NewSeriesV("alpha", "be", "gamma", "d", "epsilon"),
				)).To(Succeed())
				series := readAll(ctx, stringDB, 2)
				Expect(series).To(HaveLen(1))
				Expect(
					series[0].Unmarshal[string](),
				).To(Equal([]string{"alpha", "gamma", "epsilon"}))
			})

			It("Should size auto spans by source samples", func(ctx SpecContext) {
				writeInt64(ctx, telem.SecondTS, 4)
				writeInt64(ctx, 100*telem.SecondTS, 4)
				cfg := unary.IterRange(telem.TimeRangeMax)
				cfg.DownsampleFactor = 2
				cfg.AutoChunkSize = 6
				iter := MustSucceed(dataDB.OpenIterator(cfg))
				defer func() { Expect(iter.Close()).To(Succeed()) }()
				Expect(iter.SeekFirst(ctx)).To(BeTrue())
				Expect(iter.Next(ctx, unary.AutoSpan)).To(BeTrue())
				Expect(iter.Value().Count()).To(Equal(2))
				Expect(
					iter.Value().SeriesAt(0).Unmarshal[int64](),
				).To(Equal([]int64{1, 3}))
				Expect(
					iter.Value().SeriesAt(1).Unmarshal[int64](),
				).To(Equal([]int64{1}))
			})
		})
	}
})
