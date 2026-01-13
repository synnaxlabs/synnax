// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fgrpc

import (
	"context"
	"io"

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
	RequestTranslator  Translator[RQ, RQT]
	ResponseTranslator Translator[RS, RST]
	CreateTranslators  func() (Translator[RQ, RQT], Translator[RS, RST])
	ServiceDesc        *grpc.ServiceDesc
	Internal           bool
	handler            func(context.Context, freighter.ServerStream[RQ, RS]) error
	freighter.MiddlewareCollector
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
				return md, err
			}
			return freighter.Context{
				Context:  md.Context,
				Protocol: md.Protocol,
				Params:   make(freighter.Params),
			}, s.handler(md, s.adaptStream(stream))
		}),
	)
	if !attachedInitialMetaData {
		md := metadata.Pairs()
		for k, v := range oCtx.Params {
			if vStr, ok := v.(string); ok {
				md.Append(k, vStr)
			}
		}
		if err != nil {
			md.Append("error", errors.Encode(ctx, err, s.Internal).Error())
		}
		if err := stream.SendHeader(md); err != nil {
			return err
		}
	}
	if err == nil {
		return nil
	}
	oCtx = attachContext(oCtx)
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
		freighter.FinalizerFunc(func(ctx freighter.Context) (oCtx freighter.Context, err error) {
			ctx = attachContext(ctx)
			conn, err := s.Pool.Acquire(target)
			if err != nil {
				return oCtx, err
			}
			grpcClient, err := s.ClientFunc(ctx, conn.ClientConn)
			stream = s.adaptStream(grpcClient)
			return freighter.Context{
				Context:  ctx.Context,
				Role:     ctx.Role,
				Protocol: ctx.Protocol,
				Target:   target,
				Params:   make(freighter.Params),
				Variant:  ctx.Variant,
			}, err
		}),
	)
	return stream, err
}

func (s *StreamServerCore[RQ, RQT, RS, RST]) adaptStream(
	stream grpcServerStream[RQT, RST],
) freighter.ServerStream[RQ, RS] {
	var (
		requestTranslator  = s.RequestTranslator
		responseTranslator = s.ResponseTranslator
	)
	if s.CreateTranslators != nil {
		requestTranslator, responseTranslator = s.CreateTranslators()
	}
	return &ServerStream[RQ, RQT, RS, RST]{
		requestTranslator:  requestTranslator,
		responseTranslator: responseTranslator,
		internal:           stream,
	}
}

func (s *StreamClient[RQ, RQT, RS, RST]) adaptStream(
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
	return s.requestTranslator.Backward(s.internal.Context(), tReq)
}

// Send implements the freighter.ClientStream interface.
func (s *ServerStream[RQ, RQT, RS, RST]) Send(res RS) error {
	tRes, err := s.responseTranslator.Forward(s.internal.Context(), res)
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
	return c.responseTranslator.Backward(c.internal.Context(), tRes)
}

// Send implements the freighter.ClientStream interface.
func (c *ClientStream[RQ, RQT, RS, RST]) Send(req RQ) error {
	tReq, err := c.requestTranslator.Forward(c.internal.Context(), req)
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
		err = context.Canceled
	}
	return errors.WithStackDepth(err, 1)
}
