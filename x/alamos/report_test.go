package alamos_test

import (
	"encoding/json"
	"github.com/arya-analytics/x/alamos"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"io/ioutil"
)

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
		file, _ := json.Marshal(exp.Report())
		err := ioutil.WriteFile("report.json", file, 0644)
		Expect(err).To(BeNil())
	})
})
