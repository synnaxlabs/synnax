// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video_test

import (
	"time"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/video"
)

var (
	sps = []byte{0x67, 0x42, 0xC0, 0x1E}
	pps = []byte{0x68, 0xCE, 0x3C, 0x80}
	idr = []byte{0x65, 0x88, 0x84}
	// The slice headers below hold first_mb_in_slice = 0, then slice_type.
	pSlice  = []byte{0x41, 0xC0}
	bSlice  = []byte{0x41, 0xA0}
	bSlice6 = []byte{0x41, 0x9C}
	pSlice5 = []byte{0x41, 0x98}
)

func annexB(au ...[]byte) []byte { return MustSucceed(h264.AnnexB(au).Marshal()) }

var _ = Describe("H264", func() {
	var h *video.H264
	BeforeEach(func() { h = &video.H264{} })

	It(
		"Should return an IDR access unit with its parameter sets as a keyframe",
		func() {
			Expect(h.Frame([][]byte{sps, pps, idr})).To(Equal(telem.VideoFrame{
				Keyframe: true,
				Codec:    telem.VideoCodecH264,
				Data:     annexB(sps, pps, idr),
			}))
		},
	)

	It(
		"Should put the parameter sets in front of an IDR access unit that lacks them",
		func() {
			MustSucceed(h.Frame([][]byte{sps, pps, idr}))
			Expect(h.Frame([][]byte{idr})).To(Equal(telem.VideoFrame{
				Keyframe: true,
				Codec:    telem.VideoCodecH264,
				Data:     annexB(sps, pps, idr),
			}))
		},
	)

	It("Should use parameter sets from outside the stream", func() {
		h.SPS, h.PPS = sps, pps
		Expect(
			MustSucceed(h.Frame([][]byte{idr})).Data,
		).To(Equal(annexB(sps, pps, idr)))
	})

	It(
		"Should not return an IDR access unit as a keyframe with no parameter sets",
		func() {
			f := MustSucceed(h.Frame([][]byte{idr}))
			Expect(f.Keyframe).To(BeFalse())
			Expect(f.Data).To(Equal(annexB(idr)))
		},
	)

	It("Should return a P slice as it is", func() {
		Expect(h.Frame([][]byte{pSlice})).To(Equal(telem.VideoFrame{
			Codec: telem.VideoCodecH264,
			Data:  annexB(pSlice),
		}))
		Expect(MustSucceed(h.Frame([][]byte{pSlice5})).Keyframe).To(BeFalse())
	})

	DescribeTable("Should reject an access unit with a B slice", func(slice []byte) {
		Expect(h.Frame([][]byte{slice})).Error().To(MatchError(video.ErrBFrames))
	},
		Entry("slice_type 1", bSlice),
		Entry("slice_type 6", bSlice6),
	)
})

var _ = Describe("KeyframeWatch", func() {
	var (
		w     *video.KeyframeWatch
		start time.Time
	)
	keyframe := telem.VideoFrame{Keyframe: true}
	frame := telem.VideoFrame{}
	BeforeEach(func() {
		w = &video.KeyframeWatch{Limit: 10 * time.Second}
		start = time.Now()
	})

	It("Should accept frames within the limit of a keyframe", func() {
		Expect(w.Observe(keyframe, start)).To(BeTrue())
		Expect(w.Observe(frame, start.Add(10*time.Second))).To(BeTrue())
	})

	It("Should report each frame past the limit of the last keyframe", func() {
		Expect(w.Observe(keyframe, start)).To(BeTrue())
		for _, after := range []time.Duration{11 * time.Second, 12 * time.Second} {
			ok, err := w.Observe(frame, start.Add(after))
			Expect(ok).To(BeTrue())
			Expect(err).To(MatchError(video.ErrNoKeyframe))
		}
		Expect(w.Observe(keyframe, start.Add(13*time.Second))).To(BeTrue())
	})

	It("Should count from the first frame of a stream with no keyframe", func() {
		Expect(w.Observe(frame, start)).To(BeFalse())
		Expect(w.Observe(frame, start.Add(11*time.Second))).Error().
			To(MatchError(video.ErrNoKeyframe))
	})

	It("Should reject the frames before the first keyframe", func() {
		Expect(w.Observe(frame, start)).To(BeFalse())
		Expect(w.Observe(frame, start.Add(time.Second))).To(BeFalse())
		Expect(w.Observe(keyframe, start.Add(2*time.Second))).To(BeTrue())
		Expect(w.Observe(frame, start.Add(3*time.Second))).To(BeTrue())
	})

	It("Should never report with no limit", func() {
		w.Limit = 0
		Expect(w.Observe(frame, start)).To(BeFalse())
		Expect(w.Observe(frame, start.Add(time.Hour))).To(BeFalse())
	})
})
