package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Validation", Ordered, func() {
	var db cesium.DB
	BeforeAll(func() { db = openMemDB() })

	Describe("Invalid Channel Key", func() {
		It("Should return a validation error", func() {
			chs, w := createWriter(db, cesium.Channel{Rate: 10 * telem.Hz, Density: telem.Bit64})
			Expect(w.Write([]cesium.Segment{{ChannelKey: chs[0].Key + 1, Start: 10 * telem.SecondTS, Data: Marshal([]float64{1, 2, 3, 4})}})).To(BeTrue())
			Expect(w.Commit()).To(BeFalse())
			Expect(w.Close()).To(MatchError(validate.Error))
		})
	})

	Describe("Rate Based Channel", func() {
		Describe("Overlapping segments", func() {
			var (
				w        cesium.Writer
				segments []cesium.Segment
			)
			BeforeEach(func() {
				var chs []cesium.Channel
				chs, w = createWriter(
					db,
					cesium.Channel{Rate: 10 * telem.Hz, Density: telem.Bit64},
				)
				key := cesium.Keys(chs)[0]
				segments = []cesium.Segment{
					{
						ChannelKey: key,
						Start:      10 * telem.SecondTS,
						Data:       Marshal([]float64{1, 2, 3, 4, 5, 7, 8, 9, 10, 12, 13, 14, 15, 16, 17, 18, 19, 20}),
					},
					{
						ChannelKey: key,
						Start:      11 * telem.SecondTS,
						Data:       Marshal([]float64{1, 2, 3, 4, 5, 7, 8, 9, 10, 12, 13, 14, 15, 16, 17, 18, 19, 20}),
					},
				}
			})
			Specify("Separate Writes", func() {
				Expect(w.Write([]cesium.Segment{segments[0]})).To(BeTrue())
				Expect(w.Write([]cesium.Segment{segments[1]})).To(BeTrue())
				Expect(w.Commit()).To(BeFalse())
				Expect(w.Close()).To(MatchError(validate.Error))
			})
			Specify("Single Add", func() {
				Expect(w.Write(segments)).To(BeTrue())
				Expect(w.Commit()).To(BeFalse())
				Expect(w.Close()).To(MatchError(validate.Error))
			})
		})
	})

	Describe("Indexed Channel", func() {
		Describe("Overlapping Segments", func() {
			It("Should return a validation error", func() {
				idxCh := cesium.Channel{IsIndex: true, Density: telem.Bit64}
				Expect(db.CreateChannel(&idxCh)).To(Succeed())
				ch := cesium.Channel{Density: telem.Bit64, Index: idxCh.Key}
				Expect(db.CreateChannel(&ch)).To(Succeed())
				w := MustSucceed(db.NewWriter(idxCh.Key, ch.Key))
				Expect(w.Write([]cesium.Segment{
					{
						ChannelKey: idxCh.Key,
						Start:      10 * telem.SecondTS,
						Data: MarshalTimeStamps([]telem.TimeStamp{
							10 * telem.SecondTS,
							11 * telem.SecondTS,
							14 * telem.SecondTS,
							15 * telem.SecondTS,
							18 * telem.SecondTS,
						}),
					},
				})).To(BeTrue())
				Expect(w.Write([]cesium.Segment{
					{
						ChannelKey: ch.Key,
						Start:      10 * telem.SecondTS,
						Data:       Marshal([]float64{1, 2, 3, 4, 5}),
					},
				})).To(BeTrue())
				Expect(w.Write([]cesium.Segment{
					{
						ChannelKey: ch.Key,
						Start:      10 * telem.SecondTS,
						Data:       Marshal([]float64{1, 2, 3, 4, 5}),
					},
				}))
				Expect(w.Commit()).To(BeFalse())
				err := w.Close()
				Expect(err).To(MatchError(validate.Error))
				Expect(err.Error()).To(ContainSubstring("overlap"))
			})

		})
		Describe("Segment does not align with index", func() {
			It("Should return a validation error", func() {
				idxCh := cesium.Channel{IsIndex: true, Density: telem.Bit64}
				Expect(db.CreateChannel(&idxCh)).To(Succeed())
				ch := cesium.Channel{Density: telem.Bit64, Index: idxCh.Key}
				Expect(db.CreateChannel(&ch)).To(Succeed())
				w := MustSucceed(db.NewWriter(idxCh.Key, ch.Key))
				Expect(w.Write([]cesium.Segment{
					{
						ChannelKey: idxCh.Key,
						Start:      10 * telem.SecondTS,
						Data: MarshalTimeStamps([]telem.TimeStamp{
							10 * telem.SecondTS,
							11 * telem.SecondTS,
							14 * telem.SecondTS,
							15 * telem.SecondTS,
							18 * telem.SecondTS,
						}),
					},
				})).To(BeTrue())
				Expect(w.Write([]cesium.Segment{
					{
						ChannelKey: ch.Key,
						Start:      11 * telem.SecondTS,
						Data:       Marshal([]float64{1, 2, 3, 4, 5}),
					},
				})).To(BeTrue())
				Expect(w.Commit()).To(BeFalse())
				Expect(w.Close()).To(MatchError(validate.Error))
			})
		})
		Describe("Segment is not the same size as index", func() {
			It("Should return a validation error", func() {
				idxCh := cesium.Channel{IsIndex: true, Density: telem.Bit64}
				Expect(db.CreateChannel(&idxCh)).To(Succeed())
				ch := cesium.Channel{Density: telem.Bit64, Index: idxCh.Key}
				Expect(db.CreateChannel(&ch)).To(Succeed())
				w := MustSucceed(db.NewWriter(idxCh.Key, ch.Key))
				Expect(w.Write([]cesium.Segment{
					{
						ChannelKey: idxCh.Key,
						Start:      10 * telem.SecondTS,
						Data: MarshalTimeStamps([]telem.TimeStamp{
							10 * telem.SecondTS,
							11 * telem.SecondTS,
							14 * telem.SecondTS,
							15 * telem.SecondTS,
							18 * telem.SecondTS,
						}),
					},
				})).To(BeTrue())
				Expect(w.Write([]cesium.Segment{
					{
						ChannelKey: ch.Key,
						Start:      10 * telem.SecondTS,
						Data:       Marshal([]float64{1, 2, 3, 4, 5, 6}),
					},
				})).To(BeTrue())
				Expect(w.Commit()).To(BeFalse())
				Expect(w.Close()).To(MatchError(validate.Error))
			})
		})
	})
})
