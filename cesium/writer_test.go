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
	"time"
)

func marshalTimeSpanBigEndian(values []telem.TimeSpan) []byte {
	b := make([]byte, len(values)*8)
	for i, v := range values {
		binary.BigEndian.PutUint64(b[i*8:], uint64(v))
	}
	return b
}

func marshalInt64BigEndian(values []int64) []byte {
	b := make([]byte, len(values)*8)
	for i, v := range values {
		binary.BigEndian.PutUint64(b[i*8:], uint64(v))
	}
	return b
}

func decodeInt64BigEndian(b []byte) []int64 {
	values := make([]int64, len(b)/8)
	for i := range values {
		values[i] = int64(binary.BigEndian.Uint64(b[i*8:]))
	}
	return values
}

var _ = Describe("Writer", func() {
	var db cesium.DB
	BeforeEach(func() {
		db = MustSucceed(cesium.Open(
			"./v",
			cesium.MemBacked(),
			cesium.WithLogger(logger),
		))
	})

	AfterEach(func() { Expect(db.Close()).To(Succeed()) })

	Describe("db.Write", func() {
		It("Should write the segment correctly", func() {
			ch := cesium.Channel{Rate: 1 * telem.Hz, Density: telem.Bit64}
			Expect(db.CreateChannel(&ch)).To(Succeed())
			segments := []cesium.Segment{{
				ChannelKey: ch.Key,
				Start:      telem.TimeStamp(12 * telem.Second),
				Data:       marshalInt64BigEndian([]int64{1, 2, 3, 4}),
			}}
			Expect(db.Write(segments)).To(Succeed())
			_, err := db.Read(telem.TimeStamp(10*telem.Second).SpanRange(4*telem.Second), ch.Key)
			Expect(err).To(Succeed())
		})
	})
	Describe("Write to an indexed channel", func() {
		FIt("Should read and write the segment correctly", func() {
			index := cesium.Channel{
				Rate:    1e9 * telem.Hz,
				IsIndex: true,
				Density: telem.Bit64,
			}
			Expect(db.CreateChannel(&index)).To(Succeed())
			ch := cesium.Channel{Index: index.Key, Density: telem.Bit64}
			Expect(db.CreateChannel(&ch)).To(Succeed())
			indexSegments := []cesium.Segment{{
				ChannelKey: index.Key,
				Start:      0,
				Data: marshalTimeSpanBigEndian([]telem.TimeSpan{
					1 * telem.Second,
					3 * telem.Second,
					5 * telem.Second,
					7 * telem.Second,
					9 * telem.Second,
					12 * telem.Second,
					15 * telem.Second,
					//17 * telem.Second,
					//18 * telem.Second,
				}),
			}}
			Expect(db.Write(indexSegments)).To(Succeed())
			segments := []cesium.Segment{{
				ChannelKey: ch.Key,
				Start:      0,
				Data:       marshalInt64BigEndian([]int64{2, 4, 6, 8, 10, 13, 16}),
			}}
			Expect(db.Write(segments)).To(Succeed())
			t0 := time.Now()
			segments, err := db.Read(telem.TimeRange{
				Start: telem.TimeStamp(1 * telem.Second),
				End:   telem.TimeStamp(12 * telem.Second),
			}, ch.Key)
			logrus.Info(time.Since(t0))
			logrus.Info(segments)
			Expect(err).To(Succeed())
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
