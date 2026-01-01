// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Accuracy", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db      *cesium.DB
				fs      xfs.FS
				cleanUp func() error
			)
			BeforeAll(func() {
				fs, cleanUp = makeFS()
				db = openDBOnFS(fs)
			})
			AfterAll(func() {
				Expect(db.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})
			Context("Single Channel", func() {
				var (
					key    cesium.ChannelKey = 2
					idxKey cesium.ChannelKey = 3
					first                    = []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
					second                   = []int64{13, 14, 15, 16, 17, 18, 19, 20, 21, 22}
					// Converted to seconds on write
					firstTS  = []telem.TimeStamp{2, 4, 6, 8, 10, 12, 13, 17, 18, 20}
					secondTS = []telem.TimeStamp{22, 24, 29, 32, 33, 34, 35, 36, 38, 40}
				)
				BeforeAll(func() {
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Name: "Rufus", Key: idxKey, IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Name: "Du", Key: key, Index: idxKey, DataType: telem.Int64T},
					)).To(Succeed())
					Expect(db.WriteSeries(ctx, idxKey, 2*telem.SecondTS, telem.NewSeriesSecondsTSV(firstTS...))).To(Succeed())
					Expect(db.WriteSeries(ctx, idxKey, 22*telem.SecondTS, telem.NewSeriesSecondsTSV(secondTS...))).To(Succeed())
					Expect(db.WriteSeries(ctx, key, 2*telem.SecondTS, telem.NewSeries(first))).To(Succeed())
					Expect(db.WriteSeries(ctx, key, 22*telem.SecondTS, telem.NewSeries(second))).To(Succeed())
				})
				DescribeTable("Accuracy",
					func(
						tr telem.TimeRange,
						expected []int64,
					) {
						var (
							frame  = MustSucceed(db.Read(ctx, tr, key))
							actual = make([]int64, 0, len(expected))
						)
						for series := range frame.Series() {
							actual = append(actual, telem.UnmarshalSeries[int64](series)...)
						}
						Expect(actual).To(Equal(expected))
					},
					Entry("Max range",
						telem.TimeRangeMax,
						append(first, second...),
					),
					Entry("Empty range - not on known timestamp",
						(9*telem.SecondTS).SpanRange(0),
						[]int64{},
					),
					Entry("Empty range - on known timestamp",
						(10*telem.SecondTS).SpanRange(0),
						[]int64{},
					),
					Entry("Single, even range",
						(2*telem.SecondTS).Range(21*telem.SecondTS),
						first,
					),
					Entry("Single, Exclusive End",
						(2*telem.SecondTS).Range(20*telem.SecondTS),
						first[:len(first)-1],
					),
					Entry("Single, partial range - start and end on known timestamps",
						(4*telem.SecondTS).SpanRange(4*telem.Second),
						[]int64{2, 3},
					),
					Entry("Single, partial range - start known end unknown",
						(4*telem.SecondTS).SpanRange(7*telem.Second),
						[]int64{2, 3, 4, 5},
					),
					Entry("Single, partial range - start unknown end unknown",
						(7*telem.SecondTS).Range(11*telem.SecondTS),
						[]int64{4, 5},
					),
					Entry("Multi, End at Second Start",
						(2*telem.SecondTS).Range(22*telem.SecondTS),
						first,
					),
					Entry("Multi, End slightly above Second Start",
						(2*telem.SecondTS).Range(22*telem.SecondTS+1),
						append(first, 13),
					),
				)
			})

			Context("Multi Channel", func() {
				Context("Different Indexes", Ordered, func() {
					var (
						idxKey1 cesium.ChannelKey = 6
						idxKey2 cesium.ChannelKey = 7
						key1    cesium.ChannelKey = 8
						key2    cesium.ChannelKey = 9
						data1                     = []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
						data2                     = []int64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
						// converted to seconds on write
						idxData1 = []telem.TimeStamp{1, 3, 5, 7, 9, 11, 18, 22, 31, 35}
						idxData2 = []telem.TimeStamp{1, 2, 6, 7, 12, 14, 17, 21, 27, 33}
					)
					BeforeAll(func() {
						Expect(db.CreateChannel(
							ctx,
							cesium.Channel{Name: "Albert", Key: idxKey1, DataType: telem.TimeStampT, IsIndex: true},
							cesium.Channel{Name: "Park", Key: idxKey2, DataType: telem.TimeStampT, IsIndex: true},
							cesium.Channel{Name: "Sheffield", Key: key1, Index: idxKey1, DataType: telem.Int64T},
							cesium.Channel{Name: "London", Key: key2, Index: idxKey2, DataType: telem.Int64T},
						)).To(Succeed())
						Expect(db.WriteSeries(ctx, idxKey1, 1*telem.SecondTS, telem.NewSeriesSecondsTSV(idxData1...))).To(Succeed())
						Expect(db.WriteSeries(ctx, idxKey2, 1*telem.SecondTS, telem.NewSeriesSecondsTSV(idxData2...))).To(Succeed())
						Expect(db.WriteSeries(ctx, key1, 1*telem.SecondTS, telem.NewSeries(data1))).To(Succeed())
						Expect(db.WriteSeries(ctx, key2, 1*telem.SecondTS, telem.NewSeries(data2))).To(Succeed())
					})
					DescribeTable("Accuracy",
						func(
							tr telem.TimeRange,
							expected1 []int64,
							expected2 []int64,
						) {
							var (
								frame   = MustSucceed(db.Read(ctx, tr, key1, key2))
								actual1 = make([]int64, 0, len(expected1))
								actual2 = make([]int64, 0, len(expected2))
							)
							for k, series := range frame.Entries() {
								if k == key1 {
									actual1 = append(actual1, telem.UnmarshalSeries[int64](series)...)
								} else {
									actual2 = append(actual2, telem.UnmarshalSeries[int64](series)...)
								}
							}
							Expect(actual1).To(Equal(expected1))
							Expect(actual2).To(Equal(expected2))
						},
						Entry("Max TimeRange",
							telem.TimeRangeMax,
							data1,
							data2,
						),
						Entry("Empty TimeRange",
							(10*telem.SecondTS).SpanRange(0),
							[]int64{},
							[]int64{},
						),
						Entry("Partial TimeRange",
							(3*telem.SecondTS).Range(11*telem.SecondTS),
							[]int64{2, 3, 4, 5},
							[]int64{13, 14},
						),
						Entry("Even on one Index",
							(3*telem.SecondTS).Range(33*telem.SecondTS),
							[]int64{2, 3, 4, 5, 6, 7, 8, 9},
							[]int64{13, 14, 15, 16, 17, 18, 19},
						),
					)
				})
			})
		})
	}
})
