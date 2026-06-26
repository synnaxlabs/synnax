// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/freighter/recovery"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("Recovery Interceptors", func() {
	Describe("Unary", func() {
		It("should recover a panic and return a codes.Internal status", func() {
			interceptor := fgrpc.RecoveryUnaryServerInterceptor(alamos.Instrumentation{})
			info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
			_, err := interceptor(context.Background(), nil, info,
				func(context.Context, any) (any, error) { panic("boom") },
			)
			Expect(status.Code(err)).To(Equal(codes.Internal))
			Expect(err).To(MatchError(ContainSubstring(recovery.ErrPanic.Error())))
		})

		It("should pass through a non-panicking handler", func() {
			interceptor := fgrpc.RecoveryUnaryServerInterceptor(alamos.Instrumentation{})
			info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
			resp := MustSucceed(interceptor(context.Background(), "req", info,
				func(_ context.Context, req any) (any, error) { return req, nil },
			))
			Expect(resp).To(Equal("req"))
		})
	})

	Describe("Stream", func() {
		It("should recover a panic and return a codes.Internal status", func() {
			interceptor := fgrpc.RecoveryStreamServerInterceptor(alamos.Instrumentation{})
			info := &grpc.StreamServerInfo{FullMethod: "/test.Service/Stream"}
			err := interceptor(nil, nil, info,
				func(any, grpc.ServerStream) error { panic("boom") },
			)
			Expect(status.Code(err)).To(Equal(codes.Internal))
			Expect(err).To(MatchError(ContainSubstring(recovery.ErrPanic.Error())))
		})
	})
})
