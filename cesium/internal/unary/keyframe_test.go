// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/encoding/json"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// videoFrames returns count frames with a keyframe at each index in keys. Each frame
// carries its index as data.
func videoFrames(count int, keys ...int) telem.Series {
	frames := make([]telem.VideoFrame, count)
	for i := range frames {
		frames[i] = telem.VideoFrame{Codec: telem.VideoCodecH264, Data: []byte{byte(i)}}
	}
	for _, k := range keys {
		frames[k].Keyframe = true
	}
	return telem.NewVideoSeries(frames)
}

// seconds returns count stamps one second apart from start.
func seconds(start telem.TimeStamp, count int) telem.Series {
	stamps := make([]telem.TimeStamp, count)
	for i := range stamps {
		stamps[i] = start + telem.TimeStamp(i)*telem.SecondTS
	}
	return telem.NewSeriesV(stamps...)
}

var _ = Describe("Keyframes", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, func() {
			var (
				fs    xfs.FS
				idx   *unary.DB
				video *unary.DB
				rec   *Recorder
			)
			openVideo := func(ctx SpecContext, cache *unary.OffsetCache) {
				rec = NewRecorder(MustSucceed(fs.Sub("video")))
				video = MustSucceed(unary.Open(ctx, unary.Config{
					FS:        rec,
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "video",
						DataType: telem.VideoT,
						Index:    idx.Channel().Key,
					},
					OffsetCache: cache,
				}))
				video.SetIndex(idx.Index())
			}
			BeforeEach(func(ctx SpecContext) {
				fs = openFS()
				idx = MustSucceed(unary.Open(ctx, unary.Config{
					FS:        MustSucceed(fs.Sub("idx")),
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "idx",
						DataType: telem.TimestampT,
						IsIndex:  true,
					},
				}))
				openVideo(ctx, nil)
				// Keyframes at 10 s, 13 s, and 16 s, then at 102 s after two frames
				// that need the keyframe at 16 s.
				for _, d := range []struct {
					start  telem.TimeStamp
					frames telem.Series
				}{
					{10 * telem.SecondTS, videoFrames(8, 0, 3, 6)},
					{100 * telem.SecondTS, videoFrames(4, 2)},
				} {
					Expect(
						unary.Write(
							ctx,
							idx,
							d.start,
							seconds(d.start, int(d.frames.Len())),
						),
					).
						To(Succeed())
					Expect(unary.Write(ctx, video, d.start, d.frames)).To(Succeed())
				}
				DeferCleanup(func() {
					Expect(video.Close()).To(Succeed())
					Expect(idx.Close()).To(Succeed())
				})
			})

			// seek opens an iterator over bounds, seeks to the keyframe at or before
			// ts, and returns the iterator.
			seek := func(
				ctx SpecContext,
				bounds telem.TimeRange,
				ts telem.TimeStamp,
			) (*unary.Iterator, bool) {
				iter := MustSucceed(video.OpenIterator(unary.IterRange(bounds)))
				DeferCleanup(func() { Expect(iter.Close()).To(Succeed()) })
				return iter, iter.SeekKeyframe(ctx, ts)
			}

			DescribeTable(
				"should move the view to the last keyframe at or before ts",
				func(
					ctx SpecContext,
					ts telem.TimeStamp,
					expected telem.TimeStamp,
				) {
					iter, ok := seek(ctx, telem.TimeRangeMax, ts)
					Expect(ok).To(BeTrue())
					Expect(iter.View()).To(Equal(expected.SpanRange(0)))
				},
				Entry("at the first keyframe", 10*telem.SecondTS, 10*telem.SecondTS),
				Entry("between keyframes", 12*telem.SecondTS, 10*telem.SecondTS),
				Entry("at a later keyframe", 13*telem.SecondTS, 13*telem.SecondTS),
				Entry("between samples", 15*telem.SecondTS+1, 13*telem.SecondTS),
				Entry(
					"in the gap after a domain",
					50*telem.SecondTS,
					16*telem.SecondTS,
				),
				Entry(
					"before the first keyframe of a domain",
					101*telem.SecondTS,
					16*telem.SecondTS,
				),
				Entry(
					"at a keyframe of a later domain",
					102*telem.SecondTS,
					102*telem.SecondTS,
				),
				Entry("after the last sample", telem.TimeStampMax, 102*telem.SecondTS),
			)

			It("should read forward from the keyframe", func(ctx SpecContext) {
				iter, ok := seek(ctx, telem.TimeRangeMax, 15*telem.SecondTS)
				Expect(ok).To(BeTrue())
				Expect(iter.Next(ctx, 2*telem.Second)).To(BeTrue())
				frames := MustSucceed(iter.Value().SeriesAt(0).DecodeVideo())
				Expect(frames).To(HaveLen(2))
				Expect(frames[0].Keyframe).To(BeTrue())
				Expect(frames[0].Data).To(Equal([]byte{3}))
				Expect(frames[1].Keyframe).To(BeFalse())
			})

			It(
				"should not find a keyframe before the first one",
				func(ctx SpecContext) {
					_, ok := seek(ctx, telem.TimeRangeMax, 5*telem.SecondTS)
					Expect(ok).To(BeFalse())
				},
			)

			It("should not find a keyframe before the bounds", func(ctx SpecContext) {
				bounds := (14 * telem.SecondTS).Range(200 * telem.SecondTS)
				_, ok := seek(ctx, bounds, 15*telem.SecondTS)
				Expect(ok).To(BeFalse())
				iter, ok := seek(ctx, bounds, 17*telem.SecondTS)
				Expect(ok).To(BeTrue())
				Expect(iter.View().Start).To(Equal(16 * telem.SecondTS))
			})

			It(
				"should read only the length and header of each frame on a cold seek",
				func(
					ctx SpecContext,
				) {
					Expect(video.Close()).To(Succeed())
					// A budget of one byte caches no table, so each seek scans.
					openVideo(ctx, unary.NewOffsetCache(1))
					rec.Reset()
					iter, ok := seek(ctx, telem.TimeRangeMax, 17*telem.SecondTS)
					Expect(ok).To(BeTrue())
					Expect(iter.View().Start).To(Equal(16 * telem.SecondTS))
					Expect(rec.Count(MatchOp(OpReadAt))).To(Equal(8))
					Expect(rec.Count(MatchOp(OpReadAt), MatchLength(6))).To(Equal(8))
				},
			)

			It("should fail on a channel that is not video", func(ctx SpecContext) {
				iter := MustSucceed(
					idx.OpenIterator(unary.IterRange(telem.TimeRangeMax)),
				)
				Expect(iter.SeekKeyframe(ctx, 10*telem.SecondTS)).To(BeFalse())
				Expect(iter.Error()).To(MatchError(validate.ErrValidation))
				Expect(iter.Error()).To(MatchError(ContainSubstring(
					"cannot seek to a keyframe of a timestamp channel",
				)))
				Expect(iter.Close()).To(MatchError(validate.ErrValidation))
			})
		})
	}
})
