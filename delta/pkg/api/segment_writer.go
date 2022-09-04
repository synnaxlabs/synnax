package api

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/api/errors"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/segment"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/freighter/ferrors"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
	roacherrors "github.com/cockroachdb/errors"
)

// WriterRequest represents a request to write segment data for a set of channels.
type WriterRequest struct {
	// OpenKeys is a slice of channel keys that the client plans to write to.
	// OpenKeys should only be specified in the	first request to the server, and will be
	// ignored in future requests.
	OpenKeys []string `json:"open_keys" msgpack:"open_keys"`
	// Segments is the slice of segments to write. The segments must have keys that are
	// elements of OpenKeys. The Segments field will be ignored in the first request to
	// the server, and will only be used once an OpenKeys request has been issued.
	Segments []Segment `json:"segments" msgpack:"segments"`
}

type WriterResponse struct {
	// Ack is used to acknowledge requests issued by the client.
	Ack bool `json:"ack" msgpack:"ack"`
	// Err is a transient error encountered during writer operation, such as an invalid
	// segment data type or channel key.
	Err ferrors.Payload `json:"error" msgpack:"error"`
}

type WriterStream = freighter.ServerStream[WriterRequest, WriterResponse]

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
// implement the WriterStream interface according to the protocol defined in
// the freighter.StreamTransportServer interface.
//
// When Write returns an error that is not errors.Canceled, the api
// implementation is expected to return a WriterResponse.CloseMsg with the error,
// and then wait for a reasonable amount of time for the client to close the
// connection before forcibly terminating the connection.
func (s *SegmentService) Write(_ctx context.Context, stream WriterStream) errors.Typed {
	ctx, cancel := signal.WithCancel(_ctx, signal.WithLogger(s.Logger.Desugar()))
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
				s.Logger.Error(err)
				return
			}
			segments, tErr := translateSegments(req.Segments)
			if tErr.Occurred() {
				parseErrors <- tErr
				continue
			}
			requests.Inlet() <- writer.Request{Segments: segments}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return errors.Canceled
		case err := <-parseErrors:
			if err := stream.Send(WriterResponse{Err: ferrors.Encode(err)}); err != nil {
				return errors.Unexpected(err)
			}
		case resp, ok := <-responses.Outlet():
			if !ok {
				return errors.Nil
			}
			if err := stream.Send(WriterResponse{
				Err: ferrors.Encode(errors.General(resp.Error)),
			}); err != nil {
				return errors.Unexpected(err)
			}
		}
	}
}

func translateSegments(seg []Segment) ([]core.Segment, errors.Typed) {
	segments := make([]core.Segment, len(seg))
	for i, seg := range seg {
		ch, err := channel.ParseKey(seg.ChannelKey)
		if err != nil {
			return nil, errors.Validation(errors.Field{
				Field:   "channel_key",
				Message: err.Error(),
			})
		}
		segments[i] = core.Segment{ChannelKey: ch, Segment: cesium.Segment{Start: seg.Start, Data: seg.Data}}
	}
	return segments, errors.Nil
}

func (s *SegmentService) openWriter(ctx context.Context, srv WriterStream) (segment.StreamWriter, errors.Typed) {
	keys, _err := receiveWriterOpenArgs(srv)
	if _err.Occurred() {
		return nil, _err
	}
	w, err := s.Internal.NewStreamWriter(ctx, keys...)
	if err != nil {
		return nil, errors.Query(err)
	}
	// Let the client know the writer is ready to receive segments.
	return w, errors.MaybeUnexpected(srv.Send(WriterResponse{Ack: true}))
}

func receiveWriterOpenArgs(srv WriterStream) (channel.Keys, errors.Typed) {
	req, err := srv.Receive()
	if err != nil {
		return nil, errors.Unexpected(err)
	}
	keys, err := channel.ParseKeys(req.OpenKeys)
	if err != nil {
		return nil, errors.Validation(errors.Field{Field: "open_keys", Message: err.Error()})
	}
	if len(keys) == 0 {
		return nil, errors.Validation(errors.Field{
			Field:   "open_keys",
			Message: "must contain at least one key",
		})
	}
	return keys, errors.Nil
}
