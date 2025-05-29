// Copyright 2025 Synnax Labs, Inc.
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
	"go/types"
	"io"
	"net/http"
	"sync"
	"time"

	ws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/httputil"
	"go.uber.org/zap"
)

var (
	_ freighter.StreamClient[any, types.Nil] = (*streamClient[any, types.Nil])(nil)
	_ freighter.ClientStream[any, types.Nil] = (*clientStream[any, types.Nil])(nil)
	_ config.Config[ClientFactoryConfig]     = ClientFactoryConfig{}
)

// WSMessageType is used to differentiate between the different types of messages
// use to implement the websocket stream transport.
type WSMessageType string

const (
	// WSMessageTypeData is used for normal data movement between the ClientStream and
	// ServerStream implementations.
	WSMessageTypeData WSMessageType = "data"
	// WSMessageTypeClose is used to signal the end of the stream. We need to use this
	// instead of the regular websocket Close message because the 'reason' can't
	// have more than 123 bytes.
	WSMessageTypeClose WSMessageType = "close"
	// WSMessageTypeOpen is used to acknowledge the successful opening of the stream.
	// We need to do this to correctly handle the case where middleware
	// returns an error early. We can't just use the regular HTTP request/response
	// cycle because JavaScript implementations of the WebSockets don't allow for
	// accessing the response body.
	WSMessageTypeOpen WSMessageType = "open"
)

// WSMessage wraps a user payload with additional information needed for the websocket
// transport to correctly implement the Stream interface. Namely, we need a custom
// close WSMessage type to correctly encode and transfer information about a closure
// error across the socket.
type WSMessage[P freighter.Payload] struct {
	// Type represents the type of WSMessage being sent. One of WSMessageTypeData
	// or WSMessageTypeClose.
	Type WSMessageType `json:"type" msgpack:"type"`
	// Err is the error payload to send if the WSMessage type is WSMessageTypeClose.
	Err errors.Payload `json:"error" msgpack:"error"`
	// Payload is the user payload to send if the WSMessage type is WSMessageTypeData.
	Payload P `json:"payload" msgpack:"payload"`
}

func newStreamCore[RQ, RS freighter.Payload](
	ctx context.Context,
	cfg coreConfig,
) streamCore[RQ, RS] {
	ctx, cancel := context.WithCancel(ctx)
	b := streamCore[RQ, RS]{
		ctx:             ctx,
		cancel:          cancel,
		contextListener: make(chan struct{}),
		coreConfig:      cfg,
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
	ctx             context.Context
	cancel          context.CancelFunc
	contextListener chan struct{}
	peerClosed      error
}

func (c *streamCore[I, O]) send(msg WSMessage[O]) (err error) {
	var w io.WriteCloser
	w, err = c.conn.NextWriter(ws.BinaryMessage)
	if err != nil {
		return
	}
	defer func() {
		err = errors.Combine(err, w.Close())
	}()
	if err = c.codec.EncodeStream(nil, w, msg); err != nil {
		return
	}
	if c.writeDeadline <= 0 {
		return
	}
	err = c.conn.SetWriteDeadline(time.Now().Add(c.writeDeadline))
	return
}

func (c *streamCore[I, O]) receive() (msg WSMessage[I], err error) {
	var r io.Reader
	_, r, err = c.conn.NextReader()
	if err != nil {
		return msg, err
	}
	return msg, c.codec.DecodeStream(nil, r, &msg)
}

func (c *streamCore[I, O]) cancelStream() error {
	close(c.contextListener)
	c.peerClosed = context.Canceled
	c.cancel()
	return c.peerClosed
}

// listenForContextCancellation is a goroutine that listens for the context to be
// canceled and shuts down the stream forcefully if it is. We need this as
// the websocket implementation itself doesn't support context cancellation.
func (c *streamCore[I, O]) listenForContextCancellation() {
	select {
	case <-c.contextListener:
		return
	case <-c.ctx.Done():
		if err := c.conn.WriteControl(
			ws.CloseMessage,
			ws.FormatCloseMessage(ws.CloseGoingAway, ""),
			time.Now().Add(time.Second),
		); err != nil && !errors.Is(err, ws.ErrCloseSent) {
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
				return oCtx, errors.New("[ws] - unable to upgrade connection")
			}
			core := newStreamCore[RS, RQ](
				ctx,
				coreConfig{
					conn:            conn,
					codec:           s.codec,
					Instrumentation: s.Instrumentation,
				},
			)
			msg, err := core.receive()
			if err != nil {
				return
			}
			if msg.Type != WSMessageTypeOpen {
				return oCtx, errors.Decode(ctx, msg.Err)
			}
			stream = &clientStream[RQ, RS]{streamCore: core}
			return
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
	if err := s.streamCore.send(WSMessage[RQ]{Type: WSMessageTypeData, Payload: req}); err != nil {
		close(s.contextListener)
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
	// A close WSMessage means the server handler exited.
	if msg.Type == WSMessageTypeClose {
		close(s.contextListener)
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
	return s.streamCore.send(WSMessage[RQ]{Type: WSMessageTypeClose})
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
	serverOptions
	freighter.Reporter
	freighter.MiddlewareCollector
	alamos.Instrumentation
	serverCtx     context.Context
	path          string
	handler       func(ctx context.Context, server freighter.ServerStream[RQ, RS]) error
	writeDeadline time.Duration
	wg            *sync.WaitGroup
}

func (s *streamServer[RQ, RS]) BindHandler(
	handler func(ctx context.Context, server freighter.ServerStream[RQ, RS]) error,
) {
	s.handler = handler
}

const closeReadWriteDeadline = 500 * time.Millisecond

// fiberHandler handles the incoming websocket connection and upgrades the connection
// to a websocket connection.
//
// NOTE: shortLivedFiberCtx is a temporary fiber context
func (s *streamServer[RQ, RS]) fiberHandler(fiberCtx *fiber.Ctx) error {
	// If the caller is hitting this endpoint with a standard HTTP request, tell them
	// they can only use websockets.
	if !fiberws.IsWebSocketUpgrade(fiberCtx) {
		return fiber.ErrUpgradeRequired
	}
	// Parse the incoming request context. Used to pull various headers and parameters
	// from the request (e.g. content-type or authorization). fiberCtx is only valid
	// for the lifetime of this function. As this function will exit long before the
	// stream stops processing values, we need to use the underlying server ctx as the
	// valid context instead of the fiber context itself.
	iCtx := parseRequestCtx(s.serverCtx, fiberCtx, address.Address(s.path))
	headerContentType := iCtx.Params.GetDefault(fiber.HeaderContentType, "").(string)
	codec, err := s.codecResolver(headerContentType)
	if err != nil {
		// If we can't determine the encoder/decoder, we can't continue, so we send
		// a best effort string.
		return fiberCtx.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	// Upgrade the connection to a websocket connection.
	return fiberws.New(func(conn *fiberws.Conn) {
		s.handleSocket(iCtx, codec, conn)
	})(fiberCtx)
}

func (s *streamServer[RQ, RS]) handleSocket(
	ctx freighter.Context,
	codec binary.Codec,
	c *fiberws.Conn,
) {
	// Wrap everything in an error closure so we can log any errors that occur.
	if err := func() error {
		stream := newServerStream[RQ, RS](
			// We use s.serverCtx here so we can correctly cancel the stream if
			// the server is shutting down.
			ctx,
			coreConfig{writeDeadline: s.writeDeadline, conn: c.Conn, codec: codec},
		)
		// Register the stream with the server so it gets gracefully shut down.
		s.wg.Add(1)
		defer s.wg.Done()
		defer func() {
			if err := stream.conn.Close(); err != nil {
				s.L.Error("error closing connection", zap.Error(err))
			}
		}()
		oCtx, handlerErr := s.MiddlewareCollector.Exec(
			ctx,
			freighter.FinalizerFunc(func(iFreighterCtx freighter.Context) (oFreighterCtx freighter.Context, err error) {
				// Send a confirmation message to the client that the stream is open.
				if err = stream.send(WSMessage[RS]{Type: WSMessageTypeOpen}); err != nil {
					return
				}
				err = s.handler(iFreighterCtx, stream)
				oFreighterCtx = freighter.Context{
					Context:  ctx,
					Target:   iFreighterCtx.Target,
					Protocol: s.Protocol,
					Params:   make(freighter.Params),
				}
				return
			}),
		)
		errPld := errors.Encode(oCtx, handlerErr, s.internal)
		if errPld.Type == errors.TypeNil {
			// If everything went well, we use an EOF to signal smooth closure of
			// the stream.
			errPld = errors.Encode(oCtx, freighter.EOF, s.internal)
		}
		if stream.ctx.Err() != nil {
			return stream.ctx.Err()
		}
		if err := stream.send(WSMessage[RS]{Type: WSMessageTypeClose, Err: errPld}); err != nil {
			return err
		}
		stream.peerClosed = freighter.StreamClosed
		if handlerErr != nil {
			time.Sleep(closeReadWriteDeadline)
		}
		// Tell the client we're closing the connection. Make sure to include
		// a write deadline here in-case the client is stuck.
		if err := stream.conn.WriteControl(
			ws.CloseMessage,
			ws.FormatCloseMessage(ws.CloseNormalClosure, ""),
			time.Now().Add(closeReadWriteDeadline),
		); err != nil {
			return err
		}
		// Again, make sure a stuck client doesn't cause problems with shutdown.
		if err := stream.conn.SetReadDeadline(
			time.Now().Add(closeReadWriteDeadline),
		); err != nil {
			return err
		}
		for {
			_, err := stream.receive()
			if err != nil {
				if !ws.IsCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway) {
					s.L.Error("expected normal closure, received error instead", zap.Error(err))
				}
				break
			}
		}
		// Shut down the routine that listens for context cancellation, as we don't
		// want to leak
		close(stream.contextListener)
		stream.cancel()
		return nil
	}(); err != nil {
		s.L.Error("error handling websocket connection", zap.Error(err))
	}
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
	// A close WSMessage means the client called CloseSend.
	if msg.Type == WSMessageTypeClose {
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
	return s.streamCore.send(WSMessage[RS]{Payload: res, Type: WSMessageTypeData})
}

func isRemoteContextCancellation(err error) bool {
	return ws.IsCloseError(err, ws.CloseGoingAway)
}
