package unary_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"math"
)

var _ = Describe("Garbage Collection", func() {
	for fsName, makeFS := range fileSystems {
		fs := makeFS()
		Context("FS: "+fsName, func() {
			var (
				rateDB    *unary.DB
				dataDB    *unary.DB
				indexDB   *unary.DB
				rateKey   core.ChannelKey = 1
				dataKey   core.ChannelKey = 2
				indexKey  core.ChannelKey = 3
				pth_rate                  = rootPath + "/garbage_test/rate"
				pth_index                 = rootPath + "/garbage_test/index"
				pth_data                  = rootPath + "/garbage_test/data"
			)
			Describe("Garbage collection without threshold", func() {
				BeforeEach(func() {
					rateDB = MustSucceed(unary.Open(unary.Config{
						FS: MustSucceed(fs.Sub(pth_rate)),
						Channel: core.Channel{
							Key:      rateKey,
							DataType: telem.Int64T,
							Rate:     1 * telem.Hz,
						},
						GCThreshold: math.SmallestNonzeroFloat32,
					}))
					indexDB = MustSucceed(unary.Open(unary.Config{
						FS: MustSucceed(fs.Sub(pth_index)),
						Channel: core.Channel{
							Key:      indexKey,
							DataType: telem.TimeStampT,
							IsIndex:  true,
							Index:    indexKey,
						},
						GCThreshold: math.SmallestNonzeroFloat32,
					}))

					dataDB = MustSucceed(unary.Open(unary.Config{
						FS: MustSucceed(fs.Sub(pth_data)),
						Channel: core.Channel{
							Key:      dataKey,
							DataType: telem.Int64T,
							Index:    indexKey,
						},
						GCThreshold: math.SmallestNonzeroFloat32,
					}))
					dataDB.SetIndex(indexDB.Index())
				})
				AfterEach(func() {
					Expect(indexDB.Close()).To(Succeed())
					Expect(dataDB.Close()).To(Succeed())
					Expect(rateDB.Close()).To(Succeed())
					Expect(fs.Remove(rootPath + "/garbage_test")).To(Succeed())
				})

				Describe("Rate DB", func() {
					Specify("One pointer", func() {
						Expect(unary.Write(ctx, rateDB, 10*telem.SecondTS, telem.NewSeriesV[int64](100, 101, 102, 103, 104, 105, 106))).To(Succeed())
						Expect(rateDB.Delete(ctx, (10*telem.SecondTS + 1).Range(12*telem.SecondTS+1))).To(Succeed())

						Expect(MustSucceed(rateDB.FS.Stat("1.domain")).Size()).To(Equal(int64(7 * telem.Int64T.Density())))
						Expect(rateDB.GarbageCollect(ctx)).To(Succeed())
						Expect(MustSucceed(rateDB.FS.Stat("1.domain")).Size()).To(Equal(int64(5 * telem.Int64T.Density())))

						By("Writing some new data")
						Expect(unary.Write(ctx, rateDB, 17*telem.SecondTS, telem.NewSeriesV[int64](107, 108, 109))).To(Succeed())

						f := MustSucceed(rateDB.Read(ctx, telem.TimeRangeMax))
						Expect(f.Series).To(HaveLen(3))

						Expect(f.Series[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(10*telem.SecondTS + 1)))
						Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int64](100).Data))
						Expect(f.Series[1].TimeRange).To(Equal((12*telem.SecondTS + 1).Range(16*telem.SecondTS + 1)))
						Expect(f.Series[1].Data).To(Equal(telem.NewSeriesV[int64](103, 104, 105, 106).Data))
						Expect(f.Series[2].TimeRange).To(Equal((17 * telem.SecondTS).Range(19*telem.SecondTS + 1)))
						Expect(f.Series[2].Data).To(Equal(telem.NewSeriesV[int64](107, 108, 109).Data))
					})
					Specify("Multiple pointers", func() {
						By("Writing data to the channel")
						for i := 1; i <= 9; i++ {
							var data []int64
							for j := 0; j <= 9; j++ {
								data = append(data, int64(i*10+j))
							}
							Expect(unary.Write(ctx, rateDB, telem.TimeStamp(10*i)*telem.SecondTS, telem.NewSeriesV[int64](data...))).To(Succeed())
						}

						By("Deleting data from the channel")
						Expect(rateDB.Delete(ctx, telem.TimeRange{
							Start: 33 * telem.SecondTS,
							End:   75 * telem.SecondTS,
						})).To(Succeed())

						Expect(MustSucceed(rateDB.FS.Stat("1.domain")).Size()).To(Equal(int64(720)))
						Expect(rateDB.GarbageCollect(ctx)).To(Succeed())
						Expect(MustSucceed(rateDB.FS.Stat("1.domain")).Size()).To(Equal(int64(384)))

						By("Writing some new data")
						Expect(unary.Write(ctx, rateDB, 100*telem.SecondTS, telem.NewSeriesV[int64](100, 101)))

						By("Asserting that the data is still correct", func() {
							frame := MustSucceed(rateDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 105 * telem.SecondTS}))
							Expect(err).To(BeNil())
							Expect(frame.Series).To(HaveLen(7))

							Expect(frame.Series[2].TimeRange.End).To(Equal(33 * telem.SecondTS))
							series2Data := telem.UnmarshalSlice[int](frame.Series[2].Data, telem.Int64T)
							Expect(series2Data).To(ConsistOf(30, 31, 32))

							Expect(frame.Series[3].TimeRange.Start).To(Equal(75 * telem.SecondTS))
							series3Data := telem.UnmarshalSlice[int](frame.Series[3].Data, telem.Int64T)
							Expect(series3Data).To(ConsistOf(75, 76, 77, 78, 79))

							Expect(frame.Series[6].TimeRange.Start).To(Equal(100 * telem.SecondTS))
							series6Data := telem.UnmarshalSlice[int](frame.Series[6].Data, telem.Int64T)
							Expect(series6Data).To(ConsistOf(100, 101))
						})
					})
				})

				Describe("Channel-based DB", func() {
					Specify("One pointer", func() {
						By("Writing data to the channel")
						Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18, 19))).To(Succeed())
						Expect(unary.Write(ctx, dataDB, 10*telem.SecondTS, telem.NewSeriesV[int64](0, 1, 2, 3, 4, 5, 6, 7, 8, 9))).To(Succeed())

						By("Deleting channel data")
						Expect(dataDB.Delete(ctx, telem.TimeRange{
							Start: 12 * telem.SecondTS,
							End:   17 * telem.SecondTS,
						})).To(Succeed())

						Expect(MustSucceed(dataDB.FS.Stat("1.domain")).Size()).To(Equal(int64(80)))
						Expect(dataDB.GarbageCollect(ctx)).To(Succeed())
						Expect(MustSucceed(dataDB.FS.Stat("1.domain")).Size()).To(Equal(int64(40)))

						By("Writing some new data")
						Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSecondsTSV(20, 23, 25))).To(Succeed())
						Expect(unary.Write(ctx, dataDB, 20*telem.SecondTS, telem.NewSeriesV[int64](20, 23, 25))).To(Succeed())

						frame := MustSucceed(dataDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 30 * telem.SecondTS}))
						Expect(frame.Series).To(HaveLen(3))

						Expect(frame.Series[0].TimeRange.End).To(Equal(12 * telem.SecondTS))
						series0Data := telem.UnmarshalSlice[int](frame.Series[0].Data, telem.Int64T)
						Expect(series0Data).To(ConsistOf(0, 1))

						Expect(frame.Series[1].TimeRange.Start).To(Equal(17 * telem.SecondTS))
						series1Data := telem.UnmarshalSlice[int](frame.Series[1].Data, telem.Int64T)
						Expect(series1Data).To(ConsistOf(7, 8, 9))

						Expect(frame.Series[2].TimeRange.Start).To(Equal(20 * telem.SecondTS))
						series2Data := telem.UnmarshalSlice[int](frame.Series[2].Data, telem.Int64T)
						Expect(series2Data).To(ConsistOf(20, 23, 25))
					})
				})
				Specify("Multiple pointers", func() {
					By("Writing data to the channel")
					for i := 1; i <= 9; i++ {
						var data []int64
						var index []telem.TimeStamp

						for j := 0; j <= 9; j++ {
							data = append(data, int64(i*10+j))
							index = append(index, telem.TimeStamp(i*10+j))
						}
						Expect(unary.Write(ctx, indexDB, telem.TimeStamp(i*10)*telem.SecondTS, telem.NewSecondsTSV(index...))).To(Succeed())
						Expect(unary.Write(ctx, dataDB, telem.TimeStamp(i*10)*telem.SecondTS, telem.NewSeriesV[int64](data...))).To(Succeed())
					}

					By("Deleting data from the channel")
					Expect(dataDB.Delete(ctx, telem.TimeRange{
						Start: 33 * telem.SecondTS,
						End:   75 * telem.SecondTS,
					})).To(Succeed())

					Expect(MustSucceed(dataDB.FS.Stat("1.domain")).Size()).To(Equal(int64(720)))
					Expect(dataDB.GarbageCollect(ctx)).To(Succeed())
					Expect(MustSucceed(dataDB.FS.Stat("1.domain")).Size()).To(Equal(int64(384)))

					By("Reading data from the channel")
					frame := MustSucceed(dataDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS}))
					Expect(frame.Series).To(HaveLen(6))

					Expect(frame.Series[2].TimeRange.End).To(Equal(33 * telem.SecondTS))
					series2Data := telem.UnmarshalSlice[int](frame.Series[2].Data, telem.Int64T)
					Expect(series2Data).To(ConsistOf(30, 31, 32))

					Expect(frame.Series[3].TimeRange.Start).To(Equal(75 * telem.SecondTS))
					series3Data := telem.UnmarshalSlice[int](frame.Series[3].Data, telem.Int64T)
					Expect(series3Data).To(ConsistOf(75, 76, 77, 78, 79))
				})
			})

			Describe("GC with threshold and many files", func() {
				BeforeEach(func() {
					indexDB = MustSucceed(unary.Open(unary.Config{
						FS: MustSucceed(fs.Sub(pth_index)),
						Channel: core.Channel{
							Key:      indexKey,
							DataType: telem.TimeStampT,
							IsIndex:  true,
							Index:    indexKey,
						},
						FileSize: 50 * telem.ByteSize,
					}))

					dataDB = MustSucceed(unary.Open(unary.Config{
						FS: MustSucceed(fs.Sub(pth_data)),
						Channel: core.Channel{
							Key:      dataKey,
							DataType: telem.Int64T,
							Index:    indexKey,
						},
						GCThreshold: 0.5,
						FileSize:    50 * telem.ByteSize,
					}))
					dataDB.SetIndex(indexDB.Index())
				})
				AfterEach(func() {
					Expect(indexDB.Close()).To(Succeed())
					Expect(dataDB.Close()).To(Succeed())
					Expect(fs.Remove(rootPath + "/garbage_test")).To(Succeed())
				})

				Specify("Only some files GC", func() {
					Expect(unary.Write(ctx, indexDB, 10*telem.SecondTS, telem.NewSecondsTSV(10, 11, 12, 13, 14, 15, 16, 17, 18)))
					Expect(unary.Write(ctx, indexDB, 20*telem.SecondTS, telem.NewSecondsTSV(20, 21, 22, 23, 24, 25, 26)))
					Expect(unary.Write(ctx, indexDB, 30*telem.SecondTS, telem.NewSecondsTSV(30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41)))
					Expect(unary.Write(ctx, indexDB, 50*telem.SecondTS, telem.NewSecondsTSV(50, 51, 52, 53, 54, 55, 56)))

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
					Expect(MustSucceed(dataDB.FS.Stat("1.domain")).Size()).To(Equal(int64(72)))
					Expect(MustSucceed(dataDB.FS.Stat("2.domain")).Size()).To(Equal(int64(8)))
					Expect(MustSucceed(dataDB.FS.Stat("3.domain")).Size()).To(Equal(int64(64)))

					By("Writing more data")
					Expect(unary.Write(ctx, dataDB, 50*telem.SecondTS, telem.NewSeriesV[int64](50, 51, 52, 53, 54, 55, 56))).To(Succeed())
					Expect(unary.Write(ctx, dataDB, 17*telem.SecondTS, telem.NewSeriesV[int64](17, 18))).To(Succeed())

					By("Asserting that the data is correct", func() {
						f := MustSucceed(dataDB.Read(ctx, telem.TimeRangeMax))
						Expect(f.Series).To(HaveLen(5))

						Expect(f.Series[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(17 * telem.SecondTS)))
						Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15, 16).Data))
						Expect(f.Series[1].TimeRange).To(Equal((17 * telem.SecondTS).Range(18*telem.SecondTS + 1)))
						Expect(f.Series[1].Data).To(Equal(telem.NewSeriesV[int64](17, 18).Data))
						Expect(f.Series[2].TimeRange).To(Equal((26 * telem.SecondTS).Range(26*telem.SecondTS + 1)))
						Expect(f.Series[2].Data).To(Equal(telem.NewSeriesV[int64](26).Data))
						Expect(f.Series[3].TimeRange).To(Equal((34 * telem.SecondTS).Range(41*telem.SecondTS + 1)))
						Expect(f.Series[3].Data).To(Equal(telem.NewSeriesV[int64](34, 35, 36, 37, 38, 39, 40, 41).Data))
						Expect(f.Series[4].TimeRange).To(Equal((50 * telem.SecondTS).Range(56*telem.SecondTS + 1)))
						Expect(f.Series[4].Data).To(Equal(telem.NewSeriesV[int64](50, 51, 52, 53, 54, 55, 56).Data))
					})
				})
			})
		})
	}
})
