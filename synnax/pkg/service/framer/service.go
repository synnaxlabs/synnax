// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/telem"
)

type Service struct {
	Internal *framer.Service // distribution layer frame service
}

// got these from api/framer.go calls to Internal
func (s *Service) OpenIterator(ctx context.Context, cfg framer.IteratorConfig) (*framer.Iterator, error) {
	return s.Internal.OpenIterator(ctx, cfg)
}

func (s *Service) NewStreamIterator(ctx context.Context, cfg framer.IteratorConfig) (framer.StreamIterator, error) {
	return s.Internal.NewStreamIterator(ctx, cfg)
}

func (s *Service) NewStreamWriter(ctx context.Context, cfg framer.WriterConfig) (framer.StreamWriter, error) {
	return s.Internal.NewStreamWriter(ctx, cfg)
}

func (s *Service) NewDeleter() framer.Deleter {
	return s.Internal.NewDeleter()
}

func (s *Service) NewStreamer(ctx context.Context, cfg framer.StreamerConfig) (framer.Streamer, error) {
	return s.Internal.NewStreamer(ctx, cfg)
}

func NewService(framerSvc *framer.Service) (*Service, error) {
	return &Service{
		Internal: framerSvc,
	}, nil
}

func downSample(ctx context.Context, response framer.StreamerResponse, factors map[channel.Key]int) framer.StreamerResponse {
	dsFrame := framer.Frame{Keys: response.Frame.Keys, Series: []telem.Series{}}

	// how to get the key from the frame
	for _, k := range response.Frame.Keys {
		dsFrame.Series = append(dsFrame.Series, downSampleSeries(response.Frame.Get(k)[0], factors[k]))
	}
	dsResponse := framer.StreamerResponse{Frame: dsFrame, Error: nil}
	return dsResponse
}

func downSampleSeries(series telem.Series, factor int) telem.Series {
	// print function has been entered
	length := len(series.Data)
	if factor <= 1 || length <= factor {
		return series
	}

	seriesLength := (len(series.Data) / factor) // / factor * int(series.DataType.Density())
	downSampledData := make([]byte, 0, seriesLength)

	for i := int64(0); i < series.Len(); i += int64(factor) {
		start := i * int64(series.DataType.Density())
		end := start + int64(series.DataType.Density())
		downSampledData = append(downSampledData, series.Data[start:end]...)
		logrus.Info("i = ", i)
	}

	downSampledSeries := telem.Series{
		TimeRange: series.TimeRange,
		DataType:  series.DataType,
		Data:      downSampledData,
		Alignment: series.Alignment,
	}
	logrus.Info("original series", telem.Unmarshal[int64](series))
	logrus.Info("downsampled series", telem.Unmarshal[int64](downSampledSeries))
	return downSampledSeries
}
