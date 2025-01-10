package framer

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/x/errors"
)

type (
	StreamerRequest  = api.FrameStreamerRequest
	StreamerResponse = api.FrameStreamerResponse
	StreamerStream   = freighter.ClientStream[StreamerRequest, StreamerResponse]
	StreamerClient   = freighter.StreamClient[StreamerRequest, StreamerResponse]
	StreamerConfig   = api.FrameStreamerConfig
)

type Streamer struct {
	stream StreamerStream
}

func openStreamer(ctx context.Context, client StreamerClient, cfg StreamerConfig) (*Streamer, error) {
	s, err := client.Stream(ctx, "")
	if err != nil {
		return nil, err
	}
	if err := s.Send(cfg); err != nil {
		return nil, err
	}
	return &Streamer{stream: s}, nil
}

func (s *Streamer) Read() core.Frame {
	res, _ := s.stream.Receive()
	return res.Frame
}

func (s *Streamer) Close() error {
	if err := s.stream.CloseSend(); err != nil {
		return err
	}
	for {
		if _, err := s.stream.Receive(); err != nil {
			return errors.Skip(err, freighter.EOF)
		}
	}
}
