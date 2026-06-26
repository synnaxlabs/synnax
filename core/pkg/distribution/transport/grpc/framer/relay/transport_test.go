// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay_test

import (
	"context"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distrelay "github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Transport", func() {
	It("Should round-trip a request over the wire", func(ctx SpecContext) {
		transport.Server().BindHandler(func(
			_ context.Context,
			srv freighter.ServerStream[distrelay.Request, distrelay.Response],
		) error {
			if _, err := srv.Receive(); err != nil {
				return err
			}
			return srv.Send(distrelay.Response{Group: 7})
		})
		stream := MustSucceed(transport.Client().Stream(ctx, addr))
		Expect(stream.Send(distrelay.Request{Keys: channel.Keys{1, 2}})).To(Succeed())
		Expect(MustSucceed(stream.Receive()).Group).To(Equal(uint32(7)))
		Expect(stream.CloseSend()).To(Succeed())
	})

	It("Should expose non-nil client and server", func() {
		Expect(transport.Client()).ToNot(BeNil())
		Expect(transport.Server()).ToNot(BeNil())
	})

	Describe("Use", func() {
		It("Should apply middleware to both the client and server endpoints", func(ctx SpecContext) {
			var clientCalls, serverCalls atomic.Int32
			transport.Use(freighter.MiddlewareFunc(func(
				mCtx freighter.Context,
				next freighter.Next,
			) (freighter.Context, error) {
				switch mCtx.Role {
				case freighter.RoleClient:
					clientCalls.Add(1)
				case freighter.RoleServer:
					serverCalls.Add(1)
				}
				return next(mCtx)
			}))
			transport.Server().BindHandler(func(
				_ context.Context,
				srv freighter.ServerStream[distrelay.Request, distrelay.Response],
			) error {
				if _, err := srv.Receive(); err != nil {
					return err
				}
				return srv.Send(distrelay.Response{})
			})
			stream := MustSucceed(transport.Client().Stream(ctx, addr))
			Expect(stream.Send(distrelay.Request{})).To(Succeed())
			MustSucceed(stream.Receive())
			Expect(stream.CloseSend()).To(Succeed())
			Expect(clientCalls.Load()).To(Equal(int32(1)))
			Expect(serverCalls.Load()).To(Equal(int32(1)))
		})
	})
})
