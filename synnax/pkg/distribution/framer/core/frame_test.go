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
	Describe("SplitByLeaseholder", func() {
		It("Should split the frame into separate frames by the channels leaseholder", func() {
			node1ch1 := channel.NewKey(1, 1)
			node1ch2 := channel.NewKey(1, 1)
			node2ch1 := channel.NewKey(2, 1)
			node2ch2 := channel.NewKey(2, 1)
			f := core.MultiFrame(
				[]channel.Key{node1ch1, node1ch2, node2ch1, node2ch2},
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
					telem.NewSeriesV[int64](7, 8, 9),
					telem.NewSeriesV[int64](10, 11, 12),
				},
			)
			frames := f.SplitByLeaseholder()
			Expect(frames).To(HaveLen(2))
			Expect(frames[1]).To(Equal(core.MultiFrame(
				[]channel.Key{node1ch1, node1ch2},
				[]telem.Series{telem.NewSeriesV[int64](1, 2, 3), telem.NewSeriesV[int64](4, 5, 6)},
			)))
			Expect(frames[2]).To(Equal(core.MultiFrame(
				[]channel.Key{node2ch1, node2ch2},
				[]telem.Series{telem.NewSeriesV[int64](7, 8, 9), telem.NewSeriesV[int64](10, 11, 12)},
			)))
		})
	})
	Describe("SplitByHost", func() {
		It("Should split a frame into a local, remote, and free frame", func() {
			localNodeCh := channel.NewKey(1, 1)
			remoteNodeCh := channel.NewKey(2, 1)
			freeNodeCh := channel.NewKey(dcore.Free, 1)
			f := core.MultiFrame(
				[]channel.Key{localNodeCh, remoteNodeCh, freeNodeCh},
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
					telem.NewSeriesV[int64](7, 8, 9),
				},
			)
			local, remote, free := f.SplitByHost(1)
			Expect(local).To(Equal(core.UnaryFrame(
				localNodeCh,
				telem.NewSeriesV[int64](1, 2, 3),
			)))
			Expect(remote).To(Equal(core.UnaryFrame(
				remoteNodeCh,
				telem.NewSeriesV[int64](4, 5, 6),
			)))
			Expect(free).To(Equal(core.UnaryFrame(
				freeNodeCh,
				telem.NewSeriesV[int64](7, 8, 9),
			)))
		})
	})

	Describe("ToStorage", func() {
		It("Should convert to storage frame", func() {
			f := core.MultiFrame([]channel.Key{1, 2}, []telem.Series{telem.NewSeriesV[int64](1, 2, 3), telem.NewSeriesV[int64](4, 5, 6)})
			sf := f.ToStorage()
			Expect(sf.KeysSlice()).To(Equal([]ts.ChannelKey{1, 2}))
			Expect(sf.SeriesSlice()).To(Equal(f.SeriesSlice()))
		})
	})

	Describe("FilterKeys", func() {
		It("Should filter frame to only include specified keys", func() {
			f := core.MultiFrame([]channel.Key{1, 2, 3}, []telem.Series{
				telem.NewSeriesV[int64](1, 2, 3),
				telem.NewSeriesV[int64](4, 5, 6),
				telem.NewSeriesV[int64](7, 8, 9),
			})
			filtered := f.FilterKeys([]channel.Key{1, 3})
			Expect(filtered.KeysSlice()).To(Equal([]channel.Key{1, 3}))
			Expect(filtered.Count()).To(Equal(2))
			Expect(filtered.SeriesAt(0)).To(Equal(telem.NewSeriesV[int64](1, 2, 3)))
			Expect(filtered.SeriesAt(1)).To(Equal(telem.NewSeriesV[int64](7, 8, 9)))
		})
	})

	Describe("MergeFrames", func() {
		It("Should merge multiple frames into one", func() {
			f1 := core.MultiFrame(
				[]channel.Key{1, 2},
				[]telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](3, 4)},
			)
			f2 := core.MultiFrame(
				[]channel.Key{3, 4},
				[]telem.Series{telem.NewSeriesV[int64](5, 6), telem.NewSeriesV[int64](7, 8)},
			)
			merged := core.MergeFrames([]core.Frame{f1, f2})
			Expect(merged.Count()).To(Equal(4))
			Expect(merged.KeysSlice()).To(Equal([]channel.Key{1, 2, 3, 4}))
		})

		It("Should return empty frame for empty input", func() {
			merged := core.MergeFrames([]core.Frame{})
			Expect(merged.Count()).To(Equal(0))
			Expect(merged.KeysSlice()).To(BeEmpty())
		})

		It("Should return same frame for single input", func() {
			f := core.MultiFrame(
				[]channel.Key{1, 2},
				[]telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](3, 4)},
			)
			merged := core.MergeFrames([]core.Frame{f})
			Expect(merged).To(Equal(f))
		})
	})
})
