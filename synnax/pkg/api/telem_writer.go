// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type FrameWriterConfig struct {
	Start telem.TimeStamp `json:"start" msgpack:"start"`
	Keys  channel.Keys    `json:"keys" msgpack:"keys"`
}

// FrameWriterRequest represents a request to write Framer data for a set of channels.
type FrameWriterRequest struct {
	Config  FrameWriterConfig `json:"config" msgpack:"config"`
	Command WriterCommand     `json:"command" msgpack:"command"`
	Frame   framer.Frame      `json:"frame" msgpack:"frame"`
}

type (
	WriterCommand       = writer.Command
	FrameWriterResponse = framer.WriteResponse
)

type SegmentWriterStream = freighter.ServerStream[FrameWriterRequest, framer.WriteResponse]

// Write exposes a high level api for writing segmented telemetry to the delta
// cluster. The client is expected to send an initial request containing the
// keys of the channels to write to. The server will acquire an exclusive lock on
// these channels. If the channels are already locked, Write will return with
// an error. After sending the initial request, the client is free to send segments.
// The server will route the segments to the appropriate nodes in the cluster,
// persisting them to disk.
//
// If the client cancels the provided context, the server will immediately
// abort all pending writes, release the locks, and return an errors.Canceled.
//
// To ensure writes are durable, the client can issue a Close request
// (i.e. calling freighter.ClientStream.CloseSend()) after sending all segments,
// and then wait for the server to acknowledge the request with a Close response
// of its own.
//
// Concrete api implementations (GRPC, Websocket, etc.) are expected to
// implement the SegmentWriterStream interface according to the protocol defined in
// the freighter.StreamServer interface.
//
// When Write returns an error that is not errors.Canceled, the api
// implementation is expected to return a FrameWriterResponse.CloseMsg with the error,
// and then wait for a reasonable amount of time for the client to close the
// connection before forcibly terminating the connection.
func (s *TelemService) Write(_ctx context.Context, stream SegmentWriterStream) errors.Typed {
	ctx, cancel := signal.WithCancel(_ctx, signal.WithInstrumentation(s.Instrumentation))
	// cancellation here would occur for one of two reasons. Either we encounter
	// a fatal error (transport or writer internal) and we need to free all
	// resources, OR the client executed the close command on the writer (in
	// which case resources have already been freed and cancel does nothing).
	defer cancel()

	w, err := s.openWriter(ctx, stream)
	if err.Occurred() {
		return err
	}

	receiver := &freightfluence.TransformReceiver[framer.WriteRequest, FrameWriterRequest]{}
	receiver.Transform = func(ctx context.Context, req FrameWriterRequest) (framer.WriteRequest, bool, error) {
		return framer.WriteRequest{Command: req.Command, Frame: req.Frame}, false, nil
	}
	sender := &freightfluence.TransformSender[framer.WriteResponse, framer.WriteResponse]{}
	sender.Transform = func(ctx context.Context, resp framer.WriteResponse) (framer.WriteResponse, bool, error) {
		if resp.Err != nil {
			resp.Err = ferrors.Encode(errors.Unexpected(resp.Err))
		}
		return resp, false, nil
	}

	pipe := plumber.New()

	plumber.SetSegment(pipe, "writer", w)
	plumber.SetSource[framer.WriteRequest](pipe, "receiver", receiver)
	plumber.SetSink(pipe, "sender", sender)
	plumber.MustConnect[framer.WriteRequest](pipe, "receiver", "writer", 1)
	plumber.MustConnect[FrameWriterResponse](pipe, "writer", "sender", 1)

	pipe.Flow(ctx, confluence.CloseInletsOnExit())
	return errors.MaybeUnexpected(ctx.Wait())
}

func (s *TelemService) openWriter(ctx context.Context, srv SegmentWriterStream) (framer.StreamWriter, errors.Typed) {
	req, err := srv.Receive()
	if err != nil {
		return nil, errors.Unexpected(err)
	}
	w, err := s.Framer.NewStreamWriter(ctx, writer.Config{
		Keys: req.Config.Keys,
	})
	if err != nil {
		return nil, errors.Query(err)
	}
	// Let the client know the writer is ready to receive segments.
	return w, errors.MaybeUnexpected(srv.Send(FrameWriterResponse{}))
}
