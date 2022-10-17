package cesium_test

import (
	"encoding/binary"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"sort"
)

func marshalInt64BigEndian(values []int64) []byte {
	b := make([]byte, len(values)*8)
	for i, v := range values {
		binary.BigEndian.PutUint64(b[i*8:], uint64(v))
	}
	return b
}

var _ = Describe("Writer", func() {
	var db cesium.DB
	BeforeEach(func() {
		db = MustSucceed(cesium.Open("", cesium.MemBacked(), cesium.WithLogger(logger)))
	})

	AfterEach(func() { Expect(db.Close()).To(Succeed()) })

	Describe("db.Write", func() {
		It("Should write the segment correctly", func() {
			ch := cesium.Channel{Rate: 1 * telem.Hz, Density: telem.Bit64}
			Expect(db.CreateChannel(&ch)).To(Succeed())
			segments := []cesium.Segment{{
				ChannelKey: ch.Key,
				Start:      12,
				Data:       []byte{1, 2, 3, 4, 5, 6, 7, 8},
			}}
			Expect(db.Write(segments)).To(Succeed())
			ExpectSegmentsEqual(MustSucceed(db.Read(telem.TimeRangeMax, ch.Key)), segments)
		})
	})
	Describe("Write to an indexed channel", func() {
		It("Should read and write the segment correctly", func() {
			index := cesium.Channel{Rate: 1e9 * telem.Hz, Density: telem.Bit64}
			Expect(db.CreateChannel(&index)).To(Succeed())
			ch := cesium.Channel{Index: index.Key, Density: telem.Bit64}
			Expect(db.CreateChannel(&ch)).To(Succeed())
			indexSegments := []cesium.Segment{{
				ChannelKey: index.Key,
				Start:      0,
				Data:       marshalInt64BigEndian([]int64{1, 3, 5, 7, 9, 12, 15}),
			}}
			Expect(db.Write(indexSegments)).To(Succeed())
			segments := []cesium.Segment{{
				ChannelKey: ch.Key,
				Start:      0,
				Data:       marshalInt64BigEndian([]int64{2, 4, 6, 8, 10, 13, 16}),
			}}
			Expect(db.Write(segments)).To(Succeed())
			segs, err := db.Read(telem.TimeRange{Start: 7, End: 15}, ch.Key)
			logrus.Info(segs, err)
		})
	})
	Describe("Writer", func() {
		DescribeTable("Basic Writes", func(channels []cesium.Channel, requests [][]cesium.Segment) {
			for i := range channels {
				Expect(db.CreateChannel(&channels[i])).To(Succeed())
			}
			keys := lo.Map(
				channels,
				func(ch cesium.Channel, _ int) cesium.ChannelKey { return ch.Key },
			)
			w := MustSucceed(db.NewWriter(keys...))
			for _, req := range requests {
				w.Write(req)
			}
			Expect(w.Close()).To(Succeed())
			res := MustSucceed(db.Read(telem.TimeRangeMax, keys...))
			ExpectSegmentsEqual(res, lo.Flatten(requests))
		},
			Entry("Single channel, single segment, single request",
				[]cesium.Channel{{Rate: 1 * telem.Hz, Density: telem.Bit64}},
				[][]cesium.Segment{{
					{ChannelKey: 1, Start: 12, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
				}},
			),
			Entry("Single channel, multi segment, single request",
				[]cesium.Channel{{Rate: 1 * telem.Hz, Density: telem.Bit64}},
				[][]cesium.Segment{{
					{ChannelKey: 1, Start: 12, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
					{ChannelKey: 1, Start: 13, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
				}},
			),
			Entry("Single channel, multi segment, multi request",
				[]cesium.Channel{{Rate: 1 * telem.Hz, Density: telem.Bit64}},
				[][]cesium.Segment{
					{{ChannelKey: 1, Start: 12, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}}},
					{{ChannelKey: 1, Start: 13, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}}},
				},
			),
			Entry("Multi channel, multi segment, multi request",
				[]cesium.Channel{
					{Rate: 1 * telem.Hz, Density: telem.Bit64},
					{Rate: 1 * telem.Hz, Density: telem.Bit64},
				},
				[][]cesium.Segment{
					{
						{ChannelKey: 1, Start: 12, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
						{ChannelKey: 2, Start: 12, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
					},
					{
						{ChannelKey: 1, Start: 13, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
						{ChannelKey: 2, Start: 13, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
					},
				},
			),
		)
	})

	Describe("Write Lock Violation", func() {
		It("Should return an err when another query has a write lock on the channel", func() {
			By("Creating a new channel")
			ch := cesium.Channel{
				Rate:    1 * telem.Hz,
				Density: telem.Bit64,
			}
			Expect(db.CreateChannel(&ch)).To(Succeed())

			By("Opening the first query")
			w := MustSucceed(db.NewWriter(ch.Key))

			By("Failing to open the second query")
			_, err := db.NewWriter(ch.Key)
			Expect(err).To(HaveOccurredAs(cesium.ErrChannelLocked))
			Expect(w.Close()).To(Succeed())
		})
	})

})

func ExpectSegmentsEqual(value []cesium.Segment, expected []cesium.Segment) {
	Expect(prepareSegments(value)).To(Equal(prepareSegments(expected)))
}

func prepareSegments(segments []cesium.Segment) map[cesium.ChannelKey][]cesium.Segment {
	m := make(map[cesium.ChannelKey][]cesium.Segment)
	for _, seg := range segments {
		m[seg.ChannelKey] = append(m[seg.ChannelKey], seg)
	}
	for _, _segments := range m {
		sort.Slice(_segments, func(i, j int) bool { return _segments[i].Start.Before(_segments[j].Start) })
	}
	return m
}
