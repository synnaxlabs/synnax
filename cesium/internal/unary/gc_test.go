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
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/unary"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Garbage Collection", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, func() {
			var (
				dataDB   *unary.DB
				dataFS   xfs.FS
				indexDB  *unary.DB
				indexFS  xfs.FS
				dataKey  core.ChannelKey = 2
				indexKey core.ChannelKey = 3
				cleanUp  func() error
			)
			Describe("Garbage collection without threshold", func() {
				BeforeEach(func() {
					var fs xfs.FS
					fs, cleanUp = makeFS()
					indexFS = MustSucceed(fs.Sub("index"))
					indexDB = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        indexFS,
						MetaCodec: codec,
						Channel: core.Channel{
							Name:     "Chin",
							Key:      indexKey,
							DataType: telem.TimeStampT,
							IsIndex:  true,
							Index:    indexKey,
						},
						FileSize:        899 * telem.Byte,
						GCThreshold:     math.SmallestNonzeroFloat32,
						Instrumentation: PanicLogger(),
					}))
					dataFS = MustSucceed(fs.Sub("data"))
					dataDB = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        dataFS,
						MetaCodec: codec,
						Channel: core.Channel{
							Name:     "Renan",
							Key:      dataKey,
							DataType: telem.Int64T,
							Index:    indexKey,
						},
						FileSize:        899 * telem.Byte,
						GCThreshold:     math.SmallestNonzeroFloat32,
						Instrumentation: PanicLogger(),
					}))
					dataDB.SetIndex(indexDB.Index())
				})
				AfterEach(func() {
					Expect(indexDB.Close()).To(Succeed())
					Expect(dataDB.Close()).To(Succeed())
					Expect(cleanUp()).To(Succeed())
				})

				It("Should correctly delete and re-read data from the channel", func() {
					By("Writing data to the channel")
					for i := 1; i <= 9; i++ {
						var data []int64
						var index []telem.TimeStamp

						for j := 0; j <= 9; j++ {
							data = append(data, int64(i*10+j))
							index = append(index, telem.TimeStamp(i*10+j))
						}
						Expect(unary.Write(ctx, indexDB, telem.TimeStamp(i*10)*telem.SecondTS, telem.NewSeriesSecondsTSV(index...))).To(Succeed())
						Expect(unary.Write(ctx, dataDB, telem.TimeStamp(i*10)*telem.SecondTS, telem.NewSeriesV[int64](data...))).To(Succeed())
					}

					By("Deleting data from the channel")
					Expect(dataDB.Delete(ctx, telem.TimeRange{
						Start: 33 * telem.SecondTS,
						End:   75 * telem.SecondTS,
					})).To(Succeed())

					Expect(MustSucceed(dataFS.Stat("1.domain")).Size()).To(Equal(int64(720)))
					Expect(dataDB.GarbageCollect(ctx)).To(Succeed())
					Expect(MustSucceed(dataFS.Stat("1.domain")).Size()).To(Equal(int64(384)))

					By("Reading data from the channel")
					frame := MustSucceed(dataDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS}))
					Expect(frame.Count()).To(Equal(6))

					Expect(frame.SeriesAt(2).TimeRange.End).To(Equal(33 * telem.SecondTS))
					series2Data := telem.UnmarshalSlice[int64](frame.SeriesAt(2).Data, telem.Int64T)
					Expect(series2Data).To(ConsistOf(int64(30), int64(31), int64(32)))

					Expect(frame.SeriesAt(3).TimeRange.Start).To(Equal(75 * telem.SecondTS))
					series3Data := telem.UnmarshalSlice[int64](frame.SeriesAt(3).Data, telem.Int64T)
					Expect(series3Data).To(ConsistOf(
						int64(75),
						int64(76),
						int64(77),
						int64(78),
						int64(79),
					))
				})
			})

			Describe("GC with threshold and many files", func() {
				BeforeEach(func() {
					var fs xfs.FS
					fs, cleanUp = makeFS()
					indexFS = MustSucceed(fs.Sub("index"))
					indexDB = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        indexFS,
						MetaCodec: codec,
						Channel: core.Channel{
							Name:     "Wilkes",
							Key:      indexKey,
							DataType: telem.TimeStampT,
							IsIndex:  true,
							Index:    indexKey,
						},
						FileSize:        50 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))
					dataFS = MustSucceed(fs.Sub("data"))
					dataDB = MustSucceed(unary.Open(ctx, unary.Config{
						FS:        dataFS,
						MetaCodec: codec,
						Channel: core.Channel{
							Name:     "Lincoln",
							Key:      dataKey,
							DataType: telem.Int64T,
							Index:    indexKey,
						},
						GCThreshold:     0.5,
						FileSize:        50 * telem.Byte,
						Instrumentation: PanicLogger(),
					}))
					dataDB.SetIndex(indexDB.Index())
				})
				AfterEach(func() {
					Expect(indexDB.Close()).To(Succeed())
					Expect(dataDB.Close()).To(Succeed())
					Expect(cleanUp()).To(Succeed())
				})

				Specify("Only some files GC", func() {
					Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSeriesSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18)))
					Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSeriesSecondsTSV(20, 21, 22, 23, 24, 25, 26)))
					Expect(unary.Write(ctx, indexDB, 30*telem.SecondTS, telem.NewSeriesSecondsTSV(30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41)))
					Expect(unary.Write(ctx, indexDB, 50*telem.SecondTS, telem.NewSeriesSecondsTSV(50, 51, 52, 53, 54, 55, 56)))

					Expect(unary.Write(ctx, dataDB, 10*telem.SecondTS, telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15, 16, 17, 18)))
					Expect(unary.Write(ctx, dataDB, 20*telem.SecondTS, telem.NewSeriesV[int64](20, 21, 22, 23, 24, 25, 26)))
					Expect(unary.Write(ctx, dataDB, 30*telem.SecondTS, telem.NewSeriesV[int64](30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41)))

					Expect(dataDB.Delete(ctx, (17 * telem.SecondTS).Range(19*telem.SecondTS))).To(Succeed())
					Expect(dataDB.Delete(ctx, (20 * telem.SecondTS).Range(26*telem.SecondTS))).To(Succeed())
					Expect(dataDB.Delete(ctx, (27 * telem.SecondTS).Range(34*telem.SecondTS))).To(Succeed())

					// 1: 10, 11, 12, 13, 14, 15, 16 (size = 56, tombstone size = 16)
					// 2: 26 (size = 8, tombstone size = 48)
					// 3: 34, 35, 36, 37, 38, 39, 40, 41 (size = 64, tombstone size = 32)

					By("Expecting files 2 and 3 to garbage collect")
					Expect(dataDB.GarbageCollect(ctx)).To(Succeed())
					Expect(MustSucceed(dataFS.Stat("1.domain")).Size()).To(Equal(int64(72)))
					Expect(MustSucceed(dataFS.Stat("2.domain")).Size()).To(Equal(int64(8)))
					Expect(MustSucceed(dataFS.Stat("3.domain")).Size()).To(Equal(int64(64)))

					By("Writing more data")
					Expect(unary.Write(ctx, dataDB, 50*telem.SecondTS, telem.NewSeriesV[int64](50, 51, 52, 53, 54, 55, 56))).To(Succeed())
					Expect(unary.Write(ctx, dataDB, 17*telem.SecondTS, telem.NewSeriesV[int64](17, 18))).To(Succeed())

					By("Asserting that the data is correct")
					f := MustSucceed(dataDB.Read(ctx, telem.TimeRangeMax))
					Expect(f.Count()).To(Equal(5))

					first := f.SeriesAt(0)
					Expect(first.TimeRange).To(Equal((10 * telem.SecondTS).Range(17 * telem.SecondTS)))
					Expect(first.Data).To(Equal(telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15, 16).Data))

					second := f.SeriesAt(1)
					Expect(second.TimeRange).To(Equal((17 * telem.SecondTS).Range(18*telem.SecondTS + 1)))
					Expect(second.Data).To(Equal(telem.NewSeriesV[int64](17, 18).Data))

					third := f.SeriesAt(2)
					Expect(third.TimeRange).To(Equal((26 * telem.SecondTS).Range(26*telem.SecondTS + 1)))
					Expect(third.Data).To(Equal(telem.NewSeriesV[int64](26).Data))

					fourth := f.SeriesAt(3)
					Expect(fourth.TimeRange).To(Equal((34 * telem.SecondTS).Range(41*telem.SecondTS + 1)))
					Expect(fourth.Data).To(Equal(telem.NewSeriesV[int64](34, 35, 36, 37, 38, 39, 40, 41).Data))

					fifth := f.SeriesAt(4)
					Expect(fifth.TimeRange).To(Equal((50 * telem.SecondTS).Range(56*telem.SecondTS + 1)))
					Expect(fifth.Data).To(Equal(telem.NewSeriesV[int64](50, 51, 52, 53, 54, 55, 56).Data))
				})
			})
		})
	}
})
