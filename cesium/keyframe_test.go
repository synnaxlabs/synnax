// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// keyframes returns count frames with a keyframe at each index in keys.
func keyframes(count int, keys ...int) telem.Series {
	frames := make([]telem.VideoFrame, count)
	for i := range frames {
		frames[i] = telem.VideoFrame{Codec: telem.VideoCodecH264, Data: []byte{byte(i)}}
	}
	for _, k := range keys {
		frames[k].Keyframe = true
	}
	return telem.NewVideoSeries(frames)
}

var _ = Describe("SeekKeyframe", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, func() {
			var (
				db               *cesium.DB
				idx, front, back cesium.ChannelKey
				iter             *cesium.Iterator
				openIterator     func(keys ...cesium.ChannelKey)
			)
			BeforeEach(func(ctx SpecContext) {
				db = mustOpenDBOnFS(ctx, openFS())
				idx, front, back = GenerateChannelKey(),
					GenerateChannelKey(),
					GenerateChannelKey()
				Expect(db.CreateChannel(
					ctx,
					cesium.Channel{
						Key:      idx,
						Name:     "Daguerre",
						IsIndex:  true,
						DataType: telem.TimestampT,
					},
					cesium.Channel{
						Key:      front,
						Name:     "Niepce",
						Index:    idx,
						DataType: telem.VideoT,
					},
					cesium.Channel{
						Key:      back,
						Name:     "Talbot",
						Index:    idx,
						DataType: telem.VideoT,
					},
				)).To(Succeed())
				// front has keyframes at 1 s and 4 s, and back at 3 s.
				Expect(db.Write(ctx, telem.SecondTS, telem.MultiFrame(
					[]cesium.ChannelKey{idx, front, back},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6),
						keyframes(6, 0, 3),
						keyframes(6, 2),
					},
				))).To(Succeed())
				openIterator = func(keys ...cesium.ChannelKey) {
					iter = MustSucceed(db.OpenIterator(cesium.IteratorConfig{
						Bounds:   telem.TimeRangeMax,
						Channels: keys,
					}))
					DeferCleanup(func() { _ = iter.Close() })
				}
			})

			It("should read every channel from the keyframe", func() {
				openIterator(idx, front)
				Expect(iter.SeekKeyframe(5 * telem.SecondTS)).To(BeTrue())
				Expect(iter.Next(2 * telem.Second)).To(BeTrue())
				Expect(iter.Value().Get(idx).Series[0]).
					To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(4, 5)))
				frames := MustSucceed(iter.Value().Get(front).Series[0].DecodeVideo())
				Expect(frames[0].Keyframe).To(BeTrue())
				Expect(frames[0].Data).To(Equal([]byte{3}))
			})

			It(
				"should move to the earliest keyframe of several video channels",
				func() {
					openIterator(idx, front, back)
					Expect(iter.SeekKeyframe(5 * telem.SecondTS)).To(BeTrue())
					Expect(iter.Next(telem.Second)).To(BeTrue())
					Expect(iter.Value().Get(idx).Series[0]).
						To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(3)))
				},
			)

			It("should return false when no video channel has a keyframe", func() {
				openIterator(idx, back)
				Expect(iter.SeekKeyframe(2 * telem.SecondTS)).To(BeFalse())
			})

			It("should fail without a video channel", func() {
				openIterator(idx)
				Expect(iter.SeekKeyframe(5 * telem.SecondTS)).To(BeFalse())
				Expect(iter.Error()).To(MatchError(validate.ErrValidation))
				Expect(iter.Error()).To(MatchError(ContainSubstring(
					"cannot seek to a keyframe of a timestamp channel",
				)))
			})
		})
	}
})
