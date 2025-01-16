package computron_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/computron"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Computron Operations", Ordered, func() {
	var (
		svc *computron.Interpreter
	)
	BeforeAll(func() {
		svc = MustSucceed(computron.New())
	})

	// Basic arithmetic operations
	DescribeTable("Addition",
		func(data1 telem.Series, data2 telem.Series, expected telem.Series) {
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

	DescribeTable("Multiplication",
		func(data1 telem.Series, data2 telem.Series, expected telem.Series) {
			calc := MustSucceed(svc.NewCalculation("result = data1 * data2"))
			ds1 := MustSucceed(computron.NewSeries(data1))
			ds2 := MustSucceed(computron.NewSeries(data2))
			result := MustSucceed(calc.Run(map[string]interface{}{
				"data1": ds1,
				"data2": ds2,
			}))
			Expect(result).To(Equal(expected))
		},
		Entry("Float64", telem.NewSeriesV[float64](1.0, 2.0, 3.0), telem.NewSeriesV[float64](4.0, 5.0, 6.0), telem.NewSeriesV[float64](4.0, 10.0, 18.0)),
		Entry("Float32", telem.NewSeriesV[float32](1.0, 2.0, 3.0), telem.NewSeriesV[float32](4.0, 5.0, 6.0), telem.NewSeriesV[float32](4.0, 10.0, 18.0)),
		Entry("Int64", telem.NewSeriesV[int64](1, 2, 3), telem.NewSeriesV[int64](4, 5, 6), telem.NewSeriesV[int64](4, 10, 18)),
		Entry("Int32", telem.NewSeriesV[int32](1, 2, 3), telem.NewSeriesV[int32](4, 5, 6), telem.NewSeriesV[int32](4, 10, 18)),
		Entry("Int16", telem.NewSeriesV[int16](1, 2, 3), telem.NewSeriesV[int16](4, 5, 6), telem.NewSeriesV[int16](4, 10, 18)),
		Entry("Int8", telem.NewSeriesV[int8](1, 2, 3), telem.NewSeriesV[int8](4, 5, 6), telem.NewSeriesV[int8](4, 10, 18)),
		Entry("Uint64", telem.NewSeriesV[uint64](1, 2, 3), telem.NewSeriesV[uint64](4, 5, 6), telem.NewSeriesV[uint64](4, 10, 18)),
		Entry("Uint32", telem.NewSeriesV[uint32](1, 2, 3), telem.NewSeriesV[uint32](4, 5, 6), telem.NewSeriesV[uint32](4, 10, 18)),
		Entry("Uint16", telem.NewSeriesV[uint16](1, 2, 3), telem.NewSeriesV[uint16](4, 5, 6), telem.NewSeriesV[uint16](4, 10, 18)),
		Entry("Uint8", telem.NewSeriesV[uint8](1, 2, 3), telem.NewSeriesV[uint8](4, 5, 6), telem.NewSeriesV[uint8](4, 10, 18)),
	)

	DescribeTable("NumPy Statistical",
		func(data telem.Series, operation string, expected telem.Series) {
			calc := MustSucceed(svc.NewCalculation(operation))
			ds := MustSucceed(computron.NewSeries(data))
			result := MustSucceed(calc.Run(map[string]interface{}{
				"data": ds,
			}))
			Expect(result).To(Equal(expected))
		},
		Entry("Mean",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0, 4.0, 5.0),
			`result = np.array([np.mean(data)])`,
			telem.NewSeriesV[float64](3.0)),
		Entry("Median",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0, 4.0, 5.0),
			`result = np.array([np.median(data)])`,
			telem.NewSeriesV[float64](3.0)),
		Entry("Std",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0, 4.0, 5.0),
			`result = np.array([np.std(data)])`,
			telem.NewSeriesV[float64](1.4142135623730951)), // sqrt(2)
		Entry("Var",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0, 4.0, 5.0),
			`result = np.array([np.var(data)])`,
			telem.NewSeriesV[float64](2.0)),
	)

	// NumPy Min/Max Operations
	DescribeTable("NumPy MinMax",
		func(data telem.Series, operation string, expected telem.Series) {
			calc := MustSucceed(svc.NewCalculation(operation))
			ds := MustSucceed(computron.NewSeries(data))
			result := MustSucceed(calc.Run(map[string]interface{}{
				"data": ds,
			}))
			Expect(result).To(Equal(expected))
		},
		Entry("Min",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0, 4.0, 5.0),
			`result = np.array([np.min(data)])`,
			telem.NewSeriesV[float64](1.0)),
		Entry("Max",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0, 4.0, 5.0),
			`result = np.array([np.max(data)])`,
			telem.NewSeriesV[float64](5.0)),
		Entry("Ptp", // Peak to peak (max - min)
			telem.NewSeriesV[float64](1.0, 2.0, 3.0, 4.0, 5.0),
			`result = np.array([np.ptp(data)])`,
			telem.NewSeriesV[float64](4.0)),
	)

	// NumPy Element-wise Operations
	DescribeTable("NumPy Element-wise",
		func(data telem.Series, operation string, expected telem.Series) {
			calc := MustSucceed(svc.NewCalculation(operation))
			ds := MustSucceed(computron.NewSeries(data))
			result := MustSucceed(calc.Run(map[string]interface{}{
				"data": ds,
			}))
			Expect(result).To(Equal(expected))
		},
		Entry("Square",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			`result = np.square(data)`,
			telem.NewSeriesV[float64](1.0, 4.0, 9.0)),
		Entry("Sqrt",
			telem.NewSeriesV[float64](1.0, 4.0, 9.0),
			`result = np.sqrt(data)`,
			telem.NewSeriesV[float64](1.0, 2.0, 3.0)),
		Entry("Abs",
			telem.NewSeriesV[float64](-1.0, -2.0, 3.0),
			`result = np.abs(data)`,
			telem.NewSeriesV[float64](1.0, 2.0, 3.0)),
		Entry("Negative",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			`result = np.negative(data)`,
			telem.NewSeriesV[float64](-1.0, -2.0, -3.0)),
		Entry("Exp",
			telem.NewSeriesV[float64](0.0, 1.0),
			`result = np.exp(data)`,
			telem.NewSeriesV[float64](1.0, 2.718281828459045)),
		Entry("Log",
			telem.NewSeriesV[float64](1.0, 2.718281828459045),
			`result = np.log(data)`,
			telem.NewSeriesV[float64](0.0, 1.0)),
	)

	// NumPy Array Operations
	DescribeTable("NumPy Array",
		func(data1 telem.Series, data2 telem.Series, operation string, expected telem.Series) {
			calc := MustSucceed(svc.NewCalculation(operation))
			ds1 := MustSucceed(computron.NewSeries(data1))
			ds2 := MustSucceed(computron.NewSeries(data2))
			result := MustSucceed(calc.Run(map[string]interface{}{
				"data1": ds1,
				"data2": ds2,
			}))
			Expect(result).To(Equal(expected))
		},
		Entry("Add",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			telem.NewSeriesV[float64](4.0, 5.0, 6.0),
			`result = np.add(data1, data2)`,
			telem.NewSeriesV[float64](5.0, 7.0, 9.0)),
		Entry("Subtract",
			telem.NewSeriesV[float64](4.0, 5.0, 6.0),
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			`result = np.subtract(data1, data2)`,
			telem.NewSeriesV[float64](3.0, 3.0, 3.0)),
		Entry("Multiply",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			telem.NewSeriesV[float64](4.0, 5.0, 6.0),
			`result = np.multiply(data1, data2)`,
			telem.NewSeriesV[float64](4.0, 10.0, 18.0)),
		Entry("Divide",
			telem.NewSeriesV[float64](4.0, 10.0, 18.0),
			telem.NewSeriesV[float64](2.0, 5.0, 6.0),
			`result = np.divide(data1, data2)`,
			telem.NewSeriesV[float64](2.0, 2.0, 3.0)),
		Entry("Power",
			telem.NewSeriesV[float64](2.0, 3.0, 4.0),
			telem.NewSeriesV[float64](2.0, 2.0, 2.0),
			`result = np.power(data1, data2)`,
			telem.NewSeriesV[float64](4.0, 9.0, 16.0)),
	)

	// Rounding Operations
	DescribeTable("NumPy Rounding",
		func(data telem.Series, operation string, expected telem.Series) {
			calc := MustSucceed(svc.NewCalculation(operation))
			ds := MustSucceed(computron.NewSeries(data))
			result := MustSucceed(calc.Run(map[string]interface{}{
				"data": ds,
			}))
			Expect(result).To(Equal(expected))
		},
		Entry("Round",
			telem.NewSeriesV[float64](1.4, 1.5, 1.6),
			`result = np.round(data)`,
			telem.NewSeriesV[float64](1.0, 2.0, 2.0)),
		Entry("Floor",
			telem.NewSeriesV[float64](1.4, 1.5, 1.6),
			`result = np.floor(data)`,
			telem.NewSeriesV[float64](1.0, 1.0, 1.0)),
		Entry("Ceil",
			telem.NewSeriesV[float64](1.4, 1.5, 1.6),
			`result = np.ceil(data)`,
			telem.NewSeriesV[float64](2.0, 2.0, 2.0)),
	)

	// Advanced Statistical Operations
	DescribeTable("NumPy Advanced Statistical",
		func(data telem.Series, operation string, expected telem.Series) {
			calc := MustSucceed(svc.NewCalculation(operation))
			ds := MustSucceed(computron.NewSeries(data))
			result := MustSucceed(calc.Run(map[string]interface{}{
				"data": ds,
			}))
			Expect(result).To(Equal(expected))
		},
		Entry("Percentile 50",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0, 4.0, 5.0),
			`result = np.array([np.percentile(data, 50)])`,
			telem.NewSeriesV[float64](3.0)),
		Entry("Percentile 75",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0, 4.0, 5.0),
			`result = np.array([np.percentile(data, 75)])`,
			telem.NewSeriesV[float64](4.0)),
	)

	// Cumulative Operations
	DescribeTable("NumPy Cumulative",
		func(data telem.Series, operation string, expected telem.Series) {
			calc := MustSucceed(svc.NewCalculation(operation))
			ds := MustSucceed(computron.NewSeries(data))
			result := MustSucceed(calc.Run(map[string]interface{}{
				"data": ds,
			}))
			Expect(result).To(Equal(expected))
		},
		Entry("Cumsum",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			`result = np.cumsum(data)`,
			telem.NewSeriesV[float64](1.0, 3.0, 6.0)),
		Entry("Cumprod",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			`result = np.cumprod(data)`,
			telem.NewSeriesV[float64](1.0, 2.0, 6.0)),
		Entry("Diff",
			telem.NewSeriesV[float64](1.0, 2.0, 4.0, 7.0),
			`result = np.diff(data)`,
			telem.NewSeriesV[float64](1.0, 2.0, 3.0)),
	)

	DescribeTable("NumPy Logical",
		func(data1 telem.Series, data2 telem.Series, operation string, expected telem.Series) {
			calc := MustSucceed(svc.NewCalculation(operation))
			ds1 := MustSucceed(computron.NewSeries(data1))
			ds2 := MustSucceed(computron.NewSeries(data2))
			result := MustSucceed(calc.Run(map[string]interface{}{
				"data1": ds1,
				"data2": ds2,
			}))
			Expect(result).To(Equal(expected))
		},
		Entry("Greater",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			telem.NewSeriesV[float64](2.0, 2.0, 2.0),
			`result = np.greater(data1, data2).astype(np.uint8)`,
			telem.NewSeriesV[uint8](0, 0, 1)),
		Entry("Less",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			telem.NewSeriesV[float64](2.0, 2.0, 2.0),
			`result = np.less(data1, data2).astype(np.uint8)`,
			telem.NewSeriesV[uint8](1, 0, 0)),
		Entry("Equal",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			telem.NewSeriesV[float64](2.0, 2.0, 2.0),
			`result = np.equal(data1, data2).astype(np.uint8)`,
			telem.NewSeriesV[uint8](0, 1, 0)),
	)
})
