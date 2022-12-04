package fgrpc

import (
	"context"
	"io"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	"google.golang.org/grpc"
)

type StreamClientCore[RQ, RQT, RS, RST freighter.Payload] struct {
	RequestTranslator  Translator[RQ, RQT]
	ResponseTranslator Translator[RS, RST]
	ServiceDesc        *grpc.ServiceDesc
	Pool               *Pool
	ClientFunc         func(context.Context, grpc.ClientConnInterface) (GRPCClientStream[RQT, RST], error)
	freighter.MiddlewareCollector
}

type StreamServerCore[RQ, RQT, RS, RST freighter.Payload] struct {
	RequestTranslator  Translator[RQ, RQT]
	ResponseTranslator Translator[RS, RST]
	ServiceDesc        *grpc.ServiceDesc
	handler            func(context.Context, freighter.ServerStream[RQ, RS]) error
	freighter.MiddlewareCollector
}

func (s *StreamClientCore[RQ, RQT, RS, RST]) Report() alamos.Report {
	return reporter.Report()
}

func (s *StreamServerCore[RQ, RQT, RS, RST]) Report() alamos.Report {
	return reporter.Report()
}

func (s *StreamServerCore[RQ, RQT, RS, RST]) BindHandler(
	handler func(ctx context.Context, stream freighter.ServerStream[RQ, RS]) error,
) {
	s.handler = handler
}

func (s *StreamServerCore[RQ, RQT, RS, RST]) Handler(
	ctx context.Context, stream freighter.ServerStream[RQ, RS],
) error {
	oMD, err := s.MiddlewareCollector.Exec(
		ctx,
		parseMetaData(ctx, s.ServiceDesc.ServiceName),
		freighter.FinalizerFunc(func(ctx context.Context, md freighter.MD) (freighter.MD, error) {
			return freighter.MD{Protocol: md.Protocol, Params: make(freighter.Params)}, s.handler(ctx, stream)
		}),
	)
	attachMetaData(ctx, oMD)
	return err
}

func (s *StreamClientCore[RQ, RQT, RS, RST]) Stream(
	ctx context.Context,
	target address.Address,
) (stream freighter.ClientStream[RQ, RS], _ error) {
	_, err := s.MiddlewareCollector.Exec(
		ctx,
		freighter.MD{Target: target, Protocol: reporter.Protocol},
		freighter.FinalizerFunc(func(ctx context.Context, md freighter.MD) (oMD freighter.MD, err error) {
			conn, err := s.Pool.Acquire(target)
			if err != nil {
				return oMD, err
			}
			grpcClient, err := s.ClientFunc(ctx, conn.ClientConn)
			stream = s.Client(grpcClient)
			return parseMetaData(ctx, s.ServiceDesc.ServiceName), err
		}),
	)
	return stream, err
}

func (s *StreamServerCore[RQ, RQT, RS, RST]) Server(
	stream grpcServerStream[RQT, RST],
) freighter.ServerStream[RQ, RS] {
	return &ServerStream[RQ, RQT, RS, RST]{
		requestTranslator:  s.RequestTranslator,
		responseTranslator: s.ResponseTranslator,
		internal:           stream,
	}
}

func (s *StreamClientCore[RQ, RQT, RS, RST]) Client(
	stream GRPCClientStream[RQT, RST],
) freighter.ClientStream[RQ, RS] {
	return &ClientStream[RQ, RQT, RS, RST]{
		requestTranslator:  s.RequestTranslator,
		responseTranslator: s.ResponseTranslator,
		internal:           stream,
	}
}

// ServerStream wraps a grpc stream to implement the freighter.ServerStream interface.
type ServerStream[RQ, RQT, RS, RST freighter.Payload] struct {
	internal           grpcServerStream[RQT, RST]
	requestTranslator  Translator[RQ, RQT]
	responseTranslator Translator[RS, RST]
}

// Receive implements the freighter.ClientStream interface.
func (s *ServerStream[RQ, RQT, RS, RST]) Receive() (req RQ, err error) {
	tReq, err := s.internal.Recv()
	if err != nil {
		return req, translateGRPCError(err)
	}
	return s.requestTranslator.Backward(tReq)
}

// Send implements the freighter.ClientStream interface.
func (s *ServerStream[RQ, RQT, RS, RST]) Send(res RS) error {
	tRes, err := s.responseTranslator.Forward(res)
	if err != nil {
		return err
	}
	return translateGRPCError(s.internal.Send(tRes))
}

// ClientStream wraps a grpc stream to implement the freighter.ClientStream interface.
type ClientStream[RQ, RQT, RS, RST freighter.Payload] struct {
	internal           GRPCClientStream[RQT, RST]
	requestTranslator  Translator[RQ, RQT]
	responseTranslator Translator[RS, RST]
}

// Receive implements the freighter.ClientStream interface.
func (c *ClientStream[RQ, RQT, RS, RST]) Receive() (res RS, err error) {
	tRes, err := c.internal.Recv()
	if err != nil {
		return res, translateGRPCError(err)
	}
	return c.responseTranslator.Backward(tRes)
}

// Send implements the freighter.ClientStream interface.
func (c *ClientStream[RQ, RQT, RS, RST]) Send(req RQ) error {
	tReq, err := c.requestTranslator.Forward(req)
	if err != nil {
		return err
	}
	return translateGRPCError(c.internal.Send(tReq))
}

// CloseSend implements the freighter.ClientStream interface.
func (c *ClientStream[RQ, RQT, RS, RST]) CloseSend() error {
	return translateGRPCError(c.internal.CloseSend())
}

type grpcServerStream[RQ, RS freighter.Payload] interface {
	Send(msg RS) error
	Recv() (RQ, error)
}

type GRPCClientStream[RQ, RS freighter.Payload] interface {
	grpcServerStream[RS, RQ]
	CloseSend() error
}

func translateGRPCError(err error) error {
	if err == nil {
		return nil
	}
	if err == io.EOF {
		return freighter.EOF
	}
	return err
}
