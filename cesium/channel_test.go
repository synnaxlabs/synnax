package cesium_test

import (
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ChannelKey", Ordered, func() {
	var db cesium.DB
	BeforeAll(func() { db = openMemDB() })
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })
	Describe("Create", func() {
		Describe("Key Assignment", func() {
			It("Should assign an auto-incremented key if a key is not present", func() {
				ch := cesium.Channel{Rate: 10 * telem.Hz, Density: telem.Bit64}
				Expect(db.CreateChannel(&ch)).To(Succeed())
				Expect(ch.Key).To(Equal(cesium.ChannelKey(1)))
				ch.Key = 0
				Expect(db.CreateChannel(&ch)).To(Succeed())
				Expect(ch.Key).To(Equal(cesium.ChannelKey(2)))
			})
		})
		DescribeTable("Validation", func(expected error, channels ...cesium.Channel) {
			for i, ch := range channels {
				if i == len(channels)-1 {
					Expect(db.CreateChannel(&ch)).To(HaveOccurredAs(expected))
				} else {
					Expect(db.CreateChannel(&ch)).To(Succeed())
				}
			}
		},
			Entry("ChannelKey has no density",
				errors.Wrap(validate.Error, "[cesium] - density must be positive"),
				cesium.Channel{Rate: 10 * telem.Hz},
			),
			Entry("ChannelKey key already exists",
				errors.Wrap(validate.Error, "[cesium] - provided key 1 already assigned"),
				cesium.Channel{Key: 1, Rate: 10 * telem.Hz, Density: telem.Bit64},
			),
			Entry("ChannelKey IsIndex - Non Bit64 Density",
				errors.Wrap(validate.Error, "[cesium] - index channel must use int64 timestamps"),
				cesium.Channel{IsIndex: true, Density: telem.Bit32},
			),
			Entry("ChannelKey IsIndex - Index non-zero",
				errors.Wrap(validate.Error, "[cesium] - index channel can not be indexed"),
				cesium.Channel{Key: 45, IsIndex: true, Density: telem.Bit64},
				cesium.Channel{IsIndex: true, Index: 45},
			),
			Entry("ChannelKey has index - Index does not exist",
				errors.Wrapf(validate.Error, "[cesium] - provided index %s does not exist", cesium.ChannelKey(40000)),
				cesium.Channel{Index: 40000, Density: telem.Bit32},
			),
			Entry("ChannelKey has no index - fixed rate not provided",
				errors.Wrap(validate.Error, "[cesium] - rate must be positive"),
				cesium.Channel{Density: telem.Bit32},
			),
			Entry("ChannelKey has index - provided index key is not an indexed channel",
				errors.Wrapf(validate.Error, "[cesium] - provided channel %s is not an index", cesium.ChannelKey(60)),
				cesium.Channel{Key: 60, Density: telem.Bit32, Rate: 1 * telem.Hz},
				cesium.Channel{Key: 61, Index: 60, Density: telem.Bit32},
			),
		)
	})
	Describe("Retrieve", func() {
		Context("RetrieveChannel", func() {
			It("Should retrieve a channel by its key", func() {
				ch := cesium.Channel{Rate: 10 * telem.Hz, Density: telem.Bit64}
				Expect(db.CreateChannel(&ch)).To(Succeed())
				Expect(db.RetrieveChannel(ch.Key)).To(Equal(ch))
			})
		})
		Context("RetrieveChannels", func() {
			It("Should retrieve multiple channels by their key", func() {
				channels := []cesium.Channel{
					{Rate: 10 * telem.Hz, Density: telem.Bit64},
					{Rate: 9 * telem.Hz, Density: telem.Bit64},
				}
				for i := range channels {
					Expect(db.CreateChannel(&channels[i])).To(Succeed())
				}
				Expect(db.RetrieveChannels(channels[0].Key, channels[1].Key)).To(Equal(channels))
			})
		})
	})
})
