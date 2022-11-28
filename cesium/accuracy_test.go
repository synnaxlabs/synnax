package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Accuracy", Ordered, func() {
	var db cesium.DB
	BeforeAll(func() { db = openMemDB() })
	AfterAll(func() {
		Expect(db.Close()).To(Succeed())
	})
	Context("Single Channel", func() {

		Context("Rate Based", Ordered, func() {
			key := "rateTest"
			first := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
			second := []int64{13, 14, 15, 16, 17, 18, 19, 20, 21, 22}
			BeforeAll(func() {
				Expect(db.CreateChannel(cesium.Channel{Key: key, Rate: 1 * telem.Hz, DataType: telem.Int64T})).To(Succeed())
				Expect(db.WriteArray(
					10*telem.SecondTS,
					telem.NewKeyedArrayV[int64](key, first...),
				)).To(Succeed())
				Expect(db.WriteArray(
					20*telem.SecondTS,
					telem.NewKeyedArrayV[int64](key, second...),
				)).To(Succeed())
			})
			DescribeTable("Accuracy",
				func(
					tr telem.TimeRange,
					expected []int64,
				) {
					frame := MustSucceed(db.Read(tr, key))
					actual := []int64{}
					for _, arr := range frame.Arrays {
						actual = append(actual, telem.Unmarshal[int64](arr)...)
					}
					Expect(actual).To(Equal(expected))
				},
				Entry("Max Range",
					telem.TimeRangeMax,
					append(first, second...),
				),
				Entry("Empty Range",
					(12*telem.SecondTS).SpanRange(0),
					[]int64{},
				),
				Entry("Single, Even Range",
					(10*telem.SecondTS).Range(20*telem.SecondTS),
					first,
				),
				Entry("Single, Partial Range",
					(12*telem.SecondTS).SpanRange(2*telem.Second),
					[]int64{3, 4},
				),
				Entry("Multiple, Even Range",
					(10*telem.SecondTS).Range(30*telem.SecondTS),
					append(first, second...),
				),
				Entry("Multiple, Partial Range",
					(15*telem.SecondTS).Range(25*telem.SecondTS),
					[]int64{6, 7, 8, 9, 10, 13, 14, 15, 16, 17},
				),
			)
		})

		Context("Indexed", Ordered, func() {
			key := "idx1Test"
			idxKey := "idx1TestIdx"
			first := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
			second := []int64{13, 14, 15, 16, 17, 18, 19, 20, 21, 22}
			// Converted to seconds on write
			firstTS := []telem.TimeStamp{2, 4, 6, 8, 10, 12, 13, 17, 18, 20}
			secondTS := []telem.TimeStamp{22, 24, 29, 32, 33, 34, 35, 36, 38, 40}
			BeforeAll(func() {
				Expect(db.CreateChannel(
					cesium.Channel{Key: idxKey, IsIndex: true, DataType: telem.TimeStampT},
					cesium.Channel{Key: key, Index: idxKey, DataType: telem.Int64T},
				)).To(Succeed())
				Expect(db.WriteArray(
					2*telem.SecondTS,
					telem.NewKeyedSecondsTSV(idxKey, firstTS...),
				)).To(Succeed())
				Expect(db.WriteArray(
					22*telem.SecondTS,
					telem.NewKeyedSecondsTSV(idxKey, secondTS...),
				)).To(Succeed())
				Expect(db.WriteArray(
					2*telem.SecondTS,
					telem.NewKeyedArrayV[int64](key, first...),
				)).To(Succeed())
				Expect(db.WriteArray(
					22*telem.SecondTS,
					telem.NewKeyedArrayV[int64](key, second...),
				)).To(Succeed())
			})
			DescribeTable("Accuracy",
				func(
					tr telem.TimeRange,
					expected []int64,
				) {
					frame := MustSucceed(db.Read(tr, key))
					actual := []int64{}
					for _, arr := range frame.Arrays {
						actual = append(actual, telem.Unmarshal[int64](arr)...)
					}
					Expect(actual).To(Equal(expected))
				},
				Entry("Max Range",
					telem.TimeRangeMax,
					append(first, second...),
				),
				Entry("Empty Range",
					(12*telem.SecondTS).SpanRange(0),
					[]int64{},
				),
				Entry("Single, Even Range",
					(2*telem.SecondTS).Range(21*telem.SecondTS),
					first,
				),
				Entry("Single, Exclusive End",
					(2*telem.SecondTS).Range(20*telem.SecondTS),
					first[:len(first)-1],
				),
				Entry("Single, Partial Range",
					(4*telem.SecondTS).SpanRange(4*telem.Second),
					[]int64{2, 3},
				),
				Entry("Multi, End at Second Start",
					(2*telem.SecondTS).Range(22*telem.SecondTS),
					first,
				),
				Entry("Multi, End slightly above Second Start",
					(2*telem.SecondTS).Range(22*telem.SecondTS+1),
					append(first, 13),
				),
			)
		})
	})

	Context("Multi Channel", func() {
		Context("Rate Based", func() {
			Context("Shared Rate", Ordered, func() {
				key1 := "multiChRateShared1"
				key2 := "multiChRateShared2"
				data1 := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
				data2 := []int64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
				BeforeAll(func() {
					Expect(db.CreateChannel(
						cesium.Channel{Key: key1, Rate: 1 * telem.Hz, DataType: telem.Int64T},
						cesium.Channel{Key: key2, Rate: 1 * telem.Hz, DataType: telem.Int64T},
					)).To(Succeed())
					Expect(db.WriteArray(
						2*telem.SecondTS,
						telem.NewKeyedArrayV[int64](key1, data1...),
					)).To(Succeed())
					Expect(db.WriteArray(
						2*telem.SecondTS,
						telem.NewKeyedArrayV[int64](key2, data2...),
					)).To(Succeed())
				})

				DescribeTable("Accuracy",
					func(
						tr telem.TimeRange,
						expected1 []int64,
						expected2 []int64,
					) {
						frame := MustSucceed(db.Read(tr, key1, key2))
						actual1 := []int64{}
						actual2 := []int64{}
						for _, arr := range frame.Arrays {
							if arr.Key == key1 {
								actual1 = append(actual1, telem.Unmarshal[int64](arr)...)
							} else {
								actual2 = append(actual2, telem.Unmarshal[int64](arr)...)
							}
						}
						Expect(actual1).To(Equal(expected1))
						Expect(actual2).To(Equal(expected2))
					},
					Entry("Max Range",
						telem.TimeRangeMax,
						data1,
						data2,
					),
					Entry("Empty Range",
						(12*telem.SecondTS).SpanRange(0),
						[]int64{},
						[]int64{},
					),
					Entry("Single, Even Range",
						(2*telem.SecondTS).Range(12*telem.SecondTS),
						data1,
						data2,
					),
					Entry("Single, Exclusive End",
						(2*telem.SecondTS).Range(11*telem.SecondTS),
						data1[:len(data1)-1],
						data2[:len(data2)-1],
					),
					Entry("Single, Partial Range",
						(4*telem.SecondTS).SpanRange(4*telem.Second),
						[]int64{3, 4, 5, 6},
						[]int64{13, 14, 15, 16},
					),
				)
			})
		})

	})
})
