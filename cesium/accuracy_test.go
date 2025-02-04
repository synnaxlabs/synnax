// Copyright 2025 Synnax Labs, Inc.
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
				Context("Rate Based", Ordered, func() {
					var key cesium.ChannelKey = 1
					first := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
					second := []int64{13, 14, 15, 16, 17, 18, 19, 20, 21, 22}
					BeforeAll(func() {
						Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, Rate: 1 * telem.Hz, DataType: telem.Int64T})).To(Succeed())
						Expect(db.WriteArray(ctx, key, 10*telem.SecondTS, telem.NewSeries(first))).To(Succeed())
						Expect(db.WriteArray(ctx, key, 20*telem.SecondTS, telem.NewSeries(second))).To(Succeed())
					})
					DescribeTable("Accuracy",
						func(
							tr telem.TimeRange,
							expected []int64,
						) {
							frame := MustSucceed(db.Read(ctx, tr, key))
							actual := make([]int64, 0, len(expected))
							for _, series := range frame.Series {
								actual = append(actual, telem.Unmarshal[int64](series)...)
							}
							Expect(actual).To(Equal(expected))
						},
						Entry("Max TimeRange",
							telem.TimeRangeMax,
							append(first, second...),
						),
						Entry("Empty TimeRange",
							(12*telem.SecondTS).SpanRange(0),
							[]int64{},
						),
						Entry("Single, Even TimeRange",
							(10*telem.SecondTS).Range(20*telem.SecondTS),
							first,
						),
						Entry("Single, Partial TimeRange",
							(12*telem.SecondTS).SpanRange(2*telem.Second),
							[]int64{3, 4},
						),
						Entry("Multiple, Even TimeRange",
							(10*telem.SecondTS).Range(30*telem.SecondTS),
							append(first, second...),
						),
						Entry("Multiple, Partial TimeRange",
							(15*telem.SecondTS).Range(25*telem.SecondTS),
							[]int64{6, 7, 8, 9, 10, 13, 14, 15, 16, 17},
						),
					)
				})

				Context("Indexed", Ordered, func() {
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
							cesium.Channel{Key: idxKey, IsIndex: true, DataType: telem.TimeStampT},
							cesium.Channel{Key: key, Index: idxKey, DataType: telem.Int64T},
						)).To(Succeed())
						Expect(db.WriteArray(ctx, idxKey, 2*telem.SecondTS, telem.NewSecondsTSV(firstTS...))).To(Succeed())
						Expect(db.WriteArray(ctx, idxKey, 22*telem.SecondTS, telem.NewSecondsTSV(secondTS...))).To(Succeed())
						Expect(db.WriteArray(ctx, key, 2*telem.SecondTS, telem.NewSeries(first))).To(Succeed())
						Expect(db.WriteArray(ctx, key, 22*telem.SecondTS, telem.NewSeries(second))).To(Succeed())
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
							for _, series := range frame.Series {
								actual = append(actual, telem.Unmarshal[int64](series)...)
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
			})

			Context("Multi Channel", func() {
				Context("Rate Based", func() {
					Context("Shared Rate", Ordered, func() {
						var (
							key1  cesium.ChannelKey = 4
							key2  cesium.ChannelKey = 5
							data1                   = []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
							data2                   = []int64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
						)
						BeforeAll(func() {
							Expect(db.CreateChannel(
								ctx,
								cesium.Channel{Key: key1, Rate: 1 * telem.Hz, DataType: telem.Int64T},
								cesium.Channel{Key: key2, Rate: 1 * telem.Hz, DataType: telem.Int64T},
							)).To(Succeed())
							Expect(db.WriteArray(ctx, key1, 2*telem.SecondTS, telem.NewSeries(data1))).To(Succeed())
							Expect(db.WriteArray(ctx, key2, 2*telem.SecondTS, telem.NewSeries(data2))).To(Succeed())
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
								for i, series := range frame.Series {
									if frame.Key(i) == key1 {
										actual1 = append(actual1, telem.Unmarshal[int64](series)...)
									} else {
										actual2 = append(actual2, telem.Unmarshal[int64](series)...)
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
								(12*telem.SecondTS).SpanRange(0),
								[]int64{},
								[]int64{},
							),
							Entry("Single, Even TimeRange",
								(2*telem.SecondTS).Range(12*telem.SecondTS),
								data1,
								data2,
							),
							Entry("Single, Exclusive End",
								(2*telem.SecondTS).Range(11*telem.SecondTS),
								data1[:len(data1)-1],
								data2[:len(data2)-1],
							),
							Entry("Single, Partial TimeRange",
								(4*telem.SecondTS).SpanRange(4*telem.Second),
								[]int64{3, 4, 5, 6},
								[]int64{13, 14, 15, 16},
							),
						)
					})
				})
				Context("Indexed", func() {
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
								cesium.Channel{Key: idxKey1, DataType: telem.TimeStampT, IsIndex: true},
								cesium.Channel{Key: idxKey2, DataType: telem.TimeStampT, IsIndex: true},
								cesium.Channel{Key: key1, Index: idxKey1, DataType: telem.Int64T},
								cesium.Channel{Key: key2, Index: idxKey2, DataType: telem.Int64T},
							)).To(Succeed())
							Expect(db.WriteArray(ctx, idxKey1, 1*telem.SecondTS, telem.NewSecondsTSV(idxData1...))).To(Succeed())
							Expect(db.WriteArray(ctx, idxKey2, 1*telem.SecondTS, telem.NewSecondsTSV(idxData2...))).To(Succeed())
							Expect(db.WriteArray(ctx, key1, 1*telem.SecondTS, telem.NewSeries(data1))).To(Succeed())
							Expect(db.WriteArray(ctx, key2, 1*telem.SecondTS, telem.NewSeries(data2))).To(Succeed())
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
								for i, series := range frame.Series {
									if frame.Keys[i] == key1 {
										actual1 = append(actual1, telem.Unmarshal[int64](series)...)
									} else {
										actual2 = append(actual2, telem.Unmarshal[int64](series)...)
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
					Context("Regression Tests", func() {
					})
				})
			})
		})
	}
})
