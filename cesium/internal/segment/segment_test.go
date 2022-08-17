package segment_test

import (
	"encoding/binary"
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/segment"
	"github.com/arya-analytics/x/telem"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("segment", func() {
	Describe("Field", func() {
		It("Should generate the segments key correctly", func() {
			k := segment.NewKey(channel.Key(1), telem.TimeStampMax)
			Expect(k[0]).To(Equal(byte('s')))
			Expect(binary.BigEndian.Uint16(k[1:3])).To(Equal(uint16(1)))
			Expect(binary.BigEndian.Uint64(k[3:11])).To(Equal(uint64(telem.TimeStampMax)))
		})
		It("Should generate a key for the current timestamp correctly", func() {
			t := telem.Now()
			k := segment.NewKey(channel.Key(1), t)
			Expect(k[0]).To(Equal(byte('s')))
			Expect(binary.BigEndian.Uint16(k[1:3])).To(Equal(uint16(1)))
			Expect(binary.BigEndian.Uint64(k[3:11])).To(Equal(uint64(t)))
		})
	})
	Describe("KeyPrefix", func() {
		It("Should generate the segments key prefix correctly", func() {
			k := segment.NewKeyPrefix(channel.Key(1))
			Expect(k[0]).To(Equal(byte('s')))
			Expect(binary.BigEndian.Uint16(k[1:3])).To(Equal(uint16(1)))
		})
	})
})
