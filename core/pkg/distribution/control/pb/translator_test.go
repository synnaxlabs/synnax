// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/control"
	"github.com/synnaxlabs/synnax/pkg/distribution/control/pb"
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

var _ = Describe("Translator", func() {
	Describe("RetrieveRequest", func() {
		t := pb.RetrieveRequestTranslator
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
		t := pb.RetrieveResponseTranslator
		It("Should round-trip the states", func(ctx SpecContext) {
			msg := control.RetrieveResponse{States: states}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.States).To(HaveLen(2))
			Expect(fwd.States[0].Subject.Key).To(Equal("writer-1"))
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
		It("Should reverse a nil state into its zero value", func(ctx SpecContext) {
			res := MustSucceed(t.Backward(ctx, &pb.RetrieveResponse{
				States: []*pb.State{nil},
			}))
			Expect(res.States).To(Equal([]control.State{{}}))
		})
	})

	Describe("SubscribeRequest", func() {
		t := pb.SubscribeRequestTranslator
		It("Should round-trip the parameterless request", func(ctx SpecContext) {
			msg := control.SubscribeRequest{}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
	})

	Describe("SubscribeResponse", func() {
		t := pb.SubscribeResponseTranslator
		It("Should round-trip the states", func(ctx SpecContext) {
			msg := control.SubscribeResponse{States: states}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.States).To(HaveLen(2))
			Expect(fwd.States[1].Subject.Name).To(Equal("Writer 2"))
			Expect(fwd.States[1].Resource).To(Equal(uint32(channel.NewKey(3, 4))))
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
		It("Should round-trip an empty state list", func(ctx SpecContext) {
			msg := control.SubscribeResponse{States: []control.State{}}
			Expect(
				MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg)))),
			).To(Equal(msg))
		})
	})
})
