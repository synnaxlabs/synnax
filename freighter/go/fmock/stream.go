package fmock

import (
	"context"
	"fmt"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/freighter/ferrors"
	"github.com/arya-analytics/x/address"
)

type message[P freighter.Payload] struct {
	payload P
	error   ferrors.Payload
}

// StreamTransport is a mock implementation of the freighter.StreamTransport interface.
type StreamTransport[RQ, RS freighter.Payload] struct {
	Address    address.Address
	BufferSize int
	Network    *Network[RQ, RS]
	Handler    func(ctx context.Context, srv freighter.ServerStream[RQ, RS]) error
}

func NewStreamTransport[RQ, RS freighter.Payload](buffer int) *StreamTransport[RQ, RS] {
	return NewNetwork[RQ, RS]().RouteStream("", buffer)
}

// StreamTransport implements the freighter.StreamTransport interface.
func (s *StreamTransport[RQ, RS]) Stream(
	ctx context.Context,
	target address.Address,
) (freighter.ClientStream[RQ, RS], error) {
	if target == "" {
		target = "localhost:0"
	}
	route, ok := s.Network.StreamRoutes[target]
	if !ok || route.Handler == nil {
		return nil, address.TargetNotFound(target)
	}
	client, server := NewStreamPair[RQ, RS](ctx, s.BufferSize, route.BufferSize)
	go server.Exec(ctx, route.Handler)
	return client, nil
}

func (s *StreamTransport[RQ, RS]) Digest() freighter.Digest {
	return digest
}

// String implements the freighter interface.
func (s *StreamTransport[RQ, RS]) String() string {
	return fmt.Sprintf("mock.StreamTransport{} at %s", s.Address)
}

// BindHandler implements the freighter.StreamTransport interface.
func (s *StreamTransport[RQ, RS]) BindHandler(handler func(
	ctx context.Context,
	srv freighter.ServerStream[RQ, RS]) error) {
	s.Handler = handler
}

func NewStreamPair[RQ, RS freighter.Payload](ctx context.Context, buffers ...int) (*ClientStream[RQ, RS], *ServerStream[RQ, RS]) {
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

const defaultBuffer = 0

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
		return freighter.StreamClosed
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
		return req, freighter.StreamClosed
	case msg := <-s.requests:
		// Any error message means the StreamTransport should die.
		if msg.error.Type != ferrors.Empty {
			s.receiveErr = ferrors.Decode(msg.error)
			return req, s.receiveErr
		}
		return msg.payload, nil
	}
}

// Exec executes the provided handler function, where client is the client side of the
// server.
func (s *ServerStream[RQ, RS]) Exec(
	ctx context.Context,
	handler func(ctx context.Context, server freighter.ServerStream[RQ, RS]) error,
) {
	err := handler(ctx, s)
	errPayload := ferrors.Encode(err)
	if errPayload.Type == ferrors.Nil {
		errPayload = ferrors.Encode(freighter.EOF)
	}
	close(s.serverClosed)
	s.responses <- message[RS]{error: errPayload}
}

type ClientStream[RQ, RS freighter.Payload] struct {
	// ctx is the context the ServerStream was started with. Yes, Yes! RQ know this is a bad
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
		return freighter.StreamClosed
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
		if msg.error.Type != ferrors.Empty {
			if c.receiveErr == nil {
				c.receiveErr = ferrors.Decode(msg.error)
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
	c.sendErr = freighter.StreamClosed
	c.requests <- message[RQ]{error: ferrors.Encode(freighter.EOF)}
	close(c.clientClosed)
	return nil
}
