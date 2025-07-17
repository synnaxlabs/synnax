// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
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
// to 1 for both the request and response streams, [1, 2] will set a buffer size of 1
// for the request stream and 2 for the response stream.
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

// StreamServer implements the freighter.StreamServer interface using go channels as the
// transport.
type StreamServer[RQ, RS freighter.Payload] struct {
	Address    address.Address
	BufferSize int
	Handler    func(ctx context.Context, srv freighter.ServerStream[RQ, RS]) error
	freighter.Reporter
	freighter.MiddlewareCollector
}

var _ freighter.StreamServer[any, any] = (*StreamServer[any, any])(nil)

// BindHandler implements the freighter.StreamServer interface.
func (ss *StreamServer[RQ, RS]) BindHandler(handler func(
	ctx context.Context,
	srv freighter.ServerStream[RQ, RS]) error) {
	ss.Handler = handler
}

func (ss *StreamServer[RQ, RS]) exec(
	ctx freighter.Context,
	srv *ServerStream[RQ, RS],
) (freighter.Context, error) {
	if ss.Handler == nil {
		return ctx, errors.New("no handler bound to stream server")
	}
	return ss.MiddlewareCollector.Exec(
		ctx,
		func(md freighter.Context) (freighter.Context, error) {
			go srv.exec(ctx, ss.Handler)
			return freighter.Context{Target: ss.Address, Protocol: ss.Protocol, Params: make(freighter.Params)}, nil
		},
	)
}

// StreamClient is a mock implementation of the freighter.StreamClient interface.
type StreamClient[RQ, RS freighter.Payload] struct {
	Address    address.Address
	BufferSize int
	Network    *Network[RQ, RS]
	Server     *StreamServer[RQ, RS]
	freighter.Reporter
	freighter.MiddlewareCollector
}

var _ freighter.StreamClient[any, any] = (*StreamClient[any, any])(nil)

// Stream implements the freighter.StreamClient interface.
func (sc *StreamClient[RQ, RS]) Stream(
	ctx context.Context,
	target address.Address,
) (freighter.ClientStream[RQ, RS], error) {
	var stream freighter.ClientStream[RQ, RS]
	_, err := sc.MiddlewareCollector.Exec(
		freighter.Context{
			Context:  ctx,
			Target:   target,
			Protocol: sc.Protocol,
			Params:   make(freighter.Params),
		},
		func(ctx freighter.Context) (freighter.Context, error) {
			if target == "" {
				target = "localhost:0"
			}
			var (
				targetBufferSize int
				server           *StreamServer[RQ, RS]
			)
			if sc.Server != nil {
				server = sc.Server
				targetBufferSize = server.BufferSize
			} else if sc.Network != nil {
				srv, ok := sc.Network.resolveStreamTarget(target)
				if !ok || srv.Handler == nil {
					return freighter.Context{}, address.NewErrTargetNotFound(target)
				}
				server = srv
				targetBufferSize = srv.BufferSize
			}
			var serverStream *ServerStream[RQ, RS]
			stream, serverStream = NewStreams[RQ, RS](ctx, sc.BufferSize, targetBufferSize)
			return server.exec(ctx, serverStream)
		},
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
	// ctx is the context the ServerStream was started with. Yes, Yes! RQ know this is a
	// bad practice, but in this case we're essentially using it as a data container,
	// and we have a very good grasp on how it's used.
	ctx          context.Context
	requests     <-chan message[RQ]
	responses    chan<- message[RS]
	serverClosed chan struct{}
	clientClosed <-chan struct{}
	receiveErr   error
	sendErr      error
}

var _ freighter.ServerStream[any, any] = (*ServerStream[any, any])(nil)

// Send implements the freighter.StreamSender interface.
func (ss *ServerStream[RQ, RS]) Send(res RS) error {
	if ss.sendErr != nil {
		return ss.sendErr
	}

	if ss.ctx.Err() != nil {
		return ss.ctx.Err()
	}
	select {
	case <-ss.ctx.Done():
		return ss.ctx.Err()
	case <-ss.serverClosed:
		return freighter.ErrStreamClosed
	case ss.responses <- message[RS]{payload: res}:
		return nil
	}
}

// Receive implements the freighter.ClientStream interface.
func (ss *ServerStream[RQ, RS]) Receive() (RQ, error) {
	var req RQ
	if ss.receiveErr != nil {
		return req, ss.receiveErr
	}
	if ss.ctx.Err() != nil {
		return req, ss.ctx.Err()
	}
	select {
	case <-ss.ctx.Done():
		return req, ss.ctx.Err()
	case <-ss.serverClosed:
		return req, freighter.ErrStreamClosed
	case msg := <-ss.requests:
		// Any error message means the Stream should die.
		if msg.error.Type != errors.TypeEmpty {
			ss.receiveErr = errors.Decode(ss.ctx, msg.error)
			return req, ss.receiveErr
		}
		return msg.payload, nil
	}
}

func (ss *ServerStream[RQ, RS]) exec(
	ctx context.Context,
	handler func(context.Context, freighter.ServerStream[RQ, RS]) error,
) {
	err := handler(ctx, ss)
	errPayload := errors.Encode(ctx, err, true)
	if errPayload.Type == errors.TypeNil {
		errPayload = errors.Encode(ctx, freighter.EOF, true)
	}
	close(ss.serverClosed)
	ss.responses <- message[RS]{error: errPayload}
}

type ClientStream[RQ, RS freighter.Payload] struct {
	// ctx is the context the ServerStream was started with. Yes, Yes! I know this is a
	// bad practice, but in this case we're essentially using it as a data container,
	// and we have a very good grasp on how it's used.
	ctx          context.Context
	requests     chan<- message[RQ]
	responses    <-chan message[RS]
	serverClosed chan struct{}
	clientClosed chan struct{}
	sendErr      error
	receiveErr   error
}

var _ freighter.ClientStream[any, any] = (*ClientStream[any, any])(nil)

func (cs *ClientStream[RQ, RS]) Send(req RQ) error {
	if cs.sendErr != nil {
		return cs.sendErr
	}
	if cs.receiveErr != nil {
		return freighter.EOF
	}
	if cs.ctx.Err() != nil {
		return cs.ctx.Err()
	}
	select {
	case <-cs.ctx.Done():
		return cs.ctx.Err()
	case <-cs.clientClosed:
		return freighter.ErrStreamClosed
	case <-cs.serverClosed:
		// If the server was serverClosed, we set the sendErr to EOF and let the client
		// discover the server error by calling Receive.
		cs.sendErr = freighter.EOF
		return cs.sendErr
	case cs.requests <- message[RQ]{payload: req}:
		return nil
	}
}

func (cs *ClientStream[RQ, RS]) Receive() (RS, error) {
	var res RS
	if cs.receiveErr != nil {
		return res, cs.receiveErr
	}
	select {
	case <-cs.ctx.Done():
		return res, cs.ctx.Err()
	case msg := <-cs.responses:
		// If our message contains an error, that means the server serverClosed the
		// stream (i.e. serverClosed chan is serverClosed), so we don't need explicitly
		// listen for its closure.
		if msg.error.Type != errors.TypeEmpty {
			if cs.receiveErr == nil {
				cs.receiveErr = errors.Decode(cs.ctx, msg.error)
			}
			return res, cs.receiveErr
		}
		return msg.payload, nil
	}
}

// CloseSend implements the freighter.StreamCloser interface.
func (cs *ClientStream[RQ, RS]) CloseSend() error {
	if cs.sendErr != nil {
		return nil
	}
	cs.sendErr = freighter.ErrStreamClosed
	cs.requests <- message[RQ]{error: errors.Encode(cs.ctx, freighter.EOF, true)}
	close(cs.clientClosed)
	return nil
}

type message[P freighter.Payload] struct {
	payload P
	error   errors.Payload
}
