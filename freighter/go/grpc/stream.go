// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc

import (
	"context"
	"io"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type StreamClient[RQ, RQT, RS, RST freighter.Payload] struct {
	RequestTranslator  Translator[RQ, RQT]
	ResponseTranslator Translator[RS, RST]
	ServiceDesc        *grpc.ServiceDesc
	Pool               *Pool
	ClientFunc         func(context.Context, grpc.ClientConnInterface) (GRPCClientStream[RQT, RST], error)
	freighter.MiddlewareCollector
}

type StreamServerCore[RQ, RQT, RS, RST freighter.Payload] struct {
	RequestTranslator  BackwardTranslator[RQ, RQT]
	ResponseTranslator ForwardTranslator[RS, RST]
	CreateTranslators  func() (BackwardTranslator[RQ, RQT], ForwardTranslator[RS, RST])
	ServiceDesc        *grpc.ServiceDesc
	handler            func(context.Context, freighter.ServerStream[RQ, RS]) error
	freighter.MiddlewareCollector
	Internal bool
}

func (s *StreamClient[RQ, RQT, RS, RST]) Report() alamos.Report {
	return Reporter.Report()
}

func (s *StreamServerCore[RQ, RQT, RS, RST]) Report() alamos.Report {
	return Reporter.Report()
}

func (s *StreamServerCore[RQ, RQT, RS, RST]) BindHandler(
	handler func(ctx context.Context, stream freighter.ServerStream[RQ, RS]) error,
) {
	s.handler = handler
}

func (s *StreamServerCore[RQ, RQT, RS, RST]) Handler(
	ctx context.Context, stream grpcServerStream[RQT, RST],
) error {
	attachedInitialMetaData := false
	oCtx, err := s.Exec(
		parseServerContext(ctx, s.ServiceDesc.ServiceName, freighter.VariantStream),
		freighter.FinalizerFunc(func(md freighter.Context) (freighter.Context, error) {
			attachedInitialMetaData = true
			if err := stream.SendHeader(metadata.Pairs()); err != nil {
				if md.Err() != nil {
					return md, s.handler(md, s.adaptStream(md.Context, stream))
				}
				return md, err
			}
			return freighter.Context{
				Context:  md.Context,
				Protocol: md.Protocol,
				Params:   make(freighter.Params),
			}, s.handler(md, s.adaptStream(md.Context, stream))
		}),
	)
	if !attachedInitialMetaData {
		md := metadata.Pairs()
		if err != nil {
			md.Append("error", errors.Encode(ctx, err, s.Internal).Error())
		} else {
			for k, v := range oCtx.Params {
				if vStr, ok := v.(string); ok {
					md.Append(k, vStr)
				}
			}
		}
		if err := stream.SendHeader(md); err != nil {
			return err
		}
	}
	if err == nil || errors.Is(err, io.EOF) {
		return nil
	}
	return errors.Encode(ctx, err, s.Internal)
}

func (s *StreamClient[RQ, RQT, RS, RST]) Stream(
	ctx context.Context,
	target address.Address,
) (stream freighter.ClientStream[RQ, RS], _ error) {
	_, err := s.Exec(
		freighter.Context{
			Context:  ctx,
			Variant:  freighter.VariantStream,
			Role:     freighter.RoleClient,
			Target:   target,
			Protocol: Reporter.Protocol,
			Params:   make(freighter.Params),
		},
		freighter.FinalizerFunc(
			func(ctx freighter.Context) (oCtx freighter.Context, err error) {
				ctx = attachContext(ctx)
				conn, err := s.Pool.Acquire(target)
				if err != nil {
					return oCtx, err
				}
				grpcClient, err := s.ClientFunc(ctx, conn.ClientConn)
				oCtx = freighter.Context{
					Context:  ctx.Context,
					Role:     ctx.Role,
					Protocol: ctx.Protocol,
					Target:   target,
					Params:   make(freighter.Params),
					Variant:  ctx.Variant,
				}
				if err != nil {
					return oCtx, err
				}
				md, hdrErr := grpcClient.Header()
				if hdrErr != nil {
					return oCtx, hdrErr
				}
				if errVals := md.Get("error"); len(errVals) > 0 {
					p := &errors.Payload{}
					p.Unmarshal(errVals[0])
					return oCtx, errors.Decode(ctx, *p)
				}
				stream = s.adaptStream(grpcClient)
				return oCtx, nil
			},
		),
	)
	return stream, err
}

func (s *StreamServerCore[RQ, RQT, RS, RST]) adaptStream(
	ctx context.Context,
	stream grpcServerStream[RQT, RST],
) freighter.ServerStream[RQ, RS] {
	var (
		requestTranslator  = s.RequestTranslator
		responseTranslator = s.ResponseTranslator
	)
	if s.CreateTranslators != nil {
		requestTranslator, responseTranslator = s.CreateTranslators()
	}
	return &serverStream[RQ, RQT, RS, RST]{
		ctx:                ctx,
		requestTranslator:  requestTranslator,
		responseTranslator: responseTranslator,
		internal:           stream,
	}
}

func (s *StreamClient[RQ, RQT, RS, RST]) adaptStream(
	stream GRPCClientStream[RQT, RST],
) freighter.ClientStream[RQ, RS] {
	return &clientStream[RQ, RQT, RS, RST]{
		requestTranslator:  s.RequestTranslator,
		responseTranslator: s.ResponseTranslator,
		internal:           stream,
	}
}

type serverStream[RQ, RQT, RS, RST freighter.Payload] struct {
	// ctx is the handler's context, which a parked Receive or Send also watches.
	ctx                context.Context
	internal           grpcServerStream[RQT, RST]
	requestTranslator  BackwardTranslator[RQ, RQT]
	responseTranslator ForwardTranslator[RS, RST]
	recvOnce           sync.Once
	recv               chan recvResult[RQT]
	// recvErr is returned again by every later Receive.
	recvErr  error
	sendOnce sync.Once
	sendReq  chan RST
	// sendRes buffers one result so the loop never blocks on a Send that gave up.
	sendRes chan error
}

func (s *serverStream[RQ, RQT, RS, RST]) sendLoop() {
	for {
		select {
		case msg := <-s.sendReq:
			s.sendRes <- s.internal.Send(msg)
		case <-s.ctx.Done():
			return
		}
	}
}

type recvResult[RQT freighter.Payload] struct {
	msg RQT
	err error
}

func (s *serverStream[RQ, RQT, RS, RST]) receiveLoop() {
	defer close(s.recv)
	for {
		msg, err := s.internal.Recv()
		select {
		case s.recv <- recvResult[RQT]{msg: msg, err: err}:
		case <-s.ctx.Done():
			return
		}
		if err != nil {
			return
		}
	}
}

// Receive implements the freighter.ServerStream interface. It returns the handler
// context's cause when that context ends before a message arrives.
func (s *serverStream[RQ, RQT, RS, RST]) Receive() (req RQ, err error) {
	if s.recvErr != nil {
		return req, s.recvErr
	}
	s.recvOnce.Do(func() {
		s.recv = make(chan recvResult[RQT])
		go s.receiveLoop()
	})
	select {
	case <-s.ctx.Done():
		return req, context.Cause(s.ctx)
	case r, ok := <-s.recv:
		if !ok {
			return req, context.Cause(s.ctx)
		}
		if r.err != nil {
			s.recvErr = translateGRPCError(r.err)
			return req, s.recvErr
		}
		return s.requestTranslator.Backward(s.ctx, r.msg)
	}
}

// Send implements the freighter.ServerStream interface. It returns the handler
// context's cause when that context ends before the transport accepts the message.
func (s *serverStream[RQ, RQT, RS, RST]) Send(res RS) error {
	if s.ctx.Err() != nil {
		return context.Cause(s.ctx)
	}
	tRes, err := s.responseTranslator.Forward(s.ctx, res)
	if err != nil {
		return err
	}
	s.sendOnce.Do(func() {
		s.sendReq = make(chan RST)
		s.sendRes = make(chan error, 1)
		go s.sendLoop()
	})
	select {
	case s.sendReq <- tRes:
	case <-s.ctx.Done():
		return context.Cause(s.ctx)
	}
	select {
	case err := <-s.sendRes:
		return translateGRPCError(err)
	case <-s.ctx.Done():
		return context.Cause(s.ctx)
	}
}

type clientStream[RQ, RQT, RS, RST freighter.Payload] struct {
	internal           GRPCClientStream[RQT, RST]
	requestTranslator  Translator[RQ, RQT]
	responseTranslator Translator[RS, RST]
	closeSent          bool
}

// Receive implements the freighter.ClientStream interface.
func (c *clientStream[RQ, RQT, RS, RST]) Receive() (res RS, err error) {
	tRes, err := c.internal.Recv()
	if err != nil {
		return res, translateGRPCError(err)
	}
	return c.responseTranslator.Backward(c.internal.Context(), tRes)
}

// Send implements the freighter.ClientStream interface.
func (c *clientStream[RQ, RQT, RS, RST]) Send(req RQ) error {
	if c.closeSent {
		return freighter.ErrStreamClosed
	}
	tReq, err := c.requestTranslator.Forward(c.internal.Context(), req)
	if err != nil {
		return err
	}
	return translateGRPCError(c.internal.Send(tReq))
}

// CloseSend implements the freighter.ClientStream interface.
func (c *clientStream[RQ, RQT, RS, RST]) CloseSend() error {
	c.closeSent = true
	return translateGRPCError(c.internal.CloseSend())
}

type grpcServerStream[RQ, RS freighter.Payload] interface {
	Context() context.Context
	Send(msg RS) error
	Recv() (RQ, error)
	RecvMsg(any) error
	SetHeader(metadata.MD) error
	SendHeader(metadata.MD) error
	SetTrailer(metadata.MD)
}

type GRPCClientStream[RQ, RS freighter.Payload] interface {
	Context() context.Context
	Send(msg RQ) error
	Recv() (RS, error)
	CloseSend() error
	Header() (metadata.MD, error)
}

func translateGRPCError(err error) error {
	if err == nil {
		return err
	}
	if errors.Is(err, io.EOF) {
		return freighter.EOF
	}
	s := status.Convert(err)
	if s.Code() == codes.Canceled {
		return context.Canceled
	}
	p := &errors.Payload{}
	p.Unmarshal(s.Message())
	if decoded := errors.Decode(context.Background(), *p); decoded != nil {
		return decoded
	}
	return errors.WithStackDepth(err, 1)
}
