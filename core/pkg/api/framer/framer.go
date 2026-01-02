// Copyright 2025 Synnax Labs, Inc.
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
	"go/types"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/iterator"
	"github.com/synnaxlabs/x/address"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type Frame = framer.Frame

const (
	frameSenderAddr    address.Address = "sender"
	frameReceiverAddr  address.Address = "receiver"
	frameIteratorAddr  address.Address = "iterator"
	framerStreamerAddr address.Address = "streamer"
	frameWriterAddr    address.Address = "writer"
)

type Service struct {
	alamos.Instrumentation
	db       *gorp.DB
	access   *rbac.Service
	Channel  *channel.Service
	Internal *framer.Service
}

func NewService(cfg config.Config) *Service {
	return &Service{
		Instrumentation: cfg.Instrumentation,
		Internal:        cfg.Service.Framer,
		Channel:         cfg.Distribution.Channel,
		db:              cfg.Distribution.DB,
		access:          cfg.Service.RBAC,
	}
}

type DeleteRequest struct {
	Keys   channel.Keys    `json:"keys" msgpack:"keys" validate:"required"`
	Bounds telem.TimeRange `json:"bounds" msgpack:"bounds" validate:"bounds"`
	Names  []string        `json:"names" msgpack:"names" validate:"names"`
}

func (s *Service) Delete(
	ctx context.Context,
	req DeleteRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: framer.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		c := errors.NewCatcher(errors.WithAggregation())
		w := s.Internal.NewDeleter()
		if len(req.Keys) > 0 {
			c.Exec(func() error {
				return w.DeleteTimeRangeMany(ctx, req.Keys, req.Bounds)
			})
		} else if len(req.Names) > 0 {
			c.Exec(func() error {
				return w.DeleteTimeRangeManyByNames(ctx, req.Names, req.Bounds)
			})
		}
		return c.Error()
	})
}

type (
	IteratorRequest  = framer.IteratorRequest
	IteratorResponse = framer.IteratorResponse
	IteratorStream   = freighter.ServerStream[IteratorRequest, IteratorResponse]
)

const (
	iteratorResponseBufferSize = 50
	iteratorRequestBufferSize  = 2
)

func (s *Service) Iterate(ctx context.Context, stream IteratorStream) error {
	iter, err := s.openIterator(ctx, stream)
	if err != nil {
		return err
	}

	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(s.Child("frame_iterator")))
	// Cancellation here would occur for one of two reasons. Either we encounter
	// a fatal error (transport or iterator internal) and we need to free all
	// resources, OR the client executed the close command on the iterator (in
	// which case resources have already been freed and cancel does nothing).
	defer cancel()

	receiver := &freightfluence.Receiver[framer.IteratorRequest]{Receiver: stream}
	sender := &freightfluence.TransformSender[framer.IteratorResponse, framer.IteratorResponse]{
		Sender: freighter.SenderNopCloser[framer.IteratorResponse]{StreamSender: stream},
		Transform: func(ctx context.Context, res framer.IteratorResponse) (framer.IteratorResponse, bool, error) {
			res.Error = errors.Encode(ctx, res.Error, false)
			return res, true, nil
		},
	}
	pipe := plumber.New()
	plumber.SetSegment(pipe, frameIteratorAddr, iter)
	plumber.SetSink(pipe, frameSenderAddr, sender)
	plumber.SetSource(pipe, frameReceiverAddr, receiver)
	plumber.MustConnect[framer.IteratorResponse](pipe, frameIteratorAddr, frameSenderAddr, iteratorResponseBufferSize)
	plumber.MustConnect[framer.IteratorRequest](pipe, frameReceiverAddr, frameIteratorAddr, iteratorRequestBufferSize)

	pipe.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	return sCtx.Wait()
}

func (s *Service) openIterator(ctx context.Context, srv IteratorStream) (framer.StreamIterator, error) {
	req, err := srv.Receive()
	if err != nil {
		return nil, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: framer.OntologyIDs(req.Keys),
	}); err != nil {
		return nil, err
	}
	iter, err := s.Internal.NewStreamIterator(ctx, framer.IteratorConfig{
		Bounds:           req.Bounds,
		Keys:             req.Keys,
		ChunkSize:        req.ChunkSize,
		DownsampleFactor: req.DownsampleFactor,
	})
	if err != nil {
		return nil, err
	}
	return iter, srv.Send(framer.IteratorResponse{Variant: iterator.AckResponse, Ack: true})
}

type (
	StreamerConfig   = framer.StreamerConfig
	StreamerRequest  = framer.StreamerRequest
	StreamerResponse = framer.StreamerResponse
	StreamerStream   = freighter.ServerStream[StreamerRequest, StreamerResponse]
)

const (
	streamingRequestBufferSize  = 5
	streamingResponseBufferSize = 200
)

func (s *Service) Stream(ctx context.Context, stream StreamerStream) error {
	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(s.Child("frame_streamer")))
	defer cancel()
	streamer, err := s.openStreamer(sCtx, auth.GetSubject(ctx), stream)
	if err != nil {
		return err
	}
	var (
		receiver = &freightfluence.Receiver[StreamerRequest]{Receiver: stream}
		sender   = &freightfluence.Sender[StreamerResponse]{
			Sender: freighter.SenderNopCloser[StreamerResponse]{StreamSender: stream},
		}
		pipe = plumber.New()
	)

	plumber.SetSegment(pipe, framerStreamerAddr, streamer)
	plumber.SetSink(pipe, frameSenderAddr, sender)
	plumber.SetSource(pipe, frameReceiverAddr, receiver)
	plumber.MustConnect[StreamerRequest](pipe, frameReceiverAddr, framerStreamerAddr, streamingRequestBufferSize)
	plumber.MustConnect[StreamerResponse](pipe, framerStreamerAddr, frameSenderAddr, streamingResponseBufferSize)
	pipe.Flow(sCtx, confluence.CloseOutputInletsOnExit(), confluence.CancelOnFail())
	return sCtx.Wait()
}

func (s *Service) openStreamer(
	ctx context.Context,
	subject ontology.ID,
	stream StreamerStream,
) (streamer framer.Streamer, err error) {
	req, err := stream.Receive()
	if err != nil {
		return nil, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionRetrieve,
		Objects: framer.OntologyIDs(req.Keys),
	}); err != nil {
		return nil, err
	}
	reader, err := s.Internal.NewStreamer(ctx, req)
	if err != nil {
		return nil, err
	}
	return reader, stream.Send(framer.StreamerResponse{})
}

type WriterConfig struct {
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
	// EnableAutoCommit determines whether the writer will automatically commit after
	// each write. If EnableAutoCommit is true, then the writer will commit after each
	// write, and will flush that commit to index on FS after the specified
	// AutoIndexPersistInterval.
	//
	// [OPTIONAL] - Defaults to false.
	EnableAutoCommit bool `json:"enable_auto_commit" msgpack:"enable_auto_commit"`
	// AutoIndexPersistInterval is the interval at which commits to the index will be persisted.
	// To persist every commit to guarantee minimal loss of data, set AutoIndexPersistInterval
	// to AlwaysAutoPersist.
	// [OPTIONAL] - Defaults to 1s.
	AutoIndexPersistInterval telem.TimeSpan `json:"auto_index_persist_interval" msgpack:"auto_index_persist_interval"`
}

// WriterRequest represents a request to write CreateNet data for a set of channels.
type WriterRequest struct {
	Config  WriterConfig  `json:"config" msgpack:"config"`
	Command WriterCommand `json:"command" msgpack:"command"`
	Frame   Frame         `json:"frame" msgpack:"frame"`
}

type WriterResponse struct {
	Command    writer.Command  `json:"command" msgpack:"command"`
	End        telem.TimeStamp `json:"end" msgpack:"end"`
	Authorized bool            `json:"authorized" msgpack:"authorized"`
	Err        errors.Payload  `json:"err" msgpack:"err"`
}

type (
	WriterCommand = writer.Command
	WriterStream  = freighter.ServerStream[WriterRequest, WriterResponse]
)

const (
	writerResponseBufferSize = 2
	writerRequestBufferSize  = 50
)

// Write exposes a high-level api for writing segmented telemetry to Synnax0
// cluster. The client is expected to send an initial request containing the
// keys of the channels to write to. The server will acquire an exclusive lock on
// these channels. If the channels are already locked, Write will return with
// an error. After sending the initial request, the client is free to send segments.
// The server will route the segments to the appropriate nodes in the cluster,
// persisting them to disk.
//
// If the client cancels the provided context, the server will immediately
// abort all pending writes, release the locks, and return errors.Canceled.
//
// To ensure writes are durable, the client can issue a Close request
// (i.e. calling freighter.ClientStream.close_send()) after sending all segments,
// and then wait for the server to acknowledge the request with a Close response
// of its own.
//
// Concrete api implementations (GRPC, Websocket, etc.) are expected to
// implement the WriterStream interface according to the protocol defined in
// the freighter.StreamServer interface.
//
// When Write returns an error that is not errors.Canceled, the api
// implementation is expected to return a WriterResponse.CloseMsg with the error,
// and then wait for a reasonable amount of time for the client to close the
// connection before forcibly terminating the connection.
func (s *Service) Write(_ctx context.Context, stream WriterStream) error {
	ctx, cancel := signal.WithCancel(_ctx, signal.WithInstrumentation(s.Child("frame_writer")))
	// cancellation here would occur for one of two reasons. Either we encounter
	// a fatal error (transport or writer internal) and we need to free all
	// resources, OR the client executed the close command on the writer (in
	// which case resources have already been freed and cancel does nothing).
	defer cancel()

	w, err := s.openWriter(ctx, auth.GetSubject(_ctx), stream)
	if err != nil {
		return err
	}

	receiver := &freightfluence.TransformReceiver[framer.WriterRequest, WriterRequest]{
		Receiver: stream,
		Transform: func(_ context.Context, req WriterRequest) (framer.WriterRequest, bool, error) {
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
	sender := &freightfluence.TransformSender[framer.WriterResponse, WriterResponse]{
		Sender: freighter.SenderNopCloser[WriterResponse]{StreamSender: stream},
		Transform: func(ctx context.Context, i framer.WriterResponse) (o WriterResponse, ok bool, err error) {
			o.Command = i.Command
			o.Authorized = i.Authorized
			o.Err = errors.Encode(ctx, i.Err, false)
			o.End = i.End
			return o, true, nil
		},
	}

	pipe := plumber.New()

	plumber.SetSegment(pipe, "writer", w)
	plumber.SetSource(pipe, frameReceiverAddr, receiver)
	plumber.SetSink(pipe, frameSenderAddr, sender)
	plumber.MustConnect[framer.WriterRequest](pipe, frameReceiverAddr, frameWriterAddr, writerRequestBufferSize)
	plumber.MustConnect[framer.WriterResponse](pipe, frameWriterAddr, frameSenderAddr, writerResponseBufferSize)

	pipe.Flow(ctx, confluence.CloseOutputInletsOnExit(), confluence.CancelOnFail())
	err = ctx.Wait()
	return err
}

func (s *Service) openWriter(
	ctx context.Context,
	subject ontology.ID,
	srv WriterStream,
) (framer.StreamWriter, error) {
	req, err := srv.Receive()
	if err != nil {
		return nil, err
	}

	if err = s.access.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.ActionCreate,
		Objects: framer.OntologyIDs(req.Config.Keys),
	}); err != nil {
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
		ErrOnUnauthorized:        xconfig.Bool(req.Config.ErrOnUnauthorized),
		EnableAutoCommit:         xconfig.Bool(req.Config.EnableAutoCommit),
		AutoIndexPersistInterval: req.Config.AutoIndexPersistInterval,
	})
	if err != nil {
		return w, err
	}

	channels := make([]channel.Channel, 0, len(req.Config.Keys))
	if err = s.Channel.NewRetrieve().WhereKeys(req.Config.Keys...).Entries(&channels).Exec(ctx, nil); err != nil {
		return w, err
	}
	// Let the client know the writer is ready to receive segments.
	return w, srv.Send(WriterResponse{
		Command: writer.Open,
		Err:     errors.Encode(ctx, nil, false),
	})
}
