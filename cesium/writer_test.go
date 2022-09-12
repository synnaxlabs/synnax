package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"sort"
)

var _ = Describe("Writer", func() {
	var db cesium.DB
	BeforeEach(func() {
		db = MustSucceed(cesium.Open("", cesium.MemBacked(), cesium.WithLogger(logger)))
	})

	AfterEach(func() { Expect(db.Close()).To(Succeed()) })

	Describe("DB.Write", func() {
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
		It("Should return an error when another query has a write lock on the channel", func() {
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
