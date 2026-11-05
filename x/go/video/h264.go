// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package video turns the encoded pictures of a video stream into the frames of a
// telem.VideoT series. Each streaming protocol is an adapter that feeds it.
package video

import (
	"slices"
	"time"

	"github.com/bluenviron/mediacommon/v2/pkg/bits"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

var (
	// ErrBFrames is returned for a stream that holds B-frames, which a viewer cannot
	// decode in arrival order.
	ErrBFrames = errors.New(
		"the stream has B-frames. Set B-frames to 0 in the encoder, or use the " +
			"baseline profile",
	)
	// ErrNoKeyframe is returned for a stream with no keyframe for longer than a
	// KeyframeWatch allows, since no viewer can start to decode it.
	ErrNoKeyframe = errors.New(
		"the stream has no keyframe, so viewers cannot start to show it. Set a " +
			"keyframe interval in the encoder, and turn off intra refresh",
	)
)

// sliceTypeB is the value of slice_type modulo 5 for a B slice.
const sliceTypeB = 1

// H264 assembles H.264 access units into frames. The zero value is ready for use. It
// is not safe for concurrent use.
type H264 struct {
	// SPS is the last sequence parameter set of the stream.
	SPS []byte
	// PPS is the last picture parameter set of the stream.
	PPS []byte
}

// Frame returns the frame of one access unit, in Annex B format. The frame of an IDR
// access unit is a keyframe when H264 knows the parameter sets of the stream, and
// Frame puts them in front of an IDR access unit that lacks them. Frame returns
// ErrBFrames for an access unit with a B slice.
func (h *H264) Frame(au [][]byte) (telem.VideoFrame, error) {
	var hasSPS, hasPPS, idr bool
	for _, nalu := range au {
		if len(nalu) == 0 {
			continue
		}
		switch h264.NALUType(nalu[0] & 0x1F) {
		case h264.NALUTypeSPS:
			h.SPS, hasSPS = slices.Clone(nalu), true
		case h264.NALUTypePPS:
			h.PPS, hasPPS = slices.Clone(nalu), true
		case h264.NALUTypeIDR:
			idr = true
		case h264.NALUTypeNonIDR:
			if isBSlice(nalu) {
				return telem.VideoFrame{}, ErrBFrames
			}
		}
	}
	if idr && h.SPS != nil && h.PPS != nil {
		var sets [][]byte
		if !hasSPS {
			sets = append(sets, h.SPS)
		}
		if !hasPPS {
			sets = append(sets, h.PPS)
		}
		au = append(sets, au...)
	}
	data, err := h264.AnnexB(au).Marshal()
	if err != nil {
		return telem.VideoFrame{}, err
	}
	return telem.VideoFrame{
		Keyframe: idr && h.SPS != nil && h.PPS != nil,
		Codec:    telem.VideoCodecH264,
		Data:     data,
	}, nil
}

func isBSlice(nalu []byte) bool {
	header := h264.EmulationPreventionRemove(nalu[1:min(len(nalu), 32)])
	pos := 0
	if _, err := bits.ReadGolombUnsigned(header, &pos); err != nil {
		return false
	}
	sliceType, err := bits.ReadGolombUnsigned(header, &pos)
	return err == nil && sliceType%5 == sliceTypeB
}

// KeyframeWatch follows the keyframes of one stream. The zero value never reports a
// stream. It is not safe for concurrent use.
type KeyframeWatch struct {
	// Limit is the longest time that a stream may go without a keyframe.
	Limit time.Duration
	last  time.Time
	keyed bool
}

// Observe records a frame that arrived at now. It returns false for a frame before the
// first keyframe of the stream, since no decoder can use it. It returns ErrNoKeyframe
// for each frame while the stream has had no keyframe for longer than Limit, counted
// from the first frame it observes.
func (w *KeyframeWatch) Observe(f telem.VideoFrame, now time.Time) (bool, error) {
	w.keyed = w.keyed || f.Keyframe
	if f.Keyframe || w.last.IsZero() {
		w.last = now
	}
	if w.Limit > 0 && now.Sub(w.last) > w.Limit {
		return w.keyed, ErrNoKeyframe
	}
	return w.keyed, nil
}
