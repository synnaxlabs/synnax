// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/control"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	grpccontrol "github.com/synnaxlabs/synnax/pkg/transport/grpc/control"
	xcontrol "github.com/synnaxlabs/x/control"
	. "github.com/synnaxlabs/x/testutil"
)

type retrieveServer = grpc.UnaryServer[
	control.RetrieveRequest,
	*grpccontrol.RetrieveRequest,
	control.RetrieveResponse,
	*grpccontrol.RetrieveResponse,
]

var states = []control.State{
	{
		Subject:   xcontrol.Subject{Key: "writer-1", Name: "Writer 1"},
		Resource:  distchannel.NewKey(1, 2),
		Authority: xcontrol.AuthorityAbsolute,
	},
	{
		Subject:   xcontrol.Subject{Key: "writer-2", Name: "Writer 2"},
		Resource:  distchannel.NewKey(3, 4),
		Authority: xcontrol.AuthorityAbsolute - 5,
	},
}

var _ = Describe("gRPC Control Translators", func() {
	var (
		transport api.Transport
		bindable  = grpccontrol.New(&transport)
		server    = transport.ControlRetrieve.(*retrieveServer)
	)

	Describe("RetrieveRequest", func() {
		t := server.RequestTranslator
		It("Should translate the channel keys", func(ctx SpecContext) {
			keys := channel.Keys{distchannel.NewKey(1, 2), distchannel.NewKey(3, 4)}
			pb := &grpccontrol.RetrieveRequest{Keys: keys.Uint32()}
			Expect(
				MustSucceed(t.Backward(ctx, pb)),
			).To(Equal(control.RetrieveRequest{Keys: keys}))
		})
		It("Should translate the empty key set that requests every channel", func(
			ctx SpecContext,
		) {
			Expect(
				MustSucceed(t.Backward(ctx, &grpccontrol.RetrieveRequest{})).Keys,
			).To(BeEmpty())
		})
	})

	Describe("RetrieveResponse", func() {
		t := server.ResponseTranslator
		It("Should translate the states", func(ctx SpecContext) {
			pb := MustSucceed(t.Forward(ctx, control.RetrieveResponse{States: states}))
			Expect(pb.States).To(HaveLen(2))
			Expect(pb.States[0].Subject.Key).To(Equal("writer-1"))
			Expect(pb.States[0].Subject.Name).To(Equal("Writer 1"))
			Expect(pb.States[0].Resource).To(Equal(uint32(distchannel.NewKey(1, 2))))
			Expect(
				pb.States[0].Authority,
			).To(Equal(uint32(xcontrol.AuthorityAbsolute)))
			Expect(pb.States[1].Subject.Key).To(Equal("writer-2"))
			Expect(pb.States[1].Resource).To(Equal(uint32(distchannel.NewKey(3, 4))))
			Expect(
				pb.States[1].Authority,
			).To(Equal(uint32(xcontrol.AuthorityAbsolute - 5)))
		})
		It("Should translate an empty state list", func(ctx SpecContext) {
			pb := MustSucceed(
				t.Forward(ctx, control.RetrieveResponse{States: []control.State{}}),
			)
			Expect(pb.States).To(BeEmpty())
		})
	})

	Describe("New", func() {
		It("Should bind the retrieve server onto the API transport", func() {
			Expect(bindable).To(BeIdenticalTo(server))
		})
	})
})
