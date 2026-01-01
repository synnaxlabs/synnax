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
	"path"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	_ freighter.UnaryClient[any, any] = (*UnaryClient[any, any, any, any])(nil)
	_ freighter.UnaryServer[any, any] = (*UnaryServer[any, any, any, any])(nil)
)

// UnaryClient wraps a unary gRPC client to provide a freighter compatible interface.
// The unary gRPC service being internal must meet two requirements:
//
//  1. It contains only a single RPC method.
//  2. That RPC method must be named Exec for UnaryClient to properly implement the
//     interface.
type UnaryClient[RQ, RQT, RS, RST freighter.Payload] struct {
	TargetPrefix address.Address
	// RequestTranslator translates the given go request into a GRPC payload.
	// See Translator for more information.
	RequestTranslator Translator[RQ, RQT]
	// ResponseTranslator translates the given GRPC response into a go type.
	// See Translator for more information.
	ResponseTranslator Translator[RS, RST]
	// Pool is the pool of gRPC connections the transport will use to make requests.
	Pool *Pool
	// ServiceDesc is the gRPC service description that the transport will use to bind
	// to a gRPC service registrar.
	ServiceDesc *grpc.ServiceDesc
	// Exec is a function that executes the grpc request.
	Exec func(context.Context, grpc.ClientConnInterface, RQT) (RST, error)
	freighter.MiddlewareCollector
}

type UnaryServer[RQ, RQT, RS, RST freighter.Payload] struct {
	// Internal indicates whether the service is for go-to-go communication only, allowing
	// for more advanced error encoding that propagates stack traces, causes, etc.
	Internal bool
	// RequestTranslator translates the given GRPC request into a go type.
	// See Translator for more information.
	RequestTranslator Translator[RQ, RQT]
	// ResponseTranslator translates the given go response into a GRPC payload.
	// See Translator for more information.
	ResponseTranslator Translator[RS, RST]
	// ServiceDesc is the gRPC service description that the transport will use to bind
	// to a gRPC service registrar.
	ServiceDesc *grpc.ServiceDesc
	// handler is the handler that will be called when a request is received.
	handler func(context.Context, RQ) (RS, error)
	freighter.MiddlewareCollector
}

func (u *UnaryClient[RQ, RQT, RS, RST]) Report() alamos.Report {
	return Reporter.Report()
}

func (u *UnaryServer[RQ, RQT, RS, RST]) Report() alamos.Report {
	return Reporter.Report()
}

// BindTo implements the BindableTransport interface.
func (u *UnaryServer[RQ, RQT, RS, RST]) BindTo(reg grpc.ServiceRegistrar) {
	reg.RegisterService(u.ServiceDesc, u)
}

// Send implements the freighter.Transport interface.
func (u *UnaryClient[RQ, RQT, RS, RST]) Send(
	ctx context.Context,
	target address.Address,
	req RQ,
) (res RS, err error) {
	target = address.Address(path.Join(u.TargetPrefix.String(), target.String()))
	_, err = u.MiddlewareCollector.Exec(
		freighter.Context{
			Context:  ctx,
			Target:   address.Newf("%s.%s", target, u.ServiceDesc.ServiceName),
			Role:     freighter.Client,
			Protocol: Reporter.Protocol,
			Params:   make(freighter.Params),
			Variant:  freighter.Unary,
		},
		freighter.FinalizerFunc(func(iCtx freighter.Context) (oCtx freighter.Context, err error) {
			iCtx = attachContext(iCtx)
			conn, err := u.Pool.Acquire(target)
			if err != nil {
				return oCtx, err
			}
			tReq, err := u.RequestTranslator.Forward(iCtx, req)
			if err != nil {
				return oCtx, err
			}
			tRes, err := u.Exec(iCtx, conn.ClientConn, tReq)
			oCtx = freighter.Context{
				Context:  iCtx.Context,
				Protocol: iCtx.Protocol,
				Target:   address.Address(u.ServiceDesc.ServiceName),
				Params:   make(freighter.Params),
				Role:     iCtx.Role,
			}
			if err != nil {
				p := &errors.Payload{}
				p.Unmarshal(status.Convert(err).Message())
				return oCtx, errors.Decode(iCtx, *p)
			}
			res, err = u.ResponseTranslator.Backward(iCtx, tRes)
			return oCtx, err
		}),
	)
	return res, err
}

// Exec implements the GRPC service interface.
func (u *UnaryServer[RQ, RQT, RS, RST]) Exec(ctx context.Context, tReq RQT) (tRes RST, err error) {
	oCtx, err := u.MiddlewareCollector.Exec(
		parseServerContext(ctx, u.ServiceDesc.ServiceName, freighter.Unary),
		freighter.FinalizerFunc(func(ctx freighter.Context) (freighter.Context, error) {
			oCtx := freighter.Context{
				Context:  ctx.Context,
				Protocol: Reporter.Protocol,
				Target:   ctx.Target,
				Params:   ctx.Params,
				Role:     freighter.Server,
				Variant:  freighter.Unary,
			}
			if u.handler == nil {
				return oCtx, errors.New("[freighter] - no handler registered")
			}
			req, err := u.RequestTranslator.Backward(ctx, tReq)
			if err != nil {
				return oCtx, err
			}
			res, err := u.handler(ctx, req)
			if err != nil {
				return oCtx, err
			}
			tRes, err = u.ResponseTranslator.Forward(ctx, res)
			return oCtx, err
		},
		),
	)
	oCtx = attachContext(oCtx)
	if err == nil {
		return tRes, nil
	}
	return tRes, errors.Encode(oCtx, err, u.Internal)
}

func (u *UnaryServer[RQ, RQT, RS, RST]) BindHandler(
	handler func(context.Context, RQ) (RS, error),
) {
	u.handler = handler
}
