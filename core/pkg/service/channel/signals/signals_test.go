// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type channelPayload struct {
	Name string `json:"name"`
}

// openStreamer opens a streamer on the signal channel with the given name and returns
// once the streamer is subscribed. The streamer closes when the spec ends.
func openStreamer(
	ctx context.Context,
	name string,
) confluence.Outlet[framer.StreamerResponse] {
	GinkgoHelper()
	var sigCh channel.Channel
	Expect(channelSvc.NewRetrieve().
		Where(channel.MatchNames(name)).
		Entry(&sigCh).
		Exec(ctx, nil),
	).To(Succeed())
	streamer := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
		Keys:        channel.Keys{sigCh.Key()},
		SendOpenAck: true,
	}))
	requests, responses := confluence.Attach(streamer, 2)
	sCtx, cancel := signal.Isolated()
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	DeferClose(signal.NewHardShutdown(sCtx, cancel))
	DeferCleanup(func() {
		requests.Close()
		Eventually(responses.Outlet()).Should(BeClosed())
	})
	Eventually(responses.Outlet()).Should(Receive())
	return responses
}

// receiveSeries receives frames until the first series of one passes check. Earlier
// frames can carry changes that other specs made.
func receiveSeries(
	responses confluence.Outlet[framer.StreamerResponse],
	check func(g Gomega, series telem.Series),
) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		var res framer.StreamerResponse
		g.Expect(responses.Outlet()).To(Receive(&res))
		check(g, res.Frame.SeriesAt(0))
	}).Should(Succeed())
}

var _ = Describe("Signals", func() {
	It("Should propagate a channel creation to the set channel", func(ctx SpecContext) {
		responses := openStreamer(ctx, "sy_channel_set")
		ch := channel.Channel{
			Name: UniqueChannelName(), DataType: telem.TimestampT, IsIndex: true,
		}
		Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
		receiveSeries(responses, func(g Gomega, s telem.Series) {
			g.Expect(s.DecodeJSON[channelPayload]()).
				To(ContainElement(HaveField("Name", ch.Name)))
		})
	})

	It(
		"Should not marshal zero-length operations to the set channel",
		func(ctx SpecContext) {
			responses := openStreamer(ctx, "sy_channel_set")
			ch := channel.Channel{
				Name: UniqueChannelName(), DataType: telem.TimestampT, IsIndex: true,
			}
			Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			var payloads []map[string]any
			receiveSeries(responses, func(g Gomega, s telem.Series) {
				g.Expect(s.DecodeJSON[map[string]any]()).To(ContainElement(
					HaveKeyWithValue("name", ch.Name),
					&payloads,
				))
			})
			Expect(payloads).To(ConsistOf(Not(HaveKey("operations"))))
		},
	)

	It(
		"Should propagate a channel deletion to the delete channel",
		func(ctx SpecContext) {
			ch := channel.Channel{
				Name: UniqueChannelName(), DataType: telem.TimestampT, IsIndex: true,
			}
			Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			responses := openStreamer(ctx, "sy_channel_delete")
			Expect(channelSvc.NewWriter(nil).Delete(ctx, ch.Key(), false)).To(Succeed())
			receiveSeries(responses, func(g Gomega, s telem.Series) {
				g.Expect(s.Unmarshal[uint32]()).To(ContainElement(uint32(ch.Key())))
			})
		},
	)
})
