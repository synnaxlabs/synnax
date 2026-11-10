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
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	. "github.com/synnaxlabs/freighter/grpc/testutil"
	v1 "github.com/synnaxlabs/freighter/grpc/v1"
	"github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
)

type streamServer struct {
	fgrpc.StreamServerCore[
		testutil.Request, *v1.Request,
		testutil.Response, *v1.Response,
	]
}

func (s *streamServer) Exec(
	stream v1.TestStreamService_ExecServer,
) error {
	return s.Handler(stream.Context(), stream)
}

func (s *streamServer) BindTo(reg grpc.ServiceRegistrar) {
	v1.RegisterTestStreamServiceServer(reg, s)
}

func startPair() (
	freighter.StreamServer[testutil.Request, testutil.Response],
	freighter.StreamClient[testutil.Request, testutil.Response],
	address.Address,
) {
	GinkgoHelper()
	var (
		server freighter.StreamServer[testutil.Request, testutil.Response]
		client freighter.StreamClient[testutil.Request, testutil.Response]
	)
	addr := StartServer(func(reg grpc.ServiceRegistrar, pool *fgrpc.Pool) {
		sServer := &streamServer{
			RequestTranslator:  requestBackwardTranslator{},
			ResponseTranslator: responseForwardTranslator{},
			ServiceDesc:        &v1.TestStreamService_ServiceDesc,
			Internal:           true,
		}
		sServer.BindTo(reg)
		server = &sServer.StreamServerCore

		client = &fgrpc.StreamClient[
			testutil.Request, *v1.Request,
			testutil.Response, *v1.Response,
		]{
			RequestTranslator:  requestTranslator{},
			ResponseTranslator: responseTranslator{},
			Pool:               pool,
			ServiceDesc:        &v1.TestStreamService_ServiceDesc,
			ClientFunc: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
			) (fgrpc.GRPCClientStream[*v1.Request, *v1.Response], error) {
				return v1.NewTestStreamServiceClient(conn).Exec(ctx)
			},
		}
	})
	return server, client, addr
}

var _ = Describe("Stream", Ordered, Serial, func() {
	var (
		server freighter.StreamServer[testutil.Request, testutil.Response]
		client freighter.StreamClient[testutil.Request, testutil.Response]
		addr   address.Address
	)

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		server, client, addr = startPair()
	})

	testutil.StreamSuite(func() (
		freighter.StreamServer[testutil.Request, testutil.Response],
		freighter.StreamClient[testutil.Request, testutil.Response],
		address.Address,
	) {
		return server, client, addr
	})

	It("Should return EOF on every receive after the client closes its send side",
		func(ctx SpecContext) {
			server, client, addr := startPair()
			errs := make(chan error, 2)
			server.BindHandler(func(
				_ context.Context,
				stream freighter.ServerStream[testutil.Request, testutil.Response],
			) error {
				_, err := stream.Receive()
				errs <- err
				_, err = stream.Receive()
				errs <- err
				return nil
			})
			stream := MustSucceed(client.Stream(ctx, addr))
			Expect(stream.CloseSend()).To(Succeed())
			Eventually(errs).Should(Receive(MatchError(freighter.EOF)))
			Eventually(errs).Should(Receive(MatchError(freighter.EOF)))
			Expect(stream.Receive()).Error().To(MatchError(freighter.EOF))
		},
	)

	It(
		"Should unblock a parked handler and tell the peer why when middleware cancels",
		func(ctx SpecContext) {
			server, client, addr := startPair()
			errEnded := errors.New("session ended")
			endSession := make(chan struct{})
			server.Use(freighter.MiddlewareFunc(func(
				ctx freighter.Context,
				next freighter.Next,
			) (freighter.Context, error) {
				dCtx, cancel := context.WithCancelCause(ctx.Context)
				defer cancel(nil)
				go func() {
					select {
					case <-endSession:
						cancel(errEnded)
					case <-dCtx.Done():
					}
				}()
				ctx.Context = dCtx
				oCtx, err := next(ctx)
				if cause := context.Cause(dCtx); errors.Is(cause, errEnded) {
					return oCtx, cause
				}
				return oCtx, err
			}))
			handlerEntered := make(chan struct{})
			received := make(chan error, 1)
			server.BindHandler(func(
				_ context.Context,
				stream freighter.ServerStream[testutil.Request, testutil.Response],
			) error {
				close(handlerEntered)
				_, err := stream.Receive()
				received <- err
				return err
			})
			stream := MustSucceed(client.Stream(ctx, addr))
			Eventually(handlerEntered).Should(BeClosed())
			close(endSession)
			Eventually(received).Should(Receive(MatchError(errEnded)))
			Expect(stream.Receive()).Error().
				To(MatchError(ContainSubstring("session ended")))
		},
	)

	It(
		"Should unblock a handler stuck sending to a peer that stopped reading",
		func(ctx SpecContext) {
			server, client, addr := startPair()
			errEnded := errors.New("session ended")
			endSession := make(chan struct{})
			server.Use(freighter.MiddlewareFunc(func(
				ctx freighter.Context,
				next freighter.Next,
			) (freighter.Context, error) {
				dCtx, cancel := context.WithCancelCause(ctx.Context)
				defer cancel(nil)
				go func() {
					select {
					case <-endSession:
						cancel(errEnded)
					case <-dCtx.Done():
					}
				}()
				ctx.Context = dCtx
				return next(ctx)
			}))
			var (
				sent     atomic.Int64
				lastSend atomic.Int64
				sendErr  = make(chan error, 1)
				payload  = strings.Repeat("x", 256*1024)
			)
			server.BindHandler(func(
				_ context.Context,
				stream freighter.ServerStream[testutil.Request, testutil.Response],
			) error {
				for {
					if err := stream.Send(
						testutil.Response{Message: payload},
					); err != nil {
						sendErr <- err
						return err
					}
					sent.Add(1)
					lastSend.Store(time.Now().UnixNano())
				}
			})
			stream := MustSucceed(client.Stream(ctx, addr))
			Eventually(func() bool {
				last := lastSend.Load()
				return sent.Load() > 0 &&
					time.Since(time.Unix(0, last)) > 250*time.Millisecond
			}).WithTimeout(10 * time.Second).Should(BeTrue())
			close(endSession)
			Eventually(sendErr).Should(Receive(MatchError(errEnded)))
			Expect(stream.CloseSend()).To(Succeed())
		},
	)
})
