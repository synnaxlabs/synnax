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
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type FrameWriterConfig struct {
	// Authorities is the authority to use when writing to the channels. We set this
	// as an int and not control.Authorities because msgpack has a tough time decoding
	// lists of uint8.
	Authorities []uint32 `json:"authorities" msgpack:"authorities"`
	// ControlSubject is an identifier for the writer.
	ControlSubject control.Subject `json:"control_subject" msgpack:"control_subject"`
	// Start marks the starting timestamp of the first sample in the first frame. If
	// telemetry occupying the given timestamp already exists for the provided keys,
	// the writer will fail to open.
	// [REQUIRED]
	Start telem.TimeStamp `json:"start" msgpack:"start"`
	// Keys is keys to write to. At least one key must be provided. All keys must
	// have the same data rate OR the same index. All Frames written to the Writer must
	// have an array specified for each key, and all series must be the same length (i.e.
	// calls to Frame.Even must return true).
	// [REQUIRED]
	Keys channel.Keys `json:"keys" msgpack:"keys"`
	// Mode sets the persistence and streaming mode for the writer. The default mode is
	// WriterModePersistStream. See the ts.WriterMode documentation for more.
	// [OPTIONAL]
	Mode writer.Mode `json:"mode" msgpack:"mode"`
	// ErrOnUnauthorized controls whether the writer will return an error when
	// attempting to write to a channel that it does not have authority over.
	// In non-control scenarios, this value should be set to true. In scenarios
	// that require control handoff, this value should be set to false.
	// [OPTIONAL] - Defaults to false.
	ErrOnUnauthorized bool `json:"err_on_unauthorized" msgpack:"err_on_unauthorized"`
	// EnableAutoCommit determines whether the writer will automatically commit after each write.
	// If EnableAutoCommit is true, then the writer will commit after each write, and will
	// flush that commit to index on FS after the specified AutoIndexPersistInterval.
	// [OPTIONAL] - Defaults to false.
	EnableAutoCommit bool `json:"enable_auto_commit" msgpack:"enable_auto_commit"`
	// AutoIndexPersistInterval is the interval at which commits to the index will be persisted.
	// To persist every commit to guarantee minimal loss of data, set AutoIndexPersistInterval
	// to AlwaysAutoPersist.
	// [OPTIONAL] - Defaults to 1s.
	AutoIndexPersistInterval telem.TimeSpan `json:"auto_index_persist_interval" msgpack:"auto_index_persist_interval"`
}

// FrameWriterRequest represents a request to write CreateNet data for a set of channels.
type FrameWriterRequest struct {
	Config  FrameWriterConfig `json:"config" msgpack:"config"`
	Command WriterCommand     `json:"command" msgpack:"command"`
	Frame   framer.Frame      `json:"frame" msgpack:"frame"`
}

type (
	WriterCommand       = writer.Command
	FrameWriterResponse = framer.WriterResponse
	FrameWriterStream   = freighter.ServerStream[FrameWriterRequest, FrameWriterResponse]
)

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
// implement the FrameWriterStream interface according to the protocol defined in
// the freighter.StreamServer interface.
//
// When Write returns an error that is not errors.Canceled, the api
// implementation is expected to return a FrameWriterResponse.CloseMsg with the error,
// and then wait for a reasonable amount of time for the client to close the
// connection before forcibly terminating the connection.
func (s *FrameService) Write(_ctx context.Context, stream FrameWriterStream) error {
	ctx, cancel := signal.WithCancel(_ctx, signal.WithInstrumentation(s.Instrumentation.Child("frame_writer")))
	// cancellation here would occur for one of two reasons. Either we encounter
	// a fatal error (transport or writer internal) and we need to free all
	// resources, OR the client executed the close command on the writer (in
	// which case resources have already been freed and cancel does nothing).
	defer cancel()

	w, err := s.openWriter(ctx, stream)
	if err != nil {
		return err
	}

	receiver := &freightfluence.TransformReceiver[framer.WriterRequest, FrameWriterRequest]{
		Receiver: stream,
		Transform: func(_ context.Context, req FrameWriterRequest) (framer.WriterRequest, bool, error) {
			r := framer.WriterRequest{Command: req.Command, Frame: req.Frame}
			if r.Command == writer.SetAuthority {
				// We decode like this because msgpack has a tough time decoding slices of uint8.
				r.Config.Authorities = make([]control.Authority, len(req.Config.Authorities))
				for i, a := range req.Config.Authorities {
					r.Config.Authorities[i] = control.Authority(a)
				}
				r.Config.Keys = req.Config.Keys
			}
			return r, true, nil
		},
	}
	sender := &freightfluence.TransformSender[framer.WriterResponse, framer.WriterResponse]{
		Sender: freighter.SenderNopCloser[framer.WriterResponse]{StreamSender: stream},
		Transform: func(ctx context.Context, resp framer.WriterResponse) (framer.WriterResponse, bool, error) {
			if resp.Error != nil {
				resp.Error = errors.Encode(ctx, resp.Error, false)
			}
			return resp, true, nil
		},
	}

	pipe := plumber.New()

	plumber.SetSegment(pipe, "writer", w)
	plumber.SetSource[framer.WriterRequest](pipe, "receiver", receiver)
	plumber.SetSink[framer.WriterResponse](pipe, "sender", sender)
	plumber.MustConnect[framer.WriterRequest](pipe, "receiver", "writer", 1)
	plumber.MustConnect[FrameWriterResponse](pipe, "writer", "sender", 1)

	pipe.Flow(ctx, confluence.CloseInletsOnExit(), confluence.RecoverWithErrOnPanic())
	err = ctx.Wait()
	return err
}

func (s *FrameService) openWriter(ctx context.Context, srv FrameWriterStream) (framer.StreamWriter, error) {
	req, err := srv.Receive()
	if err != nil {
		return nil, err
	}

	authorities := make([]control.Authority, len(req.Config.Authorities))
	for i, a := range req.Config.Authorities {
		authorities[i] = control.Authority(a)
	}

	w, err := s.Internal.NewStreamWriter(ctx, writer.Config{
		ControlSubject:           req.Config.ControlSubject,
		Start:                    req.Config.Start,
		Keys:                     req.Config.Keys,
		Authorities:              authorities,
		Mode:                     req.Config.Mode,
		ErrOnUnauthorized:        config.Bool(req.Config.ErrOnUnauthorized),
		EnableAutoCommit:         config.Bool(req.Config.EnableAutoCommit),
		AutoIndexPersistInterval: req.Config.AutoIndexPersistInterval,
	})
	if err != nil {
		return nil, err
	}
	// Let the client know the writer is ready to receive segments.
	return w, srv.Send(FrameWriterResponse{
		Command: writer.Open,
		Ack:     true,
	})
}
