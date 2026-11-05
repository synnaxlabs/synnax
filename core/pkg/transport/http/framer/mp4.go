// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"bytes"
	"context"
	"io"

	"github.com/synnaxlabs/x/errors"
	xhttp "github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/video"
)

// MP4Encoder encodes a read of one video channel as a fragmented MP4 file, with no
// re-encoding. The file starts at the first keyframe in the time range. A read of
// anything other than one video channel and its index, a read with a reduction, or a
// read with no keyframe in the time range fails before the first byte. A read that
// fails after the first byte truncates the body.
var MP4Encoder mp4Encoder

type mp4Encoder struct{}

var _ xhttp.Encoder = mp4Encoder{}

// ContentType implements http.Encoder.
func (mp4Encoder) ContentType() string { return "video/mp4" }

// Encode implements encoding.Encoder. It holds the whole file in memory, so prefer
// EncodeStream.
func (e mp4Encoder) Encode(ctx context.Context, value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := e.EncodeStream(ctx, &buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeStream implements encoding.Encoder. value must be a ReadResponse.
func (mp4Encoder) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	res, err := readResponse(value)
	if err != nil {
		return err
	}
	if res.Reduction.Variant != nil {
		return errors.Wrap(
			validate.ErrValidation,
			"an MP4 read cannot take a reduction",
		)
	}
	groups, err := groupByIndex(res)
	if err != nil {
		return err
	}
	if len(groups) != 1 || len(groups[0]) != 2 ||
		groups[0][1].DataType != telem.VideoT {
		return errors.Wrap(
			validate.ErrValidation,
			"an MP4 read takes exactly one video channel",
		)
	}
	index, frames := newColumn(groups[0][0]), newColumn(groups[0][1])
	columns := []*column{&index, &frames}
	if err = openColumns(ctx, res, columns); err != nil {
		return err
	}
	m := video.NewMP4(w)
	err = errors.Combine(writeMP4(m, &index, &frames), closeColumns(columns))
	if err != nil {
		return err
	}
	if err = m.Close(); errors.Is(err, video.ErrEmptyMP4) {
		return errors.Wrapf(
			validate.ErrValidation,
			"channel %s has no keyframe in the time range",
			frames.name,
		)
	}
	return err
}

// writeMP4 writes every frame of the frames column at the time of its index sample.
func writeMP4(m *video.MP4, index, frames *column) error {
	for {
		sample, n := frames.sample()
		if n == 0 {
			return nil
		}
		ts, err := indexTime(index, frames)
		if err != nil {
			return err
		}
		f, err := telem.DecodeVideoFrame(sample)
		if err != nil {
			return err
		}
		if err = m.Write(f, ts); err != nil {
			return err
		}
		frames.advance(n)
	}
}

// indexTime moves the index column to the alignment of the sample that the frames
// column points at, and returns the timestamp there.
func indexTime(index, frames *column) (telem.TimeStamp, error) {
	alignment := frames.alignment()
	for {
		sample, n := index.sample()
		if n == 0 {
			return 0, frames.unindexedError()
		}
		if a := index.alignment(); a > alignment {
			return 0, frames.unindexedError()
		} else if a == alignment {
			return telem.TimeStamp(telem.ByteOrder.Uint64(sample)), nil
		}
		index.advance(n)
	}
}
