// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/x/address"
)

var _ = Describe("Unary", func() {
	var (
		net    *testutil.Network[testutil.Request, testutil.Response]
		client *testutil.UnaryClient[testutil.Request, testutil.Response]
	)

	BeforeEach(func() {
		net = testutil.NewNetwork[testutil.Request, testutil.Response]()
		client = net.UnaryClient()
	})

	It("Should exchange a request and a response with the bound handler", func(
		ctx SpecContext,
	) {
		server := net.UnaryServer("localhost:1")
		server.BindHandler(func(
			_ context.Context,
			req testutil.Request,
		) (testutil.Response, error) {
			return testutil.Response{ID: req.ID + 1, Message: req.Message}, nil
		})
		Expect(
			client.Send(ctx, "localhost:1", testutil.Request{ID: 1, Message: "Hello"}),
		).
			To(Equal(testutil.Response{ID: 2, Message: "Hello"}))
		Expect(net.Entries()).To(HaveLen(1))
		Expect(net.Entries()[0].Target).To(Equal(address.Address("localhost:1")))
	})

	It("Should return the handler error to the caller", func(ctx SpecContext) {
		server := net.UnaryServer("localhost:1")
		server.BindHandler(func(
			_ context.Context,
			_ testutil.Request,
		) (testutil.Response, error) {
			return testutil.Response{}, testutil.ErrCustom
		})
		Expect(client.Send(ctx, "localhost:1", testutil.Request{})).
			Error().
			To(MatchError(testutil.ErrCustom))
	})

	DescribeTable(
		"Should report the address the network assigned to the server",
		func(host address.Address, match OmegaMatcher) {
			Expect(net.UnaryServer(host).Address()).To(match)
		},
		Entry(
			"named host",
			address.Address("localhost:1"),
			Equal(address.Address("localhost:1")),
		),
		Entry("empty host", address.Address(""), Not(BeEmpty())),
	)

	DescribeTable(
		"Should return a target not found error",
		func(ctx SpecContext, target address.Address, bind bool) {
			server := net.UnaryServer("localhost:1")
			if bind {
				server.BindHandler(func(
					_ context.Context,
					_ testutil.Request,
				) (testutil.Response, error) {
					return testutil.Response{}, nil
				})
			}
			Expect(client.Send(ctx, target, testutil.Request{})).
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
