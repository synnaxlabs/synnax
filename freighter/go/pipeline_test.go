// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter_test

import (
	"context"
	"io"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
)

type (
	pipelineServer = freighter.PipelineServer[request]
	pipelineClient = freighter.PipelineClient[request]
)

type testPipelineServer struct {
	address address.Address
	handler func(context.Context, request) (io.Reader, error)
	freighter.Reporter
	freighter.MiddlewareCollector
}

var _ pipelineServer = (*testPipelineServer)(nil)

func (tps *testPipelineServer) BindHandler(handler func(context.Context, request) (io.Reader, error)) {
	tps.handler = handler
}

func (tps *testPipelineServer) exec(ctx freighter.Context, req request) (io.Reader, freighter.Context, error) {
	var reader io.Reader
	oMD, err := tps.MiddlewareCollector.Exec(
		ctx,
		freighter.FinalizerFunc(func(ctx freighter.Context) (freighter.Context, error) {
			var err error
			reader, err = tps.handler(ctx.Context, req)
			if err != nil {
				return freighter.Context{}, err
			}
			return freighter.Context{
				Context:  ctx,
				Target:   ctx.Target,
				Protocol: tps.Reporter.Protocol,
			}, nil
		}),
	)
	if err != nil {
		return nil, freighter.Context{}, err
	}
	return reader, oMD, nil
}

type testPipelineClient struct {
	server *testPipelineServer
	freighter.Reporter
	freighter.MiddlewareCollector
}

var _ pipelineClient = (*testPipelineClient)(nil)

func (tpc *testPipelineClient) Send(ctx context.Context, target address.Address, req request) (io.ReadCloser, error) {
	var reader io.Reader
	_, err := tpc.MiddlewareCollector.Exec(
		freighter.Context{Context: ctx, Target: target, Protocol: tpc.Reporter.Protocol},
		freighter.FinalizerFunc(func(ctx freighter.Context) (freighter.Context, error) {
			var err error
			reader, _, err = tpc.server.exec(ctx, req)
			if err != nil {
				return freighter.Context{}, err
			}
			return freighter.Context{
				Context:  ctx,
				Target:   target,
				Protocol: tpc.Reporter.Protocol,
			}, nil
		}),
	)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(reader), nil
}

type pipelineImplementation interface {
	start(address.Address) (pipelineServer, pipelineClient)
	stop() error
}

type testPipelineImplementation struct {
}

var _ pipelineImplementation = (*testPipelineImplementation)(nil)

func (tpi *testPipelineImplementation) start(host address.Address) (pipelineServer, pipelineClient) {
	server := &testPipelineServer{
		MiddlewareCollector: freighter.MiddlewareCollector{},
	}
	client := &testPipelineClient{
		server:              server,
		MiddlewareCollector: freighter.MiddlewareCollector{},
	}
	return server, client
}

func (tpi *testPipelineImplementation) stop() error {
	return nil
}

var pipelineImplementations = []pipelineImplementation{&testPipelineImplementation{}}

var _ = Describe("Pipeline", Ordered, Serial, func() {
	Describe("Implementation Tests", func() {
		for _, impl := range pipelineImplementations {
			const addr = "localhost:8081"
			var (
				server pipelineServer
				client pipelineClient
			)
			BeforeAll(func() {
				server, client = impl.start(addr)
			})
			AfterAll(func() {
				Expect(impl.stop()).To(Succeed())
			})
			Describe("Normal Operation", func() {
				It("should send a request", func() {
					server.BindHandler(func(_ context.Context, req request) (io.Reader, error) {
						return strings.NewReader(req.Message), nil
					})
					req := request{ID: 1, Message: "hello"}
					Expect(client.Send(context.Background(), addr, req)).To(Equal(io.NopCloser(strings.NewReader(req.Message))))
				})
			})
			Describe("Error handling", func() {
				It("should return an error if the server returns an error", func() {
					server.BindHandler(func(_ context.Context, _ request) (io.Reader, error) {
						return nil, errors.New("test error")
					})
					req := request{ID: 1, Message: "hello"}
					Expect(client.Send(context.Background(), addr, req)).Error().To(MatchError("test error"))
				})
			})
			Describe("Middleware", Focus, func() {
				It("should call the middleware", func() {
					c := 0
					client.Use(freighter.MiddlewareFunc(func(ctx freighter.Context, next freighter.Next) (freighter.Context, error) {
						c++
						return next(ctx)
					}))
					s := 0
					server.Use(freighter.MiddlewareFunc(func(ctx freighter.Context, next freighter.Next) (freighter.Context, error) {
						s++
						return next(ctx)
					}))
					server.BindHandler(func(_ context.Context, req request) (io.Reader, error) {
						return strings.NewReader(req.Message), nil
					})
					req := request{ID: 1, Message: "hello"}
					Expect(client.Send(context.Background(), addr, req)).To(Equal(io.NopCloser(strings.NewReader(req.Message))))
					Expect(c).To(Equal(1))
					Expect(s).To(Equal(1))
				})
			})
		}
	})
})
