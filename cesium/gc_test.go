package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"math"
	"path"
)

var _ = Describe("Garbage collection", Ordered, func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, func() {
			var (
				db      *cesium.DB
				rate    = testutil.GenerateChannelKey()
				basic   = testutil.GenerateChannelKey()
				index   = testutil.GenerateChannelKey()
				fs      xfs.FS
				cleanUp func() error
			)

			Context("GCThreshold = 0", Ordered, func() {
				BeforeAll(func() {
					fs, cleanUp = makeFS()
					db = MustSucceed(cesium.Open("",
						cesium.WithGC(&cesium.GCConfig{
							MaxGoroutine:  10,
							GCTryInterval: 10 * telem.Millisecond.Duration(),
							GCThreshold:   math.SmallestNonzeroFloat32,
						}),
						cesium.WithFS(fs),
						cesium.WithFileSize(899*telem.ByteSize)))
				})
				AfterAll(func() {
					Expect(db.Close()).To(Succeed())
					Expect(cleanUp()).To(Succeed())
				})
				It("Should recycle properly for a deletion on a rate channel", func() {
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: rate, DataType: telem.Uint64T, Rate: 1 * telem.Hz},
					)).To(Succeed())

					By("Writing data to the channel")
					for i := 1; i <= 9; i++ {
						var data []uint64
						for j := 0; j <= 9; j++ {
							data = append(data, uint64(i*100+j*10))
						}

						Expect(db.Write(ctx, telem.TimeStamp(10*i)*telem.SecondTS, cesium.NewFrame(
							[]cesium.ChannelKey{rate},
							[]telem.Series{
								telem.NewSeriesV[uint64](data...),
							},
						))).To(Succeed())
					}

					By("Deleting channel data")
					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{rate}, telem.TimeRange{
						Start: 20 * telem.SecondTS,
						End:   50 * telem.SecondTS,
					})).To(Succeed())

					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{rate}, telem.TimeRange{
						Start: 60 * telem.SecondTS,
						End:   66 * telem.SecondTS,
					})).To(Succeed())

					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{rate}, telem.TimeRange{
						Start: 63 * telem.SecondTS,
						End:   78 * telem.SecondTS,
					}))

					By("Checking the resulting file size")
					Eventually(func(g Gomega) uint32 {
						i, err := fs.Stat(channelKeyToPath(rate) + "/1.domain")
						g.Expect(err).ToNot(HaveOccurred())
						return uint32(i.Size())
					}).Should(Equal(uint32(42 * telem.Uint64T.Density())))
				})

				It("Should recycle properly for deletion on an indexed channel", func() {
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: index, DataType: telem.TimeStampT, IsIndex: true},
						cesium.Channel{Key: basic, DataType: telem.Int64T, Index: index},
					)).To(Succeed())

					By("Writing data to the channel")
					for i := 1; i <= 9; i++ {
						var data []int64
						var timestamps []telem.TimeStamp
						for j := 0; j <= 9; j++ {
							data = append(data, int64(i*100+j*10))
							timestamps = append(timestamps, telem.TimeStamp(i*10+j))
						}

						Expect(db.Write(ctx, telem.TimeStamp(10*i)*telem.SecondTS, cesium.NewFrame(
							[]cesium.ChannelKey{basic, index},
							[]telem.Series{
								telem.NewSeriesV[int64](data...),
								telem.NewSecondsTSV(timestamps...),
							},
						))).To(Succeed())
					}

					By("Deleting channel data")
					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{basic}, telem.TimeRange{
						Start: 20 * telem.SecondTS,
						End:   50 * telem.SecondTS,
					})).To(Succeed())

					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{basic}, telem.TimeRange{
						Start: 60 * telem.SecondTS,
						End:   66 * telem.SecondTS,
					})).To(Succeed())

					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{basic}, telem.TimeRange{
						Start: 63 * telem.SecondTS,
						End:   78 * telem.SecondTS,
					})).To(Succeed())

					By("Checking the resulting file size")
					Eventually(func(g Gomega) uint32 {
						i, err := fs.Stat(path.Join(channelKeyToPath(basic) + "/1.domain"))
						g.Expect(err).ToNot(HaveOccurred())
						return uint32(i.Size())
					}).Should(Equal(uint32(42 * telem.Int64T.Density())))
				})
			})

			Context("GCThreshold != 0", Ordered, func() {
				BeforeAll(func() {
					fs, cleanUp = makeFS()
					db = MustSucceed(cesium.Open("",
						cesium.WithGC(&cesium.GCConfig{
							MaxGoroutine:  10,
							GCTryInterval: 10 * telem.Millisecond.Duration(),
							GCThreshold:   float32(250) / 719,
						}),
						cesium.WithFS(fs),
						cesium.WithFileSize(899*telem.ByteSize)))
				})
				AfterAll(func() {
					Expect(db.Close()).To(Succeed())
					Expect(cleanUp()).To(Succeed())
				})
				It("Should only garbage collect after a certain amount garbage has accumulated", func() {
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: index, DataType: telem.TimeStampT, IsIndex: true},
						cesium.Channel{Key: basic, DataType: telem.Int64T, Index: index},
					)).To(Succeed())

					By("Writing data to the channel")
					for i := 1; i <= 9; i++ {
						var data []int64
						var timestamps []telem.TimeStamp
						for j := 0; j <= 9; j++ {
							data = append(data, int64(i*100+j*10))
							timestamps = append(timestamps, telem.TimeStamp(i*10+j))
						}

						Expect(db.Write(ctx, telem.TimeStamp(10*i)*telem.SecondTS, cesium.NewFrame(
							[]cesium.ChannelKey{basic, index},
							[]telem.Series{
								telem.NewSeriesV[int64](data...),
								telem.NewSecondsTSV(timestamps...),
							},
						))).To(Succeed())
					}

					By("Deleting channel data, this should not trigger GC since we only deleted 240 bytes")
					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{basic}, (20 * telem.SecondTS).Range(50*telem.SecondTS))).To(Succeed())

					Consistently(func(g Gomega) uint32 {
						i, err := fs.Stat(path.Join(channelKeyToPath(basic) + "/1.domain"))
						g.Expect(err).ToNot(HaveOccurred())
						return uint32(i.Size())
					}).Should(Equal(uint32(90 * telem.Int64T.Density())))

					By("Deleting more data, which should trigger GC")
					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{basic}, (60 * telem.SecondTS).Range(66*telem.SecondTS))).To(Succeed())

					By("Checking the resulting file size")
					Eventually(func(g Gomega) uint32 {
						i, err := fs.Stat(path.Join(channelKeyToPath(basic) + "/1.domain"))
						g.Expect(err).ToNot(HaveOccurred())
						return uint32(i.Size())
					}).Should(Equal(uint32(54 * telem.Int64T.Density())))

					By("Asserting that the data is still correct", func() {
						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, basic))
						Expect(f.Series).To(HaveLen(6))
						Expect(f.Series[0].TimeRange).To(Equal((10 * telem.SecondTS).Range(19*telem.SecondTS + 1)))
						Expect(f.Series[0].Data).To(Equal(telem.NewSeriesV[int64](100, 110, 120, 130, 140, 150, 160, 170, 180, 190).Data))

						Expect(f.Series[1].TimeRange).To(Equal((50 * telem.SecondTS).Range(59*telem.SecondTS + 1)))
						Expect(f.Series[1].Data).To(Equal(telem.NewSeriesV[int64](500, 510, 520, 530, 540, 550, 560, 570, 580, 590).Data))

						Expect(f.Series[2].TimeRange).To(Equal((66 * telem.SecondTS).Range(69*telem.SecondTS + 1)))
						Expect(f.Series[2].Data).To(Equal(telem.NewSeriesV[int64](660, 670, 680, 690).Data))
					})
				})
			})
			Context("Multiple files", func() {
				BeforeAll(func() {
					fs, cleanUp = makeFS()
					db = MustSucceed(cesium.Open("",
						cesium.WithGC(&cesium.GCConfig{
							MaxGoroutine:  10,
							GCTryInterval: 10 * telem.Millisecond.Duration(),
							GCThreshold:   1,
						}),
						cesium.WithFS(fs), cesium.WithFileSize(49*telem.ByteSize)))
				})
				AfterAll(func() {
					Expect(db.Close()).To(Succeed())
					Expect(cleanUp()).To(Succeed())
				})
				It("Should only garbage collect after a certain amount garbage has accumulated", func() {
					By("Creating channels")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: index, DataType: telem.TimeStampT, IsIndex: true},
						cesium.Channel{Key: basic, DataType: telem.Int64T, Index: index},
					)).To(Succeed())

					By("Writing data to the channel")
					for i := 1; i <= 9; i++ {
						var data []int64
						var timestamps []telem.TimeStamp
						for j := 0; j <= 9; j++ {
							data = append(data, int64(i*100+j*10))
							timestamps = append(timestamps, telem.TimeStamp(i*10+j))
						}

						Expect(db.Write(ctx, telem.TimeStamp(10*i)*telem.SecondTS, cesium.NewFrame(
							[]cesium.ChannelKey{basic, index},
							[]telem.Series{
								telem.NewSeriesV[int64](data...),
								telem.NewSecondsTSV(timestamps...),
							},
						))).To(Succeed())
					}

					By("Deleting channel data")
					Expect(db.DeleteTimeRange(ctx, []cesium.ChannelKey{basic}, (26 * telem.SecondTS).Range(55*telem.SecondTS))).To(Succeed())
					// File 2 should not be garbage collected (4 * 8 < 39).
					// Files 3, 4 should be garbage collected (10 * 8 > 39).
					// File 5 should be garbage collected (5 * 8 > 39).

					Consistently(func(g Gomega) uint32 {
						i, err := fs.Stat(path.Join(channelKeyToPath(basic) + "/2.domain"))
						g.Expect(err).ToNot(HaveOccurred())
						return uint32(i.Size())
					}).Should(Equal(uint32(10 * telem.Int64T.Density())))

					Eventually(func(g Gomega) {
						g.Expect(MustSucceed(fs.Stat(path.Join(channelKeyToPath(basic) + "/3.domain"))).Size()).To(Equal(int64(0)))
						g.Expect(MustSucceed(fs.Stat(path.Join(channelKeyToPath(basic) + "/4.domain"))).Size()).To(Equal(int64(0)))
						g.Expect(MustSucceed(fs.Stat(path.Join(channelKeyToPath(basic) + "/5.domain"))).Size()).To(Equal(int64(40)))
					}).Should(Succeed())

					By("Writing more data – they should go to the newly freed files, i.e. file 3 or file 4")
					// This should go to file 10.
					Expect(db.Write(ctx, 200*telem.SecondTS, cesium.NewFrame(
						[]cesium.ChannelKey{basic, index},
						[]telem.Series{
							telem.NewSeriesV[int64](2000, 2010, 2020, 2030, 2040),
							telem.NewSecondsTSV(200, 201, 202, 203, 204),
						},
					))).To(Succeed())
					Expect(db.Write(ctx, 300*telem.SecondTS, cesium.NewFrame(
						[]cesium.ChannelKey{basic, index},
						[]telem.Series{
							telem.NewSeriesV[int64](3000, 3010, 3020),
							telem.NewSecondsTSV(300, 301, 302),
						},
					))).To(Succeed())
					Expect([]int64{MustSucceed(fs.Stat(path.Join(channelKeyToPath(basic) + "/3.domain"))).Size(),
						MustSucceed(fs.Stat(path.Join(channelKeyToPath(basic) + "/4.domain"))).Size()}).
						To(ConsistOf(int64(24), int64(0)))

					By("Asserting that the data is correct", func() {
						f := MustSucceed(db.Read(ctx, telem.TimeRangeMax, basic))
						Expect(f.Series).To(HaveLen(9))
						Expect(f.Series[1].TimeRange).To(Equal((20 * telem.SecondTS).Range(26 * telem.SecondTS)))
						Expect(f.Series[1].Data).To(Equal(telem.NewSeriesV[int64](200, 210, 220, 230, 240, 250).Data))
						Expect(f.Series[2].TimeRange).To(Equal((55 * telem.SecondTS).Range(59*telem.SecondTS + 1)))
						Expect(f.Series[2].Data).To(Equal(telem.NewSeriesV[int64](550, 560, 570, 580, 590).Data))
						Expect(f.Series[7].TimeRange).To(Equal((200 * telem.SecondTS).Range(204*telem.SecondTS + 1)))
						Expect(f.Series[7].Data).To(Equal(telem.NewSeriesV[int64](2000, 2010, 2020, 2030, 2040).Data))
						Expect(f.Series[8].TimeRange).To(Equal((300 * telem.SecondTS).Range(302*telem.SecondTS + 1)))
						Expect(f.Series[8].Data).To(Equal(telem.NewSeriesV[int64](3000, 3010, 3020).Data))
					})
				})
			})
		})
	}
})
