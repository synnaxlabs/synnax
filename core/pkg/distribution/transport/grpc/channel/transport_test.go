// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"context"
	"go/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Transport", func() {
	Describe("Create", func() {
		It("Should round-trip a create request over the wire", func(ctx SpecContext) {
			transport.CreateServer().BindHandler(
				func(_ context.Context, req distchannel.CreateMessage) (distchannel.CreateMessage, error) {
					return req, nil
				},
			)
			res := MustSucceed(transport.CreateClient().Send(
				ctx,
				addr,
				distchannel.CreateMessage{Channels: []distchannel.Channel{{Name: "alpha"}}},
			))
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.Channels[0].Name).To(Equal("alpha"))
		})
	})

	Describe("Delete", func() {
		It("Should round-trip a delete request over the wire", func(ctx SpecContext) {
			var received distchannel.DeleteRequest
			transport.DeleteServer().BindHandler(
				func(_ context.Context, req distchannel.DeleteRequest) (types.Nil, error) {
					received = req
					return types.Nil{}, nil
				},
			)
			MustSucceed(transport.DeleteClient().Send(
				ctx,
				addr,
				distchannel.DeleteRequest{Keys: distchannel.Keys{1, 2, 3}},
			))
			Expect(received.Keys).To(Equal(distchannel.Keys{1, 2, 3}))
		})
	})

	Describe("Rename", func() {
		It("Should round-trip a rename request over the wire", func(ctx SpecContext) {
			var received distchannel.RenameRequest
			transport.RenameServer().BindHandler(
				func(_ context.Context, req distchannel.RenameRequest) (types.Nil, error) {
					received = req
					return types.Nil{}, nil
				},
			)
			MustSucceed(transport.RenameClient().Send(
				ctx,
				addr,
				distchannel.RenameRequest{
					Keys:  distchannel.Keys{1, 2},
					Names: []string{"beta", "gamma"},
				},
			))
			Expect(received.Keys).To(Equal(distchannel.Keys{1, 2}))
			Expect(received.Names).To(Equal([]string{"beta", "gamma"}))
		})
	})
})
