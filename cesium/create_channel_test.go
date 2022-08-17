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
			DataRate: 25 * cesium.Hz,
			DataType: cesium.Int8,
		}
		key, err := db.CreateChannel(c)
		Expect(key).To(Equal(cesium.ChannelKey(1)))
		Expect(err).ToNot(HaveOccurred())
		Expect(c.DataRate).To(Equal(cesium.DataRate(25)))
		Expect(c.DataType).To(Equal(cesium.Density(1)))
	})
	Specify("The channel can be retrieved after creation", func() {
		c := cesium.Channel{
			DataRate: 25 * cesium.Hz,
			DataType: cesium.Int8,
		}
		key, err := db.CreateChannel(c)
		Expect(key).To(Equal(cesium.ChannelKey(1)))
		Expect(err).ToNot(HaveOccurred())
		Expect(c.DataRate).To(Equal(cesium.DataRate(25)))
		Expect(c.DataType).To(Equal(cesium.Density(1)))
		resC, err := db.RetrieveChannel(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(resC[0].DataRate).To(Equal(cesium.DataRate(25)))
		Expect(resC[0].DataType).To(Equal(cesium.Density(1)))
	})
})
