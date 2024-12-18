package computron_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/computron"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Computron", Ordered, func() {
	var (
		svc *computron.Interpreter
	)
	BeforeAll(func() {
		svc = MustSucceed(computron.New())
	})
	DescribeTable("Data Types", func(
		data1 telem.Series,
		data2 telem.Series,
		expected telem.Series,
	) {
		calc := MustSucceed(svc.NewCalculation("result = data1 + data2"))
		ds1 := MustSucceed(computron.NewSeries(data1))
		ds2 := MustSucceed(computron.NewSeries(data2))
		result := MustSucceed(calc.Run(map[string]interface{}{
			"data1": ds1,
			"data2": ds2,
		}))
		Expect(result).To(Equal(expected))
	},
		Entry("Float64", telem.NewSeriesV[float64](1.0, 2.0, 3.0), telem.NewSeriesV[float64](4.0, 5.0, 6.0), telem.NewSeriesV[float64](5.0, 7.0, 9.0)),
		Entry("Float32", telem.NewSeriesV[float32](1.0, 2.0, 3.0), telem.NewSeriesV[float32](4.0, 5.0, 6.0), telem.NewSeriesV[float32](5.0, 7.0, 9.0)),
		Entry("Int64", telem.NewSeriesV[int64](1, 2, 3), telem.NewSeriesV[int64](4, 5, 6), telem.NewSeriesV[int64](5, 7, 9)),
		Entry("Int32", telem.NewSeriesV[int32](1, 2, 3), telem.NewSeriesV[int32](4, 5, 6), telem.NewSeriesV[int32](5, 7, 9)),
		Entry("Int16", telem.NewSeriesV[int16](1, 2, 3), telem.NewSeriesV[int16](4, 5, 6), telem.NewSeriesV[int16](5, 7, 9)),
		Entry("Int8", telem.NewSeriesV[int8](1, 2, 3), telem.NewSeriesV[int8](4, 5, 6), telem.NewSeriesV[int8](5, 7, 9)),
		Entry("Uint64", telem.NewSeriesV[uint64](1, 2, 3), telem.NewSeriesV[uint64](4, 5, 6), telem.NewSeriesV[uint64](5, 7, 9)),
		Entry("Uint32", telem.NewSeriesV[uint32](1, 2, 3), telem.NewSeriesV[uint32](4, 5, 6), telem.NewSeriesV[uint32](5, 7, 9)),
		Entry("Uint16", telem.NewSeriesV[uint16](1, 2, 3), telem.NewSeriesV[uint16](4, 5, 6), telem.NewSeriesV[uint16](5, 7, 9)),
		Entry("Uint8", telem.NewSeriesV[uint8](1, 2, 3), telem.NewSeriesV[uint8](4, 5, 6), telem.NewSeriesV[uint8](5, 7, 9)),
	)
})
