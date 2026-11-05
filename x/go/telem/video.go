// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"iter"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// VideoCodec identifies the encoding of the access unit in a VideoFrame.
type VideoCodec uint8

// VideoCodecH264 is H.264 in Annex B format, with no B-frames.
const VideoCodecH264 VideoCodec = 1

const (
	videoHeaderSize   = 2
	videoKeyframeFlag = 1
)

// VideoFrame is one encoded picture. It is a single sample of a VideoT series.
type VideoFrame struct {
	// Keyframe is true when the frame decodes without the frames before it. A keyframe
	// carries its codec parameter sets.
	Keyframe bool
	// Codec is the encoding of Data.
	Codec VideoCodec
	// Data is one access unit.
	Data []byte
}

// Encode returns the frame as a VideoT sample.
func (f VideoFrame) Encode() []byte {
	b := make([]byte, videoHeaderSize+len(f.Data))
	if f.Keyframe {
		b[0] = videoKeyframeFlag
	}
	b[1] = byte(f.Codec)
	copy(b[videoHeaderSize:], f.Data)
	return b
}

// DecodeVideoFrame parses a VideoT sample. The Data of the returned frame aliases
// sample. It returns validate.ErrValidation when the header is malformed or names an
// unknown codec.
func DecodeVideoFrame(sample []byte) (VideoFrame, error) {
	if len(sample) < videoHeaderSize {
		return VideoFrame{}, errors.Wrapf(
			validate.ErrValidation,
			"video sample has %d bytes, but the header needs %d",
			len(sample),
			videoHeaderSize,
		)
	}
	if sample[0]&^videoKeyframeFlag != 0 {
		return VideoFrame{}, errors.Wrapf(
			validate.ErrValidation,
			"video sample has unknown flags %#02x",
			sample[0],
		)
	}
	codec := VideoCodec(sample[1])
	if codec != VideoCodecH264 {
		return VideoFrame{}, errors.Wrapf(
			validate.ErrValidation,
			"video sample has unknown codec %d",
			codec,
		)
	}
	return VideoFrame{
		Keyframe: sample[0] == videoKeyframeFlag,
		Codec:    codec,
		Data:     sample[videoHeaderSize:],
	}, nil
}

// IsVideoKeyframe reports whether a VideoT sample is a keyframe. It reads only the
// flags byte, so sample may be the header alone. It expects a sample that passed
// Validate.
func IsVideoKeyframe(sample []byte) bool { return sample[0]&videoKeyframeFlag != 0 }

// NewVideoSeries creates a VideoT series with one sample per frame.
func NewVideoSeries(frames []VideoFrame) Series {
	samples := make([][]byte, len(frames))
	for i, f := range frames {
		samples[i] = f.Encode()
	}
	return Series{DataType: VideoT, Data: marshalVariable(samples)}
}

// NewVideoSeriesV is a variadic version of NewVideoSeries.
func NewVideoSeriesV(frames ...VideoFrame) Series { return NewVideoSeries(frames) }

// Keyframes returns the indices of the keyframes in a VideoT series, in order.
// Keyframes does NOT check the Series' DataType, and it expects samples that passed
// Validate.
func (s Series) Keyframes() iter.Seq[int] {
	return func(yield func(int) bool) {
		i := 0
		for sample := range s.Samples() {
			if IsVideoKeyframe(sample) && !yield(i) {
				return
			}
			i++
		}
	}
}

// DecodeVideo returns the frames of a VideoT series. The Data of each frame aliases
// the series. DecodeVideo does NOT check the Series' DataType. It returns
// validate.ErrValidation when a sample is not a valid frame.
func (s Series) DecodeVideo() ([]VideoFrame, error) {
	frames := make([]VideoFrame, 0, s.Len())
	for sample := range s.Samples() {
		f, err := DecodeVideoFrame(sample)
		if err != nil {
			return nil, err
		}
		frames = append(frames, f)
	}
	return frames, nil
}
