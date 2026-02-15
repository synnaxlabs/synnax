// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
)

func UnarySuite(
	deps func() (
		freighter.UnaryServer[Request, Response],
		freighter.UnaryClient[Request, Response],
		address.Address,
	),
) {
	Describe("Normal Operation", func() {
		It("should send a request", func() {
			server, client, addr := deps()
			server.BindHandler(func(ctx context.Context, req Request) (Response, error) {
				return Response(req), nil
			})
			res := MustSucceed(client.Send(context.TODO(), addr, Request{ID: 1, Message: "hello"}))
			Expect(res).To(Equal(Response{ID: 1, Message: "hello"}))
		})
	})

	Describe("Details Handling", func() {
		It("Should correctly return a custom error to the client", func() {
			server, client, addr := deps()
			server.BindHandler(func(ctx context.Context, req Request) (Response, error) {
				return Response{}, ErrCustom
			})
			Expect(client.Send(context.TODO(), addr, Request{ID: 1, Message: "hello"})).Error().To(MatchError(ErrCustom))
		})
	})

	Describe("Middleware", func() {
		It("Should correctly call the middleware", func() {
			server, client, addr := deps()
			c := 0
			server.Use(freighter.MiddlewareFunc(func(
				ctx freighter.Context,
				next freighter.Next,
			) (freighter.Context, error) {
				c++
				oMd, err := next(ctx)
				c++
				return oMd, err
			}))
			server.BindHandler(func(ctx context.Context, req Request) (Response, error) {
				return Response{}, nil
			})
			Expect(client.Send(context.TODO(), addr, Request{ID: 1, Message: "hello"})).Error().ToNot(HaveOccurred())
			Expect(c).To(Equal(2))
		})
	})
}
