package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"math"
	"path"
)

var _ = Describe("Garbage collection", Ordered, func() {
	for fsName, makeFS := range fileSystems {
		fs := makeFS()
		Context("FS: "+fsName, func() {
			var (
				db    *cesium.DB
				rate  = GenerateChannelKey()
				basic = GenerateChannelKey()
				index = GenerateChannelKey()
			)

			Context("GCThreshold = 0", Ordered, func() {
				BeforeAll(func() {
					db = MustSucceed(cesium.Open(rootPath,
						cesium.WithGC(&cesium.GCConfig{
							MaxGoroutine:  10,
							GCTryInterval: 10 * telem.Millisecond.Duration(),
							GCThreshold:   math.SmallestNonzeroFloat32,
						}),
						cesium.WithFS(fs)))
				})
				AfterAll(func() {
					Expect(db.Close()).To(Succeed())
					Expect(fs.Remove(rootPath)).To(Succeed())
				})
				It("Should recycle properly for a deletion on a rate channel", func() {
					By("Creating a channel")
					Expect(db.CreateChannel(
						ctx,
						cesium.Channel{Key: rate, DataType: telem.Int64T, Rate: 1 * telem.Hz},
					)).To(Succeed())

					By("Writing data to the channel")
					Expect(db.WriteArray(ctx, rate, 10*telem.SecondTS,
						telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15, 16, 17, 18))).To(Succeed())

					By("Deleting channel data")
					Expect(db.DeleteTimeRange(ctx, rate, telem.TimeRange{
						Start: 12 * telem.SecondTS,
						End:   15 * telem.SecondTS,
					})).To(Succeed())

					By("Checking the resulting file size")
					Eventually(func(g Gomega) uint32 {
						i, err := fs.Stat(channelKeyToPath(rate) + "/1.domain")
						g.Expect(err).ToNot(HaveOccurred())
						return uint32(i.Size())
					}).Should(Equal(uint32(6 * telem.Int64T.Density())))
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
					Expect(db.DeleteTimeRange(ctx, basic, telem.TimeRange{
						Start: 20 * telem.SecondTS,
						End:   50 * telem.SecondTS,
					})).To(Succeed())

					Expect(db.DeleteTimeRange(ctx, basic, telem.TimeRange{
						Start: 60 * telem.SecondTS,
						End:   66 * telem.SecondTS,
					})).To(Succeed())

					Expect(db.DeleteTimeRange(ctx, basic, telem.TimeRange{
						Start: 63 * telem.SecondTS,
						End:   78 * telem.SecondTS,
					}))

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
					db = MustSucceed(cesium.Open(rootPath,
						cesium.WithGC(&cesium.GCConfig{
							MaxGoroutine:  10,
							GCTryInterval: 10 * telem.Millisecond.Duration(),
							GCThreshold:   float32(250*telem.ByteSize) / float32(telem.Gigabyte),
						}),
						cesium.WithFS(fs)))
				})
				AfterAll(func() {
					Expect(db.Close()).To(Succeed())
					Expect(fs.Remove(rootPath)).To(Succeed())
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
					Expect(db.DeleteTimeRange(ctx, basic, (20 * telem.SecondTS).Range(50*telem.SecondTS))).To(Succeed())

					Consistently(func(g Gomega) uint32 {
						i, err := fs.Stat(path.Join(channelKeyToPath(basic) + "/1.domain"))
						g.Expect(err).ToNot(HaveOccurred())
						return uint32(i.Size())
					}).Should(Equal(uint32(90 * telem.Int64T.Density())))

					By("Deleting more data, which should trigger GC")
					Expect(db.DeleteTimeRange(ctx, basic, (60 * telem.SecondTS).Range(66*telem.SecondTS))).To(Succeed())

					By("Checking the resulting file size")
					Eventually(func(g Gomega) uint32 {
						i, err := fs.Stat(path.Join(channelKeyToPath(basic) + "/1.domain"))
						g.Expect(err).ToNot(HaveOccurred())
						return uint32(i.Size())
					}).Should(Equal(uint32(54 * telem.Int64T.Density())))

					By("Deleting more data, which should not trigger GC")
					Expect(db.DeleteTimeRange(ctx, basic, (25 * telem.SecondTS).Range(65*telem.SecondTS))).To(Succeed())
					Consistently(func(g Gomega) uint32 {
						i, err := fs.Stat(path.Join(channelKeyToPath(basic) + "/1.domain"))
						g.Expect(err).ToNot(HaveOccurred())
						return uint32(i.Size())
					}).Should(Equal(uint32(54 * telem.Int64T.Density())))

					By("Deleting more data, which should trigger GC")
					Expect(db.DeleteTimeRange(ctx, basic, (25 * telem.SecondTS).Range(97*telem.SecondTS))).To(Succeed())
					Eventually(func(g Gomega) uint32 {
						i, err := fs.Stat(path.Join(channelKeyToPath(basic) + "/1.domain"))
						g.Expect(err).ToNot(HaveOccurred())
						return uint32(i.Size())
					}).Should(Equal(uint32(13 * telem.Int64T.Density())))
				})
			})
		})
	}
})
