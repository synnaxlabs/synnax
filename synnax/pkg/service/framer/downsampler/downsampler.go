// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package downsampler

import (
	"bytes"
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/telem"
)

const defaultBuffer = 25

func NewStreamer(
	ctx context.Context,
	cfg framer.StreamerConfig,
	service *framer.Service,
) (framer.Streamer, error) {
	s, err := service.NewStreamer(ctx, cfg)
	if err != nil {
		return nil, err
	}
	downsampler := &confluence.LinearTransform[
		framer.StreamerResponse,
		framer.StreamerResponse,
	]{
		Transform: func(ctx context.Context, i framer.StreamerResponse) (
			o framer.StreamerResponse,
			ok bool,
			err error,
		) {
			i = downsample(ctx, i, cfg.DownsampleFactor)
			return i, true, nil
		},
	}

	pipe := plumber.New()
	plumber.SetSegment[framer.StreamerRequest, framer.StreamerResponse](
		pipe,
		"dist-streamer",
		s,
	)

	plumber.SetSegment[framer.StreamerResponse, framer.StreamerResponse](
		pipe,
		"downsampler",
		downsampler,
	)
	plumber.MustConnect[framer.StreamerResponse](
		pipe,
		"dist-streamer",
		"downsampler",
		defaultBuffer,
	)
	seg := &plumber.Segment[framer.StreamerRequest, framer.StreamerResponse]{
		Pipeline:         pipe,
		RouteInletsTo:    []address.Address{"dist-streamer"},
		RouteOutletsFrom: []address.Address{"downsampler"},
	}

	return seg, nil
}

func downsample(
	ctx context.Context,
	response framer.StreamerResponse,
	factor int,
) framer.StreamerResponse {
	if factor <= 1 {
		return response
	}
	for i, series := range response.Frame.Series {
		response.Frame.Series[i] = downsampleSeries(series, factor)
	}
	return response
}

func downsampleSeries(series telem.Series, factor int) telem.Series {
	length := len(series.Data)
	if factor <= 1 || length <= factor {
		return series
	}

	if series.DataType.IsVariable() {
		return downsampleVariable(series, factor)
	}

	// Original downsampling logic for non-variable types
	seriesLength := (len(series.Data) / factor)
	downsampledData := make([]byte, 0, seriesLength)
	for i := int64(0); i < series.Len(); i += int64(factor) {
		start := i * int64(series.DataType.Density())
		end := start + int64(series.DataType.Density())
		downsampledData = append(downsampledData, series.Data[start:end]...)
	}

	downsampledSeries := telem.Series{
		TimeRange: series.TimeRange,
		DataType:  series.DataType,
		Data:      downsampledData,
		Alignment: series.Alignment,
	}
	return downsampledSeries
}

func downsampleVariable(series telem.Series, factor int) telem.Series {
	lines := bytes.Split(series.Data, []byte("\n"))
	downsampledLines := make([][]byte, 0, len(lines)/factor+1)

	for i := 0; i < len(lines); i += factor {
		if i < len(lines) {
			downsampledLines = append(downsampledLines, lines[i])
		}
	}

	downsampledData := bytes.Join(downsampledLines, []byte("\n"))

	downsampledSeries := telem.Series{
		TimeRange: series.TimeRange,
		DataType:  series.DataType,
		Data:      downsampledData,
		Alignment: series.Alignment,
	}
	return downsampledSeries
}
