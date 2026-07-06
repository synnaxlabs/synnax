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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

func createChannel(ctx SpecContext, dt telem.DataType) channel.Channel {
	GinkgoHelper()
	ch := channel.Channel{
		Name:     UniqueChannelName(),
		DataType: dt,
		Virtual:  true,
	}
	Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
	return ch
}

var _ = Describe("API Channel Resolver", func() {
	Describe("RetrieveDataTypes", func() {
		It("Should resolve the data types of the channels in key order", func(ctx SpecContext) {
			ch1 := createChannel(ctx, telem.Float32T)
			ch2 := createChannel(ctx, telem.Int64T)
			ch3 := createChannel(ctx, telem.Uint8T)
			Expect(apiSvc.RetrieveDataTypes(
				ctx, channel.Keys{ch1.Key(), ch2.Key(), ch3.Key()},
			)).To(Equal([]telem.DataType{telem.Float32T, telem.Int64T, telem.Uint8T}))
		})

		It("Should resolve the data type of a single channel", func(ctx SpecContext) {
			ch := createChannel(ctx, telem.Float64T)
			Expect(apiSvc.RetrieveDataTypes(ctx, channel.Keys{ch.Key()})).
				To(Equal([]telem.DataType{telem.Float64T}))
		})

		It("Should return an empty slice when no keys are provided", func(ctx SpecContext) {
			Expect(apiSvc.RetrieveDataTypes(ctx, channel.Keys{})).To(BeEmpty())
		})

		It("Should return a not found error when a key does not resolve to a channel", func(ctx SpecContext) {
			ch := createChannel(ctx, telem.Int32T)
			Expect(apiSvc.RetrieveDataTypes(
				ctx, channel.Keys{ch.Key(), channel.Key(999999)},
			)).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("RetrieveName", func() {
		It("Should resolve the name of an existing channel", func(ctx SpecContext) {
			ch := createChannel(ctx, telem.Float32T)
			Expect(apiSvc.RetrieveName(ctx, ch.Key())).To(Equal(ch.Name))
		})

		It("Should return an empty string when no channel has the key", func(ctx SpecContext) {
			Expect(apiSvc.RetrieveName(ctx, channel.Key(999999))).To(BeEmpty())
		})
	})
})
