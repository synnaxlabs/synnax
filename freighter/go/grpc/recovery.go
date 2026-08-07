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
	"runtime/debug"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryUnaryServerInterceptor returns a grpc.UnaryServerInterceptor that recovers
// from panics in unary handlers, records the stack via ins, and returns a
// codes.Internal status. It is a transport-level backstop for gRPC handlers that do not
// pass through the freighter recovery Middleware, such as cluster-internal services.
func RecoveryUnaryServerInterceptor(
	ins alamos.Instrumentation,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				recovery.LogPanic(ins, info.FullMethod, r, debug.Stack())
				err = status.Error(codes.Internal, recovery.ErrPanic.Error())
			}
		}()
		return handler(ctx, req)
	}
}

// RecoveryStreamServerInterceptor returns a grpc.StreamServerInterceptor that recovers
// from panics in streaming handlers, mirroring RecoveryUnaryServerInterceptor.
func RecoveryStreamServerInterceptor(
	ins alamos.Instrumentation,
) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				recovery.LogPanic(ins, info.FullMethod, r, debug.Stack())
				err = status.Error(codes.Internal, recovery.ErrPanic.Error())
			}
		}()
		return handler(srv, ss)
	}
}
