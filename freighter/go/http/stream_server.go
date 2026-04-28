// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"context"
	"sync"
	"syscall"
	"time"

	ws "github.com/fasthttp/websocket"
	fiberws "github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

var streamReporter = freighter.Reporter{
	Protocol:  "websocket",
	Encodings: SupportedContentTypes(),
}

// additionalCodec pairs a content type with a constructor that returns a fresh Codec
// instance per request. Per-request construction matters for stateful codecs (e.g. the
// streaming framer codec, which tracks channel keys for the active stream).
type additionalCodec struct {
	contentType string
	new         func() Codec
}

// streamServerOptions configures a streaming HTTP server. Stream handlers can register
// stateful codecs whose state lives for the duration of a single websocket connection;
// per-request construction is supported.
type streamServerOptions struct {
	// additionalCodecs are codecs registered alongside the defaults. They are matched
	// by content type before the default registry, with a fresh codec instance
	// constructed per request via the registered factory.
	additionalCodecs []additionalCodec
}

// StreamServerOption configures a streaming HTTP server.
type StreamServerOption func(*streamServerOptions)

// WithAdditionalCodec registers a stream-server codec on top of the default codec list.
// The constructor is invoked once per matching request so the codec may be stateful and
// hold per-stream state.
func WithAdditionalCodec(
	contentType string,
	new func() Codec,
) StreamServerOption {
	return func(o *streamServerOptions) {
		o.additionalCodecs = append(o.additionalCodecs, additionalCodec{
			contentType: contentType,
			new:         new,
		})
	}
}

func newStreamServerOptions(opts []StreamServerOption) streamServerOptions {
	so := streamServerOptions{}
	for _, opt := range opts {
		opt(&so)
	}
	return so
}

type streamServer[RQ, RS freighter.Payload] struct {
	alamos.Instrumentation
	serverCtx context.Context
	streamServerOptions
	handler func(context.Context, freighter.ServerStream[RQ, RS]) error
	wg      *sync.WaitGroup
	path    string
	freighter.Reporter
	freighter.MiddlewareCollector
	writeDeadline time.Duration
}

func (s *streamServer[RQ, RS]) resolveStreamCodec(contentType string) (Codec, error) {
	for _, ac := range s.additionalCodecs {
		if ac.contentType == contentType {
			return ac.new(), nil
		}
	}
	return ResolveCodec(contentType)
}

func (s *streamServer[RQ, RS]) BindHandler(
	handler func(context.Context, freighter.ServerStream[RQ, RS]) error,
) {
	s.handler = handler
}

// fiberHandler handles the incoming websocket connection and upgrades the connection to
// a websocket connection.
func (s *streamServer[RQ, RS]) fiberHandler(upgradeCtx fiber.Ctx) error {
	// If the caller is hitting this endpoint with a standard HTTP request, tell them
	// they can only use websockets.
	if !fiberws.IsWebSocketUpgrade(upgradeCtx) {
		return fiber.ErrUpgradeRequired
	}
	// Parse the incoming request context. Used to pull various headers and parameters
	// from the request (e.g., content-type or authorization). upgradeCtx is only valid
	// for the lifetime of this function. As this function will exit long before the
	// stream stops processing values, we need to use the underlying server ctx as the
	// valid context instead of the fiber context itself.
	iCtx := parseRequestCtx(s.serverCtx, upgradeCtx, address.Address(s.path))
	headerContentType := iCtx.GetDefault(fiber.HeaderContentType, "").(string)
	codec, err := s.resolveStreamCodec(headerContentType)
	if err != nil {
		// If we can't determine the encoder/decoder, we can't continue, so we send a
		// best effort string.
		return upgradeCtx.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	// Upgrade the connection to a websocket connection.
	return fiberws.New(func(c *fiberws.Conn) { s.handleSocket(iCtx, codec, c) })(
		upgradeCtx,
	)
}

func (s *streamServer[RQ, RS]) handleSocket(
	ctx freighter.Context,
	codec encoding.Codec,
	c *fiberws.Conn,
) {
	stream := &serverStream[RQ, RS]{streamCore: newStreamCore[RQ, RS](
		coreConfig{writeDeadline: s.writeDeadline, conn: c.Conn, codec: codec},
		ctx.Done(),
	)}
	// Register the stream with the server so it gets gracefully shut down.
	s.wg.Add(1)
	defer s.wg.Done()
	_, handlerErr := s.Exec(
		ctx,
		freighter.FinalizerFunc(func(ctx freighter.Context) (freighter.Context, error) {
			oCtx := ctx
			oCtx.Params = make(freighter.Params)
			// Send a confirmation message to the client that the stream is open.
			if err := stream.send(WSMessage[RS]{Type: WSMessageTypeOpen}); err != nil {
				return oCtx, err
			}
			return oCtx, s.handler(ctx, stream)
		}),
	)
	// These errors occur when the client abruptly closes the connection (e.g. reloading
	// web pages, closing the app). They're not anomalous.
	if err := errors.Skip(
		stream.close(handlerErr),
		ws.ErrCloseSent,
		syscall.EPIPE,
		syscall.ECONNRESET,
	); err != nil {
		s.L.Error("error closing connection", zap.Error(err))
	}
}

type serverStream[RQ, RS freighter.Payload] struct{ streamCore[RQ, RS] }

// Send implements the freighter.ServerStream interface.
func (s *serverStream[RQ, RS]) Send(res RS) error {
	err := s.send(WSMessage[RS]{Payload: res, Type: WSMessageTypeData})
	if errors.IsAny(err, ws.ErrCloseSent, syscall.EPIPE, syscall.ECONNRESET) {
		return freighter.ErrStreamClosed
	}
	return err
}

func (s *serverStream[RQ, RS]) close(err error) (closeErr error) {
	defer func() {
		closeErr = errors.Combine(closeErr, s.streamCore.close())
	}()
	if err == nil {
		err = freighter.EOF
	}
	closeCode := contextCancelledCloseCode
	if !errors.Is(err, context.Canceled) {
		closeCode = ws.CloseNormalClosure
		if err = s.send(WSMessage[RS]{
			Type: WSMessageTypeClose,
			Err:  errors.Encode(context.TODO(), err, false),
		}); err != nil {
			return err
		}
	}

	s.peerCloseErr = freighter.ErrStreamClosed

	// Tell the client we're closing the connection. Make sure to include a write
	// deadline here in-case the client is stuck.
	if err = s.conn.WriteControl(
		ws.CloseMessage,
		ws.FormatCloseMessage(closeCode, ""),
		time.Now().Add(closeReadWriteDeadline),
	); err != nil {
		return err
	}

	// Again, make sure a stuck client doesn't cause problems with shutdown.
	if err = s.conn.SetReadDeadline(
		time.Now().Add(closeReadWriteDeadline),
	); err != nil {
		return err
	}

	// Wait until the client acknowledges the closure.
	for {
		if _, err = s.receiveRaw(); err != nil {
			if !ws.IsCloseError(err, ws.CloseNormalClosure, ws.CloseGoingAway) {
				s.L.Error(
					"expected normal closure, received error instead",
					zap.Error(err),
				)
			}
			break
		}
	}
	return nil
}
