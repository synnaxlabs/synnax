// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Video", func() {
	key := telem.VideoFrame{
		Keyframe: true,
		Codec:    telem.VideoCodecH264,
		Data:     []byte{0, 0, 0, 1, 0x67},
	}
	delta := telem.VideoFrame{Codec: telem.VideoCodecH264, Data: []byte{0, 0, 1, 0x41}}

	Describe("Encode", func() {
		It("Should prefix the access unit with the flags and codec bytes", func() {
			Expect(key.Encode()).To(Equal([]byte{1, 1, 0, 0, 0, 1, 0x67}))
			Expect(delta.Encode()).To(Equal([]byte{0, 1, 0, 0, 1, 0x41}))
		})
	})

	Describe("DecodeVideoFrame", func() {
		It("Should decode an encoded frame", func() {
			Expect(telem.DecodeVideoFrame(key.Encode())).To(Equal(key))
			Expect(telem.DecodeVideoFrame(delta.Encode())).To(Equal(delta))
		})

		It("Should decode a frame with an empty access unit", func() {
			f := MustSucceed(telem.DecodeVideoFrame([]byte{0, 1}))
			Expect(f.Data).To(BeEmpty())
		})

		It("Should reject a sample shorter than the header", func() {
			Expect(telem.DecodeVideoFrame([]byte{1})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"video sample has 1 bytes, but the header needs 2",
				)),
			))
		})

		It("Should reject unknown flags", func() {
			Expect(telem.DecodeVideoFrame([]byte{0x03, 1})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("video sample has unknown flags 0x03")),
			))
		})

		It("Should reject an unknown codec", func() {
			Expect(telem.DecodeVideoFrame([]byte{1, 9})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("video sample has unknown codec 9")),
			))
		})
	})

	Describe("IsVideoKeyframe", func() {
		It("Should read the keyframe flag from the header alone", func() {
			key := telem.VideoFrame{Keyframe: true, Codec: telem.VideoCodecH264}
			Expect(telem.IsVideoKeyframe(key.Encode()[:1])).To(BeTrue())
			delta := telem.VideoFrame{Codec: telem.VideoCodecH264, Data: []byte{9}}
			Expect(telem.IsVideoKeyframe(delta.Encode())).To(BeFalse())
		})
	})

	Describe("NewVideoSeries", func() {
		It("Should create a variable density video series", func() {
			s := telem.NewVideoSeriesV(key, delta)
			Expect(s.DataType).To(Equal(telem.VideoT))
			Expect(s.Len()).To(BeEquivalentTo(2))
			Expect(s.At(0)).To(Equal(key.Encode()))
			Expect(s.At(1)).To(Equal(delta.Encode()))
		})
	})

	Describe("Keyframes", func() {
		It("Should return the indices of the keyframes", func() {
			s := telem.NewVideoSeriesV(
				telem.VideoFrame{Keyframe: true, Codec: telem.VideoCodecH264},
				telem.VideoFrame{Codec: telem.VideoCodecH264, Data: []byte{1}},
				telem.VideoFrame{Keyframe: true, Codec: telem.VideoCodecH264},
			)
			Expect(slices.Collect(s.Keyframes())).To(Equal([]int{0, 2}))
		})

		It("Should return nothing for a series with no keyframe", func() {
			s := telem.NewVideoSeriesV(
				telem.VideoFrame{Codec: telem.VideoCodecH264, Data: []byte{1}},
			)
			Expect(slices.Collect(s.Keyframes())).To(BeEmpty())
		})
	})

	Describe("DecodeVideo", func() {
		It("Should return the frames of the series", func() {
			s := telem.NewVideoSeriesV(key, delta)
			Expect(s.DecodeVideo()).To(Equal([]telem.VideoFrame{key, delta}))
		})

		It("Should return no frames for an empty series", func() {
			s := telem.Series{DataType: telem.VideoT}
			Expect(s.DecodeVideo()).To(BeEmpty())
		})

		It("Should reject a series with an invalid frame", func() {
			s := telem.Series{
				DataType: telem.VideoT,
				Data:     telem.MarshalVariableSample([]byte{0, 7}),
			}
			Expect(s.DecodeVideo()).Error().To(
				MatchError(ContainSubstring("video sample has unknown codec 7")),
			)
		})
	})

	Describe("Validate", func() {
		It("Should accept a series of valid frames", func() {
			s := telem.NewVideoSeriesV(key, delta)
			Expect(s.Validate()).To(Succeed())
		})

		It("Should reject a series with an invalid frame", func() {
			data := append(
				telem.MarshalVariableSample(key.Encode()),
				telem.MarshalVariableSample([]byte{0})...,
			)
			s := telem.Series{DataType: telem.VideoT, Data: data}
			Expect(s.Validate()).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"video sample has 1 bytes, but the header needs 2",
				)),
			))
		})
	})
})
