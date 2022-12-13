package api

import (
	"context"
	roacherrors "github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type FrameWriterConfig struct {
	Start telem.TimeStamp `json:"start" msgpack:"start"`
	Keys  []string        `json:"keys" msgpack:"keys"`
}

// FrameWriterRequest represents a request to write Framer data for a set of channels.
type FrameWriterRequest struct {
	Command WriterCommand     `json:"command" msgpack:"command"`
	Config  FrameWriterConfig `json:"config" msgpack:"config"`
	// Segments is the slice of segments to write. The segments must have keys that are
	// elements of OpenKeys. The Segments field will be ignored in the first request to
	// the server, and will only be used once an OpenKeys request has been issued.
	Frame Frame `json:"frame" msgpack:"frame"`
}

type (
	WriterCommand = writer.Command
)

type FrameWriterResponse struct {
	Command WriterCommand `json:"command" msgpack:"command"`
	Ack     bool          `json:"ack" msgpack:"ack"`
	// Err is a transient error encountered during writer operation, such as an invalid
	// Framer data type or Channel key.
	Err ferrors.Payload `json:"error" msgpack:"error"`
}

type SegmentWriterStream = freighter.ServerStream[FrameWriterRequest, FrameWriterResponse]

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
func (s *FrameService) Write(_ctx context.Context, stream SegmentWriterStream) errors.Typed {
	ctx, cancel := signal.WithCancel(_ctx, signal.WithLogger(s.logger.Desugar()))
	// cancellation here would occur for one of two reasons. Either we encounter
	// a fatal error (transport or writer internal) and we need to free all
	// resources, OR the client executed the close command on the writer (in
	// which case resources have already been freed and cancel does nothing).
	defer cancel()

	w, err := s.openWriter(ctx, stream)
	if err.Occurred() {
		return err
	}
	requests := confluence.NewStream[writer.Request]()
	w.InFrom(requests)
	responses := confluence.NewStream[writer.Response]()
	w.OutTo(responses)
	w.Flow(ctx,
		confluence.CloseInletsOnExit(),
		confluence.CancelOnExitErr(),
	)

	parseErrors := make(chan errors.Typed)
	go func() {
		for {
			req, err := stream.Receive()
			// We accept both a close request or a transport EOF as valid
			// signatures for closing the writer. Any other error is considered abnormal
			// and results in immediate cancellation and freeing of all resources.
			if roacherrors.Is(err, freighter.EOF) {
				requests.Close()
				return
			}
			if err != nil {
				cancel()
				s.logger.Error(err)
				return
			}
			frame, tErr := toDistributionFrame(req.Frame)
			if tErr.Occurred() {
				parseErrors <- tErr
				continue
			}
			requests.Inlet() <- writer.Request{Command: req.Command, Frame: frame}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return errors.Canceled
		case err := <-parseErrors:
			if err := stream.Send(FrameWriterResponse{Err: ferrors.Encode(err)}); err != nil {
				return errors.Unexpected(err)
			}
		case resp, ok := <-responses.Outlet():
			if !ok {
				return errors.Nil
			}
			if err := stream.Send(FrameWriterResponse{
				Command: resp.Command,
				Ack:     resp.Ack,
				Err:     ferrors.Encode(errors.MaybeGeneral(resp.Err)),
			}); err != nil {
				return errors.Unexpected(err)
			}
		}
	}
}

func (s *FrameService) openWriter(ctx context.Context, srv SegmentWriterStream) (framer.StreamWriter, errors.Typed) {
	cfg, _err := receiveWriterOpenConfig(srv)
	if _err.Occurred() {
		return nil, _err
	}
	w, err := s.Internal.NewStreamWriter(ctx, cfg)
	if err != nil {
		return nil, errors.Query(err)
	}
	// Let the client know the writer is ready to receive segments.
	return w, errors.MaybeUnexpected(srv.Send(FrameWriterResponse{}))
}

func receiveWriterOpenConfig(
	srv SegmentWriterStream,
) (cfg framer.WriterConfig, _ errors.Typed) {
	req, err := srv.Receive()
	if err != nil {
		return cfg, errors.Unexpected(err)
	}
	keys, err := channel.ParseKeys(req.Config.Keys)
	if err != nil {
		return cfg, errors.Validation(errors.Field{Field: "config.openKeys", Message: err.Error()})
	}
	cfg.Keys = keys
	cfg.Start = req.Config.Start
	return cfg, errors.Nil
}
