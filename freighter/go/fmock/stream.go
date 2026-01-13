// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fmock

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
)

var (
	_ freighter.StreamClient[int, types.Nil] = (*StreamClient[int, types.Nil])(nil)
	_ freighter.StreamServer[types.Nil, int] = (*StreamServer[types.Nil, int])(nil)
	_ freighter.ServerStream[int, types.Nil] = (*ServerStream[int, types.Nil])(nil)
	_ freighter.ClientStream[int, types.Nil] = (*ClientStream[int, types.Nil])(nil)
)

// NewStreamPair creates a new stream client and server pair that are directly linked to
// one another i.e. dialing any target on the client will call the server's handler.
func NewStreamPair[RQ, RS freighter.Payload](buffers ...int) (*StreamServer[RQ, RS], *StreamClient[RQ, RS]) {
	inB, outB := parseBuffers(buffers)
	ss := &StreamServer[RQ, RS]{BufferSize: outB, Reporter: reporter}
	sc := &StreamClient[RQ, RS]{BufferSize: inB, Server: ss, Reporter: reporter}
	return ss, sc
}

// NewStreams creates a set of directly linked client and server streams that can be
// used to exchange messages between each other. Buffers can be specified to set the
// buffer size of the channels used to exchange messages. [1] will set the buffer size
// to 1 for both the request and response streams, [1, 2] will set a buffer size of
// 1 for the request stream and 2 for the response stream.
func NewStreams[RQ, RS freighter.Payload](
	ctx context.Context,
	buffers ...int,
) (*ClientStream[RQ, RS], *ServerStream[RQ, RS]) {
	inB, outB := parseBuffers(buffers)
	req, res := make(chan message[RQ], inB), make(chan message[RS], outB)
	serverClosed, clientClosed := make(chan struct{}), make(chan struct{})
	return &ClientStream[RQ, RS]{
			ctx:          ctx,
			requests:     req,
			responses:    res,
			serverClosed: serverClosed,
			clientClosed: clientClosed,
		},
		&ServerStream[RQ, RS]{
			ctx:          ctx,
			requests:     req,
			responses:    res,
			serverClosed: serverClosed,
			clientClosed: clientClosed,
		}
}

// StreamServer implements the freighter.StreamSever interface using go channels as
// the transport.
type StreamServer[RQ, RS freighter.Payload] struct {
	Address    address.Address
	BufferSize int
	Handler    func(ctx context.Context, srv freighter.ServerStream[RQ, RS]) error
	freighter.Reporter
	freighter.MiddlewareCollector
}

// BindHandler implements the freighter.Stream interface.
func (s *StreamServer[RQ, RS]) BindHandler(handler func(
	ctx context.Context,
	srv freighter.ServerStream[RQ, RS]) error) {
	s.Handler = handler
}

func (s *StreamServer[RQ, RS]) exec(
	ctx freighter.Context,
	srv *ServerStream[RQ, RS],
) (freighter.Context, error) {
	if s.Handler == nil {
		return ctx, errors.New("no handler bound to stream server")
	}
	return s.Exec(
		ctx,
		freighter.FinalizerFunc(func(md freighter.Context) (freighter.Context, error) {
			go srv.exec(ctx, s.Handler)
			return freighter.Context{Target: s.Address, Protocol: s.Protocol, Params: make(freighter.Params)}, nil
		}),
	)
}

// StreamClient is a mock implementation of the freighter.Stream interface.
type StreamClient[RQ, RS freighter.Payload] struct {
	Address    address.Address
	BufferSize int
	Network    *Network[RQ, RS]
	Server     *StreamServer[RQ, RS]
	freighter.Reporter
	freighter.MiddlewareCollector
}

// Stream implements the freighter.Stream interface.
func (s *StreamClient[RQ, RS]) Stream(
	ctx context.Context,
	target address.Address,
) (stream freighter.ClientStream[RQ, RS], err error) {
	_, err = s.Exec(
		freighter.Context{
			Context:  ctx,
			Target:   target,
			Protocol: s.Protocol,
			Params:   make(freighter.Params),
		},
		freighter.FinalizerFunc(func(ctx freighter.Context) (oCtx freighter.Context, err error) {
			if target == "" {
				target = "localhost:0"
			}
			var (
				targetBufferSize int
				server           *StreamServer[RQ, RS]
			)
			if s.Server != nil {
				server = s.Server
				targetBufferSize = server.BufferSize
			} else if s.Network != nil {
				srv, ok := s.Network.resolveStreamTarget(target)
				if !ok || srv.Handler == nil {
					return oCtx, address.NewTargetNotFoundError(target)
				}
				server = srv
				targetBufferSize = srv.BufferSize
			}
			var serverStream *ServerStream[RQ, RS]
			stream, serverStream = NewStreams[RQ, RS](ctx, s.BufferSize, targetBufferSize)
			return server.exec(ctx, serverStream)
		}),
	)
	return stream, err
}

const defaultBuffer = 10

func parseBuffers(buffers []int) (int, int) {
	if len(buffers) == 0 {
		return defaultBuffer, defaultBuffer
	}
	if len(buffers) == 1 {
		return buffers[0], buffers[0]
	}
	return buffers[0], buffers[1]
}

type ServerStream[RQ, RS freighter.Payload] struct {
	// ctx is the context the ServerStream was started with. Yes, Yes! RQ know this is a bad
	// practice, but in this case we're essentially using it as a data container,
	// and we have a very good grasp on how it's used.
	ctx          context.Context
	requests     <-chan message[RQ]
	responses    chan<- message[RS]
	serverClosed chan struct{}
	clientClosed <-chan struct{}
	receiveErr   error
	sendErr      error
}

// Send implements the freighter.StreamSender interface.
func (s *ServerStream[RQ, RS]) Send(res RS) error {
	if s.sendErr != nil {
		return s.sendErr
	}

	if s.ctx.Err() != nil {
		return s.ctx.Err()
	}
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.serverClosed:
		return freighter.ErrStreamClosed
	case s.responses <- message[RS]{payload: res}:
		return nil
	}
}

// Receive implements the freighter.ClientStream interface.
func (s *ServerStream[RQ, RS]) Receive() (req RQ, err error) {
	if s.receiveErr != nil {
		return req, s.receiveErr
	}
	if s.ctx.Err() != nil {
		return req, s.ctx.Err()
	}
	select {
	case <-s.ctx.Done():
		return req, s.ctx.Err()
	case <-s.serverClosed:
		return req, freighter.ErrStreamClosed
	case msg := <-s.requests:
		// Any error message means the Stream should die.
		if msg.error.Type != errors.TypeEmpty {
			s.receiveErr = errors.Decode(s.ctx, msg.error)
			return req, s.receiveErr
		}
		return msg.payload, nil
	}
}

func (s *ServerStream[RQ, RS]) exec(
	ctx context.Context,
	handler func(ctx context.Context, server freighter.ServerStream[RQ, RS]) error,
) {
	err := handler(ctx, s)
	errPayload := errors.Encode(ctx, err, true)
	if errPayload.Type == errors.TypeNil {
		errPayload = errors.Encode(ctx, freighter.EOF, true)
	}
	close(s.serverClosed)
	s.responses <- message[RS]{error: errPayload}
}

type ClientStream[RQ, RS freighter.Payload] struct {
	// ctx is the context the ServerStream was started with. Yes, Yes! I know this is a bad
	// practice, but in this case we're essentially using it as a data container,
	// and we have a very good grasp on how it's used.
	ctx          context.Context
	requests     chan<- message[RQ]
	responses    <-chan message[RS]
	serverClosed chan struct{}
	clientClosed chan struct{}
	sendErr      error
	receiveErr   error
}

func (c *ClientStream[RQ, RS]) Send(req RQ) error {
	if c.sendErr != nil {
		return c.sendErr
	}
	if c.receiveErr != nil {
		return freighter.EOF
	}
	if c.ctx.Err() != nil {
		return c.ctx.Err()
	}
	select {
	case <-c.ctx.Done():
		return c.ctx.Err()
	case <-c.clientClosed:
		return freighter.ErrStreamClosed
	case <-c.serverClosed:
		// If the server was serverClosed, we set the sendErr to EOF and let
		// the client discover the server error by calling Receive.
		c.sendErr = freighter.EOF
		return c.sendErr
	case c.requests <- message[RQ]{payload: req}:
		return nil
	}
}

func (c *ClientStream[RQ, RS]) Receive() (res RS, err error) {
	if c.receiveErr != nil {
		return res, c.receiveErr
	}
	select {
	case <-c.ctx.Done():
		return res, c.ctx.Err()
	case msg := <-c.responses:
		// If our message contains an error, that means the server serverClosed the stream (i.e. serverClosed chan
		// is serverClosed), so we don't need explicitly listen for its closure.
		if msg.error.Type != errors.TypeEmpty {
			if c.receiveErr == nil {
				c.receiveErr = errors.Decode(c.ctx, msg.error)
			}
			return res, c.receiveErr
		}
		return msg.payload, nil
	}
}

// CloseSend implements the freighter.StreamCloser interface.
func (c *ClientStream[RQ, RS]) CloseSend() error {
	if c.sendErr != nil {
		return nil
	}
	c.sendErr = freighter.ErrStreamClosed
	c.requests <- message[RQ]{error: errors.Encode(c.ctx, freighter.EOF, true)}
	close(c.clientClosed)
	return nil
}

type message[P freighter.Payload] struct {
	payload P
	error   errors.Payload
}
