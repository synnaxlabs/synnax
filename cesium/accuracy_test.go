package cesium_test

import (
	"sort"

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

func expectSeg(
	seg cesium.Segment,
	key cesium.ChannelKey,
	start telem.TimeStamp,
	data []byte,
) {
	Expect(seg.ChannelKey).To(Equal(key))
	Expect(seg.Start).To(Equal(start))
	Expect(seg.Data).To(Equal(data))
}

func sortSegs(segments []cesium.Segment) {
	sort.Slice(segments, func(i, j int) bool {
		if segments[i].ChannelKey != segments[j].ChannelKey {
			return segments[i].ChannelKey < segments[j].ChannelKey
		}
		return segments[i].Start < segments[j].Start
	})
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
					w.Commit()
					Expect(w.Close()).To(Succeed())
					key = cesium.Keys(chs)[0]
				})

				Specify("Even Range", func() {
					segments := MustSucceed(db.Read(
						(10 * telem.SecondTS).SpanRange(10*telem.Second),
						key,
					))
					Expect(segments).To(HaveLen(2))
					sortSegs(segments)
					expectSeg(
						segments[0],
						key,
						10*telem.SecondTS,
						Marshal([]int64{1, 2, 3, 4, 5}),
					)
					expectSeg(
						segments[1],
						key,
						15*telem.SecondTS,
						Marshal([]int64{6, 7, 8, 9, 10}),
					)
				})

				Specify("Max Range", func() {
					segments := MustSucceed(db.Read(telem.TimeRangeMax, key))
					Expect(segments).To(HaveLen(2))
					sortSegs(segments)
					expectSeg(
						segments[0],
						key,
						10*telem.SecondTS,
						Marshal([]int64{1, 2, 3, 4, 5}),
					)
					expectSeg(
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
					expectSeg(
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
					sortSegs(segments)
					expectSeg(
						segments[0],
						key,
						12*telem.SecondTS,
						Marshal([]int64{3, 4, 5}),
					)
					expectSeg(
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
					w.Commit()
					Expect(w.Close()).To(Succeed())
				})

				Specify("Even Range", func() {
					segments := MustSucceed(db.Read(
						(10 * telem.SecondTS).SpanRange(30*telem.Second),
						key,
					))
					Expect(segments).To(HaveLen(3))
					expectSeg(
						segments[0],
						key,
						10*telem.SecondTS,
						Marshal([]int64{1, 2, 3}),
					)
					expectSeg(
						segments[1],
						key,
						15*telem.SecondTS,
						Marshal([]int64{5, 6, 7, 8}),
					)
					expectSeg(
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
					expectSeg(
						segments[0],
						key,
						12*telem.SecondTS,
						Marshal([]int64{3}),
					)
					expectSeg(
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
					w.Commit()
					Expect(w.Close()).To(Succeed())
					chs, w2 := createWriter(db, cesium.Channel{Index: idxKey, Density: telem.Bit64})
					key = cesium.Keys(chs)[0]
					Expect(w2.Write([]cesium.Segment{{
						ChannelKey: key,
						Start:      10 * telem.SecondTS,
						Data:       Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
					}})).To(BeTrue())
					w2.Commit()
					Expect(w2.Close()).To(Succeed())
				})

				Specify("Even Range", func() {
					segments := MustSucceed(db.Read(
						(10 * telem.SecondTS).SpanRange(31*telem.Second),
						key,
					))
					Expect(segments).To(HaveLen(1))
					expectSeg(
						segments[0],
						key,
						10*telem.SecondTS,
						Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
					)
				})

				Specify("Partial Range", func() {
					segments := MustSucceed(db.Read(
						(13 * telem.SecondTS).SpanRange(10*telem.Second),
						key,
					))
					Expect(segments).To(HaveLen(1))
					expectSeg(
						segments[0],
						key,
						13*telem.SecondTS,
						Marshal([]int64{3, 4, 5, 6}),
					)
				})
			})

			Describe("Non-Contiguous", func() {
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
						{
							ChannelKey: idxKey,
							Start:      42 * telem.SecondTS,
							Data: MarshalTimeStamps([]telem.TimeStamp{
								42 * telem.SecondTS,
								43 * telem.SecondTS,
								44 * telem.SecondTS,
								45 * telem.SecondTS,
								47 * telem.SecondTS,
								48 * telem.SecondTS,
								49 * telem.SecondTS,
								50 * telem.SecondTS,
								52 * telem.SecondTS,
								53 * telem.SecondTS,
							}),
						},
					})).To(BeTrue())
					w.Commit()
					Expect(w.Close()).To(Succeed())
					chs, w2 := createWriter(db, cesium.Channel{Index: idxKey, Density: telem.Bit64})
					key = cesium.Keys(chs)[0]
					Expect(w2.Write([]cesium.Segment{
						{
							ChannelKey: key,
							Start:      10 * telem.SecondTS,
							Data:       Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
						},
						{
							ChannelKey: key,
							Start:      42 * telem.SecondTS,
							Data:       Marshal([]int64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}),
						},
					})).To(BeTrue())
					w2.Commit()
					Expect(w2.Close()).To(Succeed())
				})

				Specify("Even Range", func() {
					segments := MustSucceed(db.Read(
						(10 * telem.SecondTS).SpanRange(44*telem.Second),
						key,
					))
					Expect(segments).To(HaveLen(2))
					expectSeg(
						segments[0],
						key,
						10*telem.SecondTS,
						Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
					)
					expectSeg(
						segments[1],
						key,
						42*telem.SecondTS,
						Marshal([]int64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}),
					)
				})

				Specify("Partial Range", func() {
					segments := MustSucceed(db.Read(
						(44 * telem.SecondTS).SpanRange(5*telem.Second),
						key,
					))
					Expect(segments).To(HaveLen(1))
					expectSeg(
						segments[0],
						key,
						44*telem.SecondTS,
						Marshal([]int64{13, 14, 15, 16}),
					)
				})
			})
		})

	})
	Context("Multi Channel", func() {
		Context("Indexed", func() {
			Context("Shared StorageIndex", func() {
				var (
					key1, key2, key3, idxKey cesium.ChannelKey
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
					w.Commit()
					Expect(w.Close()).To(Succeed())
					chs, w2 := createWriter(db,
						cesium.Channel{Index: idxKey, Density: telem.Bit64},
						cesium.Channel{Index: idxKey, Density: telem.Bit64},
						cesium.Channel{Index: idxKey, Density: telem.Bit64},
					)
					key1 = cesium.Keys(chs)[0]
					key2 = cesium.Keys(chs)[1]
					key3 = cesium.Keys(chs)[2]
					Expect(w2.Write([]cesium.Segment{
						{
							ChannelKey: key1,
							Start:      10 * telem.SecondTS,
							Data:       Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
						},
					})).To(BeTrue())
					Expect(w2.Write([]cesium.Segment{
						{
							ChannelKey: key2,
							Start:      10 * telem.SecondTS,
							Data:       Marshal([]int64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}),
						},
					})).To(BeTrue())
					Expect(w2.Write([]cesium.Segment{
						{
							ChannelKey: key3,
							Start:      10 * telem.SecondTS,
							Data:       Marshal([]int64{21, 22, 23, 24, 25, 26, 27, 28, 29, 30}),
						},
					})).To(BeTrue())
					w2.Commit()
					Expect(w2.Close()).To(Succeed())
				})

				Specify("Even Range", func() {
					segments := MustSucceed(db.Read(
						(10 * telem.SecondTS).SpanRange(31*telem.Second),
						key1, key2, key3,
					))
					Expect(segments).To(HaveLen(3))
					expectSeg(
						segments[0],
						key1,
						10*telem.SecondTS,
						Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
					)
					expectSeg(
						segments[1],
						key2,
						10*telem.SecondTS,
						Marshal([]int64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}),
					)
					expectSeg(
						segments[2],
						key3,
						10*telem.SecondTS,
						Marshal([]int64{21, 22, 23, 24, 25, 26, 27, 28, 29, 30}),
					)
				})

				Specify("Partial Range", func() {
					segments := MustSucceed(db.Read(
						(12 * telem.SecondTS).SpanRange(19*telem.Second),
						key1, key2, key3,
					))
					Expect(segments).To(HaveLen(3))
					expectSeg(
						segments[0],
						key1,
						12*telem.SecondTS,
						Marshal([]int64{2, 3, 4, 5, 6, 7, 8}),
					)
					expectSeg(
						segments[1],
						key2,
						12*telem.SecondTS,
						Marshal([]int64{12, 13, 14, 15, 16, 17, 18}),
					)
					expectSeg(
						segments[2],
						key3,
						12*telem.SecondTS,
						Marshal([]int64{22, 23, 24, 25, 26, 27, 28}),
					)
				})

			})
			Context("Multi StorageIndex", func() {
				Describe("Contiguous", func() {
					var (
						key1, key2     cesium.ChannelKey
						idxOne, idxTwo cesium.ChannelKey
					)
					BeforeEach(func() {
						chs, w := createWriter(
							db,
							cesium.Channel{IsIndex: true, Density: telem.Bit64},
							cesium.Channel{IsIndex: true, Density: telem.Bit64},
						)
						idxOne = cesium.Keys(chs)[0]
						idxTwo = cesium.Keys(chs)[1]

						Expect(w.Write([]cesium.Segment{
							{
								ChannelKey: idxOne,
								Start:      10 * telem.SecondTS,
								Data: MarshalTimeStamps([]telem.TimeStamp{
									10 * telem.SecondTS,
									11 * telem.SecondTS,
									13 * telem.SecondTS,
									14 * telem.SecondTS,
									18 * telem.SecondTS,
									22 * telem.SecondTS,
									25 * telem.SecondTS,
								}),
							},
						})).To(BeTrue())
						Expect(w.Write([]cesium.Segment{
							{
								ChannelKey: idxTwo,
								Start:      10 * telem.SecondTS,
								Data: MarshalTimeStamps([]telem.TimeStamp{
									10 * telem.SecondTS,
									13 * telem.SecondTS,
									15 * telem.SecondTS,
									16 * telem.SecondTS,
									17 * telem.SecondTS,
									24 * telem.SecondTS,
								}),
							},
						})).To(BeTrue())
						w.Commit()
						Expect(w.Close()).To(Succeed())

						chs, w = createWriter(
							db,
							cesium.Channel{Index: idxOne, Density: telem.Bit64},
							cesium.Channel{Index: idxTwo, Density: telem.Bit64},
						)
						key1 = cesium.Keys(chs)[0]
						key2 = cesium.Keys(chs)[1]
						Expect(w.Write([]cesium.Segment{
							{
								ChannelKey: key1,
								Start:      10 * telem.SecondTS,
								Data:       Marshal([]int64{1, 2, 3, 4, 5, 6, 7}),
							},
						})).To(BeTrue())
						Expect(w.Write([]cesium.Segment{
							{
								ChannelKey: key2,
								Start:      10 * telem.SecondTS,
								Data:       Marshal([]int64{11, 12, 13, 14, 15, 16}),
							},
						})).To(BeTrue())
						w.Commit()
						Expect(w.Close()).To(Succeed())
					})

					Specify("Within defined range", func() {
						segments := MustSucceed(db.Read(
							(12 * telem.SecondTS).SpanRange(7*telem.Second),
							key1, key2,
						))
						Expect(segments).To(HaveLen(2))
						sortSegs(segments)
						expectSeg(
							segments[0],
							key1,
							13*telem.SecondTS,
							Marshal([]int64{3, 4, 5}),
						)
						expectSeg(
							segments[1],
							key2,
							13*telem.SecondTS,
							Marshal([]int64{12, 13, 14, 15}),
						)
					})

					Specify("Outside defined range", func() {
						segments := MustSucceed(db.Read(
							(5 * telem.SecondTS).SpanRange(25*telem.Second),
							key1, key2,
						))
						Expect(segments).To(HaveLen(2))
						sortSegs(segments)
						expectSeg(
							segments[0],
							key1,
							10*telem.SecondTS,
							Marshal([]int64{1, 2, 3, 4, 5, 6, 7}),
						)
						expectSeg(
							segments[1],
							key2,
							10*telem.SecondTS,
							Marshal([]int64{11, 12, 13, 14, 15, 16}),
						)
					})
				})
				Describe("Non-Contiguous", func() {
					var (
						idxOne, idxTwo cesium.ChannelKey
						key1, key2     cesium.ChannelKey
					)
					BeforeEach(func() {
						chs, w := createWriter(
							db,
							cesium.Channel{IsIndex: true, Density: telem.Bit64},
							cesium.Channel{IsIndex: true, Density: telem.Bit64},
						)
						idxOne = cesium.Keys(chs)[0]
						idxTwo = cesium.Keys(chs)[1]
						// Add two segments to each idx, with a gap between them
						Expect(w.Write([]cesium.Segment{
							{
								ChannelKey: idxOne,
								Start:      10 * telem.SecondTS,
								Data: MarshalTimeStamps([]telem.TimeStamp{
									10 * telem.SecondTS,
									13 * telem.SecondTS,
									15 * telem.SecondTS,
									16 * telem.SecondTS,
									17 * telem.SecondTS,
									24 * telem.SecondTS,
								}),
							},
							{
								ChannelKey: idxOne,
								Start:      25 * telem.SecondTS,
								Data: MarshalTimeStamps([]telem.TimeStamp{
									25 * telem.SecondTS,
									28 * telem.SecondTS,
									32 * telem.SecondTS,
									35 * telem.SecondTS,
									36 * telem.SecondTS,
								}),
							},
							{
								ChannelKey: idxTwo,
								Start:      10 * telem.SecondTS,
								Data: MarshalTimeStamps([]telem.TimeStamp{
									10 * telem.SecondTS,
									13 * telem.SecondTS,
									15 * telem.SecondTS,
									16 * telem.SecondTS,
									17 * telem.SecondTS,
								}),
							},
							{
								ChannelKey: idxTwo,
								Start:      20 * telem.SecondTS,
								Data: MarshalTimeStamps([]telem.TimeStamp{
									20 * telem.SecondTS,
									23 * telem.SecondTS,
									27 * telem.SecondTS,
									30 * telem.SecondTS,
									31 * telem.SecondTS,
								}),
							},
						})).To(BeTrue())
						w.Commit()
						Expect(w.Close()).To(Succeed())
						chs, w = createWriter(
							db,
							cesium.Channel{Index: idxOne, Density: telem.Bit64},
							cesium.Channel{Index: idxTwo, Density: telem.Bit64},
						)
						key1 = cesium.Keys(chs)[0]
						key2 = cesium.Keys(chs)[1]
						Expect(w.Write([]cesium.Segment{
							{
								ChannelKey: key1,
								Start:      10 * telem.SecondTS,
								Data:       Marshal([]int64{1, 2, 3, 4, 5, 6}),
							},
							{
								ChannelKey: key1,
								Start:      25 * telem.SecondTS,
								Data:       Marshal([]int64{7, 8, 9, 10, 11}),
							},
							{
								ChannelKey: key2,
								Start:      10 * telem.SecondTS,
								Data:       Marshal([]int64{1, 2, 3, 4, 5}),
							},
							{
								ChannelKey: key2,
								Start:      20 * telem.SecondTS,
								Data:       Marshal([]int64{6, 7, 8, 9, 10}),
							},
						})).To(BeTrue())
						w.Commit()
						Expect(w.Close()).To(Succeed())
					})

					Specify("Even Range", func() {
						segments := MustSucceed(db.Read(
							(5 * telem.SecondTS).SpanRange(35*telem.Second),
							key1, key2,
						))
						Expect(segments).To(HaveLen(4))
						sortSegs(segments)
						expectSeg(
							segments[0],
							key1,
							10*telem.SecondTS,
							Marshal([]int64{1, 2, 3, 4, 5, 6}),
						)
						expectSeg(
							segments[1],
							key1,
							25*telem.SecondTS,
							Marshal([]int64{7, 8, 9, 10, 11}),
						)
						expectSeg(
							segments[2],
							key2,
							10*telem.SecondTS,
							Marshal([]int64{1, 2, 3, 4, 5}),
						)
						expectSeg(
							segments[3],
							key2,
							20*telem.SecondTS,
							Marshal([]int64{6, 7, 8, 9, 10}),
						)
					})

					Specify("Partial Range", func() {
						segments := MustSucceed(db.Read(
							(15 * telem.SecondTS).SpanRange(15*telem.Second),
							key1, key2,
						))
						Expect(segments).To(HaveLen(4))
						sortSegs(segments)
						expectSeg(
							segments[0],
							key1,
							15*telem.SecondTS,
							Marshal([]int64{3, 4, 5, 6}),
						)
						expectSeg(
							segments[1],
							key1,
							25*telem.SecondTS,
							Marshal([]int64{7, 8}),
						)
						expectSeg(
							segments[2],
							key2,
							15*telem.SecondTS,
							Marshal([]int64{3, 4, 5}),
						)
						expectSeg(
							segments[3],
							key2,
							20*telem.SecondTS,
							Marshal([]int64{6, 7, 8}),
						)
					})

				})
			})
		})
		Context("Rate Based", func() {
			Context("Shared Rate", func() {
				var (
					key1, key2 cesium.ChannelKey
				)
				BeforeEach(func() {
					dr := 1 * telem.Hz
					chs, w := createWriter(
						db,
						cesium.Channel{Rate: dr, Density: telem.Bit64},
						cesium.Channel{Rate: dr, Density: telem.Bit64},
					)
					key1 = cesium.Keys(chs)[0]
					key2 = cesium.Keys(chs)[1]
					Expect(w.Write([]cesium.Segment{
						{
							ChannelKey: key1,
							Start:      10 * telem.SecondTS,
							Data:       Marshal([]int64{1, 2, 3, 4, 5, 6}),
						},
						{
							ChannelKey: key1,
							Start:      16 * telem.SecondTS,
							Data:       Marshal([]int64{7, 8, 9, 10, 11}),
						},
						{
							ChannelKey: key2,
							Start:      10 * telem.SecondTS,
							Data:       Marshal([]int64{1, 2, 3, 4, 5}),
						},
						{
							ChannelKey: key2,
							Start:      15 * telem.SecondTS,
							Data:       Marshal([]int64{6, 7, 8, 9, 10, 11}),
						},
					})).To(BeTrue())
					Expect(w.Commit()).To(BeTrue())
					Expect(w.Close()).To(Succeed())
				})
				Specify("Even Range", func() {
					segments := MustSucceed(db.Read(
						(10 * telem.SecondTS).SpanRange(11*telem.Second),
						key1, key2,
					))
					Expect(segments).To(HaveLen(4))
					sortSegs(segments)
					expectSeg(
						segments[0],
						key1,
						10*telem.SecondTS,
						Marshal([]int64{1, 2, 3, 4, 5, 6}),
					)
					expectSeg(
						segments[1],
						key1,
						16*telem.SecondTS,
						Marshal([]int64{7, 8, 9, 10, 11}),
					)
					expectSeg(
						segments[2],
						key2,
						10*telem.SecondTS,
						Marshal([]int64{1, 2, 3, 4, 5}),
					)
					expectSeg(
						segments[3],
						key2,
						15*telem.SecondTS,
						Marshal([]int64{6, 7, 8, 9, 10, 11}),
					)
				})

				Specify("Partial Range", func() {
					segments := MustSucceed(db.Read(
						(12 * telem.SecondTS).SpanRange(6*telem.Second),
						key1, key2,
					))
					Expect(segments).To(HaveLen(4))
					sortSegs(segments)
					expectSeg(
						segments[0],
						key1,
						12*telem.SecondTS,
						Marshal([]int64{3, 4, 5, 6}),
					)
					expectSeg(
						segments[1],
						key1,
						16*telem.SecondTS,
						Marshal([]int64{7, 8}),
					)
					expectSeg(
						segments[2],
						key2,
						12*telem.SecondTS,
						Marshal([]int64{3, 4, 5}),
					)
					expectSeg(
						segments[3],
						key2,
						15*telem.SecondTS,
						Marshal([]int64{6, 7, 8}),
					)
				})
			})
			Context("Multi Rate", func() {
				var (
					key1, key2 cesium.ChannelKey
				)
				BeforeEach(func() {
					dr1 := 1 * telem.Hz
					dr2 := 2 * telem.Hz
					chs, w := createWriter(
						db,
						cesium.Channel{Rate: dr1, Density: telem.Bit64},
						cesium.Channel{Rate: dr2, Density: telem.Bit64},
					)
					key1 = cesium.Keys(chs)[0]
					key2 = cesium.Keys(chs)[1]
					Expect(w.Write([]cesium.Segment{
						{
							ChannelKey: key1,
							Start:      10 * telem.SecondTS,
							Data:       Marshal([]int64{1, 2, 3, 4, 5, 6}),
						},
						{
							ChannelKey: key1,
							Start:      16 * telem.SecondTS,
							Data:       Marshal([]int64{7, 8, 9, 10, 11}),
						},
						{
							ChannelKey: key2,
							Start:      10 * telem.SecondTS,
							Data:       Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
						},
						{
							ChannelKey: key2,
							Start:      15 * telem.SecondTS,
							Data:       Marshal([]int64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21}),
						},
					})).To(BeTrue())
					Expect(w.Commit()).To(BeTrue())
					Expect(w.Close()).To(Succeed())
				})
				Specify("Even Range", func() {
					segments := MustSucceed(db.Read(
						(10 * telem.SecondTS).SpanRange(11*telem.Second),
						key1, key2,
					))
					Expect(segments).To(HaveLen(4))
					sortSegs(segments)
					expectSeg(
						segments[0],
						key1,
						10*telem.SecondTS,
						Marshal([]int64{1, 2, 3, 4, 5, 6}),
					)
					expectSeg(
						segments[1],
						key1,
						16*telem.SecondTS,
						Marshal([]int64{7, 8, 9, 10, 11}),
					)
					expectSeg(
						segments[2],
						key2,
						10*telem.SecondTS,
						Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
					)
					expectSeg(
						segments[3],
						key2,
						15*telem.SecondTS,
						Marshal([]int64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21}),
					)
				})

				Specify("Partial Range", func() {
					segments := MustSucceed(db.Read(
						(12 * telem.SecondTS).SpanRange(6*telem.Second),
						key1, key2,
					))
					Expect(segments).To(HaveLen(4))
					sortSegs(segments)
					expectSeg(
						segments[0],
						key1,
						12*telem.SecondTS,
						Marshal([]int64{3, 4, 5, 6}),
					)
					expectSeg(
						segments[1],
						key1,
						16*telem.SecondTS,
						Marshal([]int64{7, 8}),
					)
					expectSeg(
						segments[2],
						key2,
						12*telem.SecondTS,
						Marshal([]int64{5, 6, 7, 8, 9, 10}),
					)
					expectSeg(
						segments[3],
						key2,
						15*telem.SecondTS,
						Marshal([]int64{11, 12, 13, 14, 15, 16}),
					)
				})
			})
		})
		Context("Mixed", func() {
			var (
				idxOne, idxTwo                   cesium.ChannelKey
				dr1Hz, dr2Hz, idxedOne, idxedTwo cesium.ChannelKey
			)
			BeforeEach(func() {
				idxChs, w := createWriter(
					db,
					cesium.Channel{IsIndex: true, Density: telem.Bit64},
					cesium.Channel{IsIndex: true, Density: telem.Bit64},
				)
				idxOne = cesium.Keys(idxChs)[0]
				idxTwo = cesium.Keys(idxChs)[1]
				w.Write([]cesium.Segment{
					{
						ChannelKey: idxOne,
						Start:      10 * telem.SecondTS,
						Data: MarshalTimeStamps([]telem.TimeStamp{
							10 * telem.SecondTS,
							11 * telem.SecondTS,
							14 * telem.SecondTS,
							17 * telem.SecondTS,
							18 * telem.SecondTS,
							19 * telem.SecondTS,
							22 * telem.SecondTS,
							23 * telem.SecondTS,
							24 * telem.SecondTS,
							28 * telem.SecondTS,
							30 * telem.SecondTS,
						}),
					},
					{
						ChannelKey: idxTwo,
						Start:      10 * telem.SecondTS,
						Data: MarshalTimeStamps([]telem.TimeStamp{
							10 * telem.SecondTS,
							14 * telem.SecondTS,
							15 * telem.SecondTS,
							18 * telem.SecondTS,
							20 * telem.SecondTS,
							22 * telem.SecondTS,
							24 * telem.SecondTS,
							25 * telem.SecondTS,
							26 * telem.SecondTS,
							29 * telem.SecondTS,
							30 * telem.SecondTS,
						}),
					},
				})
				Expect(w.Commit()).To(BeTrue())
				Expect(w.Close()).To(Succeed())
				dr1 := 1 * telem.Hz
				dr2 := 2 * telem.Hz
				chs, w := createWriter(
					db,
					cesium.Channel{Rate: dr1, Density: telem.Bit64},
					cesium.Channel{Rate: dr2, Density: telem.Bit64},
					cesium.Channel{Index: idxOne, Density: telem.Bit64},
					cesium.Channel{Index: idxTwo, Density: telem.Bit64},
				)
				keys := cesium.Keys(chs)
				dr1Hz = keys[0]
				dr2Hz = keys[1]
				idxedOne = keys[2]
				idxedTwo = keys[3]

				Expect(w.Write([]cesium.Segment{
					{
						ChannelKey: dr1Hz,
						Start:      10 * telem.SecondTS,
						Data:       Marshal([]int64{1, 2, 3, 4, 5, 6}),
					},
					{
						ChannelKey: dr1Hz,
						Start:      16 * telem.SecondTS,
						Data:       Marshal([]int64{7, 8, 9, 10, 11}),
					},
					{
						ChannelKey: dr2Hz,
						Start:      10 * telem.SecondTS,
						Data:       Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
					},
					{
						ChannelKey: dr2Hz,
						Start:      15 * telem.SecondTS,
						Data:       Marshal([]int64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21}),
					},
					{
						ChannelKey: idxedOne,
						Start:      10 * telem.SecondTS,
						Data:       Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}),
					},
					{
						ChannelKey: idxedTwo,
						Start:      10 * telem.SecondTS,
						Data:       Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}),
					},
				})).To(BeTrue())
				Expect(w.Commit()).To(BeTrue())
				Expect(w.Close()).To(Succeed())
			})

			It("Should read the correct values between 10 and 30 seconds", func() {
				segments := MustSucceed(db.Read(
					(10 * telem.SecondTS).SpanRange(21*telem.Second),
					dr1Hz, dr2Hz, idxedOne, idxedTwo,
				))
				Expect(segments).To(HaveLen(6))
				sortSegs(segments)
				expectSeg(
					segments[0],
					dr1Hz,
					10*telem.SecondTS,
					Marshal([]int64{1, 2, 3, 4, 5, 6}),
				)
				expectSeg(
					segments[1],
					dr1Hz,
					16*telem.SecondTS,
					Marshal([]int64{7, 8, 9, 10, 11}),
				)
				expectSeg(
					segments[2],
					dr2Hz,
					10*telem.SecondTS,
					Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
				)
				expectSeg(
					segments[3],
					dr2Hz,
					15*telem.SecondTS,
					Marshal([]int64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21}),
				)
				expectSeg(
					segments[4],
					idxedOne,
					10*telem.SecondTS,
					Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}),
				)
				expectSeg(
					segments[5],
					idxedTwo,
					10*telem.SecondTS,
					Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}),
				)
			})

			It("Should read the correct values between 5 and 25 seconds ", func() {
				segments := MustSucceed(db.Read(
					(10 * telem.SecondTS).SpanRange(8*telem.Second),
					dr1Hz, dr2Hz, idxedOne, idxedTwo,
				))
				Expect(segments).To(HaveLen(6))
				sortSegs(segments)
				expectSeg(
					segments[0],
					dr1Hz,
					10*telem.SecondTS,
					Marshal([]int64{1, 2, 3, 4, 5, 6}),
				)
				expectSeg(
					segments[1],
					dr1Hz,
					16*telem.SecondTS,
					Marshal([]int64{7, 8}),
				)
				expectSeg(
					segments[2],
					dr2Hz,
					10*telem.SecondTS,
					Marshal([]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
				)
				expectSeg(
					segments[3],
					dr2Hz,
					(15 * telem.SecondTS),
					Marshal([]int64{11, 12, 13, 14, 15, 16}),
				)
				expectSeg(
					segments[4],
					idxedOne,
					10*telem.SecondTS,
					Marshal([]int64{1, 2, 3, 4}),
				)
				expectSeg(
					segments[5],
					idxedTwo,
					10*telem.SecondTS,
					Marshal([]int64{1, 2, 3}),
				)
			})
		})
	})
})
