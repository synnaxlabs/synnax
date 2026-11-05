// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video

import (
	"io"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/fmp4"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/fmp4/seekablebuffer"
	"github.com/bluenviron/mediacommon/v2/pkg/formats/mp4/codecs"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

var (
	// ErrNoParameterSets is returned for a first keyframe that holds no SPS or no PPS,
	// since an MP4 file needs both before its first frame.
	ErrNoParameterSets = errors.New("the first keyframe has no SPS or no PPS")
	// ErrEmptyMP4 is returned by Close for an MP4 that received no keyframe, since a
	// file with no frames is not a valid MP4 file.
	ErrEmptyMP4 = errors.New("the file has no keyframe to start at")
)

// MP4Timescale is the number of ticks in one second of the files that MP4 writes.
const MP4Timescale = 90000

// defaultDuration is the duration of the last frame of a file with only one frame.
const defaultDuration = MP4Timescale / 30

// MP4 writes H.264 frames as a fragmented MP4 file, with no re-encoding. It drops the
// frames before the first keyframe, takes the parameter sets of the file from that
// keyframe, and writes one fragment for each group of pictures. It is not safe for
// concurrent use.
type MP4 struct {
	w     io.Writer
	buf   seekablebuffer.Buffer
	start telem.TimeStamp
	// started is true once the first keyframe is written.
	started  bool
	sequence uint32
	// samples hold the frames of the group of pictures that the next fragment holds,
	// and stamps hold their times.
	samples []*fmp4.Sample
	stamps  []telem.TimeStamp
	// last is the duration of the newest frame of the last fragment.
	last uint32
}

// NewMP4 returns an MP4 that writes to w.
func NewMP4(w io.Writer) *MP4 { return &MP4{w: w} }

// Write adds a frame shown at ts. Frames must arrive in increasing order of time. A
// keyframe writes the fragment of the frames before it.
func (m *MP4) Write(f telem.VideoFrame, ts telem.TimeStamp) error {
	if !m.started && !f.Keyframe {
		return nil
	}
	var au h264.AnnexB
	if err := au.Unmarshal(f.Data); err != nil {
		return err
	}
	if !m.started {
		if err := m.writeInit(au); err != nil {
			return err
		}
		m.started, m.start = true, ts
	}
	if f.Keyframe && len(m.samples) > 0 {
		if err := m.flush(ts); err != nil {
			return err
		}
	}
	payload, err := h264.AVCC(pictureNALUs(au)).Marshal()
	if err != nil {
		return err
	}
	m.samples = append(
		m.samples,
		&fmp4.Sample{IsNonSyncSample: !f.Keyframe, Payload: payload},
	)
	m.stamps = append(m.stamps, ts)
	return nil
}

// Close writes the last fragment. Its last frame lasts as long as the frame before
// it. Close does not close the writer. Close returns ErrEmptyMP4 when no keyframe
// arrived.
func (m *MP4) Close() error {
	if !m.started {
		return ErrEmptyMP4
	}
	end := m.ticks(m.stamps[len(m.stamps)-1])
	if len(m.stamps) > 1 {
		end += end - m.ticks(m.stamps[len(m.stamps)-2])
	} else if m.last > 0 {
		end += uint64(m.last)
	} else {
		end += defaultDuration
	}
	return m.flushTicks(end)
}

func (m *MP4) writeInit(au h264.AnnexB) error {
	var sps, pps []byte
	for _, nalu := range au {
		switch h264.NALUType(nalu[0] & 0x1F) {
		case h264.NALUTypeSPS:
			sps = nalu
		case h264.NALUTypePPS:
			pps = nalu
		}
	}
	if sps == nil || pps == nil {
		return ErrNoParameterSets
	}
	init := fmp4.Init{Tracks: []*fmp4.InitTrack{{
		ID:        1,
		TimeScale: MP4Timescale,
		Codec:     &codecs.H264{SPS: sps, PPS: pps},
	}}}
	return m.write(init.Marshal)
}

func (m *MP4) flush(next telem.TimeStamp) error { return m.flushTicks(m.ticks(next)) }

// flushTicks writes the pending frames as a fragment that ends at end.
func (m *MP4) flushTicks(end uint64) error {
	for i, s := range m.samples {
		next := end
		if i+1 < len(m.stamps) {
			next = m.ticks(m.stamps[i+1])
		}
		s.Duration = uint32(next - m.ticks(m.stamps[i]))
	}
	m.sequence++
	part := fmp4.Part{
		SequenceNumber: m.sequence,
		Tracks: []*fmp4.PartTrack{{
			ID:       1,
			BaseTime: m.ticks(m.stamps[0]),
			Samples:  m.samples,
		}},
	}
	m.last = m.samples[len(m.samples)-1].Duration
	m.samples, m.stamps = nil, nil
	return m.write(part.Marshal)
}

func (m *MP4) write(marshal func(io.WriteSeeker) error) error {
	m.buf.Reset()
	if err := marshal(&m.buf); err != nil {
		return err
	}
	_, err := m.w.Write(m.buf.Bytes())
	return err
}

// ticks converts ts to ticks since the first frame, in two parts so that a long file
// does not overflow.
func (m *MP4) ticks(ts telem.TimeStamp) uint64 {
	span := m.start.Span(ts)
	whole, part := span/telem.Second, span%telem.Second
	return uint64(whole)*MP4Timescale + uint64(part)*MP4Timescale/uint64(telem.Second)
}

// pictureNALUs leaves out the parameter sets, which the file holds once, and the
// access unit delimiters.
func pictureNALUs(au h264.AnnexB) [][]byte {
	out := make([][]byte, 0, len(au))
	for _, nalu := range au {
		switch h264.NALUType(nalu[0] & 0x1F) {
		case h264.NALUTypeSPS, h264.NALUTypePPS, h264.NALUTypeAccessUnitDelimiter:
			continue
		}
		out = append(out, nalu)
	}
	return out
}
