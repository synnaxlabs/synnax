// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
)

type (
	UnaryServer = freighter.UnaryServer[Request, Response]
	UnaryClient = freighter.UnaryClient[Request, Response]
)

type UnaryImplementation interface {
	Start(address.Address) (UnaryServer, UnaryClient)
	Stop() error
}

func AssertUnary(impl UnaryImplementation) {
	Describe("Fulfills UnaryImplementation", Ordered, Serial, func() {
		var (
			addr   address.Address
			server UnaryServer
			client UnaryClient
			req    Request
		)
		BeforeAll(func() {
			addr = "localhost:8081"
			server, client = impl.Start(addr)
		})
		AfterAll(func() {
			Expect(impl.Stop()).To(Succeed())
		})
		BeforeEach(func() {
			req = Request{ID: 1, Message: "hello"}
		})
		Describe("Normal Operation", func() {
			It("should send a request", func() {
				server.BindHandler(func(
					_ context.Context,
					req Request,
				) (Response, error) {
					return req, nil
				})
				Expect(client.Send(context.Background(), addr, req)).To(Equal(req))
			})
		})
		Describe("Details Handling", func() {
			It("Should correctly return a custom error to the client", func() {
				server.BindHandler(func(context.Context, Request) (Response, error) {
					return Response{}, errTest
				})
				Expect(client.Send(context.Background(), addr, req)).Error().
					To(MatchError(ContainSubstring(testErrorType)))
			})
		})
		Describe("Middleware", func() {
			It("Should correctly call the middleware", func() {
				c := 0
				server.Use(freighter.MiddlewareFunc(func(
					ctx freighter.Context,
					next freighter.MiddlewareHandler,
				) (freighter.Context, error) {
					c++
					oMd, err := next(ctx)
					if err != nil {
						return freighter.Context{}, err
					}
					c++
					return oMd, nil
				}))
				server.BindHandler(func(context.Context, Request) (Response, error) {
					return Response{}, nil
				})
				Expect(client.Send(context.Background(), addr, req)).
					Error().To(Not(HaveOccurred()))
				Expect(c).To(Equal(2))
			})
		})
	})
}
