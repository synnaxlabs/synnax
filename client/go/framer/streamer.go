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

type Streamer struct{ stream StreamerStream }

func openStreamer(ctx context.Context, client StreamerClient, cfg StreamerConfig) (*Streamer, error) {
	s, err := client.Stream(ctx, "")
	if err != nil {
		return nil, err
	}
	if err := s.Send(cfg); err != nil {
		return nil, err
	}
	_, err = s.Receive()
	return &Streamer{stream: s}, err
}

func (s *Streamer) Read() (core.Frame, error) {
	res, err := s.stream.Receive()
	return res.Frame, err
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
