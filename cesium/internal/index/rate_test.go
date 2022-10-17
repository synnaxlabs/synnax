package index_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Rate", func() {
	Describe("SeekP", func() {
		It("Should return a certain approximation", func() {
			rate := index.RateSearcher(10)
			Expect(rate.SearchP(telem.TimeStamp(0*telem.Second), position.Uncertain)).To(Equal(position.ExactlyAt(0)))
			Expect(rate.SearchP(telem.TimeStamp(10*telem.Second), position.Uncertain)).To(Equal(position.ExactlyAt(100)))
			Expect(rate.SearchP(telem.TimeStamp(20*telem.Second), position.Uncertain)).To(Equal(position.ExactlyAt(200)))
		})
	})
	Describe("SearchTS", func() {
		It("Should return a certain approximation", func() {
			rate := index.RateSearcher(10)
			Expect(rate.SearchTS(0, telem.Uncertain)).To(Equal(telem.CertainlyAt(telem.TimeStamp(0 * telem.Second))))
			Expect(rate.SearchTS(100, telem.Uncertain)).To(Equal(telem.CertainlyAt(telem.TimeStamp(10 * telem.Second))))
			Expect(rate.SearchTS(200, telem.Uncertain)).To(Equal(telem.CertainlyAt(telem.TimeStamp(20 * telem.Second))))
		})
	})
})
