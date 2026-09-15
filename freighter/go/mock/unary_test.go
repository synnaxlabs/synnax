// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/freighter/test"
	"github.com/synnaxlabs/x/address"
)

var _ = Describe("Unary", func() {
	var (
		net    *mock.Network[test.Request, test.Response]
		client *mock.UnaryClient[test.Request, test.Response]
	)

	BeforeEach(func() {
		net = mock.NewNetwork[test.Request, test.Response]()
		client = net.UnaryClient()
	})

	It("Should exchange a request and a response with the bound handler", func(
		ctx SpecContext,
	) {
		server := net.UnaryServer("localhost:1")
		server.BindHandler(func(
			_ context.Context,
			req test.Request,
		) (test.Response, error) {
			return test.Response{ID: req.ID + 1, Message: req.Message}, nil
		})
		Expect(client.Send(ctx, "localhost:1", test.Request{ID: 1, Message: "Hello"})).
			To(Equal(test.Response{ID: 2, Message: "Hello"}))
		Expect(net.Entries()).To(HaveLen(1))
		Expect(net.Entries()[0].Target).To(Equal(address.Address("localhost:1")))
	})

	It("Should return the handler error to the caller", func(ctx SpecContext) {
		server := net.UnaryServer("localhost:1")
		server.BindHandler(func(
			_ context.Context,
			_ test.Request,
		) (test.Response, error) {
			return test.Response{}, test.ErrCustom
		})
		Expect(client.Send(ctx, "localhost:1", test.Request{})).
			Error().
			To(MatchError(test.ErrCustom))
	})

	DescribeTable(
		"Should return a target not found error",
		func(ctx SpecContext, target address.Address, bind bool) {
			server := net.UnaryServer("localhost:1")
			if bind {
				server.BindHandler(func(
					_ context.Context,
					_ test.Request,
				) (test.Response, error) {
					return test.Response{}, nil
				})
			}
			Expect(client.Send(ctx, target, test.Request{})).
				Error().
				To(SatisfyAll(
					MatchError(address.ErrNotFound),
					MatchError(ContainSubstring(target.String())),
				))
		},
		Entry("when no server hosts the target", address.Address("localhost:2"), true),
		Entry("when the server has no handler", address.Address("localhost:1"), false),
	)
})
