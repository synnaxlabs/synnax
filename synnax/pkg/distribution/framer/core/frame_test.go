package core_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Frame", func() {
	Describe("Vertical", func() {
		It("Should return true if there if there is only one-series per channel", func() {
			f := core.Frame{
				Keys:   []channel.Key{123},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2, 3)},
			}
			Expect(f.Vertical()).To(BeTrue())
		})
		It("Should return false if there is more than one-series per channel", func() {
			f := core.Frame{
				Keys:   []channel.Key{123, 123},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2, 3), telem.NewSeriesV[int64](4, 5, 6)},
			}
			Expect(f.Vertical()).To(BeFalse())
		})
		It("Should return true if the frame is empty", func() {
			f := core.Frame{
				Keys:   []channel.Key{},
				Series: []telem.Series{},
			}
			Expect(f.Vertical()).To(BeTrue())
		})
	})
	Describe("SplitByLeaseholder", func() {
		It("Should split the frame into separate frames by the channels leaseholder", func() {
			node1ch1 := channel.NewKey(1, 1)
			node1ch2 := channel.NewKey(1, 1)
			node2ch1 := channel.NewKey(2, 1)
			node2ch2 := channel.NewKey(2, 1)
			f := core.Frame{
				Keys: []channel.Key{node1ch1, node1ch2, node2ch1, node2ch2},
				Series: []telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
					telem.NewSeriesV[int64](7, 8, 9),
					telem.NewSeriesV[int64](10, 11, 12),
				},
			}
			frames := f.SplitByLeaseholder()
			Expect(frames).To(HaveLen(2))
			Expect(frames[1]).To(Equal(core.Frame{
				Keys:   []channel.Key{node1ch1, node1ch2},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2, 3), telem.NewSeriesV[int64](4, 5, 6)},
			}))
			Expect(frames[2]).To(Equal(core.Frame{
				Keys:   []channel.Key{node2ch1, node2ch2},
				Series: []telem.Series{telem.NewSeriesV[int64](7, 8, 9), telem.NewSeriesV[int64](10, 11, 12)},
			}))
		})
	})
	Describe("SplitByHost", func() {
		It("Should split a frame into a local, remote, and free frame", func() {
			localNodeCh := channel.NewKey(1, 1)
			remoteNodeCh := channel.NewKey(2, 1)
			freeNodeCh := channel.NewKey(dcore.Free, 1)
			f := core.Frame{
				Keys: []channel.Key{localNodeCh, remoteNodeCh, freeNodeCh},
				Series: []telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
					telem.NewSeriesV[int64](7, 8, 9),
				},
			}
			local, remote, free := f.SplitByHost(1)
			Expect(local).To(Equal(core.Frame{
				Keys:   []channel.Key{localNodeCh},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2, 3)},
			}))
			Expect(remote).To(Equal(core.Frame{
				Keys:   []channel.Key{remoteNodeCh},
				Series: []telem.Series{telem.NewSeriesV[int64](4, 5, 6)},
			}))
			Expect(free).To(Equal(core.Frame{
				Keys:   []channel.Key{freeNodeCh},
				Series: []telem.Series{telem.NewSeriesV[int64](7, 8, 9)},
			}))
		})
	})
	Describe("Even", func() {
		It("Should return true if all series have the same length and time range", func() {
			tr := telem.TimeRange{Start: 0, End: 10}
			s1 := telem.NewSeriesV[int64](1, 2, 3)
			s2 := telem.NewSeriesV[int64](4, 5, 6)
			s1.TimeRange = tr
			s2.TimeRange = tr
			f := core.Frame{
				Keys:   []channel.Key{1, 2},
				Series: []telem.Series{s1, s2},
			}
			Expect(f.Even()).To(BeTrue())
		})

		It("Should return false if series have different lengths", func() {
			tr := telem.TimeRange{Start: 0, End: 10}
			s1 := telem.NewSeriesV[int64](1, 2, 3)
			s2 := telem.NewSeriesV[int64](4, 5)
			s1.TimeRange = tr
			s2.TimeRange = tr
			f := core.Frame{
				Keys:   []channel.Key{1, 2},
				Series: []telem.Series{s1, s2},
			}
			Expect(f.Even()).To(BeFalse())
		})

		It("Should return false if series have different time ranges", func() {
			s1 := telem.NewSeriesV[int64](1, 2, 3)
			s2 := telem.NewSeriesV[int64](4, 5, 6)
			s1.TimeRange = telem.TimeRange{Start: 0, End: 10}
			s2.TimeRange = telem.TimeRange{Start: 5, End: 15}
			f := core.Frame{
				Keys:   []channel.Key{1, 2},
				Series: []telem.Series{s1, s2},
			}
			Expect(f.Even()).To(BeFalse())
		})

		It("Should return true for an empty frame", func() {
			f := core.Frame{
				Keys:   []channel.Key{},
				Series: []telem.Series{},
			}
			Expect(f.Even()).To(BeTrue())
		})

		It("Should return true for a frame with a single series", func() {
			s1 := telem.NewSeriesV[int64](1, 2, 3)
			s1.TimeRange = telem.TimeRange{Start: 0, End: 10}
			f := core.Frame{
				Keys:   []channel.Key{1},
				Series: []telem.Series{s1},
			}
			Expect(f.Even()).To(BeTrue())
		})
	})
	Describe("ToStorage", func() {
		It("Should convert to storage frame", func() {
			f := core.Frame{
				Keys:   []channel.Key{1, 2},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2, 3), telem.NewSeriesV[int64](4, 5, 6)},
			}
			sf := f.ToStorage()
			Expect(sf.Keys).To(Equal([]ts.ChannelKey{1, 2}))
			Expect(sf.Series).To(Equal(f.Series))
		})
	})
	Describe("FilterKeys", func() {
		It("Should filter frame to only include specified keys", func() {
			f := core.Frame{
				Keys: []channel.Key{1, 2, 3},
				Series: []telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
					telem.NewSeriesV[int64](7, 8, 9),
				},
			}
			filtered := f.FilterKeys([]channel.Key{1, 3})
			Expect(filtered.Keys).To(Equal(channel.Keys{1, 3}))
			Expect(filtered.Series).To(HaveLen(2))
			Expect(filtered.Series[0]).To(Equal(telem.NewSeriesV[int64](1, 2, 3)))
			Expect(filtered.Series[1]).To(Equal(telem.NewSeriesV[int64](7, 8, 9)))
		})
	})
	Describe("Get", func() {
		It("Should return all series for a given key", func() {
			f := core.Frame{
				Keys: []channel.Key{1, 2, 1},
				Series: []telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
					telem.NewSeriesV[int64](7, 8, 9),
				},
			}
			series := f.Get(1)
			Expect(series).To(HaveLen(2))
			Expect(series[0]).To(Equal(telem.NewSeriesV[int64](1, 2, 3)))
			Expect(series[1]).To(Equal(telem.NewSeriesV[int64](7, 8, 9)))
		})
	})
	Describe("String", func() {
		It("Should return formatted string representation", func() {
			f := core.Frame{
				Keys:   []channel.Key{1, 2},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](3, 4)},
			}
			str := f.String()
			Expect(str).To(ContainSubstring("Frame{"))
			Expect(str).To(ContainSubstring("1:"))
			Expect(str).To(ContainSubstring("2:"))
			Expect(str).To(ContainSubstring("}"))
		})

		It("Should handle empty frame", func() {
			f := core.Frame{
				Keys:   []channel.Key{},
				Series: []telem.Series{},
			}
			Expect(f.String()).To(Equal("Frame{}"))
		})
	})
	Describe("MergeFrames", func() {
		It("Should merge multiple frames into one", func() {
			f1 := core.Frame{
				Keys:   []channel.Key{1, 2},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](3, 4)},
			}
			f2 := core.Frame{
				Keys:   []channel.Key{3, 4},
				Series: []telem.Series{telem.NewSeriesV[int64](5, 6), telem.NewSeriesV[int64](7, 8)},
			}
			merged := core.MergeFrames([]core.Frame{f1, f2})
			Expect(merged.Keys).To(Equal(channel.Keys{1, 2, 3, 4}))
			Expect(merged.Series).To(HaveLen(4))
		})

		It("Should return empty frame for empty input", func() {
			merged := core.MergeFrames([]core.Frame{})
			Expect(merged.Keys).To(BeEmpty())
			Expect(merged.Series).To(BeEmpty())
		})

		It("Should return same frame for single input", func() {
			f := core.Frame{
				Keys:   []channel.Key{1, 2},
				Series: []telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](3, 4)},
			}
			merged := core.MergeFrames([]core.Frame{f})
			Expect(merged).To(Equal(f))
		})
	})
})
