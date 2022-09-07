package api

import (
	"context"
	"github.com/arya-analytics/delta/pkg/api/errors"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/stream"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/x/telem"
	roacherrors "github.com/cockroachdb/errors"
	"go/types"
)

type Sample struct {
	ChannelKey string          `json:"channel_key" msgpack:"channel_key"`
	Stamp      telem.TimeStamp `json:"time_stamp" msgpack:"time_stamp"`
	Value      []byte          `json:"value" msgpack:"value"`
}

type StreamWriterRequest struct {
	Samples []Sample `json:"samples" msgpack:"samples"`
}

type StreamReaderRequest struct {
	ChannelKeys []string `json:"channel_keys" msgpack:"channel_keys"`
}

type StreamReaderResponse struct {
	Samples []Sample `json:"samples" msgpack:"samples"`
}

type StreamService struct {
	LoggingProvider
	Internal *stream.Service
}

type (
	SampleWriterStream = freighter.ServerStream[StreamWriterRequest, types.Nil]
	SampleReaderStream = freighter.ServerStream[StreamReaderRequest, StreamReaderResponse]
)

func NewStreamService(prov Provider) *StreamService {
	return &StreamService{
		Internal:        prov.Config.Stream,
		LoggingProvider: prov.Logging,
	}
}

func (s *StreamService) Write(_ctx context.Context, srv SampleWriterStream) errors.Typed {
	writer := s.Internal.NewStreamWriter()

	for {
		req, err := srv.Receive()
		if roacherrors.Is(err, freighter.EOF) {
			writer.Close()
			return errors.Nil
		}
		if err != nil {
			return errors.Unexpected(err)
		}
		var samples []stream.Sample
		for _, sample := range req.Samples {
			key, err := channel.ParseKey(sample.ChannelKey)
			if err != nil {
				return errors.Parse(err)
			}
			samples = append(samples, stream.Sample{
				ChannelKey: key,
				Stamp:      sample.Stamp,
				Value:      sample.Value,
			})
		}
		writer.Inlet() <- samples
	}
}

func (s *StreamService) Read(_ctx context.Context, srv SampleReaderStream) errors.Typed {
	// receive one request, parse the open keys, and open a reader.
	// then read from the reader and send to the stream.

	req, err := srv.Receive()
	if err != nil {
		return errors.Unexpected(err)
	}
	keys, err := channel.ParseKeys(req.ChannelKeys)
	if err != nil {
		return errors.Parse(err)
	}
	reader, closer := s.Internal.NewStreamReader(keys...)
	defer closer.Close()

	for samples := range reader.Outlet() {
		var resp []Sample
		for _, sample := range samples {
			resp = append(resp, Sample{
				ChannelKey: sample.ChannelKey.String(),
				Stamp:      sample.Stamp,
				Value:      sample.Value,
			})
		}
		err = srv.Send(StreamReaderResponse{Samples: resp})
		if err != nil {
			return errors.Unexpected(err)
		}
	}
	return errors.Nil
}
