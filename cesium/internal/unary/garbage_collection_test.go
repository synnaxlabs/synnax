package unary_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Garbage Collection", Ordered, func() {
	var (
		rateDB *unary.DB
		//dataDB *unary.DB
		//indexDB *unary.DB
		rateKey core.ChannelKey = 1
		//dataKey core.ChannelKey = 2
		//indexKey core.ChannelKey = 3
		pth = "./tests/garbage_test"
	)

	Describe("Garbage Collect after deletion", func() {
		It("Should Garbage Collect when called", func() {
			rateDB = MustSucceed(unary.Open(unary.Config{
				FS: MustSucceed(fs.Default.Sub(pth)),
				Channel: core.Channel{
					Key:      rateKey,
					DataType: telem.Int64T,
					Rate:     1 * telem.Hz,
				},
			}))

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

			Expect(rateDB.Domain.CollectTombstone(ctx, 1000)).To(Succeed())

			By("Reading data from the channel")
			frame, err := rateDB.Read(ctx, telem.TimeRange{Start: 10 * telem.SecondTS, End: 100 * telem.SecondTS})
			Expect(err).To(BeNil())
			Expect(frame.Series).To(HaveLen(6))

			Expect(frame.Series[2].TimeRange.End).To(Equal(33 * telem.SecondTS))
			series0Data := telem.UnmarshalSlice[int](frame.Series[2].Data, telem.Int64T)
			Expect(series0Data).To(ContainElement(31))
			Expect(series0Data).To(ContainElement(32))
			Expect(series0Data).ToNot(ContainElement(33))

			Expect(frame.Series[3].TimeRange.Start).To(Equal(75 * telem.SecondTS))
			series1Data := telem.UnmarshalSlice[int](frame.Series[3].Data, telem.Int64T)
			Expect(series1Data).ToNot(ContainElement(74))
			Expect(series1Data).To(ContainElement(75))

			Expect(rateDB.Close()).To(Succeed())
		})

	})
})
