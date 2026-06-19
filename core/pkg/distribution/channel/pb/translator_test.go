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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel/pb"
	"github.com/synnaxlabs/x/control"
	controlpb "github.com/synnaxlabs/x/control/pb"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Translator", func() {
	Describe("CreateMessage", func() {
		t := pb.CreateMessageTranslator
		It("Should round-trip the minimal storage and routing fields", func(ctx SpecContext) {
			msg := channel.CreateMessage{Channels: []channel.Channel{
				{
					Name:        "sensor",
					Leaseholder: 2,
					DataType:    telem.Float32T,
					IsIndex:     false,
					LocalKey:    5,
					LocalIndex:  3,
					Virtual:     false,
					Concurrency: control.ConcurrencyExclusive,
				},
				{
					Name:        "stream",
					Leaseholder: 1,
					DataType:    telem.Float64T,
					Virtual:     true,
					Concurrency: control.ConcurrencyShared,
				},
			}}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.Channels).To(HaveLen(2))
			Expect(fwd.Channels[0].Name).To(Equal("sensor"))
			Expect(fwd.Channels[0].Leaseholder).To(Equal(uint32(2)))
			Expect(fwd.Channels[0].DataType).To(Equal(string(telem.Float32T)))
			Expect(fwd.Channels[0].LocalKey).To(Equal(uint32(5)))
			Expect(fwd.Channels[0].LocalIndex).To(Equal(uint32(3)))
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
		It("Should round-trip an empty channel list", func(ctx SpecContext) {
			msg := channel.CreateMessage{Channels: []channel.Channel{}}
			Expect(MustSucceed(t.Backward(ctx, MustSucceed(t.Forward(ctx, msg))))).To(Equal(msg))
		})
		It("Should return an error when forwarding an unrecognized concurrency value", func(ctx SpecContext) {
			msg := channel.CreateMessage{Channels: []channel.Channel{
				{Name: "bad", Concurrency: control.Concurrency(99)},
			}}
			Expect(t.Forward(ctx, msg)).Error().To(
				MatchError(ContainSubstring("unrecognized control.Concurrency value")),
			)
		})
		It("Should return an error when reversing an unrecognized concurrency value", func(ctx SpecContext) {
			msg := &pb.CreateMessage{Channels: []*pb.Channel{
				{Name: "bad", Concurrency: controlpb.Concurrency(99)},
			}}
			Expect(t.Backward(ctx, msg)).Error().To(
				MatchError(ContainSubstring("unrecognized Concurrency value")),
			)
		})
	})
	Describe("DeleteRequest", func() {
		t := pb.DeleteRequestTranslator
		It("Should round-trip the channel keys", func(ctx SpecContext) {
			msg := channel.DeleteRequest{Keys: channel.Keys{
				channel.NewKey(1, 2),
				channel.NewKey(3, 4),
			}}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.Keys).To(Equal(msg.Keys.Uint32()))
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
	})
	Describe("RenameRequest", func() {
		t := pb.RenameMessageTranslator
		It("Should round-trip keys and names", func(ctx SpecContext) {
			msg := channel.RenameRequest{
				Keys:  channel.Keys{channel.NewKey(1, 2), channel.NewKey(1, 3)},
				Names: []string{"first", "second"},
			}
			fwd := MustSucceed(t.Forward(ctx, msg))
			Expect(fwd.Keys).To(Equal(msg.Keys.Uint32()))
			Expect(fwd.Names).To(Equal(msg.Names))
			Expect(MustSucceed(t.Backward(ctx, fwd))).To(Equal(msg))
		})
	})
})
