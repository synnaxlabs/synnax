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
	"context"
	"github.com/sirupsen/logrus"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/telem"
)

const defaultBuffer = 25

func NewDownsampledStreamer(ctx context.Context, cfg framer.StreamerConfig, service *framer.Service) (framer.Streamer, error) {
	s, err := service.NewStreamer(ctx, cfg)
	if err != nil {
		return nil, err
	}
	downsampler := &confluence.LinearTransform[framer.StreamerResponse, framer.StreamerResponse]{
		Transform: func(ctx context.Context, i framer.StreamerResponse) (o framer.StreamerResponse, ok bool, err error) {
			i = downsample(ctx, i, cfg.DownsampleFactor)
			return i, true, nil
		},
	}
	pipe := plumber.New()
	plumber.SetSegment[framer.StreamerRequest, framer.StreamerResponse](pipe, "dist-streamer", s)
	plumber.SetSegment[framer.StreamerResponse, framer.StreamerResponse](pipe, "downsampler", downsampler)
	plumber.MustConnect[framer.StreamerResponse](pipe, "dist-streamer", "downsampler", defaultBuffer)
	seg := &plumber.Segment[framer.StreamerRequest, framer.StreamerResponse]{
		Pipeline:         pipe,
		RouteInletsTo:    []address.Address{"dist-streamer"},
		RouteOutletsFrom: []address.Address{"downsampler"},
	}

	return seg, nil
}

func downsample(ctx context.Context, response framer.StreamerResponse, factor int) framer.StreamerResponse {
	dsFrame := framer.Frame{Keys: response.Frame.Keys, Series: []telem.Series{}}

	for _, k := range response.Frame.Keys {
		dsFrame.Series = append(dsFrame.Series, downsampleSeries(response.Frame.Get(k)[0], factor))
	}
	dsResponse := framer.StreamerResponse{Frame: dsFrame, Error: nil}
	return dsResponse
}

func downsampleSeries(series telem.Series, factor int) telem.Series {
	length := len(series.Data)
	if factor <= 1 || length <= factor {
		return series
	}

	seriesLength := (len(series.Data) / factor) // / factor * int(series.DataType.Density())
	downsampledData := make([]byte, 0, seriesLength)

	for i := int64(0); i < series.Len(); i += int64(factor) {
		start := i * int64(series.DataType.Density())
		end := start + int64(series.DataType.Density())
		downsampledData = append(downsampledData, series.Data[start:end]...)
		logrus.Info("i = ", i) //TODO: remove
	}

	downsampledSeries := telem.Series{
		TimeRange: series.TimeRange,
		DataType:  series.DataType,
		Data:      downsampledData,
		Alignment: series.Alignment,
	}
	logrus.Info("original series", telem.Unmarshal[int64](series))               // TODO: remove
	logrus.Info("downsampled series", telem.Unmarshal[int64](downsampledSeries)) // TODO: remove
	return downsampledSeries
}
