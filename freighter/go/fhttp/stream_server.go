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
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/binary"
	"go.uber.org/zap"
)

type streamServerOptions struct {
	codecs map[string]func() binary.Codec
}

type StreamServerOption func(*streamServerOptions)

// WithCodec adds a new option to get a codec to the stream server.
func WithCodec(contentType string, getCodec func() binary.Codec) StreamServerOption {
	return func(o *streamServerOptions) { o.codecs[contentType] = getCodec }
}

func newStreamServerOptions(opts []StreamServerOption) streamServerOptions {
	so := streamServerOptions{
		codecs: map[string]func() binary.Codec{
			MIMEApplicationJSON:    func() binary.Codec { return binary.JSONCodec },
			MIMEApplicationMsgPack: func() binary.Codec { return binary.MsgPackCodec },
		},
	}
	for _, opt := range opts {
		opt(&so)
	}
	return so
}

type streamServer[RQ, RS freighter.Payload] struct {
	streamServerOptions
	freighter.Reporter
	freighter.MiddlewareCollector
	alamos.Instrumentation
	serverCtx     context.Context
	path          string
	handler       func(context.Context, freighter.ServerStream[RQ, RS]) error
	writeDeadline time.Duration
	wg            *sync.WaitGroup
}

// NewStreamServer returns a new freighter stream server that uses HTTP as the
// transport.
func NewStreamServer[RQ, RS freighter.Payload](
	r *Router,
	path string,
	opts ...StreamServerOption,
) freighter.StreamServer[RQ, RS] {
	so := newStreamServerOptions(opts)
	s := &streamServer[RQ, RS]{
		streamServerOptions: newStreamServerOptions(opts),
		Reporter:            newReporter(lo.Keys(so.codecs)...),
		path:                path,
		Instrumentation:     r.Instrumentation,
		serverCtx:           r.streamCtx,
		writeDeadline:       r.StreamWriteDeadline,
		wg:                  r.streamWg,
	}
	r.register(path, fiber.MethodGet, s, s.fiberHandler)
	return s
}

func (s *streamServer[RQ, RS]) BindHandler(
	handler func(context.Context, freighter.ServerStream[RQ, RS]) error,
) {
	s.handler = handler
}

const closeReadWriteDeadline = 500 * time.Millisecond

// fiberHandler handles the incoming websocket connection and upgrades the connection to
// a websocket connection.
//
// NOTE: shortLivedFiberCtx is a temporary fiber context
func (s *streamServer[RQ, RS]) fiberHandler(upgradeCtx *fiber.Ctx) error {
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
	contentType := iCtx.Params[fiber.HeaderContentType].(string)
	getCodec, ok := s.codecs[contentType]
	if !ok {
		return upgradeCtx.Status(fiber.StatusUnsupportedMediaType).SendString(
			fmt.Sprintf("unsupported content type: %s", contentType),
		)
	}
	codec := getCodec()
	// Upgrade the connection to a websocket connection.
	handler := fiberws.New(func(c *fiberws.Conn) { s.handleSocket(iCtx, codec, c) })
	return handler(upgradeCtx)
}

func (s *streamServer[RQ, RS]) handleSocket(
	ctx freighter.Context,
	codec binary.Codec,
	c *fiberws.Conn,
) {
	stream := &serverStream[RQ, RS]{streamCore: newStreamCore[RQ, RS](
		coreConfig{writeDeadline: s.writeDeadline, conn: c.Conn, codec: codec},
		ctx.Done(),
	)}
	// Register the stream with the server so it gets gracefully shut down.
	s.wg.Add(1)
	defer s.wg.Done()
	_, handlerErr := s.MiddlewareCollector.Exec(
		ctx,
		func(ctx freighter.Context) (freighter.Context, error) {
			oCtx := ctx
			oCtx.Params = make(freighter.Params)
			// Send a confirmation message to the client that the stream is open.
			if err := stream.send(WSMessage[RS]{Type: WSMessageTypeOpen}); err != nil {
				return oCtx, err
			}
			return oCtx, s.handler(ctx, stream)
		},
	)
	if err := stream.close(handlerErr); err != nil {
		s.L.Error("error closing connection", zap.Error(err))
	}
}
