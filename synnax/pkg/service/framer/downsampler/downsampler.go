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
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type DownSampler = framer.Streamer

const defaultBuffer = 25

type downsampledStreamer struct {
	downsampleFactor int
	confluence.AbstractUnarySink[framer.StreamerRequest]
	confluence.AbstractUnarySource[framer.StreamerResponse]
	streamer    framer.Streamer
	downsampler confluence.Downsampler[framer.StreamerResponse]
}

func NewDownsampledStreamer(ctx context.Context, cfg framer.StreamerConfig, service *framer.Service) (framer.Streamer, error) {
	s, err := service.NewStreamer(ctx, cfg)
	if err != nil {
		return nil, err
	}
	downsampledStreamer := &downsampledStreamer{
		downsampleFactor: cfg.DownsampleFactor,
		streamer:         s,
	}
	// requests stream --> downsampledStreamer --> streamer
	downsampledStreamer.streamer.InFrom(downsampledStreamer.In)
	downsampledStreamer.downsampler = confluence.Downsampler[framer.StreamerResponse]{Downsample: downsample, DownsamplingFactor: downsampledStreamer.downsampleFactor}
	responses := confluence.NewStream[framer.StreamerResponse](defaultBuffer)

	// streamer --> internal responses stream --> downsampler -->downsampledStreamer.Out (inlet)
	downsampledStreamer.streamer.OutTo(responses)
	downsampledStreamer.downsampler.InFrom(responses)
	downsampledStreamer.downsampler.OutTo(downsampledStreamer.Out)
	return framer.Streamer(downsampledStreamer), nil
}

func (d *downsampledStreamer) Flow(sCtx signal.Context, opts ...confluence.Option) {
	d.downsampler.Flow(sCtx, opts...)
	d.streamer.Flow(sCtx, opts...)
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
