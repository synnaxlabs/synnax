package cesium_test

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/x/telem"
	. "github.com/arya-analytics/x/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateChannel", func() {
	var (
		db cesium.DB
		ch cesium.Channel
	)
	BeforeEach(func() {
		db = MustSucceed(cesium.Open("testdata", cesium.MemBacked()))
		ch = cesium.Channel{Rate: 25 * telem.Hz, Density: telem.Bit8}
		Expect(db.CreateChannel(&ch)).To(Succeed())
	})
	AfterEach(func() { Expect(db.Close()).To(Succeed()) })
	It("Should assign an auto-incrementing ID", func() {
		Expect(ch.Key).To(Equal(cesium.ChannelKey(1)))
	})
	Specify("The channel can be retrieved after creation", func() {
		resCh := MustSucceed(db.RetrieveChannel(ch.Key))
		Expect(resCh[0].Rate).To(Equal(telem.Rate(25)))
		Expect(resCh[0].Density).To(Equal(telem.Density(1)))
	})
})
