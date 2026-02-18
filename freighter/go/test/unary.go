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

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/testutil"
)

func UnarySuite(
	deps func() (
		freighter.UnaryServer[Request, Response],
		freighter.UnaryClient[Request, Response],
		address.Address,
	),
) {
	var ctx context.Context
	ginkgo.BeforeEach(func() {
		ctx = context.Background()
	})
	ginkgo.Describe("Normal Operation", func() {
		ginkgo.It("should send a request", func() {
			server, client, addr := deps()
			server.BindHandler(func(ctx context.Context, req Request) (Response, error) {
				return Response(req), nil
			})
			res := testutil.MustSucceed(client.Send(ctx, addr, Request{ID: 1, Message: "hello"}))
			gomega.Expect(res).To(gomega.Equal(Response{ID: 1, Message: "hello"}))
		})
	})

	ginkgo.Describe("Details Handling", func() {
		ginkgo.It("Should correctly return a custom error to the client", func() {
			server, client, addr := deps()
			server.BindHandler(func(ctx context.Context, req Request) (Response, error) {
				return Response{}, ErrCustom
			})
			gomega.Expect(client.Send(ctx, addr, Request{ID: 1, Message: "hello"})).Error().To(gomega.MatchError(ErrCustom))
		})
	})

	ginkgo.Describe("Middleware", func() {
		ginkgo.It("Should correctly call the middleware", func() {
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
			gomega.Expect(client.Send(ctx, addr, Request{ID: 1, Message: "hello"})).Error().ToNot(gomega.HaveOccurred())
			gomega.Expect(c).To(gomega.Equal(2))
		})
	})
}
