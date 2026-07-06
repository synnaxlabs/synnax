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
	"io"
	"time"

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

func openStreamer(ctx context.Context, name string) (
	confluence.Inlet[framer.StreamerRequest],
	confluence.Outlet[framer.StreamerResponse], io.Closer,
) {
	var sigCh channel.Channel
	Expect(channelSvc.NewRetrieve().
		Where(channel.MatchNames(name)).
		Entry(&sigCh).
		Exec(ctx, nil),
	).To(Succeed())
	streamer := MustSucceed(framerSvc.NewStreamer(ctx, framer.StreamerConfig{
		Keys: channel.Keys{sigCh.Key()},
	}))
	requests, responses := confluence.Attach(streamer, 2)
	sCtx, cancel := signal.Isolated()
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	// A slight delay guarantees the streamer has started up and is subscribed before
	// the change that should propagate to it is made.
	time.Sleep(5 * time.Millisecond)
	return requests, responses, signal.NewHardShutdown(sCtx, cancel)
}

var _ = Describe("Signals", func() {
	It("Should propagate a channel creation to the set channel", func(ctx SpecContext) {
		requests, responses, closeStreamer := openStreamer(ctx, "sy_channel_set")
		ch := channel.Channel{
			Name: RandomName(), DataType: telem.TimeStampT, IsIndex: true,
		}
		Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
		var res framer.StreamerResponse
		Eventually(responses.Outlet()).Should(Receive(&res))
		payloads := MustSucceed(telem.UnmarshalJSONSeries[channelPayload](
			res.Frame.SeriesAt(0),
		))
		names := make([]string, len(payloads))
		for i, p := range payloads {
			names[i] = p.Name
		}
		Expect(names).To(ContainElement(ch.Name))
		requests.Close()
		Eventually(responses.Outlet()).Should(BeClosed())
		Expect(closeStreamer.Close()).To(Succeed())
	})

	It("Should propagate a channel deletion to the delete channel", func(ctx SpecContext) {
		ch := channel.Channel{
			Name: RandomName(), DataType: telem.TimeStampT, IsIndex: true,
		}
		Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
		requests, responses, closeStreamer := openStreamer(ctx, "sy_channel_delete")
		Expect(channelSvc.NewWriter(nil).Delete(ctx, ch.Key(), false)).To(Succeed())
		var res framer.StreamerResponse
		Eventually(responses.Outlet()).Should(Receive(&res))
		keys := telem.UnmarshalSeries[uint32](res.Frame.SeriesAt(0))
		Expect(keys).To(ContainElement(uint32(ch.Key())))
		requests.Close()
		Eventually(responses.Outlet()).Should(BeClosed())
		Expect(closeStreamer.Close()).To(Succeed())
	})
})
