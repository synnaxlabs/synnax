// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"bytes"
	"context"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/fmp4"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mp4/codecs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/transport/http/framer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/video"
)

var (
	// sps is a 1920x1080 baseline SPS, which an MP4 init segment can parse.
	sps = []byte{
		0x67, 0x42, 0xc0, 0x28, 0xd9, 0x00, 0x78, 0x02, 0x27, 0xe5, 0x84, 0x00, 0x00,
		0x03, 0x00, 0x04, 0x00, 0x00, 0x03, 0x00, 0xf0, 0x3c, 0x60, 0xc9, 0x20,
	}
	pps    = []byte{0x68, 0xce, 0x3c, 0x80}
	idr    = []byte{0x65, 0x88, 0x84, 0x00}
	pSlice = []byte{0x41, 0xc0, 0x00}
)

func videoFrame(keyframe bool, au ...[]byte) telem.VideoFrame {
	return telem.VideoFrame{
		Keyframe: keyframe,
		Codec:    telem.VideoCodecH264,
		Data:     MustSucceed(h264.AnnexB(au).Marshal()),
	}
}

func keyframe() telem.VideoFrame { return videoFrame(true, sps, pps, idr) }

func delta() telem.VideoFrame { return videoFrame(false, pSlice) }

// createVideo creates the channels of createIndexed and deletes them when the spec
// ends, since the GOP cache holds a relay tap open while a video channel exists.
func createVideo(
	ctx SpecContext,
	dataTypes ...telem.DataType,
) (svcchannel.Channel, []svcchannel.Channel) {
	GinkgoHelper()
	index, channels := createIndexed(ctx, dataTypes...)
	DeferCleanup(func(ctx SpecContext) {
		keys := append(svcchannel.KeysFromChannels(channels), index.Key())
		Expect(channelWriter.DeleteMany(ctx, keys, false)).To(Succeed())
	})
	return index, channels
}

var _ = Describe("MP4", func() {
	Describe("MP4Encoder", func() {
		It("Should emit the MP4 media type", func() {
			Expect(framer.MP4Encoder.ContentType()).To(Equal("video/mp4"))
		})

		It("Should write the frames of a video channel from its first keyframe", func(
			ctx SpecContext,
		) {
			index, channels := createVideo(ctx, telem.VideoT)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1, 2, 3, 4),
				channels,
				telem.NewVideoSeriesV(delta(), keyframe(), delta(), keyframe()),
			)
			b := MustSucceed(
				framer.MP4Encoder.Encode(ctx, read(ctx, index, channels...)),
			)
			var init fmp4.Init
			Expect(init.Unmarshal(bytes.NewReader(b))).To(Succeed())
			Expect(init.Tracks[0].Codec).To(Equal(&codecs.H264{SPS: sps, PPS: pps}))
			var parts fmp4.Parts
			Expect(parts.Unmarshal(b)).To(Succeed())
			Expect(parts).To(HaveLen(2))
			second := uint64(video.MP4Timescale)
			Expect(parts[0].Tracks[0].BaseTime).To(BeZero())
			Expect(parts[0].Tracks[0].Samples).To(HaveLen(2))
			Expect(parts[1].Tracks[0].BaseTime).To(Equal(2 * second))
			Expect(parts[1].Tracks[0].Samples).To(HaveLen(1))
			for _, p := range parts {
				for _, s := range p.Tracks[0].Samples {
					Expect(s.Duration).To(Equal(uint32(second)))
				}
			}
		})

		It("Should only write the frames in the time range", func(ctx SpecContext) {
			index, channels := createVideo(ctx, telem.VideoT)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1, 2, 3),
				channels,
				telem.NewVideoSeriesV(keyframe(), delta(), keyframe()),
			)
			res := read(ctx, index, channels...)
			res.OpenIterator = func(
				ctx context.Context,
				keys svcchannel.Keys,
			) (*svcframer.Iterator, error) {
				return framerSvc.OpenIterator(ctx, svcframer.IteratorConfig{
					Keys: keys,
					Bounds: telem.TimeRange{
						Start: 2 * telem.SecondTS,
						End:   telem.TimeStampMax,
					},
				})
			}
			var parts fmp4.Parts
			Expect(parts.Unmarshal(MustSucceed(framer.MP4Encoder.Encode(ctx, res)))).
				To(Succeed())
			Expect(parts).To(HaveLen(1))
			Expect(parts[0].Tracks[0].Samples).To(HaveLen(1))
		})

		DescribeTable("Should fail a read of anything but one video channel", func(
			ctx SpecContext,
			dataTypes ...telem.DataType,
		) {
			index, channels := createVideo(ctx, dataTypes...)
			var buf bytes.Buffer
			Expect(framer.MP4Encoder.EncodeStream(
				ctx,
				&buf,
				read(ctx, index, channels...),
			)).To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"an MP4 read takes exactly one video channel",
				)),
			))
			Expect(buf.Len()).To(BeZero())
		},
			Entry("float channel", telem.Float32T),
			Entry("two video channels", telem.VideoT, telem.VideoT),
			Entry("video and float channels", telem.VideoT, telem.Float32T),
		)

		It("Should fail a read with a reduction", func(ctx SpecContext) {
			index, channels := createVideo(ctx, telem.VideoT)
			res := response(
				channels,
				[]svcchannel.Channel{index},
				svcframer.IteratorConfig{
					Reduction: telem.Reduction{
						Variant: telem.StrideReduction{Factor: 2},
					},
				},
			)
			Expect(framer.MP4Encoder.Encode(ctx, res)).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("an MP4 read cannot take a reduction")),
			))
		})

		It("Should fail a time range with no keyframe before the first byte", func(
			ctx SpecContext,
		) {
			index, channels := createVideo(ctx, telem.VideoT)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1, 2),
				channels,
				telem.NewVideoSeriesV(delta(), delta()),
			)
			var buf bytes.Buffer
			Expect(framer.MP4Encoder.EncodeStream(
				ctx,
				&buf,
				read(ctx, index, channels...),
			)).To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"channel "+channels[0].Name+" has no keyframe in the time range",
				)),
			))
			Expect(buf.Len()).To(BeZero())
		})
	})
})
