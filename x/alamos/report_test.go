package alamos_test

import (
	"bytes"
	"github.com/arya-analytics/x/alamos"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type myReporter struct {
}

func (m myReporter) Report() alamos.Report {
	return alamos.Report{
		"key": "value",
	}
}

var _ = Describe("Report", func() {
	It("Should write the experiment to JSON", func() {
		exp := alamos.New("exp")
		g := alamos.NewGauge[int](exp, alamos.Debug, "gauge")
		g.Record(1)
		g2 := alamos.NewGauge[int](exp, alamos.Debug, "gauge2")
		g2.Record(2)
		sub := alamos.Sub(exp, "sub")
		g3 := alamos.NewSeries[float64](sub, alamos.Debug, "gauge3")
		g3.Record(3.2)
		_ = alamos.NewSeries[float64](nil, alamos.Debug, "gauge4")
		w := bytes.NewBuffer([]byte{})
		err := exp.Report().WriteJSON(w)
		Expect(err).To(BeNil())
		Expect(w.String()).To(ContainSubstring("gauge"))
		Expect(exp.Report().String()).To(ContainSubstring("gauge"))
	})
	It("Should attach reporters to an experiment", func() {
		exp := alamos.New("exp")
		alamos.AttachReporter(exp, "reporter", alamos.Debug, myReporter{})
		Expect(exp.Report().String()).To(ContainSubstring("key"))
	})
})
