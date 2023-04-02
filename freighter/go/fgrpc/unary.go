// Copyright 2023 Synnax Labs, Inc.
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
	roacherrors "github.com/cockroachdb/errors"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"google.golang.org/grpc"
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
	// Client is a function that executes the grpc request.
	Client func(context.Context, grpc.ClientConnInterface, RQT) (RST, error)
	freighter.MiddlewareCollector
}

type UnaryServer[RQ, RQT, RS, RST freighter.Payload] struct {
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
	return reporter.Report()
}

func (u *UnaryServer[RQ, RQT, RS, RST]) Report() alamos.Report {
	return reporter.Report()
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
	_, err = u.MiddlewareCollector.Exec(
		ctx,
		freighter.MD{Target: target, Protocol: reporter.Protocol},
		freighter.FinalizerFunc(func(ctx context.Context, md freighter.MD) (oMD freighter.MD, err error) {
			attachMetaData(ctx, md)
			conn, err := u.Pool.Acquire(target)
			if err != nil {
				return oMD, err
			}
			tReq, err := u.RequestTranslator.Forward(req)
			if err != nil {
				return oMD, err
			}
			tRes, err := u.Client(ctx, conn.ClientConn, tReq)
			parseMetaData(ctx, u.ServiceDesc.ServiceName)
			if err != nil {
				return oMD, err
			}
			res, err = u.ResponseTranslator.Backward(tRes)
			return oMD, err
		}),
	)
	return res, err
}

// Exec implements the GRPC service interface.
func (u *UnaryServer[RQ, RQT, RS, RST]) Exec(ctx context.Context, tReq RQT) (tRes RST, err error) {
	oMD, err := u.MiddlewareCollector.Exec(
		ctx,
		parseMetaData(ctx, u.ServiceDesc.ServiceName),
		freighter.FinalizerFunc(func(ctx context.Context, md freighter.MD) (freighter.MD, error) {
			oMD := freighter.MD{Protocol: reporter.Protocol, Target: md.Target, Params: md.Params}
			if u.handler == nil {
				return oMD, roacherrors.New("[freighter] - no handler registered")
			}
			req, err := u.RequestTranslator.Backward(tReq)
			if err != nil {
				return oMD, err
			}
			res, err := u.handler(ctx, req)
			if err != nil {
				return oMD, err
			}
			tRes, err = u.ResponseTranslator.Forward(res)
			return oMD, err
		},
		),
	)
	attachMetaData(ctx, oMD)
	return tRes, err
}

func (u *UnaryServer[RQ, RQT, RS, RST]) BindHandler(
	handler func(context.Context, RQ) (RS, error),
) {
	u.handler = handler
}
