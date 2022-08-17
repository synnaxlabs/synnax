package alamos_test

import (
	"github.com/arya-analytics/x/alamos"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metric", func() {
	var (
		exp alamos.Experiment
	)
	BeforeEach(func() {
		exp = alamos.New("test")
	})
	Describe("Series", func() {
		It("Should create a Series defaultBaseMetric", func() {
			Expect(func() {
				alamos.NewSeries[int8](exp, alamos.Debug, "test.series")
			}).ToNot(Panic())
		})
		It("Should show up in the list of measurements", func() {
			alamos.NewSeries[int8](exp, alamos.Debug, "test.series")
			m := alamos.RetrieveMetric[int8](exp, "test.series")
			Expect(m.Key()).To(Equal("test.series"))
		})
		It("Should record values to the series", func() {
			series := alamos.NewSeries[float64](exp, alamos.Debug, "test.series")
			series.Record(1.0)
			Expect(series.Values()).To(Equal([]float64{1}))
		})
	})
	Describe("Gauge", func() {
		It("Should create a Gauge defaultBaseMetric", func() {
			Expect(func() { alamos.NewGauge[int8](exp, alamos.Debug, "test.gauge") }).ToNot(Panic())
		})
		It("Should set the value on teh gauge", func() {
			gauge := alamos.NewGauge[float64](exp, alamos.Debug, "test.gauge")
			gauge.Record(1)
			Expect(gauge.Values()[0]).To(Equal(1.0))
		})
	})
})
