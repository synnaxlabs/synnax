// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"context"
	"sync"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/types"
)

var (
	_ freighter.StreamClient[any, any] = (*StreamClient[any, any])(nil)
	_ freighter.StreamServer[any, any] = (*StreamServer[any, any])(nil)
	_ freighter.ServerStream[any, any] = (*ServerStream[any, any])(nil)
	_ freighter.ClientStream[any, any] = (*ClientStream[any, any])(nil)
)

// NewStreamPair creates a new stream client and server pair that are directly linked to
// one another i.e. dialing any target on the client will call the server's handler.
func NewStreamPair[RQ, RS freighter.Payload](
	buffers ...int,
) (*StreamServer[RQ, RS], *StreamClient[RQ, RS]) {
	inB, outB := parseBuffers(buffers)
	s := &StreamServer[RQ, RS]{bufferSize: outB}
	return s, &StreamClient[RQ, RS]{bufferSize: inB, server: s}
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
		}
}

// StreamServer implements the freighter.StreamServer interface using go channels as the
// transport.
type StreamServer[RQ, RS freighter.Payload] struct {
	mu      sync.RWMutex
	handler func(context.Context, freighter.ServerStream[RQ, RS]) error
	address address.Address
	reporter
	freighter.MiddlewareCollector
	bufferSize int
}

// BindHandler implements the freighter.StreamServer interface.
func (ss *StreamServer[RQ, RS]) BindHandler(handler func(
	context.Context,
	freighter.ServerStream[RQ, RS]) error,
) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.handler = handler
}

// Address returns where the server is reachable on its Network. It is empty for a
// server built by NewStreamPair. Callers need it when they let Network.StreamServer
// assign an address instead of naming one.
func (ss *StreamServer[RQ, RS]) Address() address.Address { return ss.address }

func (ss *StreamServer[RQ, RS]) boundHandler() func(
	context.Context,
	freighter.ServerStream[RQ, RS],
) error {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.handler
}

func (ss *StreamServer[RQ, RS]) exec(
	ctx freighter.Context,
	srv *ServerStream[RQ, RS],
) (freighter.Context, error) {
	h := ss.boundHandler()
	if h == nil {
		return ctx, errors.New("no handler bound to stream server")
	}
	return ss.Exec(
		ctx,
		freighter.FinalizerFunc(func(md freighter.Context) (freighter.Context, error) {
			go srv.exec(ctx, h)
			return freighter.Context{
				Target:   ss.address,
				Protocol: protocol,
				Params:   make(freighter.Params),
			}, nil
		}),
	)
}

// StreamClient implements the freighter.StreamClient interface using go channels as the
// transport.
type StreamClient[RQ, RS freighter.Payload] struct {
	network *Network[RQ, RS]
	server  *StreamServer[RQ, RS]
	reporter
	freighter.MiddlewareCollector
	bufferSize int
}

// Stream implements the freighter.StreamClient interface.
func (sc *StreamClient[RQ, RS]) Stream(
	ctx context.Context,
	target address.Address,
) (stream freighter.ClientStream[RQ, RS], err error) {
	_, err = sc.Exec(
		freighter.Context{
			Context:  ctx,
			Target:   target,
			Protocol: protocol,
			Params:   make(freighter.Params),
		},
		freighter.FinalizerFunc(
			func(ctx freighter.Context) (oCtx freighter.Context, err error) {
				if target == "" {
					target = "localhost:0"
				}
				var (
					targetBufferSize int
					server           *StreamServer[RQ, RS]
				)
				if sc.server != nil {
					server = sc.server
					targetBufferSize = server.bufferSize
				} else if sc.network != nil {
					srv, ok := sc.network.resolveStreamTarget(target)
					if !ok || srv.boundHandler() == nil {
						return oCtx, address.NewTargetNotFoundError(target)
					}
					server = srv
					targetBufferSize = srv.bufferSize
				}
				var serverStream *ServerStream[RQ, RS]
				stream, serverStream = NewStreams[RQ, RS](
					ctx,
					sc.bufferSize,
					targetBufferSize,
				)
				return server.exec(ctx, serverStream)
			},
		),
	)
	if err != nil {
		return nil, err
	}
	return stream, nil
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

// ServerStream implements the freighter.ServerStream interface using go channels as the
// transport. Use NewStreams to construct one.
type ServerStream[RQ, RS freighter.Payload] struct {
	// ctx is the context the ServerStream was started with. Yes, Yes! I know this is a
	// bad practice, but in this case we're essentially using it as a data container,
	// and we have a very good grasp on how it's used.
	ctx          context.Context
	requests     <-chan message[RQ]
	responses    chan<- message[RS]
	serverClosed chan struct{}
	receiveErr   error
	sendErr      error
}

// Send implements the freighter.ServerStream interface.
func (ss *ServerStream[RQ, RS]) Send(res RS) error {
	if ss.sendErr != nil {
		return ss.sendErr
	}
	// A free buffer slot would otherwise win the race against an earlier cancel.
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

// Receive implements the freighter.ServerStream interface.
func (ss *ServerStream[RQ, RS]) Receive() (RQ, error) {
	if ss.receiveErr != nil {
		return types.Zero[RQ](), ss.receiveErr
	}
	// A buffered request would otherwise win the race against an earlier cancel.
	if ss.ctx.Err() != nil {
		return types.Zero[RQ](), ss.ctx.Err()
	}
	select {
	case <-ss.ctx.Done():
		return types.Zero[RQ](), ss.ctx.Err()
	case <-ss.serverClosed:
		return types.Zero[RQ](), freighter.ErrStreamClosed
	case msg := <-ss.requests:
		// Any error message means the Stream should die.
		if msg.err.Type != errors.TypeEmpty {
			ss.receiveErr = errors.Decode(ss.ctx, msg.err)
			return types.Zero[RQ](), ss.receiveErr
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
	// A client that abandons the stream stops receiving, so the final error can have
	// nowhere to go. Drop it rather than parking this goroutine for good.
	select {
	case ss.responses <- message[RS]{err: errPayload}:
	case <-ss.ctx.Done():
	}
}

// ClientStream implements the freighter.ClientStream interface using go channels as the
// transport. Use NewStreams to construct one.
type ClientStream[RQ, RS freighter.Payload] struct {
	// ctx is the context the ClientStream was started with. Yes, Yes! I know this is a
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

// Send implements the freighter.ClientStream interface.
func (cs *ClientStream[RQ, RS]) Send(req RQ) error {
	if cs.sendErr != nil {
		return cs.sendErr
	}
	if cs.receiveErr != nil {
		return freighter.EOF
	}
	// A free buffer slot would otherwise win the race against an earlier cancel.
	if cs.ctx.Err() != nil {
		return cs.ctx.Err()
	}
	select {
	case <-cs.ctx.Done():
		return cs.ctx.Err()
	case <-cs.clientClosed:
		return freighter.ErrStreamClosed
	case <-cs.serverClosed:
		// If the server was serverClosed, we set the sendErr to EOF and let
		// the client discover the server error by calling Receive.
		cs.sendErr = freighter.EOF
		return cs.sendErr
	case cs.requests <- message[RQ]{payload: req}:
		return nil
	}
}

// Receive implements the freighter.ClientStream interface.
func (cs *ClientStream[RQ, RS]) Receive() (RS, error) {
	if cs.receiveErr != nil {
		return types.Zero[RS](), cs.receiveErr
	}
	// A buffered response would otherwise win the race against an earlier cancel.
	if cs.ctx.Err() != nil {
		return types.Zero[RS](), cs.ctx.Err()
	}
	select {
	case <-cs.ctx.Done():
		return types.Zero[RS](), cs.ctx.Err()
	case msg := <-cs.responses:
		// If our message contains an error, that means the server serverClosed the
		// stream (i.e. serverClosed chan is serverClosed), so we don't need explicitly
		// listen for its closure.
		if msg.err.Type != errors.TypeEmpty {
			if cs.receiveErr == nil {
				cs.receiveErr = errors.Decode(cs.ctx, msg.err)
			}
			return types.Zero[RS](), cs.receiveErr
		}
		return msg.payload, nil
	}
}

// CloseSend implements the freighter.ClientStream interface.
func (cs *ClientStream[RQ, RS]) CloseSend() error {
	if cs.sendErr != nil {
		return nil
	}
	cs.sendErr = freighter.ErrStreamClosed
	defer close(cs.clientClosed)
	// A returned server or a cancelled context leaves the EOF nowhere to go. Drop it
	// rather than parking the caller for good on a full request buffer.
	select {
	case cs.requests <- message[RQ]{err: errors.Encode(cs.ctx, freighter.EOF, true)}:
	case <-cs.serverClosed:
	case <-cs.ctx.Done():
	}
	return nil
}

type message[P freighter.Payload] struct {
	payload P
	err     errors.Payload
}
