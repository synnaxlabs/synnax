package index_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Rate", func() {
	Describe("Distance", func() {
		It("Should return the distance between two timestamps", func() {
			idx := index.NewRate(1)
			dist := MustSucceed(idx.Distance((0 * telem.SecondTS).SpanRange(8*telem.Second), true))
			Expect(dist).To(Equal(int64(7)))
		})
	})
	Describe("Stamp", func() {
		It("Should return the timestamp at the given distance", func() {
			idx := index.NewRate(1)
			ts := MustSucceed(idx.Stamp(0*telem.SecondTS, 7))
			Expect(ts).To(Equal(7 * telem.SecondTS))
		})
	})
})
