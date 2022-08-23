package cesium_test

import (
	"github.com/arya-analytics/cesium"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateChannel", func() {
	var db cesium.DB
	BeforeEach(func() {
		var err error
		db, err = cesium.Open("testdata", cesium.MemBacked())
		Expect(err).ToNot(HaveOccurred())
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})
	It("Should create the channel correctly", func() {
		c := cesium.Channel{
			Rate:    25 * cesium.Hz,
			Density: cesium.Bit8,
		}
		key, err := db.CreateChannel(c)
		Expect(key).To(Equal(cesium.ChannelKey(1)))
		Expect(err).ToNot(HaveOccurred())
		Expect(c.Rate).To(Equal(cesium.Rate(25)))
		Expect(c.Density).To(Equal(cesium.Density(1)))
	})
	Specify("The channel can be retrieved after creation", func() {
		c := cesium.Channel{
			Rate:    25 * cesium.Hz,
			Density: cesium.Bit8,
		}
		key, err := db.CreateChannel(c)
		Expect(key).To(Equal(cesium.ChannelKey(1)))
		Expect(err).ToNot(HaveOccurred())
		Expect(c.Rate).To(Equal(cesium.Rate(25)))
		Expect(c.Density).To(Equal(cesium.Density(1)))
		resC, err := db.RetrieveChannel(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(resC[0].Rate).To(Equal(cesium.Rate(25)))
		Expect(resC[0].Density).To(Equal(cesium.Density(1)))
	})
})
