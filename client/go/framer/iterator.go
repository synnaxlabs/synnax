package framer

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

const AutoSpan = api.FrameIteratorAutoSpan

type (
	IteratorRequest  = api.FrameIteratorRequest
	IteratorResponse = api.FrameIteratorResponse
	IteratorStream   = freighter.ClientStream[IteratorRequest, IteratorResponse]
	IteratorClient   = freighter.StreamClient[IteratorRequest, IteratorResponse]
	IteratorConfig   struct {
		Keys      channel.Keys
		Bounds    telem.TimeRange
		ChunkSize int64
	}
)

type Iterator struct {
	stream IteratorStream
	value  api.Frame
}

func openIterator(ctx context.Context, client IteratorClient, cfg IteratorConfig) (*Iterator, error) {
	s, err := client.Stream(ctx, "")
	if err != nil {
		return nil, err
	}
	i := &Iterator{stream: s}
	_, err = i.exec(ctx, IteratorRequest{Bounds: cfg.Bounds, ChunkSize: cfg.ChunkSize, Keys: cfg.Keys})
	return i, err
}

func (i *Iterator) Next(ctx context.Context, span telem.TimeSpan) bool {
	v, _ := i.exec(ctx, IteratorRequest{Command: iterator.Next, Span: span})
	return v
}

func (i *Iterator) Prev(ctx context.Context, span telem.TimeSpan) bool {
	v, _ := i.exec(ctx, IteratorRequest{Command: iterator.Prev, Span: span})
	return v
}

func (i *Iterator) SeekFirst(ctx context.Context) bool {
	v, _ := i.exec(ctx, IteratorRequest{Command: iterator.SeekFirst})
	return v
}

func (i *Iterator) SeekLE(ctx context.Context, stamp telem.TimeStamp) bool {
	v, _ := i.exec(ctx, IteratorRequest{Command: iterator.SeekLE, Stamp: stamp})
	return v
}

func (i *Iterator) SeekGE(ctx context.Context, stamp telem.TimeStamp) bool {
	v, _ := i.exec(ctx, IteratorRequest{Command: iterator.SeekGE, Stamp: stamp})
	return v
}

func (i *Iterator) Value() core.Frame {
	return i.value
}

func (i *Iterator) Valid() bool {
	v, _ := i.exec(context.Background(), IteratorRequest{Command: iterator.Valid})
	return v
}

func (i *Iterator) Close() error {
	if err := i.stream.CloseSend(); err != nil {
		return err
	}
	_, err := i.stream.Receive()
	return errors.Skip(err, freighter.EOF)
}

func (i *Iterator) exec(ctx context.Context, req IteratorRequest) (bool, error) {
	if err := i.stream.Send(req); err != nil {
		return false, err
	}
	i.value = core.Frame{}
	for {
		res, err := i.stream.Receive()
		if err != nil {
			return false, err
		}
		if res.Variant == iterator.AckResponse {
			return res.Ack, nil
		}
		i.value = i.value.Extend(res.Frame)
	}
}
