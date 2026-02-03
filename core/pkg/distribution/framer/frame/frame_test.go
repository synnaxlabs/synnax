// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package frame_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
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
			f := frame.NewMulti(
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
			Expect(frames[1]).To(Equal(frame.NewMulti(
				[]channel.Key{node1ch1, node1ch2},
				[]telem.Series{telem.NewSeriesV[int64](1, 2, 3), telem.NewSeriesV[int64](4, 5, 6)},
			)))
			Expect(frames[2]).To(Equal(frame.NewMulti(
				[]channel.Key{node2ch1, node2ch2},
				[]telem.Series{telem.NewSeriesV[int64](7, 8, 9), telem.NewSeriesV[int64](10, 11, 12)},
			)))
		})
	})

	Describe("SplitByHost", func() {
		It("Should split a frame into a local, remote, and free frame", func() {
			localNodeCh := channel.NewKey(1, 1)
			remoteNodeCh := channel.NewKey(2, 1)
			freeNodeCh := channel.NewKey(cluster.NodeKeyFree, 1)
			f := frame.NewMulti(
				[]channel.Key{localNodeCh, remoteNodeCh, freeNodeCh},
				[]telem.Series{
					telem.NewSeriesV[int64](1, 2, 3),
					telem.NewSeriesV[int64](4, 5, 6),
					telem.NewSeriesV[int64](7, 8, 9),
				},
			)
			local, remote, free := f.SplitByHost(1)
			Expect(local).To(Equal(frame.NewUnary(
				localNodeCh,
				telem.NewSeriesV[int64](1, 2, 3),
			)))
			Expect(remote).To(Equal(frame.NewUnary(
				remoteNodeCh,
				telem.NewSeriesV[int64](4, 5, 6),
			)))
			Expect(free).To(Equal(frame.NewUnary(
				freeNodeCh,
				telem.NewSeriesV[int64](7, 8, 9),
			)))
		})
	})

	Describe("ToStorage", func() {
		It("Should convert to storage frame", func() {
			f := frame.NewMulti([]channel.Key{1, 2}, []telem.Series{telem.NewSeriesV[int64](1, 2, 3), telem.NewSeriesV[int64](4, 5, 6)})
			sf := f.ToStorage()
			Expect(sf.KeysSlice()).To(Equal([]ts.ChannelKey{1, 2}))
			Expect(sf.SeriesSlice()).To(Equal(f.SeriesSlice()))
		})
	})

	Describe("KeepKeys", func() {
		It("Should filter frame to only include specified keys", func() {
			f := frame.NewMulti([]channel.Key{1, 2, 3}, []telem.Series{
				telem.NewSeriesV[int64](1, 2, 3),
				telem.NewSeriesV[int64](4, 5, 6),
				telem.NewSeriesV[int64](7, 8, 9),
			})
			filtered := f.KeepKeys([]channel.Key{1, 3})
			Expect(filtered.KeysSlice()).To(Equal([]channel.Key{1, 3}))
			Expect(filtered.Count()).To(Equal(2))
			Expect(filtered.SeriesAt(0)).To(Equal(telem.NewSeriesV[int64](1, 2, 3)))
			Expect(filtered.SeriesAt(1)).To(Equal(telem.NewSeriesV[int64](7, 8, 9)))
		})
	})

	Describe("Extend", func() {
		It("Should extend a frame with another frame", func() {
			f1 := frame.NewMulti(
				[]channel.Key{1, 2},
				[]telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](3, 4)},
			)
			f2 := frame.NewMulti(
				[]channel.Key{3, 4},
				[]telem.Series{telem.NewSeriesV[int64](5, 6), telem.NewSeriesV[int64](7, 8)},
			)
			extended := f1.Extend(f2)
			Expect(extended.Count()).To(Equal(4))
			Expect(extended.KeysSlice()).To(Equal([]channel.Key{1, 2, 3, 4}))
		})
		It("Should extend an empty frame", func() {
			empty := frame.Frame{}
			f := frame.NewMulti(
				[]channel.Key{1, 2},
				[]telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](3, 4)},
			)
			extended := empty.Extend(f)
			Expect(extended.Count()).To(Equal(2))
			Expect(extended.KeysSlice()).To(Equal([]channel.Key{1, 2}))
		})
		It("Should extend with an empty frame", func() {
			f := frame.NewMulti(
				[]channel.Key{1, 2},
				[]telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](3, 4)},
			)
			extended := f.Extend(frame.Frame{})
			Expect(extended.Count()).To(Equal(2))
			Expect(extended.KeysSlice()).To(Equal([]channel.Key{1, 2}))
		})
		It("Should extend with a masked frame", func() {
			f1 := frame.NewMulti(
				[]channel.Key{1},
				[]telem.Series{telem.NewSeriesV[int64](1)},
			)
			f2 := frame.NewMulti(
				[]channel.Key{2, 3, 4},
				[]telem.Series{
					telem.NewSeriesV[int64](2),
					telem.NewSeriesV[int64](3),
					telem.NewSeriesV[int64](4),
				},
			).KeepKeys([]channel.Key{2, 4})
			extended := f1.Extend(f2)
			Expect(extended.Count()).To(Equal(3))
			Expect(extended.KeysSlice()).To(Equal([]channel.Key{1, 2, 4}))
		})
	})
	Describe("Merge", func() {
		It("Should merge multiple frames into one", func() {
			f1 := frame.NewMulti(
				[]channel.Key{1, 2},
				[]telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](3, 4)},
			)
			f2 := frame.NewMulti(
				[]channel.Key{3, 4},
				[]telem.Series{telem.NewSeriesV[int64](5, 6), telem.NewSeriesV[int64](7, 8)},
			)
			merged := frame.Merge([]frame.Frame{f1, f2})
			Expect(merged.Count()).To(Equal(4))
			Expect(merged.KeysSlice()).To(Equal([]channel.Key{1, 2, 3, 4}))
		})
		It("Should return empty frame for empty input", func() {
			merged := frame.Merge([]frame.Frame{})
			Expect(merged.Count()).To(Equal(0))
			Expect(merged.KeysSlice()).To(BeEmpty())
		})
		It("Should return same frame for single input", func() {
			f := frame.NewMulti(
				[]channel.Key{1, 2},
				[]telem.Series{telem.NewSeriesV[int64](1, 2), telem.NewSeriesV[int64](3, 4)},
			)
			merged := frame.Merge([]frame.Frame{f})
			Expect(merged).To(Equal(f))
		})
		It("Should merge many frames", func() {
			frames := make([]frame.Frame, 5)
			for i := range frames {
				frames[i] = frame.NewUnary(channel.Key(i+1), telem.NewSeriesV[int64](int64(i+1)))
			}
			merged := frame.Merge(frames)
			Expect(merged.Count()).To(Equal(5))
			Expect(merged.KeysSlice()).To(Equal([]channel.Key{1, 2, 3, 4, 5}))
		})
	})

	Describe("ShallowCopy", func() {
		It("Should create a shallow copy of the frame", func() {
			original := frame.NewMulti(
				[]channel.Key{1, 2, 3},
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
	})

	Describe("NewFromStorage", func() {
		It("Should create a new frame from its storage later representation", func() {
			storageFrame := telem.UnaryFrame[cesium.ChannelKey](1, telem.NewSeriesV[float32](1, 2, 3, 4))
			distFrame := frame.NewFromStorage(storageFrame)
			Expect(distFrame.SeriesSlice()).To(HaveLen(1))
			Expect(distFrame.KeysSlice()).To(HaveLen(1))
			Expect(distFrame.KeysSlice()[0]).To(Equal(channel.Key(1)))
			Expect(distFrame.SeriesAt(0)).To(telem.MatchSeries(telem.NewSeriesV[float32](1, 2, 3, 4)))
		})
	})

	Describe("Alloc", func() {
		It("Should allocate a frame with the specified capacity", func() {
			fr := frame.Alloc(12)
			Expect(fr.RawKeys()).To(HaveCap(12))
			Expect(fr.RawSeries()).To(HaveCap(12))
		})
	})
})
