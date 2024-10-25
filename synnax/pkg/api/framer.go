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
	"github.com/synnaxlabs/freighter/fhttp"
	framercodec "github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	xbinary "github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/httputil"
	"go/types"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	framesvc "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type Frame = framer.Frame

type FrameService struct {
	alamos.Instrumentation
	authProvider
	dbProvider
	accessProvider
	Internal *framesvc.Service
	Channel  channel.Readable
}

func NewFrameService(p Provider) *FrameService {
	return &FrameService{
		Instrumentation: p.Instrumentation,
		Internal:        p.Config.Framer,
		Channel:         p.Config.Channel,
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

const FrameIteratorAutoSpan = ts.AutoSpan

type (
	FrameIteratorRequest  = framer.IteratorRequest
	FrameIteratorResponse = framer.IteratorResponse
	FrameIteratorStream   = freighter.ServerStream[FrameIteratorRequest, FrameIteratorResponse]
)

func (s *FrameService) Iterate(ctx context.Context, stream FrameIteratorStream) error {
	iter, err := s.openIterator(ctx, stream)
	if err != nil {
		return err
	}

	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(s.Instrumentation.Child("frame_iterator")))
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
	plumber.SetSegment(pipe, "iterator", iter)
	plumber.SetSink[iterator.Response](pipe, "sender", sender)
	plumber.SetSource[iterator.Request](pipe, "receiver", receiver)
	plumber.MustConnect[iterator.Response](pipe, "iterator", "sender", 1)
	plumber.MustConnect[iterator.Request](pipe, "receiver", "iterator", 1)

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

func (s *FrameService) Stream(ctx context.Context, stream StreamerStream) error {
	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(s.Instrumentation.Child("frame_streamer")))
	defer cancel()
	streamer, err := s.openStreamer(sCtx, getSubject(ctx), stream)
	if err != nil {
		return err
	}
	var (
		receiver = &freightfluence.Receiver[FrameStreamerRequest]{Receiver: stream}
		sender   = &freightfluence.TransformSender[FrameStreamerResponse, FrameStreamerResponse]{
			Sender: freighter.SenderNopCloser[FrameStreamerResponse]{StreamSender: stream},
			Transform: func(ctx context.Context, res FrameStreamerResponse) (FrameStreamerResponse, bool, error) {
				if res.Error != nil {
					res.Error = errors.Encode(ctx, res.Error, false)
				}
				return res, true, nil
			},
		}
		pipe = plumber.New()
	)

	plumber.SetSegment[FrameStreamerRequest, FrameStreamerResponse](pipe, "streamer", streamer)
	plumber.SetSink[FrameStreamerResponse](pipe, "sender", sender)
	plumber.SetSource[FrameStreamerRequest](pipe, "receiver", receiver)
	plumber.MustConnect[FrameStreamerResponse](pipe, "streamer", "sender", 70)
	plumber.MustConnect[FrameStreamerRequest](pipe, "receiver", "streamer", 70)
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
	reader, err := s.Internal.NewStreamer(ctx, framer.StreamerConfig{Keys: req.Keys, DownsampleFactor: req.DownsampleFactor})
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
	Frame   Frame             `json:"frame" msgpack:"frame"`
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
	ctx, cancel := signal.WithCancel(_ctx, signal.WithInstrumentation(s.Instrumentation.Child("frame_writer")))
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

	pipe.Flow(ctx, confluence.CloseOutputInletsOnExit())
	return ctx.Wait()
}

func (s *FrameService) openWriter(
	ctx context.Context,
	subject ontology.ID,
	srv FrameWriterStream,
) (w framer.StreamWriter, err error) {
	req, err := srv.Receive()
	if err != nil {
		return
	}

	if err = s.access.Enforce(ctx, access.Request{
		Subject: subject,
		Action:  access.Create,
		Objects: framer.OntologyIDs(req.Config.Keys),
	}); err != nil {
		return
	}

	authorities := make([]control.Authority, len(req.Config.Authorities))
	for i, a := range req.Config.Authorities {
		authorities[i] = control.Authority(a)
	}

	w, err = s.Internal.NewStreamWriter(ctx, writer.Config{
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
		return
	}

	channels := make([]channel.Channel, 0, len(req.Config.Keys))
	if err = s.Channel.NewRetrieve().WhereKeys(req.Config.Keys...).Entries(&channels).Exec(ctx, nil); err != nil {
		return
	}
	// Let the client know the writer is ready to receive segments.
	return w, srv.Send(FrameWriterResponse{
		Command: writer.Open,
		Ack:     true,
	})
}

type WSFramerCodec struct {
	BaseCodec      *framercodec.Codec
	LowerPerfCodec xbinary.Codec
	reader         channel.Readable
	keysCache      channel.Keys
}

func NewWSWriterCodec(reader channel.Readable) httputil.Codec {
	return &WSFramerCodec{reader: reader, LowerPerfCodec: httputil.JSONCodec}
}

var _ xbinary.Codec = (*WSFramerCodec)(nil)

func (w *WSFramerCodec) Decode(
	ctx context.Context,
	data []byte,
	value interface{},
) error {
	r := bytes.NewReader(data)
	return w.DecodeStream(ctx, r, value)
}

var (
	highPerfSpecialChar byte = 255
	lowPerfSpecialChar  byte = 254
	wsHeaderSize             = 1
)

func (w *WSFramerCodec) maybeCacheKeys(req FrameWriterRequest) {
	if req.Command != writer.Open {
		return
	}
	w.keysCache = req.Config.Keys
}

func (w *WSFramerCodec) maybeBoostrapCodec(ctx context.Context) error {
	if w.BaseCodec != nil {
		return nil
	}
	channels := make([]channel.Channel, 0, len(w.keysCache))
	if err := w.reader.NewRetrieve().WhereKeys(w.keysCache...).Entries(&channels).Exec(ctx, nil); err != nil {
		return err
	}
	cdc := framercodec.NewCodecFromChannels(channels)
	w.BaseCodec = &cdc
	return nil
}

func (w *WSFramerCodec) DecodeStream(
	ctx context.Context,
	r io.Reader,
	value interface{},
) error {
	switch v := value.(type) {
	case *fhttp.WSMessage[FrameWriterRequest]:
		return w.decodeWriteRequest(ctx, r, v)
	case *fhttp.WSMessage[FrameWriterResponse]:
		return w.lowPerfDecode(ctx, r, v)
	case *fhttp.WSMessage[FrameStreamerRequest]:
		return w.decodeStreamRequest(ctx, r, v)
	case *fhttp.WSMessage[FrameStreamerResponse]:
		return w.decodeStreamResponse(ctx, r, v)
	default:
		panic("incompatible type")
	}
}

func (w *WSFramerCodec) Encode(ctx context.Context, value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case fhttp.WSMessage[FrameWriterRequest]:
		return w.encodeWriteRequest(ctx, v)
	case fhttp.WSMessage[FrameWriterResponse]:
		return w.lowPerfEncode(ctx, v)
	case fhttp.WSMessage[FrameStreamerRequest]:
		return w.lowPerfEncode(ctx, v)
	case fhttp.WSMessage[FrameStreamerResponse]:
		return w.encodeStreamResponse(ctx, v)
	default:
		panic("incompatible type")
	}
}

func (w *WSFramerCodec) lowPerfEncode(ctx context.Context, value interface{}) ([]byte, error) {
	b, err := w.LowerPerfCodec.Encode(ctx, value)
	if err != nil {
		return nil, err
	}
	return append([]byte{lowPerfSpecialChar}, b...), nil
}

func (w *WSFramerCodec) lowPerfDecode(ctx context.Context, r io.Reader, value interface{}) error {
	return w.LowerPerfCodec.DecodeStream(ctx, r, value)
}

func (w *WSFramerCodec) decodeWriteRequest(
	ctx context.Context,
	r io.Reader,
	v *fhttp.WSMessage[FrameWriterRequest],
) error {
	sc := new(uint8)
	if err := binary.Read(r, binary.LittleEndian, sc); err != nil {
		return err
	}
	if *sc == lowPerfSpecialChar {
		if err := w.lowPerfDecode(ctx, r, v); err != nil {
			return err
		}
		w.maybeCacheKeys(v.Payload)
		return nil
	}
	if err := w.maybeBoostrapCodec(ctx); err != nil {
		return err
	}
	v.Type = fhttp.WSMsgTypeData
	fr, err := w.BaseCodec.DecodeStream(r)
	if err != nil {
		return err
	}
	v.Payload.Command = writer.Write
	v.Payload.Frame = fr
	return nil
}

func (w *WSFramerCodec) encodeWriteRequest(
	ctx context.Context,
	v fhttp.WSMessage[FrameWriterRequest],
) (b []byte, err error) {
	if v.Type == fhttp.WSMsgTypeClose || v.Payload.Command != writer.Write {
		return w.lowPerfEncode(ctx, v)
	}
	if b, err = w.BaseCodec.Encode(v.Payload.Frame, wsHeaderSize); err != nil {
		return
	}
	b[0] = highPerfSpecialChar
	return
}

func (w *WSFramerCodec) decodeStreamResponse(
	ctx context.Context,
	r io.Reader,
	v *fhttp.WSMessage[FrameStreamerResponse],
) error {
	sc := new(uint8)
	if err := binary.Read(r, binary.LittleEndian, sc); err != nil {
		return err
	}
	if *sc == lowPerfSpecialChar {
		if err := w.lowPerfDecode(ctx, r, v); err != nil {
			return err
		}
	}
	if err := w.maybeBoostrapCodec(ctx); err != nil {
		return err
	}
	v.Type = fhttp.WSMsgTypeData
	fr, err := w.BaseCodec.DecodeStream(r)
	if err != nil {
		return err
	}
	v.Payload.Frame = fr
	return nil
}

func (w *WSFramerCodec) encodeStreamResponse(
	ctx context.Context,
	v fhttp.WSMessage[FrameStreamerResponse],
) (b []byte, err error) {
	if v.Type == fhttp.WSMsgTypeClose {
		return w.lowPerfEncode(ctx, v)
	}
	if err = w.maybeBoostrapCodec(ctx); err != nil {
		return
	}
	if b, err = w.BaseCodec.Encode(v.Payload.Frame, 1); err != nil {
		return
	}
	b[0] = highPerfSpecialChar
	return
}

func (w *WSFramerCodec) decodeStreamRequest(
	ctx context.Context,
	r io.Reader,
	v *fhttp.WSMessage[FrameStreamerRequest],
) error {
	if err := w.lowPerfDecode(ctx, r, v); err != nil {
		return err
	}
	w.keysCache = v.Payload.Keys
	return nil
}

func (w *WSFramerCodec) ContentType() string {
	return framerContentType
}

const framerContentType = "application/sy-framer"

func NewHTTPCodecResolver(channel channel.Readable) httputil.CodecResolver {
	return func(ct string) (httputil.Codec, error) {
		if ct == framerContentType {
			return NewWSWriterCodec(channel), nil
		}
		return httputil.ResolveCodec(ct)
	}
}
