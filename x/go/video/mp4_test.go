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
	"bytes"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/fmp4"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mp4/codecs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/video"
)

// realSPS is a 1920x1080 baseline SPS, which an MP4 init segment can parse.
var realSPS = []byte{
	0x67, 0x42, 0xc0, 0x28, 0xd9, 0x00, 0x78, 0x02, 0x27, 0xe5, 0x84, 0x00, 0x00,
	0x03, 0x00, 0x04, 0x00, 0x00, 0x03, 0x00, 0xf0, 0x3c, 0x60, 0xc9, 0x20,
}

func h264Frame(keyframe bool, au ...[]byte) telem.VideoFrame {
	return telem.VideoFrame{
		Keyframe: keyframe,
		Codec:    telem.VideoCodecH264,
		Data:     annexB(au...),
	}
}

func mp4Key() telem.VideoFrame { return h264Frame(true, realSPS, pps, idr) }

func mp4Delta() telem.VideoFrame { return h264Frame(false, pSlice) }

func ms(n int) telem.TimeStamp { return telem.TimeStamp(n) * telem.MillisecondTS }

func parts(out []byte) fmp4.Parts {
	var p fmp4.Parts
	Expect(p.Unmarshal(out)).To(Succeed())
	return p
}

func durations(p *fmp4.Part) []uint32 {
	d := make([]uint32, len(p.Tracks[0].Samples))
	for i, s := range p.Tracks[0].Samples {
		d[i] = s.Duration
	}
	return d
}

var _ = Describe("MP4", func() {
	var (
		out bytes.Buffer
		m   *video.MP4
	)
	BeforeEach(func() {
		out.Reset()
		m = video.NewMP4(&out)
	})

	It("Should take the parameter sets of the file from the first keyframe", func() {
		Expect(m.Write(mp4Key(), ms(0))).To(Succeed())
		Expect(m.Close()).To(Succeed())
		var init fmp4.Init
		Expect(init.Unmarshal(bytes.NewReader(out.Bytes()))).To(Succeed())
		Expect(init.Tracks).To(HaveLen(1))
		Expect(init.Tracks[0].TimeScale).To(Equal(uint32(video.MP4Timescale)))
		Expect(init.Tracks[0].Codec).To(Equal(&codecs.H264{SPS: realSPS, PPS: pps}))
	})

	It("Should write a fragment for each group of pictures", func() {
		Expect(m.Write(mp4Key(), ms(0))).To(Succeed())
		Expect(m.Write(mp4Delta(), ms(40))).To(Succeed())
		Expect(m.Write(mp4Delta(), ms(80))).To(Succeed())
		Expect(m.Write(mp4Key(), ms(100))).To(Succeed())
		Expect(m.Write(mp4Delta(), ms(150))).To(Succeed())
		Expect(m.Close()).To(Succeed())
		p := parts(out.Bytes())
		Expect(p).To(HaveLen(2))
		Expect(p[0].SequenceNumber).To(Equal(uint32(1)))
		Expect(p[0].Tracks[0].BaseTime).To(Equal(uint64(0)))
		Expect(durations(p[0])).To(Equal([]uint32{3600, 3600, 1800}))
		Expect(p[1].SequenceNumber).To(Equal(uint32(2)))
		Expect(p[1].Tracks[0].BaseTime).To(Equal(uint64(9000)))
		Expect(durations(p[1])).To(Equal([]uint32{4500, 4500}))
		syncs := []bool{}
		for _, part := range p {
			for _, s := range part.Tracks[0].Samples {
				syncs = append(syncs, !s.IsNonSyncSample)
			}
		}
		Expect(syncs).To(Equal([]bool{true, false, false, true, false}))
	})

	It("Should store each frame without its parameter sets", func() {
		Expect(m.Write(mp4Key(), ms(0))).To(Succeed())
		Expect(m.Close()).To(Succeed())
		var au h264.AVCC
		Expect(au.Unmarshal(parts(out.Bytes())[0].Tracks[0].Samples[0].Payload)).
			To(Succeed())
		Expect(au).To(Equal(h264.AVCC{idr}))
	})

	It("Should drop the frames before the first keyframe", func() {
		Expect(m.Write(mp4Delta(), ms(0))).To(Succeed())
		Expect(m.Write(mp4Key(), ms(500))).To(Succeed())
		Expect(m.Write(mp4Delta(), ms(600))).To(Succeed())
		Expect(m.Close()).To(Succeed())
		p := parts(out.Bytes())
		Expect(p).To(HaveLen(1))
		Expect(p[0].Tracks[0].BaseTime).To(Equal(uint64(0)))
		Expect(durations(p[0])).To(Equal([]uint32{9000, 9000}))
	})

	It("Should give a file with one frame a duration", func() {
		Expect(m.Write(mp4Key(), ms(0))).To(Succeed())
		Expect(m.Close()).To(Succeed())
		Expect(durations(parts(out.Bytes())[0])).To(Equal([]uint32{3000}))
	})

	It("Should fail to close a file with no keyframe", func() {
		Expect(m.Write(mp4Delta(), ms(0))).To(Succeed())
		Expect(m.Close()).To(MatchError(video.ErrEmptyMP4))
		Expect(out.Len()).To(BeZero())
	})

	It("Should fail on a first keyframe with no parameter sets", func() {
		Expect(m.Write(h264Frame(true, idr), ms(0))).
			To(MatchError(video.ErrNoParameterSets))
	})

	It("Should not overflow the times of a long file", func() {
		day := 3 * 24 * telem.HourTS
		Expect(m.Write(mp4Key(), 0)).To(Succeed())
		Expect(m.Write(mp4Key(), day)).To(Succeed())
		Expect(m.Close()).To(Succeed())
		Expect(parts(out.Bytes())[1].Tracks[0].BaseTime).
			To(Equal(uint64(3 * 24 * 3600 * video.MP4Timescale)))
	})
})
