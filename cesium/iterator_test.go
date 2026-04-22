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
	"github.com/synnaxlabs/cesium/internal/channel"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator Behavior", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, Ordered, func() {
			ShouldNotLeakGoroutinesPerSpec()
			var (
				db      *cesium.DB
				fs      fs.FS
				cleanUp func() error
			)
			BeforeAll(func(ctx SpecContext) {
				fs, cleanUp = makeFS()
				db = openDBOnFS(ctx, fs)
			})
			AfterAll(func() {
				Expect(db.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})

			Describe("Accuracy", func() {
				var (
					data1Key, index1Key, data2Key, index2Key cesium.ChannelKey
					data1, index1, data2, index2             cesium.Channel
					i                                        *cesium.Iterator
				)
				BeforeAll(func(ctx SpecContext) {
					data1Key, index1Key, data2Key, index2Key = GenerateChannelKey(),
						GenerateChannelKey(), GenerateChannelKey(), GenerateChannelKey()
					index1 = cesium.Channel{Key: index1Key, Name: "Magellan", IsIndex: true, DataType: telem.TimeStampT}
					data1 = cesium.Channel{Key: data1Key, Name: "Columbus", Index: index1Key, DataType: telem.Uint16T}
					index2 = cesium.Channel{Key: index2Key, Name: "DaGama", IsIndex: true, DataType: telem.TimeStampT}
					data2 = cesium.Channel{Key: data2Key, Name: "Vespucci", Index: index2Key, DataType: telem.Uint16T}

					Expect(db.CreateChannel(ctx, index1, data1, index2, data2)).To(Succeed())
					Expect(db.Write(ctx, 0, telem.MultiFrame(
						[]cesium.ChannelKey{index1Key, data1Key},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(0, 1, 2),
							telem.NewSeriesV[uint16](10, 11, 12),
						},
					))).To(Succeed())
					Expect(db.Write(ctx, 10*telem.SecondTS, telem.MultiFrame(
						[]cesium.ChannelKey{index1Key, data1Key},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(10, 12, 15),
							telem.NewSeriesV[uint16](20, 22, 25),
						},
					))).To(Succeed())
					Expect(db.Write(ctx, 4*telem.SecondTS, telem.MultiFrame(
						[]cesium.ChannelKey{index1Key, data1Key},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(4, 7, 9),
							telem.NewSeriesV[uint16](14, 17, 19),
						},
					))).To(Succeed())
					Expect(db.Write(ctx, 2*telem.SecondTS, telem.MultiFrame(
						[]cesium.ChannelKey{index2Key, data2Key},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(2, 3, 6, 8),
							telem.NewSeriesV[uint16](2, 3, 6, 8),
						},
					))).To(Succeed())
					Expect(db.Write(ctx, 11*telem.SecondTS, telem.MultiFrame(
						[]cesium.ChannelKey{index2Key, data2Key},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(11, 12, 13, 15),
							telem.NewSeriesV[uint16](11, 12, 13, 15),
						},
					)))

					// Index1: 0  1  2 / _  4  _  _  7  _  9  /  10  _  12   _   _  15
					// Data1: 10 11 12 / _ 14  _  _ 17  _ 19  /  20  _  22   _   _  25
					// Index2: _  _  2   3  _  _  6  _  8  /  _  _  11  12  13   _  15
					// Data2:  _  _  2   3  _  _  6  _  8  /  _  _  11  12  13   _  15
				})

				Context("Basic", func() {
					Specify("SeekLast", func() {
						i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
							Bounds:   telem.TimeRangeMax,
							Channels: []cesium.ChannelKey{data1Key, data2Key},
						}))
						Expect(i.SeekLast()).To(BeTrue())
						Expect(i.Prev(5 * telem.Second)).To(BeTrue())
						f := i.Value()
						Expect(f.Count()).To(Equal(2))
						Expect(f.Get(data1Key).Series[0]).To(telem.MatchSeriesDataV[uint16](22, 25))
						Expect(f.Get(data2Key).Series[0]).To(telem.MatchSeriesDataV[uint16](11, 12, 13, 15))
						Expect(i.Close()).To(Succeed())
					})
					Specify("SeekLE", func() {
						i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
							Bounds:   telem.TimeRangeMax,
							Channels: []cesium.ChannelKey{data1Key, data2Key},
						}))
						Expect(i.SeekLE(4 * telem.SecondTS)).To(BeTrue())
						Expect(i.Next(6 * telem.Second)).To(BeTrue())
						f := i.Value()
						Expect(f.Count()).To(Equal(2))
						Expect(f.Get(data1Key).Series[0]).To(telem.MatchSeriesDataV[uint16](14, 17, 19))
						Expect(f.Get(data2Key).Series[0]).To(telem.MatchSeriesDataV[uint16](6, 8))
						Expect(i.Close()).To(Succeed())
					})

					Specify("SeekGE", func() {
						i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
							Bounds:   telem.TimeRangeMax,
							Channels: []cesium.ChannelKey{data1Key, data2Key},
						}))
						Expect(i.SeekGE(9 * telem.SecondTS)).To(BeTrue())
						Expect(i.Next(3 * telem.Second)).To(BeTrue())
						f := i.Value()
						Expect(f.Get(data1Key).Series[0]).To(telem.MatchSeriesData(telem.NewSeriesV[uint16](19)))
						Expect(f.Get(data1Key).Series[1]).To(telem.MatchSeriesData(telem.NewSeriesV[uint16](20)))
						Expect(f.Get(data2Key).Series[0]).To(telem.MatchSeriesData(telem.NewSeriesV[uint16](11, 12, 13)))
						Expect(i.Close()).To(Succeed())
					})

					Specify("SetBounds & Error", func() {
						i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
							Bounds:   telem.TimeRangeMax,
							Channels: []cesium.ChannelKey{data1Key, data2Key},
						}))
						Expect(i.SeekGE(12 * telem.SecondTS)).To(BeTrue())
						Expect(i.Next(3 * telem.Second)).To(BeTrue())
						i.SetBounds((6 * telem.SecondTS).Range(9 * telem.SecondTS))
						Expect(i.Valid()).To(BeFalse())
						Expect(i.Error()).ToNot(HaveOccurred())
						Expect(i.Close()).To(Succeed())
						Expect(i.SeekFirst()).To(BeFalse())
						Expect(i.Error()).To(MatchError(ContainSubstring("closed")))
					})
				})

				Specify("With bound", func() {
					i = MustSucceed(db.OpenIterator(cesium.IteratorConfig{
						Bounds:   (1 * telem.SecondTS).Range(13 * telem.SecondTS),
						Channels: []cesium.ChannelKey{data1Key, data2Key},
					}))
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(4 * telem.Second)).To(BeTrue())
					f := i.Value()
					Expect(f.Count()).To(Equal(3))
					series1 := f.Get(data1Key)
					Expect(series1.Series).To(HaveLen(2))
					Expect(f.Get(data1Key).Series[0]).To(telem.MatchSeriesDataV[uint16](11, 12))
					Expect(f.Get(data1Key).Series[0].TimeRange).To(Equal((1 * telem.SecondTS).Range(2*telem.SecondTS + 1)))
					Expect(f.Get(data1Key).Series[1]).To(telem.MatchSeriesDataV[uint16](14))
					Expect(f.Get(data1Key).Series[1].TimeRange).To(Equal((4 * telem.SecondTS).Range(5 * telem.SecondTS)))
					Expect(f.Get(data2Key).Series[0]).To(telem.MatchSeriesDataV[uint16](2, 3))
					Expect(f.Get(data2Key).TimeRange()).To(Equal((2 * telem.SecondTS).Range(6 * telem.SecondTS)))

					Expect(i.Next(20 * telem.Second)).To(BeTrue())
					f = i.Value()
					Expect(f.Count()).To(Equal(4))
					Expect(f.Get(data1Key).Series[0]).To(telem.MatchSeriesDataV[uint16](17, 19))
					Expect(f.Get(data1Key).Series[0].TimeRange).To(Equal((5 * telem.SecondTS).Range(9*telem.SecondTS + 1)))
					Expect(f.Get(data1Key).Series[1]).To(telem.MatchSeriesDataV[uint16](20, 22))
					Expect(f.Get(data1Key).Series[1].TimeRange).To(Equal((10 * telem.SecondTS).Range(13 * telem.SecondTS)))

					Expect(f.Get(data2Key).Series[0]).To(telem.MatchSeriesDataV[uint16](6, 8))
					Expect(f.Get(data2Key).Series[0].TimeRange).To(Equal((6 * telem.SecondTS).Range(8*telem.SecondTS + 1)))
					Expect(f.Get(data2Key).Series[1]).To(telem.MatchSeriesDataV[uint16](11, 12))
					Expect(f.Get(data2Key).Series[1].TimeRange).To(Equal((11 * telem.SecondTS).Range(13 * telem.SecondTS)))

					Expect(i.Next(1 * telem.Second)).To(BeFalse())

					Expect(i.Prev(20 * telem.Second)).To(BeTrue())
					f = i.Value()
					Expect(f.Count()).To(Equal(5))
					Expect(f.Get(data1Key).Series[0]).To(telem.MatchSeriesDataV[uint16](11, 12))
					Expect(f.Get(data1Key).Series[1]).To(telem.MatchSeriesDataV[uint16](14, 17, 19))
					Expect(f.Get(data1Key).Series[2]).To(telem.MatchSeriesDataV[uint16](20, 22))
					Expect(f.Get(data2Key).Series[0]).To(telem.MatchSeriesDataV[uint16](2, 3, 6, 8))
					Expect(f.Get(data2Key).Series[1]).To(telem.MatchSeriesDataV[uint16](11, 12))
					Expect(i.Close()).To(Succeed())
				})

				Specify("No bound", func() {
					i = MustSucceed(db.OpenIterator(cesium.IteratorConfig{
						Bounds:   telem.TimeRangeMax,
						Channels: []cesium.ChannelKey{data1Key, data2Key},
					}))
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(4 * telem.Second)).To(BeTrue())
					f := i.Value()
					Expect(f.Count()).To(Equal(2))
					Expect(f.Get(data1Key).Series[0]).To(telem.MatchSeriesDataV[uint16](10, 11, 12))
					Expect(f.Get(data1Key).Series[0].TimeRange).To(Equal((0 * telem.SecondTS).Range(2*telem.SecondTS + 1)))
					Expect(f.Get(data2Key).Series[0]).To(telem.MatchSeriesDataV[uint16](2, 3))
					Expect(f.Get(data2Key).Series[0].TimeRange).To(Equal((2 * telem.SecondTS).Range(6 * telem.SecondTS)))

					Expect(i.Next(20 * telem.Second)).To(BeTrue())
					f = i.Value()
					Expect(f.Count()).To(Equal(4))
					Expect(f.Get(data1Key).Series[0]).To(telem.MatchSeriesDataV[uint16](14, 17, 19))
					Expect(f.Get(data1Key).Series[0].TimeRange).To(Equal((4 * telem.SecondTS).Range(9*telem.SecondTS + 1)))
					Expect(f.Get(data1Key).Series[1]).To(telem.MatchSeriesDataV[uint16](20, 22, 25))
					Expect(f.Get(data1Key).Series[1].TimeRange).To(Equal((10 * telem.SecondTS).Range(15*telem.SecondTS + 1)))
					Expect(f.Get(data2Key).Series[0]).To(telem.MatchSeriesDataV[uint16](6, 8))
					Expect(f.Get(data2Key).Series[0].TimeRange).To(Equal((6 * telem.SecondTS).Range(8*telem.SecondTS + 1)))
					Expect(f.Get(data2Key).Series[1]).To(telem.MatchSeriesDataV[uint16](11, 12, 13, 15))
					Expect(f.Get(data2Key).Series[1].TimeRange).To(Equal((11 * telem.SecondTS).Range(15*telem.SecondTS + 1)))

					Expect(i.Next(1 * telem.Second)).To(BeFalse())
					Expect(i.Close()).To(Succeed())
				})

				Specify("Auto-Span", func() {
					// Index1: [ 0  1  2 / _  4]  _  _  [7  _  9  /  10  _  12]   _   _  15
					// Data1:  [10 11 12 / _ 14]  _  _ [17  _ 19  /  20  _  22]   _   _  25
					// Index2: _  _  [2   3  _  _  6  _  8]  _  /  _  [11  12  13   _]  15
					// Data2:  _  _  [2   3  _  _  6  _  8]  _  /  _  [11  12  13  _]  15
					i = MustSucceed(db.OpenIterator(cesium.IteratorConfig{
						Bounds:        telem.TimeRangeMax,
						Channels:      []cesium.ChannelKey{data1Key, data2Key},
						AutoChunkSize: 4,
					}))
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(cesium.AutoSpan)).To(BeTrue())
					f := i.Value()
					Expect(f.Count()).To(Equal(3))
					Expect(f.Get(data1Key).Series[0]).To(telem.MatchSeriesDataV[uint16](10, 11, 12))
					Expect(f.Get(data1Key).Series[1]).To(telem.MatchSeriesDataV[uint16](14))
					Expect(f.Get(data2Key).Series[0]).To(telem.MatchSeriesDataV[uint16](2, 3, 6, 8))

					Expect(i.Next(cesium.AutoSpan)).To(BeTrue())
					f = i.Value()
					Expect(f.Count()).To(Equal(3))
					Expect(f.Get(data1Key).Series[0]).To(telem.MatchSeriesDataV[uint16](17, 19))
					Expect(f.Get(data1Key).Series[1]).To(telem.MatchSeriesDataV[uint16](20, 22))
					Expect(f.Get(data2Key).Series[0]).To(telem.MatchSeriesDataV[uint16](11, 12, 13, 15))

					Expect(i.Next(cesium.AutoSpan)).To(BeTrue())
					f = i.Value()
					Expect(f.Count()).To(Equal(1))
					Expect(f.Get(data1Key).Series[0]).To(telem.MatchSeriesDataV[uint16](25))
					Expect(i.Close()).To(Succeed())
				})
			})

			Describe("Variable Channels", func() {
				var (
					varIdxKey  cesium.ChannelKey
					varDataKey cesium.ChannelKey
				)
				BeforeAll(func(ctx SpecContext) {
					varIdxKey = GenerateChannelKey()
					varDataKey = GenerateChannelKey()
					Expect(db.CreateChannel(ctx,
						cesium.Channel{Key: varIdxKey, Name: "var-iter-idx", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: varDataKey, Name: "var-iter-data", Index: varIdxKey, DataType: telem.StringT},
					)).To(Succeed())
					Expect(db.Write(ctx, 100*telem.SecondTS, telem.MultiFrame(
						[]cesium.ChannelKey{varIdxKey, varDataKey},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(100, 101, 102, 103, 104, 105, 106, 107, 108, 109),
							telem.NewSeriesV("s0", "s1", "s2", "s3", "s4", "s5", "s6", "s7", "s8", "s9"),
						},
					))).To(Succeed())
				})
				It("Should iterate forward through all data", func() {
					i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
						Bounds:   (100 * telem.SecondTS).Range(110 * telem.SecondTS),
						Channels: []cesium.ChannelKey{varDataKey},
					}))
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(telem.TimeSpanMax)).To(BeTrue())
					Expect(i.Value().Get(varDataKey).Series[0].Len()).To(Equal(int64(10)))
					Expect(i.Close()).To(Succeed())
				})
				It("Should iterate backward through all data", func() {
					i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
						Bounds:   (100 * telem.SecondTS).Range(110 * telem.SecondTS),
						Channels: []cesium.ChannelKey{varDataKey},
					}))
					Expect(i.SeekLast()).To(BeTrue())
					Expect(i.Prev(telem.TimeSpanMax)).To(BeTrue())
					Expect(i.Value().Get(varDataKey).Series[0].Len()).To(Equal(int64(10)))
					Expect(i.Close()).To(Succeed())
				})
				It("Should read a sub-range of data", func() {
					i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
						Bounds:   (102 * telem.SecondTS).Range(106 * telem.SecondTS),
						Channels: []cesium.ChannelKey{varDataKey},
					}))
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(telem.TimeSpanMax)).To(BeTrue())
					Expect(telem.UnmarshalSeries[string](i.Value().Get(varDataKey).Series[0])).To(
						Equal([]string{"s2", "s3", "s4", "s5"}),
					)
					Expect(i.Close()).To(Succeed())
				})
				It("Should SeekGE to a specific timestamp", func() {
					i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
						Bounds:   (100 * telem.SecondTS).Range(110 * telem.SecondTS),
						Channels: []cesium.ChannelKey{varDataKey},
					}))
					Expect(i.SeekGE(105 * telem.SecondTS)).To(BeTrue())
					Expect(i.Next(3 * telem.Second)).To(BeTrue())
					Expect(telem.UnmarshalSeries[string](i.Value().Get(varDataKey).Series[0])).To(
						Equal([]string{"s5", "s6", "s7"}),
					)
					Expect(i.Close()).To(Succeed())
				})
				It("Should iterate alongside fixed channels", func(ctx SpecContext) {
					var (
						mixIdx   = GenerateChannelKey()
						fixedKey = GenerateChannelKey()
						varKey   = GenerateChannelKey()
					)
					Expect(db.CreateChannel(ctx,
						cesium.Channel{Key: mixIdx, Name: "mix-iter-idx", IsIndex: true, DataType: telem.TimeStampT},
						cesium.Channel{Key: fixedKey, Name: "mix-iter-fixed", Index: mixIdx, DataType: telem.Int64T},
						cesium.Channel{Key: varKey, Name: "mix-iter-var", Index: mixIdx, DataType: telem.StringT},
					)).To(Succeed())
					Expect(db.Write(ctx, 200*telem.SecondTS, telem.MultiFrame(
						[]cesium.ChannelKey{mixIdx, fixedKey, varKey},
						[]telem.Series{
							telem.NewSeriesSecondsTSV(200, 201, 202, 203, 204, 205, 206, 207, 208, 209),
							telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5, 6, 7, 8, 9),
							telem.NewSeriesV("s0", "s1", "s2", "s3", "s4", "s5", "s6", "s7", "s8", "s9"),
						},
					))).To(Succeed())
					i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
						Bounds:   (203 * telem.SecondTS).Range(207 * telem.SecondTS),
						Channels: []cesium.ChannelKey{varKey, fixedKey},
					}))
					Expect(i.SeekFirst()).To(BeTrue())
					Expect(i.Next(telem.TimeSpanMax)).To(BeTrue())
					f := i.Value()
					Expect(telem.UnmarshalSeries[string](f.Get(varKey).Series[0])).To(
						Equal([]string{"s3", "s4", "s5", "s6"}),
					)
					Expect(telem.UnmarshalSeries[int64](f.Get(fixedKey).Series[0])).To(
						Equal([]int64{3, 4, 5, 6}),
					)
					Expect(i.Close()).To(Succeed())
				})
			})

			Describe("Open", func() {
				It("Should return an error when attempting to open an iterator on a virtual channel", func(ctx SpecContext) {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      key,
						Name:     "Marco",
						DataType: telem.Float32T,
						Virtual:  true,
					})).To(Succeed())
					_, err := db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{key}})
					Expect(err).To(MatchError(ContainSubstring("virtual")))
				})
			})

			Describe("Close", func() {
				It("Should not allow operations on a closed iterator", func(ctx SpecContext) {
					key := GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{
						Key:      key,
						Name:     "Cook",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					})).To(Succeed())
					i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []channel.Key{key}}))
					Expect(i.Close()).To(Succeed())
					Expect(i.Valid()).To(BeFalse())
					Expect(i.SeekFirst()).To(BeFalse())
					Expect(i.Valid()).To(BeFalse())
					Expect(i.Error()).To(MatchError(cesium.ErrIteratorClosed))
					Expect(i.Close()).To(Succeed())
				})

				It("Should not allow opening an iterator on a closed db", func(ctx SpecContext) {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(ctx, sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{
						Key:      key,
						Name:     "Drake",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					_, err := subDB.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{key}})
					Expect(err).To(MatchError(cesium.ErrDBClosed))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})

				It("Should not allow opening a stream iterator on a closed db", func(ctx SpecContext) {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(ctx, sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{
						Key:      key,
						Name:     "Polo",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					_, err := subDB.NewStreamIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{key}})
					Expect(err).To(MatchError(cesium.ErrDBClosed))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})

				It("Should not allow reading from a closed db", func(ctx SpecContext) {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(ctx, sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{
						Key:      key,
						Name:     "Zheng",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					_, err := subDB.Read(ctx, telem.TimeRangeMax, key)
					Expect(err).To(MatchError(cesium.ErrDBClosed))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})
			})
		})
	}
})
