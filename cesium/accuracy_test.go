package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func createWriter(db cesium.DB, ch ...cesium.Channel) ([]cesium.Channel, cesium.Writer) {
	for i := range ch {
		Expect(db.CreateChannel(&ch[i])).To(Succeed())
	}
	return ch, MustSucceed(db.NewWriter(cesium.Keys(ch)...))
}

func expectSegment(
	seg cesium.Segment,
	key cesium.ChannelKey,
	start telem.TimeStamp,
	data []byte,
) {
	Expect(seg.ChannelKey).To(Equal(key))
	Expect(seg.Data).To(Equal(data))
	Expect(seg.Start).To(Equal(start))
}

var _ = Describe("Accuracy", Ordered, func() {
	var db cesium.DB
	BeforeAll(func() { db = openMemDB() })
	AfterAll(func() {
		Expect(db.Close()).To(Succeed())
	})
	Context("Single Channel", func() {

		Context("Rate Based", func() {

			Context("Contiguous", Ordered, func() {

				var key cesium.ChannelKey
				BeforeAll(func() {
					chs, w := createWriter(db, cesium.Channel{Rate: 1 * telem.Hz, Density: telem.Bit64})
					Expect(w.Write([]cesium.Segment{
						{
							ChannelKey: chs[0].Key,
							Start:      10 * telem.SecondTS,
							Data:       Marshal([]int64{1, 2, 3, 4, 5}),
						},
						{
							ChannelKey: chs[0].Key,
							Start:      15 * telem.SecondTS,
							Data:       Marshal([]int64{6, 7, 8, 9, 10}),
						},
					})).To(BeTrue())
					Expect(w.Close()).To(Succeed())
					key = cesium.Keys(chs)[0]
				})

				Specify("Even Range", func() {
					segments := MustSucceed(db.Read(
						(10 * telem.SecondTS).SpanRange(10*telem.Second),
						key,
					))
					Expect(segments).To(HaveLen(2))
					expectSegment(
						segments[0],
						key,
						10*telem.SecondTS,
						Marshal([]int64{1, 2, 3, 4, 5}),
					)
					expectSegment(
						segments[1],
						key,
						15*telem.SecondTS,
						Marshal([]int64{6, 7, 8, 9, 10}),
					)
				})

				Specify("Max Range", func() {
					segments := MustSucceed(db.Read(telem.TimeRangeMax, key))
					Expect(segments).To(HaveLen(2))
					expectSegment(
						segments[0],
						key,
						10*telem.SecondTS,
						Marshal([]int64{1, 2, 3, 4, 5}),
					)
					expectSegment(
						segments[1],
						key,
						15*telem.SecondTS,
						Marshal([]int64{6, 7, 8, 9, 10}),
					)
				})

				Specify("Empty Range", func() {
					segments := MustSucceed(db.Read(telem.TimeRange{}, key))
					Expect(segments).To(HaveLen(0))
				})

				Specify("Partial Range", func() {
					segments := MustSucceed(db.Read(
						(12 * telem.SecondTS).SpanRange(2*telem.Second),
						key,
					))
					Expect(segments).To(HaveLen(1))
					expectSegment(
						segments[0],
						key,
						12*telem.SecondTS,
						Marshal([]int64{3, 4}),
					)
				})

				Specify("Partial Range, Multi Segment", func() {
					segments := MustSucceed(db.Read(
						(12 * telem.SecondTS).SpanRange(4*telem.Second),
						key,
					))
					Expect(segments).To(HaveLen(2))
					expectSegment(
						segments[0],
						key,
						12*telem.SecondTS,
						Marshal([]int64{3, 4, 5}),
					)
					expectSegment(
						segments[1],
						key,
						15*telem.SecondTS,
						Marshal([]int64{6}),
					)
				})
			})

			Context("Non-contiguous", Ordered, func() {
				var key cesium.ChannelKey
				BeforeAll(func() {
					chs, w := createWriter(db, cesium.Channel{Rate: 1 * telem.Hz, Density: telem.Bit64})
					key = cesium.Keys(chs)[0]
					Expect(w.Write([]cesium.Segment{
						{
							ChannelKey: key,
							Start:      10 * telem.SecondTS,
							Data:       Marshal([]int64{1, 2, 3}),
						},
						{
							ChannelKey: key,
							Start:      15 * telem.SecondTS,
							Data:       Marshal([]int64{5, 6, 7, 8}),
						},
						{
							ChannelKey: key,
							Start:      25 * telem.SecondTS,
							Data:       Marshal([]int64{9, 10, 11, 12}),
						},
					})).To(BeTrue())
					Expect(w.Close()).To(Succeed())
				})

				Specify("Even Range", func() {
					segments := MustSucceed(db.Read(
						(10 * telem.SecondTS).SpanRange(30*telem.Second),
						key,
					))
					Expect(segments).To(HaveLen(3))
					expectSegment(
						segments[0],
						key,
						10*telem.SecondTS,
						Marshal([]int64{1, 2, 3}),
					)
					expectSegment(
						segments[1],
						key,
						15*telem.SecondTS,
						Marshal([]int64{5, 6, 7, 8}),
					)
					expectSegment(
						segments[2],
						key,
						25*telem.SecondTS,
						Marshal([]int64{9, 10, 11, 12}),
					)
				})

				Specify("Partial Range", func() {
					segments := MustSucceed(db.Read(
						(12 * telem.SecondTS).SpanRange(4*telem.Second),
						key,
					))
					Expect(segments).To(HaveLen(2))
					expectSegment(
						segments[0],
						key,
						12*telem.SecondTS,
						Marshal([]int64{3}),
					)
					expectSegment(
						segments[1],
						key,
						15*telem.SecondTS,
						Marshal([]int64{5}),
					)
				})

				Specify("Partial Range, Range has no data", func() {
					segments := MustSucceed(db.Read(
						(13 * telem.SecondTS).SpanRange(2*telem.Second),
						key,
					))
					Expect(segments).To(HaveLen(0))
				})
			})
		})

		Context("Indexed", func() {

			Context("Contiguous", Ordered, func() {
				var (
					key    cesium.ChannelKey
					idxKey cesium.ChannelKey
				)
				BeforeAll(func() {
					chs, w := createWriter(
						db,
						cesium.Channel{IsIndex: true, Density: telem.Bit64},
					)
					idxKey = cesium.Keys(chs)[0]
					Expect(w.Write([]cesium.Segment{
						{
							ChannelKey: idxKey,
							Start:      10 * telem.SecondTS,
							Data: MarshalTimeStamps([]telem.TimeStamp{
								10 * telem.SecondTS,
								12 * telem.SecondTS,
								13 * telem.SecondTS,
								18 * telem.SecondTS,
								19 * telem.SecondTS,
								22 * telem.SecondTS,
								23 * telem.SecondTS,
								30 * telem.SecondTS,
								35 * telem.SecondTS,
								40 * telem.SecondTS,
							}),
						},
					})).To(BeTrue())
					Expect(w.Close()).To(Succeed())
					chs, w2 := createWriter(db, cesium.Channel{Index: idxKey, Density: telem.Bit64})
					key = cesium.Keys(chs)[0]
					Expect(w2.Write([]cesium.Segment{{
						ChannelKey: key,
						Start:      10 * telem.SecondTS,
						Data:       Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
					}})).To(BeTrue())
					Expect(w2.Close()).To(Succeed())
				})

				Specify("Even Range", func() {
					segments := MustSucceed(db.Read(
						(10 * telem.SecondTS).SpanRange(31*telem.Second),
						key,
					))
					Expect(segments).To(HaveLen(1))
					expectSegment(
						segments[0],
						key,
						10*telem.SecondTS,
						Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
					)
				})
			})

		})

	})
	Context("Multi Channel", func() {
		Context("Indexed", func() {
			Context("Shared Index", func() {})
			Context("Multi Index", func() {})
		})
		Context("Rate Based", func() {
			Context("Shared Rate", func() {})
			Context("Multi Rate", func() {})
		})
		Context("Mixed", func() {})
	})
})
