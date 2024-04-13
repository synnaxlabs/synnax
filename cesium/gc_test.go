package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/core"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Garbage collection", Ordered, func() {
	var (
		db     *cesium.DB
		basic1 core.ChannelKey = 1
		basic2 core.ChannelKey = 2
		index1 core.ChannelKey = 11
		pth                    = "./test/gc"
		fs                     = MustSucceed(xfs.Default.Sub(pth))
	)

	BeforeAll(func() {
		db = MustSucceed(cesium.Open("",
			cesium.WithGC(&cesium.GCConfig{
				ReadChunkSize: uint32(100 * telem.Megabyte),
				MaxGoroutine:  10,
				GcTryInterval: 10 * telem.Millisecond.Duration(),
			}),
			cesium.WithFS(fs)))
	})
	AfterAll(func() {
		Expect(db.Close()).To(Succeed())
		Expect(xfs.Default.Remove("./test")).To(Succeed())
	})

	Context("Periodic garbage collection", func() {
		It("Should recycle properly for a deletion on a rate channel", func() {
			By("Creating a channel")
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{Key: basic1, DataType: telem.Int64T, Rate: 1 * telem.Hz},
			)).To(Succeed())
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{basic1},
				Start:    10 * telem.SecondTS,
			}))

			By("Writing data to the channel")
			ok := w.Write(cesium.NewFrame(
				[]cesium.ChannelKey{basic1},
				[]telem.Series{
					telem.NewSeriesV[int64](10, 11, 12, 13, 14, 15, 16, 17, 18),
				}),
			)
			Expect(ok).To(BeTrue())
			_, ok = w.Commit()
			Expect(ok).To(BeTrue())
			Expect(w.Close()).To(Succeed())

			// Data before deletion: 10, 11, 12, 13, 14, 15, 16, 17, 18

			By("Deleting channel data")
			Expect(db.DeleteTimeRange(ctx, basic1, telem.TimeRange{
				Start: 12 * telem.SecondTS,
				End:   15 * telem.SecondTS,
			})).To(Succeed())

			// Data after deletion: 10, 11, 15, 16, 17, 18

			//By("Checking the resulting file size")
			//Eventually(func(g Gomega) uint32 {
			//	i, err := fs.Stat("./1/1.domain")
			//	g.Expect(err).ToNot(HaveOccurred())
			//	return uint32(i.Size())
			//}).Should(Equal(uint32(6 * telem.Int64T.Density())))
		})
	})

	It("Should recycle properly for deletion on an indexed channel", func() {
		By("Creating a channel")
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{Key: index1, DataType: telem.TimeStampT, IsIndex: true},
			cesium.Channel{Key: basic2, DataType: telem.Int64T, Index: index1},
		)).To(Succeed())

		By("Writing data to the channel")
		for i := 1; i <= 9; i++ {
			var data []int64
			var timestamps []telem.TimeStamp
			for j := 0; j <= 9; j++ {
				data = append(data, int64(i*100+j*10))
				timestamps = append(timestamps, telem.TimeStamp(i*10+j))
			}
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{basic2, index1},
				Start:    telem.TimeStamp(10*i) * telem.SecondTS,
			}))
			ok := w.Write(cesium.NewFrame(
				[]cesium.ChannelKey{basic2, index1},
				[]telem.Series{
					telem.NewSeriesV[int64](data...),
					telem.NewSecondsTSV(timestamps...),
				}),
			)
			Expect(ok).To(BeTrue())
			_, ok = w.Commit()
			Expect(ok).To(BeTrue())
			Expect(w.Close()).To(Succeed())
		}

		// should have been written to 10 - 99
		By("Deleting channel data")
		Expect(db.DeleteTimeRange(ctx, basic2, telem.TimeRange{
			Start: 20 * telem.SecondTS,
			End:   50 * telem.SecondTS,
		})).To(Succeed())

		Expect(db.DeleteTimeRange(ctx, basic2, telem.TimeRange{
			Start: 60 * telem.SecondTS,
			End:   66 * telem.SecondTS,
		})).To(Succeed())

		//By("Checking the resulting file size")
		//Eventually(func(g Gomega) uint32 {
		//	i, err := fs.Stat(path.Join(strconv.FormatUint(uint64(basic2), 10), "/1.domain"))
		//	g.Expect(err).ToNot(HaveOccurred())
		//	return uint32(i.Size())
		//}).Should(Equal(uint32(54 * telem.Int64T.Density())))
	})
})
