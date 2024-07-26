// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fhttp

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/x/errors"
	"go/types"
	"io"
	"net/http"
	"time"

	ws "github.com/fasthttp/websocket"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	roacherrors "github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/httputil"
	"go.uber.org/zap"
)

var (
	_ freighter.StreamClient[any, types.Nil] = (*streamClient[any, types.Nil])(nil)
	_ freighter.ClientStream[any, types.Nil] = (*clientStream[any, types.Nil])(nil)
	_ config.Config[ClientFactoryConfig]     = ClientFactoryConfig{}
)

type messageType string

const (
	messageTypeData messageType = "data"
	closeMessage    messageType = "close"
)

// message wraps a user payload with additional information needed for the websocket
// transport to correctly implement the Stream interface. Namely, we need a custom
// close message type to correctly encode and transfer information about a closure
// error across the socket.
type message[P freighter.Payload] struct {
	// Type represents the type of message being sent. One of messageTypeData
	// or closeMessage.
	Type messageType `json:"type" msgpack:"type"`
	// Err is the error payload to send if the message type is closeMessage.
	Err errors.Payload `json:"error" msgpack:"error"`
	// Payload is the user payload to send if the message type is messageTypeData.
	Payload P `json:"payload" msgpack:"payload"`
}

func newStreamCore[RQ, RS freighter.Payload](
	ctx context.Context,
	cfg coreConfig,
) streamCore[RQ, RS] {
	ctx, cancel := context.WithCancel(ctx)
	b := streamCore[RQ, RS]{
		ctx:        ctx,
		cancel:     cancel,
		contextC:   make(chan struct{}),
		coreConfig: cfg,
	}
	go b.listenForContextCancellation()
	return b
}

type coreConfig struct {
	alamos.Instrumentation
	conn          *ws.Conn
	codec         binary.Codec
	writeDeadline time.Duration
}

// streamCore is the common functionality implemented by both the client and server streams.
type streamCore[I, O freighter.Payload] struct {
	coreConfig
	ctx        context.Context
	cancel     context.CancelFunc
	contextC   chan struct{}
	peerClosed error
}

func (c *streamCore[I, O]) send(msg message[O]) error {
	b, err := c.codec.Encode(nil, msg)
	if err != nil {
		return err
	}
	if c.writeDeadline > 0 {
		if err = c.conn.SetWriteDeadline(time.Now().Add(c.writeDeadline)); err != nil {
			return err
		}
	}
	return c.conn.WriteMessage(ws.BinaryMessage, b)
}

func (c *streamCore[I, O]) receive() (msg message[I], err error) {
	var r io.Reader
	_, r, err = c.conn.NextReader()
	if err != nil {
		return msg, err
	}
	return msg, c.codec.DecodeStream(nil, r, &msg)
}

func (c *streamCore[I, O]) cancelStream() error {
	close(c.contextC)
	c.peerClosed = context.Canceled
	c.cancel()
	return c.peerClosed
}

func (c *streamCore[I, O]) listenForContextCancellation() {
	select {
	case <-c.contextC:
		return
	case <-c.ctx.Done():
		if err := c.conn.WriteControl(
			ws.CloseMessage,
			ws.FormatCloseMessage(ws.CloseGoingAway, ""),
			time.Now().Add(time.Second),
		); err != nil && !roacherrors.Is(err, ws.ErrCloseSent) {
			c.L.Error("error sending close message: %v \n", zap.Error(err))
		}
	}
}

type streamClient[RQ, RS freighter.Payload] struct {
	alamos.Instrumentation
	codec  httputil.Codec
	dialer ws.Dialer
	freighter.Reporter
	freighter.MiddlewareCollector
}

func (s *streamClient[RQ, RS]) Report() alamos.Report {
	r := streamReporter
	r.Encodings = []string{s.codec.ContentType()}
	return r.Report()
}

func (s *streamClient[RQ, RS]) Stream(
	ctx context.Context,
	target address.Address,
) (stream freighter.ClientStream[RQ, RS], err error) {
	_, err = s.MiddlewareCollector.Exec(
		freighter.Context{
			Context:  ctx,
			Target:   target,
			Protocol: s.Reporter.Protocol,
			Params:   make(freighter.Params),
		},
		freighter.FinalizerFunc(func(ctx freighter.Context) (oCtx freighter.Context, err error) {
			ctx.Params[fiber.HeaderContentType] = s.codec.ContentType()
			conn, res, err := s.dialer.DialContext(ctx, "ws://"+target.String(), mdToHeaders(ctx))
			oCtx = parseResponseCtx(res, target)
			if err != nil {
				return oCtx, err
			}
			if res.StatusCode != fiber.StatusSwitchingProtocols {
				return oCtx, roacherrors.New("[ws] - unable to upgrade connection")
			}
			stream = &clientStream[RQ, RS]{streamCore: newStreamCore[RS, RQ](ctx, coreConfig{conn: conn, codec: s.codec, Instrumentation: s.Instrumentation})}
			return oCtx, nil
		}),
	)
	return stream, err
}

type clientStream[RQ, RS freighter.Payload] struct {
	streamCore[RS, RQ]
	sendClosed bool
}

// Send implements the freighter.ClientStream interface.
func (s *clientStream[RQ, RS]) Send(req RQ) error {
	if s.peerClosed != nil {
		return freighter.EOF
	}
	if s.sendClosed {
		return freighter.StreamClosed
	}
	if s.ctx.Err() != nil {
		return s.ctx.Err()
	}
	if err := s.streamCore.send(message[RQ]{Type: messageTypeData, Payload: req}); err != nil {
		close(s.contextC)
		return freighter.EOF
	}
	return nil
}

// Receive implements the freighter.ClientStream interface.
func (s *clientStream[RQ, RS]) Receive() (res RS, err error) {
	if s.peerClosed != nil {
		return res, s.peerClosed
	}

	if s.ctx.Err() != nil {
		return res, s.ctx.Err()
	}

	msg, err := s.streamCore.receive()
	if isRemoteContextCancellation(err) {
		return res, s.cancelStream()
	}
	if err != nil {
		return res, err
	}
	// A close message means the server handler exited.
	if msg.Type == closeMessage {
		close(s.contextC)
		s.peerClosed = errors.Decode(s.ctx, msg.Err)
		return res, s.peerClosed
	}
	return msg.Payload, err
}

// CloseSend implements the freighter.ClientStream interface.
func (s *clientStream[RQ, RS]) CloseSend() error {
	if s.peerClosed != nil || s.sendClosed {
		return nil
	}
	s.sendClosed = true
	return s.streamCore.send(message[RQ]{Type: closeMessage})
}

func mdToHeaders(md freighter.Context) http.Header {
	headers := make(http.Header, len(md.Params))
	for k, v := range md.Params {
		if vStr, ok := v.(string); ok {
			headers[k] = []string{vStr}
		}
	}
	return headers
}

type streamServer[RQ, RS freighter.Payload] struct {
	freighter.Reporter
	freighter.MiddlewareCollector
	alamos.Instrumentation
	serverCtx     context.Context
	path          string
	internal      bool
	handler       func(ctx context.Context, server freighter.ServerStream[RQ, RS]) error
	writeDeadline time.Duration
}

func (s *streamServer[RQ, RS]) BindHandler(
	handler func(ctx context.Context, server freighter.ServerStream[RQ, RS]) error,
) {
	s.handler = handler
}

const closeReadDeadline = 500 * time.Millisecond

func (s *streamServer[RQ, RS]) fiberHandler(fiberCtx *fiber.Ctx) error {
	if !fiberws.IsWebSocketUpgrade(fiberCtx) {
		return fiber.ErrUpgradeRequired
	}
	iMD := parseRequestCtx(fiberCtx, address.Address(s.path))
	headerContentType := iMD.Params.GetDefault(fiber.HeaderContentType, "").(string)
	codec, err := httputil.DetermineCodec(headerContentType)
	if err != nil {
		// If we can't determine the encoder/decoder, we can't continue, so we sent a best effort string.
		return fiberCtx.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	oCtx, err := s.MiddlewareCollector.Exec(
		iMD,
		freighter.FinalizerFunc(func(ctx freighter.Context) (freighter.Context, error) {
			oCtx := freighter.Context{
				Target:   iMD.Target,
				Protocol: s.Reporter.Protocol,
				Params:   make(freighter.Params),
			}
			return oCtx, fiberws.New(func(c *fiberws.Conn) {
				if err := func() error {
					stream := newServerStream[RQ, RS](s.serverCtx, coreConfig{
						writeDeadline: s.writeDeadline,
						conn:          c.Conn,
						codec:         codec,
					})
					defer func() {
						if err := stream.conn.Close(); err != nil {
							s.L.Error("error closing connection", zap.Error(err))
						}
					}()
					errPld := errors.Encode(ctx, s.handler(stream.ctx, stream), s.internal)
					if errPld.Type == errors.TypeNil {
						errPld = errors.Encode(ctx, freighter.EOF, s.internal)
					}
					if stream.ctx.Err() != nil {
						return stream.ctx.Err()
					}
					if err = stream.send(message[RS]{Type: closeMessage, Err: errPld}); err != nil {
						return err
					}
					stream.peerClosed = freighter.StreamClosed
					if err = stream.conn.WriteControl(
						ws.CloseMessage,
						ws.FormatCloseMessage(ws.CloseNormalClosure, ""),
						time.Now().Add(500*time.Millisecond),
					); err != nil {
						return err
					}
					if err = stream.conn.SetReadDeadline(time.Now().Add(closeReadDeadline)); err != nil {
						return err
					}
					if _, err = stream.receive(); err != nil &&
						!ws.IsCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway) {
						s.L.Error("expected normal closure, received error instead", zap.Error(err))
					}
					close(stream.contextC)
					stream.cancel()
					return nil
				}(); err != nil && !errors.Is(err, context.Canceled) {
					s.L.Error("stream server handler error", zap.Error(err))
				}
			})(fiberCtx)
		}))
	setResponseCtx(fiberCtx, oCtx)
	fErr := errors.Encode(oCtx, err, s.internal)
	if fErr.Type == errors.TypeNil {
		return nil
	}
	fiberCtx.Status(fiber.StatusBadRequest)
	return encodeAndWrite(fiberCtx, codec, fErr)
}

type serverStream[RQ, RS freighter.Payload] struct{ streamCore[RQ, RS] }

func newServerStream[RQ, RS freighter.Payload](
	ctx context.Context,
	cfg coreConfig,
) *serverStream[RQ, RS] {
	return &serverStream[RQ, RS]{streamCore: newStreamCore[RQ, RS](ctx, cfg)}
}

// Receive implements the freighter.ServerStream interface.
func (s *serverStream[RQ, RS]) Receive() (req RQ, err error) {
	if s.peerClosed != nil {
		return req, s.peerClosed
	}
	if s.ctx.Err() != nil {
		return req, s.ctx.Err()
	}
	msg, err := s.streamCore.receive()
	if isRemoteContextCancellation(err) {
		return req, s.cancelStream()
	}
	if err != nil {
		return req, err
	}
	// A close message means the client called CloseSend.
	if msg.Type == closeMessage {
		s.peerClosed = freighter.EOF
		return req, s.peerClosed
	}
	return msg.Payload, err
}

// Send implements the freighter.ServerStream interface.
func (s *serverStream[RQ, RS]) Send(res RS) error {
	if s.ctx.Err() != nil {
		return s.ctx.Err()
	}
	return s.streamCore.send(message[RS]{Payload: res, Type: messageTypeData})
}

func isRemoteContextCancellation(err error) bool {
	return ws.IsCloseError(err, ws.CloseGoingAway)
}
