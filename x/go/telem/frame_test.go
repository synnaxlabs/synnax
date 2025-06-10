package telem_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type AltKey int32

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

		It("Should correctly append a key and a series to a pre-allocated frame", func() {
			fr := telem.AllocFrame[int](12)
			Expect(fr.Count()).To(Equal(0))
			fr = fr.Append(2, telem.NewSeriesV[int32](1, 2, 3))
			Expect(fr.Count()).To(Equal(1))
			Expect(fr.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
			}))
			Expect(fr.KeysSlice()).To(Equal([]int{2}))
		})
	})

	Describe("Prepend", func() {
		It("Should correctly prepend a key and a series to the frame", func() {
			fr := telem.UnaryFrame[int](1, telem.NewSeriesV[int32](1, 2, 3))
			fr = fr.Prepend(2, telem.NewSeriesV[int32](4, 5, 6))
			Expect(fr.KeysSlice()).To(Equal([]int{2, 1}))
			Expect(fr.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](4, 5, 6),
				telem.NewSeriesV[int32](1, 2, 3),
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
			for i := range 256 {
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
			Expect(series).To(Equal(telem.NewMultiSeriesV(
				telem.NewSeriesV[int32](1, 2),
				telem.NewSeriesV[int32](5, 6),
			)))
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
			Expect(series).To(Equal(telem.NewMultiSeriesV(
				telem.NewSeriesV[int32](1, 2),
				telem.NewSeriesV[int32](5, 6),
			)))
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
			count := 0
			for range fr.Series() {
				count++
				if count == 2 {
					break
				}
			}

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

	Describe("Len", func() {
		It("Should return the number of samples in the largest series in the frame", func() {
			fr := telem.MultiFrame(
				[]int{1, 2},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5),
				})
			Expect(fr.Len()).To(Equal(int64(3)))
		})
	})

	Describe("Sort", func() {
		It("Should correctly sort an unmasked frame by keys", func() {
			fr := telem.MultiFrame(
				[]int{3, 1, 2},
				[]telem.Series{
					telem.NewSeriesV[int32](7, 8, 9),
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
				})

			fr.Sort()

			Expect(fr.KeysSlice()).To(Equal([]int{1, 2, 3}))
			Expect(fr.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](4, 5, 6),
				telem.NewSeriesV[int32](7, 8, 9),
			}))
		})

		It("Should handle sorting a frame with duplicate keys", func() {
			fr := telem.MultiFrame(
				[]int{3, 1, 2, 1},
				[]telem.Series{
					telem.NewSeriesV[int32](7, 8, 9),
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](10, 11, 12),
				})

			fr.Sort()

			Expect(fr.KeysSlice()).To(Equal([]int{1, 1, 2, 3}))

			series := fr.SeriesSlice()
			Expect(series[0]).To(Equal(telem.NewSeriesV[int32](1, 2, 3)))
			Expect(series[1]).To(Equal(telem.NewSeriesV[int32](10, 11, 12)))
			Expect(series[2]).To(Equal(telem.NewSeriesV[int32](4, 5, 6)))
			Expect(series[3]).To(Equal(telem.NewSeriesV[int32](7, 8, 9)))
		})

		It("Should respect masking when sorting", func() {
			fr := telem.MultiFrame(
				[]int{3, 1, 2, 5, 4},
				[]telem.Series{
					telem.NewSeriesV[int32](7, 8, 9),
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](13, 14, 15),
					telem.NewSeriesV[int32](10, 11, 12),
				})

			filtered := fr.FilterKeys([]int{1, 3, 5})

			filtered.Sort()

			Expect(filtered.KeysSlice()).To(Equal([]int{1, 3, 5}))

			series := filtered.SeriesSlice()
			Expect(series[0]).To(Equal(telem.NewSeriesV[int32](1, 2, 3)))
			Expect(series[1]).To(Equal(telem.NewSeriesV[int32](7, 8, 9)))
			Expect(series[2]).To(Equal(telem.NewSeriesV[int32](13, 14, 15)))

			Expect(fr.KeysSlice()).To(Equal([]int{1, 2, 3, 4, 5}))
		})

		It("Should sort an empty frame without errors", func() {
			fr := telem.Frame[int]{}
			Expect(func() { fr.Sort() }).NotTo(Panic())
			Expect(fr.Empty()).To(BeTrue())
		})

		It("Should maintain consistency between keys and series after sorting", func() {
			fr := telem.MultiFrame(
				[]int{3, 1, 2},
				[]telem.Series{
					telem.NewSeriesV[int32](7, 8, 9),
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
				})

			fr.Sort()

			keys := fr.KeysSlice()
			series := fr.SeriesSlice()
			Expect(keys[0]).To(Equal(1))
			Expect(series[0]).To(Equal(telem.NewSeriesV[int32](1, 2, 3)))
			Expect(keys[1]).To(Equal(2))
			Expect(series[1]).To(Equal(telem.NewSeriesV[int32](4, 5, 6)))
			Expect(keys[2]).To(Equal(3))
			Expect(series[2]).To(Equal(telem.NewSeriesV[int32](7, 8, 9)))
		})

		It("Should correctly handle a filtered frame with more than 128 entries", func() {
			keys := make([]int, 200)
			series := make([]telem.Series, 200)
			for i := range 200 {
				keys[i] = 199 - i
				series[i] = telem.NewSeriesV[int32](int32(i), int32(i+1))
			}
			fr := telem.MultiFrame(keys, series)
			filteredKeys := []int{10, 50, 100, 150, 190}
			filtered := fr.FilterKeys(filteredKeys)
			filtered.Sort()
			sortedKeys := filtered.KeysSlice()
			Expect(sortedKeys).To(Equal([]int{10, 50, 100, 150, 190}))
			resultSeries := filtered.SeriesSlice()
			Expect(resultSeries[0]).To(Equal(telem.NewSeriesV[int32](189, 190)))
			Expect(resultSeries[1]).To(Equal(telem.NewSeriesV[int32](149, 150)))
			Expect(resultSeries[2]).To(Equal(telem.NewSeriesV[int32](99, 100)))
			Expect(resultSeries[3]).To(Equal(telem.NewSeriesV[int32](49, 50)))
			Expect(resultSeries[4]).To(Equal(telem.NewSeriesV[int32](9, 10)))
		})
	})

	Describe("SeriesAt and RawSeriesAt", func() {
		It("Should correctly get series at index", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
				})

			Expect(fr.SeriesAt(0)).To(Equal(telem.NewSeriesV[int32](1, 2, 3)))
			Expect(fr.SeriesAt(1)).To(Equal(telem.NewSeriesV[int32](4, 5, 6)))
			Expect(fr.SeriesAt(2)).To(Equal(telem.NewSeriesV[int32](7, 8, 9)))

			Expect(fr.SeriesAt(-1)).To(Equal(telem.NewSeriesV[int32](7, 8, 9)))
			Expect(fr.SeriesAt(-3)).To(Equal(telem.NewSeriesV[int32](1, 2, 3)))

			filtered := fr.FilterKeys([]int{1, 3})
			Expect(filtered.SeriesAt(0)).To(Equal(telem.NewSeriesV[int32](1, 2, 3)))
			Expect(filtered.SeriesAt(1)).To(Equal(telem.NewSeriesV[int32](7, 8, 9)))

			Expect(fr.RawSeriesAt(0)).To(Equal(telem.NewSeriesV[int32](1, 2, 3)))
			Expect(fr.RawSeriesAt(1)).To(Equal(telem.NewSeriesV[int32](4, 5, 6)))
			Expect(fr.RawSeriesAt(2)).To(Equal(telem.NewSeriesV[int32](7, 8, 9)))
		})

		It("Should panic when accessing out of bounds indices", func() {
			fr := telem.MultiFrame(
				[]int{1, 2},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
				})

			Expect(func() { fr.SeriesAt(2) }).To(Panic())
			Expect(func() { fr.SeriesAt(-3) }).To(Panic())

			filtered := fr.FilterKeys([]int{1})
			Expect(func() { filtered.SeriesAt(1) }).To(Panic())
		})
	})

	Describe("SetSeriesAt and SetRawSeriesAt", func() {
		It("Should correctly set series at index", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
				})

			newSeries := telem.NewSeriesV[int32](10, 11, 12)
			fr.SetSeriesAt(1, newSeries)
			Expect(fr.SeriesAt(1)).To(Equal(newSeries))

			anotherSeries := telem.NewSeriesV[int32](13, 14, 15)
			fr.SetSeriesAt(-1, anotherSeries)
			Expect(fr.SeriesAt(2)).To(Equal(anotherSeries))

			filtered := fr.FilterKeys([]int{1, 3})
			replacementSeries := telem.NewSeriesV[int32](16, 17, 18)
			filtered.SetSeriesAt(1, replacementSeries)
			Expect(filtered.SeriesAt(1)).To(Equal(replacementSeries))

			rawSeries := telem.NewSeriesV[int32](20, 21, 22)
			fr.SetRawSeriesAt(0, rawSeries)
			Expect(fr.RawSeriesAt(0)).To(Equal(rawSeries))
		})
	})

	Describe("RawSeries and RawKeys", func() {
		It("Should return raw unfiltered series and keys", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
				})

			filtered := fr.FilterKeys([]int{1, 3})

			Expect(filtered.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](7, 8, 9),
			}))

			Expect(filtered.RawSeries()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](4, 5, 6),
				telem.NewSeriesV[int32](7, 8, 9),
			}))

			Expect(filtered.KeysSlice()).To(Equal([]int{1, 3}))
			Expect(filtered.RawKeys()).To(Equal([]int{1, 2, 3}))
		})
	})

	Describe("UnsafeReinterpretKeysAs", func() {
		It("Should reinterpret key type correctly", func() {
			fr := telem.MultiFrame(
				[]int32{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
				})
			reinterpreted := telem.UnsafeReinterpretKeysAs[int32, AltKey](fr)
			keys := reinterpreted.KeysSlice()
			Expect(keys).To(HaveLen(3))
			Expect(keys[0]).To(Equal(AltKey(1)))
			Expect(keys[1]).To(Equal(AltKey(2)))
			Expect(keys[2]).To(Equal(AltKey(3)))
			Expect(reinterpreted.SeriesSlice()).To(Equal(fr.SeriesSlice()))
			filtered := reinterpreted.FilterKeys([]AltKey{1, 3})
			Expect(filtered.KeysSlice()).To(Equal([]AltKey{1, 3}))
			Expect(filtered.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](7, 8, 9),
			}))
		})
	})

	Describe("ShallowCopy", func() {
		It("Should create a shallow copy of an unmasked frame", func() {
			original := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
				})

			copied := original.ShallowCopy()

			Expect(copied.KeysSlice()).To(Equal(original.KeysSlice()))
			Expect(copied.SeriesSlice()).To(Equal(original.SeriesSlice()))

			copied = copied.Append(4, telem.NewSeriesV[int32](10, 11, 12))
			Expect(copied.KeysSlice()).To(HaveLen(4))
			Expect(original.KeysSlice()).To(HaveLen(3))

			newSeries := telem.NewSeriesV[int32](13, 14, 15)
			copied.SetSeriesAt(0, newSeries)
			Expect(copied.SeriesAt(0)).To(Equal(newSeries))

			Expect(original.SeriesAt(0)).NotTo(Equal(newSeries))
		})

		It("Should preserve masking in the copy", func() {
			original := telem.MultiFrame(
				[]int{1, 2, 3, 4},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
					telem.NewSeriesV[int32](10, 11, 12),
				})

			filtered := original.FilterKeys([]int{1, 3})
			copied := filtered.ShallowCopy()

			Expect(copied.KeysSlice()).To(Equal([]int{1, 3}))
			Expect(copied.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](7, 8, 9),
			}))

			furtherFiltered := copied.FilterKeys([]int{1})
			Expect(furtherFiltered.KeysSlice()).To(Equal([]int{1}))
			Expect(filtered.KeysSlice()).To(Equal([]int{1, 3}))
		})

		It("Should handle empty frames", func() {
			empty := telem.Frame[int]{}
			copied := empty.ShallowCopy()
			Expect(copied.Empty()).To(BeTrue())
		})
	})

	Describe("Extend", func() {
		It("Should extend a frame with another frame's keys and series", func() {
			frame1 := telem.MultiFrame(
				[]int{1, 2},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
				})

			frame2 := telem.MultiFrame(
				[]int{3, 4},
				[]telem.Series{
					telem.NewSeriesV[int32](7, 8, 9),
					telem.NewSeriesV[int32](10, 11, 12),
				})

			extended := frame1.Extend(frame2)

			Expect(extended.KeysSlice()).To(Equal([]int{1, 2, 3, 4}))
			Expect(extended.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](4, 5, 6),
				telem.NewSeriesV[int32](7, 8, 9),
				telem.NewSeriesV[int32](10, 11, 12),
			}))

			Expect(frame2.KeysSlice()).To(Equal([]int{3, 4}))
		})

		It("Should handle extending with an empty frame", func() {
			frame := telem.MultiFrame(
				[]int{1, 2},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
				})

			empty := telem.Frame[int]{}
			extended := frame.Extend(empty)

			Expect(extended.KeysSlice()).To(Equal([]int{1, 2}))
			Expect(extended.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](4, 5, 6),
			}))

			extendedEmpty := empty.Extend(frame)
			Expect(extendedEmpty.KeysSlice()).To(Equal([]int{1, 2}))
			Expect(extendedEmpty.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](4, 5, 6),
			}))
		})

		It("Should respect masking when extending", func() {
			frame1 := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
				})

			frame2 := telem.MultiFrame(
				[]int{4, 5, 6},
				[]telem.Series{
					telem.NewSeriesV[int32](10, 11, 12),
					telem.NewSeriesV[int32](13, 14, 15),
					telem.NewSeriesV[int32](16, 17, 18),
				})
			filtered1 := frame1.FilterKeys([]int{1, 3})
			filtered2 := frame2.FilterKeys([]int{4, 6})
			extended := filtered1.Extend(filtered2)

			Expect(extended.KeysSlice()).To(Equal([]int{1, 3, 4, 6}))
			Expect(extended.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](7, 8, 9),
				telem.NewSeriesV[int32](10, 11, 12),
				telem.NewSeriesV[int32](16, 17, 18),
			}))
		})

		It("Should handle duplicate keys when extending", func() {
			frame1 := telem.MultiFrame(
				[]int{1, 2},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
				})

			frame2 := telem.MultiFrame(
				[]int{2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](7, 8, 9),
					telem.NewSeriesV[int32](10, 11, 12),
				})

			extended := frame1.Extend(frame2)

			Expect(extended.KeysSlice()).To(Equal([]int{1, 2, 2, 3}))
			Expect(extended.SeriesSlice()).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](4, 5, 6),
				telem.NewSeriesV[int32](7, 8, 9),
				telem.NewSeriesV[int32](10, 11, 12),
			}))

			series := extended.Get(2)
			Expect(series.Series).To(HaveLen(2))
			Expect(series.Series[0]).To(Equal(telem.NewSeriesV[int32](4, 5, 6)))
			Expect(series.Series[1]).To(Equal(telem.NewSeriesV[int32](7, 8, 9)))
		})
	})

	Describe("SeriesI", func() {
		It("Should iterate over all series and indices in an unmasked frame", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
				})

			indices := make([]int, 0)
			series := make([]telem.Series, 0)

			for i, s := range fr.SeriesI() {
				indices = append(indices, i)
				series = append(series, s)
			}

			Expect(indices).To(Equal([]int{0, 1, 2}))
			Expect(series).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](4, 5, 6),
				telem.NewSeriesV[int32](7, 8, 9),
			}))
		})

		It("Should respect masking when iterating over series and indices", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3, 4, 5},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
					telem.NewSeriesV[int32](10, 11, 12),
					telem.NewSeriesV[int32](13, 14, 15),
				})

			filtered := fr.FilterKeys([]int{1, 3, 5})

			indices := make([]int, 0)
			series := make([]telem.Series, 0)

			for i, s := range filtered.SeriesI() {
				indices = append(indices, i)
				series = append(series, s)
			}

			Expect(indices).To(Equal([]int{0, 1, 2}))
			Expect(series).To(Equal([]telem.Series{
				telem.NewSeriesV[int32](1, 2, 3),
				telem.NewSeriesV[int32](7, 8, 9),
				telem.NewSeriesV[int32](13, 14, 15),
			}))
		})

		It("Should handle early exits in SeriesI iteration", func() {
			fr := telem.MultiFrame(
				[]int{1, 2, 3, 4, 5},
				[]telem.Series{
					telem.NewSeriesV[int32](1, 2, 3),
					telem.NewSeriesV[int32](4, 5, 6),
					telem.NewSeriesV[int32](7, 8, 9),
					telem.NewSeriesV[int32](10, 11, 12),
					telem.NewSeriesV[int32](13, 14, 15),
				})

			// Test early exit
			count := 0
			lastIndex := -1
			lastValue := int32(0)

			for i, s := range fr.SeriesI() {
				count++
				lastIndex = i
				lastValue = telem.ValueAt[int32](s, 0)
				if count == 3 {
					break
				}
			}

			Expect(count).To(Equal(3))
			Expect(lastIndex).To(Equal(2))
			Expect(lastValue).To(Equal(int32(7)))

			// Test early exit with masked frame
			filtered := fr.FilterKeys([]int{1, 3, 5})
			count = 0
			lastIndex = -1
			lastValue = int32(0)

			for i, s := range filtered.SeriesI() {
				count++
				lastIndex = i
				lastValue = telem.ValueAt[int32](s, 0)
				if count == 2 {
					break
				}
			}

			Expect(count).To(Equal(2))
			Expect(lastIndex).To(Equal(1))
			Expect(lastValue).To(Equal(int32(7)))
		})

		It("Should maintain correct indexing even with sparse filtering", func() {
			fr := telem.MultiFrame(
				[]int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
				[]telem.Series{
					telem.NewSeriesV[int32](1),
					telem.NewSeriesV[int32](2),
					telem.NewSeriesV[int32](3),
					telem.NewSeriesV[int32](4),
					telem.NewSeriesV[int32](5),
					telem.NewSeriesV[int32](6),
					telem.NewSeriesV[int32](7),
					telem.NewSeriesV[int32](8),
					telem.NewSeriesV[int32](9),
					telem.NewSeriesV[int32](10),
				})

			// Select sparse keys
			filtered := fr.FilterKeys([]int{10, 50, 90})

			indices := make([]int, 0)
			seriesValues := make([]int32, 0)

			for i, s := range filtered.SeriesI() {
				indices = append(indices, i)
				seriesValues = append(seriesValues, telem.ValueAt[int32](s, 0))
			}

			// Expect consecutive indices 0, 1, 2 despite filtering out many elements
			Expect(indices).To(Equal([]int{0, 1, 2}))
			Expect(seriesValues).To(Equal([]int32{1, 5, 9}))
		})

		It("Should handle empty frames", func() {
			fr := telem.Frame[int]{}

			hasEntries := false
			for range fr.SeriesI() {
				hasEntries = true
			}

			Expect(hasEntries).To(BeFalse())
		})
	})

	Describe("HasData", func() {
		It("Should return true when the frame contains at least one sample", func() {
			Expect(telem.UnaryFrame[int](1, telem.NewSecondsTSV(1, 2, 3)).HasData()).To(BeTrue())
		})

		It("Should return false when the series in the frame is empty", func() {
			Expect(telem.UnaryFrame[int](1, telem.NewSecondsTSV()).HasData()).To(BeFalse())
		})

		It("Should return false when the frame has no series", func() {
			Expect(telem.Frame[int]{}.HasData()).To(BeFalse())
		})
	})

	Describe("At", func() {
		It("Should return the correct key and series for valid indices", func() {
			fr := telem.MultiFrame(
				[]int{10, 20, 30},
				[]telem.Series{
					telem.NewSeriesV[int32](100),
					telem.NewSeriesV[int32](200),
					telem.NewSeriesV[int32](300),
				},
			)
			k, s := fr.At(0)
			Expect(k).To(Equal(10))
			Expect(s).To(Equal(telem.NewSeriesV[int32](100)))

			k, s = fr.At(2)
			Expect(k).To(Equal(30))
			Expect(s).To(Equal(telem.NewSeriesV[int32](300)))

			// Negative index
			k, s = fr.At(-1)
			Expect(k).To(Equal(30))
			Expect(s).To(Equal(telem.NewSeriesV[int32](300)))
		})

		It("Should panic for out-of-bounds indices", func() {
			fr := telem.MultiFrame(
				[]int{10, 20},
				[]telem.Series{
					telem.NewSeriesV[int32](100),
					telem.NewSeriesV[int32](200),
				},
			)
			Expect(func() { fr.At(2) }).To(Panic())
			Expect(func() { fr.At(-3) }).To(Panic())
		})
	})

	Describe("RawKeyAt", func() {
		It("Should return the correct key for valid indices", func() {
			fr := telem.MultiFrame(
				[]int{10, 20, 30},
				[]telem.Series{
					telem.NewSeriesV[int32](100),
					telem.NewSeriesV[int32](200),
					telem.NewSeriesV[int32](300),
				},
			)
			Expect(fr.RawKeyAt(0)).To(Equal(10))
			Expect(fr.RawKeyAt(2)).To(Equal(30))
		})

		It("Should panic for out-of-bounds indices", func() {
			fr := telem.MultiFrame(
				[]int{10, 20},
				[]telem.Series{
					telem.NewSeriesV[int32](100),
					telem.NewSeriesV[int32](200),
				},
			)
			Expect(func() { _ = fr.RawKeyAt(2) }).To(Panic())
		})
	})
})
