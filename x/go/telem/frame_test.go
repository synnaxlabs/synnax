package telem_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Frame", func() {
	Describe("UnaryFrame", func() {
		It("Should construct a frame from a single key and series", func() {
			fr := telem.UnaryFrame[int](1, telem.NewSeriesV[int32](1, 2, 3))
			Expect(fr.KeysSlice()).To(Equal([]int{1}))
			Expect(fr.SeriesSlice()).To(Equal([]telem.Series{telem.NewSeriesV[int32](1, 2, 3)}))
		})
	})

	Describe("MultiFrame", func() {
		It("Should construct a frame from multiple keys and series", func() {
			fr := telem.MultiFrame(
				[]int{1, 2},
				[]telem.Series{telem.NewSeriesV[int32](1, 2, 3), telem.NewSeriesV[int32](4, 5, 6)})
			Expect(fr.KeysSlice()).To(Equal([]int{1, 2}))
			Expect(fr.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](4, 5, 6),
			}))
		})
		It("Should panic if the keys and series are not the same length", func() {
			Expect(func() {
				telem.MultiFrame([]int{1, 2}, []telem.Series{telem.NewSeriesV[int32](1, 2, 3)})
			}).To(Panic())
		})
	})

	Describe("Append", func() {
		It("Should append a key and series to the frame", func() {
			fr := telem.UnaryFrame[int](1, telem.NewSeriesV[int32](1, 2, 3))
			fr = fr.Append(2, telem.NewSeriesV[int32](4, 5, 6))
			Expect(fr.KeysSlice()).To(Equal([]int{1, 2}))
			Expect(fr.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](4, 5, 6),
			}))
		})
	})

	Describe("Empty", func() {
		It("Should return true if the frame is empty", func() {
			fr := telem.Frame[int]{}
			Expect(fr.Empty()).To(BeTrue())
		})
	})

	Describe("FilterKeys", func() {
		It("Should correctly filter keys for a frame with less than 128 entries", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
				})
			filtered := fr.FilterKeys([]int{1, 3})

			By("Filtering out the new frame")
			Expect(filtered.KeysSlice()).To(Equal([]int{1, 3}))
			Expect(filtered.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](7, 8, 9),
			}))

			By("Keeping the original frame")
			Expect(fr.KeysSlice()).To(Equal([]int{1, 2, 3}))
			Expect(fr.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](4, 5, 6),
				telem.NewSeriesV[int32](7, 8, 9),
			}))
		})

		It("Should correctly filter keys for a frame with more than 128 entries", func() {
			keys := make([]int, 256)
			series := make([]telem.Series, 256)
			for i := 0; i < 256; i++ {
				keys[i] = i
				series[i] = telem.NewSeriesV[int32](int32(i), int32(i+1), int32(i+2))
			}
			fr := telem.MultiFrame(keys, series)
			filtered := fr.FilterKeys([]int{1, 3, 5, 7, 9})
			Expect(filtered.KeysSlice()).To(Equal([]int{1, 3, 5, 7, 9}))
			Expect(filtered.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](3, 4, 5),
				telem.NewSeriesV[int32](5, 6, 7),
				telem.NewSeriesV[int32](7, 8, 9),
				telem.NewSeriesV[int32](9, 10, 11),
			}))
		})
	})

	Describe("Series", func() {
		It("Should iterate over all series in an unmasked frame", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
				})

			series := make([]telem.Series, 0)
			for s := range fr.Series() {
				series = append(series, s)
			}

			Expect(series).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](4, 5, 6),
				telem.NewSeriesV[int32](7, 8, 9),
			}))
		})

		It("Should respect masking when iterating", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
				})

			filtered := fr.FilterKeys([]int{1, 3})
			series := make([]telem.Series, 0)
			for s := range filtered.Series() {
				series = append(series, s)
			}

			Expect(series).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](7, 8, 9),
			}))
		})
	})

	Describe("Keys", func() {
		It("Should iterate over all keys in an unmasked frame", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
				})

			keys := make([]int, 0)
			for k := range fr.Keys() {
				keys = append(keys, k)
			}

			Expect(keys).To(Equal([]int{1, 2, 3}))
		})

		It("Should respect masking when iterating over keys", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
				})

			filtered := fr.FilterKeys([]int{1, 3})
			keys := make([]int, 0)
			for k := range filtered.Keys() {
				keys = append(keys, k)
			}

			Expect(keys).To(Equal([]int{1, 3}))
		})
	})

	Describe("Entries", func() {
		It("Should iterate over all key-series pairs", func() {
			fr := telem.MultiFrame(
				[]int{1, 2},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2),
					telem.NewSeriesV[int32](3, 4),
				})

			keys := make([]int, 0)
			series := make([]telem.Series, 0)

			for k, s := range fr.Entries() {
				keys = append(keys, k)
				series = append(series, s)
			}

			Expect(keys).To(Equal([]int{1, 2}))
			Expect(series).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2),
				telem.NewSeriesV[int32](3, 4),
			}))
		})

		It("Should respect masking when iterating over entries", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2),
					telem.NewSeriesV[int32](3, 4),
					telem.NewSeriesV[int32](5, 6),
				})

			filtered := fr.FilterKeys([]int{1, 3})
			keys := make([]int, 0)
			series := make([]telem.Series, 0)

			for k, s := range filtered.Entries() {
				keys = append(keys, k)
				series = append(series, s)
			}

			Expect(keys).To(Equal([]int{1, 3}))
			Expect(series).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2),
				telem.NewSeriesV[int32](5, 6),
			}))
		})
	})

	Describe("Get", func() {
		It("Should return all series matching a key", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 1}, // Note: duplicate key
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2),
					telem.NewSeriesV[int32](3, 4),
					telem.NewSeriesV[int32](5, 6),
				})

			series := fr.Get(1)
			Expect(series).To(Equal(telem.MultiSeries{Series: []telem.Series{
				telem.NewSeriesV[int32](1, 2),
				telem.NewSeriesV[int32](5, 6),
			}}))
		})

		It("Should respect masking when getting series", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 1},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2),
					telem.NewSeriesV[int32](3, 4),
					telem.NewSeriesV[int32](5, 6),
				})

			filtered := fr.FilterKeys([]int{1})
			series := filtered.Get(1)
			Expect(series).To(Equal(telem.MultiSeries{Series: []telem.Series{
				telem.NewSeriesV[int32](1, 2),
				telem.NewSeriesV[int32](5, 6),
			}}))
		})

		It("Should return empty slice for non-existent key", func() {
			fr := telem.MultiFrame(
				[]int{1, 2},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2),
					telem.NewSeriesV[int32](3, 4),
				})

			series := fr.Get(3)
			Expect(series.Series).To(BeEmpty())
		})
	})

	Describe("Vertical", func() {
		It("Should return true when all keys are unique", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2),
					telem.NewSeriesV[int32](3, 4),
					telem.NewSeriesV[int32](5, 6),
				})

			Expect(fr.Vertical()).To(BeTrue())
		})

		It("Should return false when keys are not unique", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 1},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2),
					telem.NewSeriesV[int32](3, 4),
					telem.NewSeriesV[int32](5, 6),
				})

			Expect(fr.Vertical()).To(BeFalse())
		})

		It("Should respect masking when checking verticality", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 1},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2),
					telem.NewSeriesV[int32](3, 4),
					telem.NewSeriesV[int32](5, 6),
				})

			filtered := fr.FilterKeys([]int{2})
			Expect(filtered.Vertical()).To(BeTrue())
		})
	})

	// Empty() is already tested in the existing test suite, but let's add more edge cases
	Describe("Empty", func() {
		It("Should return true for a filtered frame with no matching keys", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2),
					telem.NewSeriesV[int32](3, 4),
					telem.NewSeriesV[int32](5, 6),
				})

			filtered := fr.FilterKeys([]int{4, 5})
			Expect(filtered.Empty()).To(BeTrue())
		})

		It("Should return false for a filtered frame with matching keys", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2),
					telem.NewSeriesV[int32](3, 4),
					telem.NewSeriesV[int32](5, 6),
				})

			filtered := fr.FilterKeys([]int{1, 2})
			Expect(filtered.Empty()).To(BeFalse())
		})
	})

	Describe("Even", func() {
		It("Should return true for an empty frame", func() {
			fr := telem.Frame[int]{}
			Expect(fr.Even()).To(BeTrue())
		})

		It("Should return true when all series have the same length and time range", func() {
			tr := telem.TimeRange{Start: 0, End: 100}
			s1 := telem.NewSeriesV[int32](1, 2, 3)
			s1.TimeRange = tr
			s2 := telem.NewSeriesV[int32](4, 5, 6)
			s2.TimeRange = tr
			s3 := telem.NewSeriesV[int32](7, 8, 9)
			s3.TimeRange = tr
			fr := telem.MultiFrame([]int{1, 2, 3}, []telem.Series{s1, s2, s3})
			Expect(fr.Even()).To(BeTrue())
		})

		It("Should return false when series have different lengths", func() {
			tr := telem.TimeRange{Start: 0, End: 100}
			s1 := telem.NewSeriesV[int32](1, 2, 3)
			s1.TimeRange = tr
			s2 := telem.NewSeriesV[int32](4, 5)
			s2.TimeRange = tr
			fr := telem.MultiFrame([]int{1, 2}, []telem.Series{s1, s2})
			Expect(fr.Even()).To(BeFalse())
		})

		It("Should return false when series have different time ranges", func() {
			tr1 := telem.TimeRange{Start: 0, End: 100}
			tr2 := telem.TimeRange{Start: 0, End: 200}
			s1 := telem.NewSeriesV[int32](1, 2, 3)
			s1.TimeRange = tr1
			s2 := telem.NewSeriesV[int32](4, 5, 6)
			s2.TimeRange = tr2
			fr := telem.MultiFrame([]int{1, 2}, []telem.Series{s1, s2})
			Expect(fr.Even()).To(BeFalse())
		})

		It("Should respect masking when checking evenness", func() {
			tr := telem.TimeRange{Start: 0, End: 100}
			s1 := telem.NewSeriesV[int32](1, 2, 3)
			s1.TimeRange = tr
			s2 := telem.NewSeriesV[int32](4, 5)
			s2.TimeRange = telem.TimeRange{Start: 10, End: 100}
			s3 := telem.NewSeriesV[int32](7, 8, 9)
			s3.TimeRange = tr
			fr := telem.MultiFrame([]int{1, 2, 3}, []telem.Series{s1, s2, s3})
			filtered := fr.FilterKeys([]int{1, 3})
			Expect(filtered.Even()).To(BeTrue())
		})
	})

	Describe("Early Iterator Exits", func() {
		var fr telem.Frame[int]

		BeforeEach(func() {
			fr = telem.MultiFrame(
				[]int{1, 2, 3, 4, 5},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2),
					telem.NewSeriesV[int32](3, 4),
					telem.NewSeriesV[int32](5, 6),
					telem.NewSeriesV[int32](7, 8),
					telem.NewSeriesV[int32](9, 10),
				})
		})

		It("Should support early exit in Series iteration", func() {
			count := 0
			for range fr.Series() {
				count++
				if count == 2 {
					break
				}
			}
			Expect(count).To(Equal(2))
		})

		It("Should support early exit in Keys iteration", func() {
			count := 0
			for range fr.Keys() {
				count++
				if count == 3 {
					break
				}
			}
			Expect(count).To(Equal(3))
		})

		It("Should support early exit in Entries iteration", func() {
			keyCount := 0
			seriesSum := int32(0)
			for _, s := range fr.Entries() {
				keyCount++
				seriesSum += telem.ValueAt[int32](s, 0)
				if keyCount == 2 {
					break
				}
			}
			Expect(keyCount).To(Equal(2))
			Expect(seriesSum).To(Equal(int32(4))) // 1 + 3
		})

		It("Should support early exit with masked frame", func() {
			filtered := fr.FilterKeys([]int{1, 3, 5})
			count := 0
			for range filtered.Series() {
				count++
				if count == 2 {
					break
				}
			}
			Expect(count).To(Equal(2))
		})

		It("Should maintain consistency after early exit", func() {
			// First iteration with early exit
			count := 0
			for range fr.Series() {
				count++
				if count == 2 {
					break
				}
			}

			// Second iteration should start fresh
			series := make([]telem.Series, 0)
			for s := range fr.Series() {
				series = append(series, s)
			}
			Expect(len(series)).To(Equal(5))
		})
	})

	Describe("String", func() {
		It("Should return formatted string representation", func() {
			f := telem.MultiFrame[int](
				[]int{1, 2},
				[]telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](3, 4)},
			)
			str := f.String()
			Expect(str).To(ContainSubstring("Frame{"))
			Expect(str).To(ContainSubstring("1:"))
			Expect(str).To(ContainSubstring("2:"))
			Expect(str).To(ContainSubstring("}"))
		})

		It("Should handle empty frame", func() {
			f := telem.MultiFrame[int]([]int{}, []telem.Series{})
			Expect(f.String()).To(Equal("Frame{}"))
		})
	})

	Describe("Encode + Decode", func() {
		codecs := []binary.Codec{
			&binary.JSONCodec{},
			&binary.MsgPackCodec{},
		}
		for _, codec := range codecs {
			It("Should encode and decode a frame", func() {
				original := telem.MultiFrame[int](
					[]int{1, 2},
					[]telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](3, 4)},
				)
				buf := MustSucceed(codec.Encode(ctx, original))
				var decoded telem.Frame[int]
				Expect(codec.Decode(ctx, buf, &decoded)).To(Succeed())
				Expect(decoded).To(Equal(original))
			})
			It("Should respect masking", func() {
				original := telem.MultiFrame[int](
					[]int{1, 2},
					[]telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](3, 4)},
				).FilterKeys([]int{1})
				buf := MustSucceed(codec.Encode(ctx, original))
				var decoded telem.Frame[int]
				Expect(codec.Decode(ctx, buf, &decoded)).To(Succeed())
				Expect(decoded).To(Equal(telem.UnaryFrame[int](
					1,
					telem.NewSeriesV[int64](1, 2),
				)))
			})
		}

	})
})
