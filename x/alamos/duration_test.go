package alamos_test

import (
	"github.com/arya-analytics/x/alamos"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Duration", func() {
	Describe("Stopwatch", func() {
		It("Should start and stop a stopwatch", func() {
			exp := alamos.New("test")
			dur := alamos.NewGaugeDuration(exp, alamos.Debug, "test")
			sw := dur.Stopwatch()
			sw.Start()
			Expect(sw.Elapsed()).To(BeNumerically(">", 0))
			sw.Stop()
		})
	})
	Describe("SeriesDuration", func() {
		It("Should append stopwatch values to a slice", func() {
			exp := alamos.New("test")
			dur := alamos.NewSeriesDuration(exp, alamos.Debug, "test")
			sw := dur.Stopwatch()
			sw.Start()
			sw.Stop()
			sw2 := dur.Stopwatch()
			sw2.Start()
			sw2.Stop()
			Expect(dur.Values()).To(HaveLen(2))
		})
	})
	Describe("Empty Stopwatch", func() {
		It("Should do nothing", func() {
			dur := alamos.NewGaugeDuration(nil, alamos.Debug, "test")
			sw := dur.Stopwatch()
			sw.Start()
			sw.Stop()
			Expect(sw.Elapsed()).To(Equal(time.Duration(0)))
			dur2 := alamos.NewSeriesDuration(nil, alamos.Debug, "test")
			sw2 := dur2.Stopwatch()
			sw2.Start()
			sw2.Stop()
			Expect(dur2.Values()).To(HaveLen(0))
		})
	})

})
