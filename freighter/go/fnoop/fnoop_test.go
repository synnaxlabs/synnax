// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fnoop_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fnoop"
)

var _ = Describe("Fnoop", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
	})
	Describe("Unary", func() {
		Describe("Client", func() {
			var client fnoop.UnaryClient[any, any]
			It("should return zero values on calls to Send", func() {
				Expect(client.Send(ctx, "", nil)).To(BeZero())
			})
			It("should be able to call Use", func() {
				var m freighter.Middleware
				client.Use(m)
			})
		})
		Describe("Server", func() {
			var server fnoop.UnaryServer[any, any]
			It("should be able to call BindHandler", func() {
				server.BindHandler(func(context.Context, any) (any, error) {
					return nil, nil
				})
			})
			It("should be able to call Use", func() {
				var m freighter.Middleware
				server.Use(m)
			})
		})
	})
	Describe("Stream", func() {
		Describe("Client", func() {
			var client fnoop.StreamClient[any, any]
			It("should be able to call Use", func() {
				var m freighter.Middleware
				client.Use(m)
			})
			It("should return nil on calls to Stream", func() {
				Expect(client.Stream(ctx, "")).To(BeNil())
			})
		})
		Describe("Server", func() {
			var server fnoop.StreamServer[any, any]
			It("should be able to call BindHandler", func() {
				server.BindHandler(
					func(context.Context, freighter.ServerStream[any, any]) error {
						return nil
					},
				)
			})
			It("should be able to call Use", func() {
				var m freighter.Middleware
				server.Use(m)
			})
		})
	})
})
