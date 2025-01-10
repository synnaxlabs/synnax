package framer

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/errors"
)

type (
	WriterRequest  = api.FrameWriterRequest
	WriterResponse = api.FrameWriterResponse
	WriterStream   = freighter.ClientStream[WriterRequest, WriterResponse]
	WriterClient   = freighter.StreamClient[WriterRequest, WriterResponse]
	WriterConfig   = api.FrameWriterConfig
)

type Writer struct {
	stream WriterStream
}

func openWriter(ctx context.Context, client WriterClient, cfg WriterConfig) (*Writer, error) {
	s, err := client.Stream(ctx, "")
	if err != nil {
		return nil, err
	}
	if err := s.Send(WriterRequest{Config: cfg}); err != nil {
		return nil, err
	}
	return &Writer{stream: s}, nil
}

func (w *Writer) Write(ctx context.Context, frame core.Frame) bool {
	if err := w.stream.Send(WriterRequest{Command: writer.Data, Frame: frame}); err != nil {
		return false
	}
	return true
}

func (w *Writer) Commit(ctx context.Context) bool {
	if err := w.stream.Send(WriterRequest{Command: writer.Commit}); err != nil {
		return false
	}
	for {
		resp, err := w.stream.Receive()
		if err != nil {
			return false
		}
		if resp.Command == writer.Commit {
			return true
		}
	}
}

func (w *Writer) Close(ctx context.Context) error {
	if err := w.stream.CloseSend(); err != nil {
		return err
	}
	for {
		if _, err := w.stream.Receive(); err != nil {
			return errors.Skip(err, freighter.EOF)
		}
	}

}
