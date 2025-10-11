// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"go/types"
	"io"
	"reflect"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	framercodec "github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/address"
	xbinary "github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/httputil"
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

type FrameService struct {
	alamos.Instrumentation
	authProvider
	dbProvider
	accessProvider
	Channel  channel.Readable
	Internal *framer.Service
}

func NewFrameService(p Provider) *FrameService {
	return &FrameService{
		Instrumentation: p.Instrumentation,
		Internal:        p.Service.Framer,
		Channel:         p.Distribution.Channel,
		authProvider:    p.auth,
		dbProvider:      p.db,
		accessProvider:  p.access,
	}
}

type FrameDeleteRequest struct {
	Keys   channel.Keys    `json:"keys" msgpack:"keys" validate:"required"`
	Bounds telem.TimeRange `json:"bounds" msgpack:"bounds" validate:"bounds"`
	Names  []string        `json:"names" msgpack:"names" validate:"names"`
}

func (s *FrameService) FrameDelete(
	ctx context.Context,
	req FrameDeleteRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: framer.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
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
	FrameIteratorRequest  = framer.IteratorRequest
	FrameIteratorResponse = framer.IteratorResponse
	FrameIteratorStream   = freighter.ServerStream[FrameIteratorRequest, FrameIteratorResponse]
)

const (
	iteratorResponseBufferSize = 50
	iteratorRequestBufferSize  = 2
)

func (s *FrameService) Iterate(ctx context.Context, stream FrameIteratorStream) error {
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

	receiver := &freightfluence.Receiver[iterator.Request]{Receiver: stream}
	sender := &freightfluence.TransformSender[iterator.Response, iterator.Response]{
		Sender: freighter.SenderNopCloser[iterator.Response]{StreamSender: stream},
		Transform: func(ctx context.Context, res iterator.Response) (iterator.Response, bool, error) {
			res.Error = errors.Encode(ctx, res.Error, false)
			return res, true, nil
		},
	}
	pipe := plumber.New()
	plumber.SetSegment(pipe, frameIteratorAddr, iter)
	plumber.SetSink[iterator.Response](pipe, frameSenderAddr, sender)
	plumber.SetSource[iterator.Request](pipe, frameReceiverAddr, receiver)
	plumber.MustConnect[iterator.Response](pipe, frameIteratorAddr, frameSenderAddr, iteratorResponseBufferSize)
	plumber.MustConnect[iterator.Request](pipe, frameReceiverAddr, frameIteratorAddr, iteratorRequestBufferSize)

	pipe.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	return sCtx.Wait()
}

func (s *FrameService) openIterator(ctx context.Context, srv FrameIteratorStream) (framer.StreamIterator, error) {
	req, err := srv.Receive()
	if err != nil {
		return nil, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: framer.OntologyIDs(req.Keys),
	}); err != nil {
		return nil, err
	}
	iter, err := s.Internal.NewStreamIterator(ctx, framer.IteratorConfig{
		Bounds:    req.Bounds,
		Keys:      req.Keys,
		ChunkSize: req.ChunkSize,
	})
	if err != nil {
		return nil, err
	}
	return iter, srv.Send(framer.IteratorResponse{Variant: iterator.AckResponse, Ack: true})
}

type (
	FrameStreamerConfig   = framer.StreamerConfig
	FrameStreamerRequest  = framer.StreamerRequest
	FrameStreamerResponse = framer.StreamerResponse
	StreamerStream        = freighter.ServerStream[FrameStreamerRequest, FrameStreamerResponse]
)

const (
	streamingRequestBufferSize  = 5
	streamingResponseBufferSize = 200
)

func (s *FrameService) Stream(ctx context.Context, stream StreamerStream) error {
	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(s.Child("frame_streamer")))
	defer cancel()
	streamer, err := s.openStreamer(sCtx, getSubject(ctx), stream)
	if err != nil {
		return err
	}
	var (
		receiver = &freightfluence.Receiver[FrameStreamerRequest]{Receiver: stream}
		sender   = &freightfluence.Sender[FrameStreamerResponse]{
			Sender: freighter.SenderNopCloser[FrameStreamerResponse]{StreamSender: stream},
		}
		pipe = plumber.New()
	)

	plumber.SetSegment[FrameStreamerRequest, FrameStreamerResponse](pipe, framerStreamerAddr, streamer)
	plumber.SetSink[FrameStreamerResponse](pipe, frameSenderAddr, sender)
	plumber.SetSource[FrameStreamerRequest](pipe, frameReceiverAddr, receiver)
	plumber.MustConnect[FrameStreamerRequest](pipe, frameReceiverAddr, framerStreamerAddr, streamingRequestBufferSize)
	plumber.MustConnect[FrameStreamerResponse](pipe, framerStreamerAddr, frameSenderAddr, streamingResponseBufferSize)
	pipe.Flow(sCtx, confluence.CloseOutputInletsOnExit(), confluence.CancelOnFail())
	return sCtx.Wait()
}

func (s *FrameService) openStreamer(
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
		Action:  access.Retrieve,
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
	ErrOnUnauthorized *bool `json:"err_on_unauthorized" msgpack:"err_on_unauthorized"`
	// EnableAutoCommit determines whether the writer will automatically commit after each write.
	// If EnableAutoCommit is true, then the writer will commit after each write, and will
	// flush that commit to index on FS after the specified AutoIndexPersistInterval.
	// [OPTIONAL] - Defaults to false.
	EnableAutoCommit *bool `json:"enable_auto_commit" msgpack:"enable_auto_commit"`
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
	Frame   Frame             `json:"frame" msgpack:"frame"`
}

type FrameWriterResponse struct {
	Command    writer.Command  `json:"command" msgpack:"command"`
	End        telem.TimeStamp `json:"end" msgpack:"end"`
	Authorized bool            `json:"authorized" msgpack:"authorized"`
	Err        errors.Payload  `json:"err" msgpack:"err"`
}

type (
	WriterCommand     = writer.Command
	FrameWriterStream = freighter.ServerStream[FrameWriterRequest, FrameWriterResponse]
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
// implement the FrameWriterStream interface according to the protocol defined in
// the freighter.StreamServer interface.
//
// When Write returns an error that is not errors.Canceled, the api
// implementation is expected to return a FrameWriterResponse.CloseMsg with the error,
// and then wait for a reasonable amount of time for the client to close the
// connection before forcibly terminating the connection.
func (s *FrameService) Write(_ctx context.Context, stream FrameWriterStream) error {
	ctx, cancel := signal.WithCancel(_ctx, signal.WithInstrumentation(s.Child("frame_writer")))
	// cancellation here would occur for one of two reasons. Either we encounter
	// a fatal error (transport or writer internal) and we need to free all
	// resources, OR the client executed the close command on the writer (in
	// which case resources have already been freed and cancel does nothing).
	defer cancel()

	w, err := s.openWriter(ctx, getSubject(_ctx), stream)
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
	sender := &freightfluence.TransformSender[framer.WriterResponse, FrameWriterResponse]{
		Sender: freighter.SenderNopCloser[FrameWriterResponse]{StreamSender: stream},
		Transform: func(ctx context.Context, i framer.WriterResponse) (o FrameWriterResponse, ok bool, err error) {
			o.Command = i.Command
			o.Authorized = i.Authorized
			o.Err = errors.Encode(ctx, i.Err, false)
			o.End = i.End
			return o, true, nil
		},
	}

	pipe := plumber.New()

	plumber.SetSegment(pipe, "writer", w)
	plumber.SetSource[framer.WriterRequest](pipe, frameReceiverAddr, receiver)
	plumber.SetSink[framer.WriterResponse](pipe, frameSenderAddr, sender)
	plumber.MustConnect[framer.WriterRequest](pipe, frameReceiverAddr, frameWriterAddr, writerRequestBufferSize)
	plumber.MustConnect[framer.WriterResponse](pipe, frameWriterAddr, frameSenderAddr, writerResponseBufferSize)

	pipe.Flow(ctx, confluence.CloseOutputInletsOnExit(), confluence.CancelOnFail())
	err = ctx.Wait()
	return err
}

func (s *FrameService) openWriter(
	ctx context.Context,
	subject ontology.ID,
	srv FrameWriterStream,
) (framer.StreamWriter, error) {
	req, err := srv.Receive()
	if err != nil {
		return nil, err
	}

	if err = s.access.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.Create,
		Objects: framer.OntologyIDs(req.Config.Keys),
	}); err != nil {
		return nil, err
	}

	authorities := make([]control.Authority, len(req.Config.Authorities))
	for i, a := range req.Config.Authorities {
		authorities[i] = control.Authority(a)
	}

	errOnUnauthorized := false
	if req.Config.ErrOnUnauthorized != nil {
		errOnUnauthorized = *req.Config.ErrOnUnauthorized
	}
	enableAutoCommit := false
	if req.Config.EnableAutoCommit != nil {
		enableAutoCommit = *req.Config.EnableAutoCommit
	}

	w, err := s.Internal.NewStreamWriter(ctx, writer.Config{
		ControlSubject:           req.Config.ControlSubject,
		Start:                    req.Config.Start,
		Keys:                     req.Config.Keys,
		Authorities:              authorities,
		Mode:                     req.Config.Mode,
		ErrOnUnauthorized:        config.Bool(errOnUnauthorized),
		EnableAutoCommit:         config.Bool(enableAutoCommit),
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
	return w, srv.Send(FrameWriterResponse{
		Command: writer.Open,
		Err:     errors.Encode(ctx, nil, false),
	})
}

type WSFramerCodec struct {
	*framercodec.Codec
	LowerPerfCodec xbinary.Codec
}

func NewWSFramerCodec(channels channel.Readable) httputil.Codec {
	return &WSFramerCodec{
		LowerPerfCodec: httputil.JSONCodec,
		Codec:          framercodec.NewDynamic(channels),
	}
}

var _ xbinary.Codec = (*WSFramerCodec)(nil)

func (c *WSFramerCodec) Decode(
	ctx context.Context,
	data []byte,
	value any,
) error {
	r := bytes.NewReader(data)
	return c.DecodeStream(ctx, r, value)
}

var (
	highPerfSpecialChar byte = 255
	lowPerfSpecialChar  byte = 254
)

func (c *WSFramerCodec) DecodeStream(
	ctx context.Context,
	r io.Reader,
	value any,
) error {
	switch v := value.(type) {
	case *fhttp.WSMessage[FrameWriterRequest]:
		return c.decodeWriteRequest(ctx, r, v)
	case *fhttp.WSMessage[FrameWriterResponse]:
		return c.lowPerfDecode(ctx, r, v)
	case *fhttp.WSMessage[FrameStreamerRequest]:
		return c.decodeStreamRequest(ctx, r, v)
	case *fhttp.WSMessage[FrameStreamerResponse]:
		return c.decodeStreamResponse(ctx, r, v)
	default:
		panic(fmt.Sprintf("incompatible type %s provided to framer codec", reflect.TypeOf(value)))
	}
}

func (c *WSFramerCodec) Encode(ctx context.Context, value any) ([]byte, error) {
	wr := &bytes.Buffer{}
	err := c.EncodeStream(ctx, wr, value)
	return wr.Bytes(), err
}

func (c *WSFramerCodec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	switch v := value.(type) {
	case fhttp.WSMessage[FrameWriterRequest]:
		return c.encodeWriteRequest(ctx, w, v)
	case fhttp.WSMessage[FrameWriterResponse]:
		return c.lowPerfEncode(ctx, w, v)
	case fhttp.WSMessage[FrameStreamerRequest]:
		return c.lowPerfEncode(ctx, w, v)
	case fhttp.WSMessage[FrameStreamerResponse]:
		return c.encodeStreamResponse(ctx, w, v)
	default:
		panic("incompatible type")
	}
}

func (c *WSFramerCodec) lowPerfEncode(ctx context.Context, w io.Writer, value any) error {
	if _, err := w.Write([]byte{lowPerfSpecialChar}); err != nil {
		return err
	}
	b, err := c.LowerPerfCodec.Encode(ctx, value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func (c *WSFramerCodec) lowPerfDecode(ctx context.Context, r io.Reader, value any) error {
	return c.LowerPerfCodec.DecodeStream(ctx, r, value)
}

func (c *WSFramerCodec) decodeWriteRequest(
	ctx context.Context,
	r io.Reader,
	v *fhttp.WSMessage[FrameWriterRequest],
) error {
	sc := new(uint8)
	if err := binary.Read(r, binary.LittleEndian, sc); err != nil {
		return err
	}
	if *sc == lowPerfSpecialChar {
		if err := c.lowPerfDecode(ctx, r, v); err != nil {
			return err
		}
		if v.Payload.Command == writer.Open {
			return c.Update(ctx, v.Payload.Config.Keys)
		}
		return nil
	}
	v.Type = fhttp.WSMessageTypeData
	fr, err := c.Codec.DecodeStream(r)
	if err != nil {
		return err
	}
	v.Payload.Command = writer.Write
	v.Payload.Frame = fr
	return nil
}

func (c *WSFramerCodec) encodeWriteRequest(
	ctx context.Context,
	w io.Writer,
	v fhttp.WSMessage[FrameWriterRequest],
) error {
	if v.Type != fhttp.WSMessageTypeData || v.Payload.Command != writer.Write {
		return c.lowPerfEncode(ctx, w, v)
	}
	if _, err := w.Write([]byte{highPerfSpecialChar}); err != nil {
		return err
	}
	return c.Codec.EncodeStream(ctx, w, v.Payload.Frame)
}

func (c *WSFramerCodec) decodeStreamResponse(
	ctx context.Context,
	r io.Reader,
	v *fhttp.WSMessage[FrameStreamerResponse],
) error {
	sc := new(uint8)
	if err := binary.Read(r, binary.LittleEndian, sc); err != nil {
		return err
	}
	if *sc == lowPerfSpecialChar {
		if err := c.lowPerfDecode(ctx, r, v); err != nil {
			return err
		}
	}
	v.Type = fhttp.WSMessageTypeData
	fr, err := c.Codec.DecodeStream(r)
	if err != nil {
		return err
	}
	v.Payload.Frame = fr
	return nil
}

func (c *WSFramerCodec) encodeStreamResponse(
	ctx context.Context,
	w io.Writer,
	v fhttp.WSMessage[FrameStreamerResponse],
) error {
	if v.Type != fhttp.WSMessageTypeData || v.Payload.Frame.Empty() {
		return c.lowPerfEncode(ctx, w, v)
	}
	if _, err := w.Write([]byte{highPerfSpecialChar}); err != nil {
		return err
	}
	return c.Codec.EncodeStream(ctx, w, v.Payload.Frame)
}

func (c *WSFramerCodec) decodeStreamRequest(
	ctx context.Context,
	r io.Reader,
	v *fhttp.WSMessage[FrameStreamerRequest],
) error {
	if err := c.lowPerfDecode(ctx, r, v); err != nil {
		return err
	}
	return c.Update(ctx, v.Payload.Keys)
}

func (c *WSFramerCodec) ContentType() string {
	return framerContentType
}

const framerContentType = "application/sy-framer"

func NewHTTPCodecResolver(channel channel.Readable) httputil.CodecResolver {
	return func(ct string) (httputil.Codec, error) {
		if ct == framerContentType {
			return NewWSFramerCodec(channel), nil
		}
		return httputil.ResolveCodec(ct)
	}
}
