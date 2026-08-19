// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/control"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	xcontrol "github.com/synnaxlabs/x/control"
	. "github.com/synnaxlabs/x/testutil"
)

var states = []control.State{
	{
		Subject:   xcontrol.Subject{Key: "writer-1", Name: "Writer 1"},
		Resource:  channel.NewKey(1, 2),
		Authority: xcontrol.AuthorityAbsolute,
	},
	{
		Subject:   xcontrol.Subject{Key: "writer-2", Name: "Writer 2"},
		Resource:  channel.NewKey(3, 4),
		Authority: xcontrol.AuthorityAbsolute - 5,
	},
}

var _ = Describe("gRPC Control Translators", func() {
	Describe("RetrieveRequest", func() {
		t := retrieveRequestTranslator{}
		It("Should round-trip the channel keys", func(ctx SpecContext) {
			msg := control.RetrieveRequest{Keys: channel.Keys{
				channel.NewKey(1, 2),
				channel.NewKey(3, 4),
			}}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.Keys).To(Equal(msg.Keys.Uint32()))
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
		It("Should round-trip the empty key set that requests every channel", func(
			ctx SpecContext,
		) {
			fwd := MustSucceed(t.Forward(ctx, control.RetrieveRequest{}))
			Expect(fwd.Keys).To(BeEmpty())
			Expect(MustSucceed(t.Backward(ctx, fwd)).Keys).To(BeEmpty())
		})
	})

	Describe("RetrieveResponse", func() {
		t := retrieveResponseTranslator{}
		It("Should round-trip the states", func(ctx SpecContext) {
			msg := control.RetrieveResponse{States: states}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.States).To(HaveLen(2))
			Expect(fwd.States[0].Subject.Name).To(Equal("Writer 1"))
			Expect(fwd.States[0].Resource).To(Equal(uint32(channel.NewKey(1, 2))))
			Expect(
				fwd.States[0].Authority,
			).To(Equal(uint32(xcontrol.AuthorityAbsolute)))
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
		It("Should round-trip an empty state list", func(ctx SpecContext) {
			msg := control.RetrieveResponse{States: []control.State{}}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
	})

	Describe("New", func() {
		It("Should bind the retrieve server onto the API transport", func() {
			var transport api.Transport
			bindable := New(&transport)
			Expect(bindable).ToNot(BeNil())
			Expect(transport.ControlRetrieve).To(
				BeIdenticalTo(bindable.(*retrieveServer)),
			)
		})
	})
})
