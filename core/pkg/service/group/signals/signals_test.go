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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	groupcdc "github.com/synnaxlabs/synnax/pkg/service/group/signals"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Signals", Ordered, func() {
	var (
		builder   *mock.Cluster
		dist      mock.Node
		cdcCloser io.Closer
	)
	BeforeAll(func(ctx SpecContext) {
		builder = mock.NewCluster()
		dist = builder.Provision(ctx)
		sigs := MustSucceed(signals.New(signals.Config{
			Channel: dist.Channel,
			Framer:  dist.Framer,
		}))
		cdcCloser = MustSucceed(groupcdc.Publish(ctx, sigs, dist.Group.Observe()))
	})
	AfterAll(func() {
		Expect(cdcCloser.Close()).To(Succeed())
		Expect(builder.Close()).To(Succeed())
	})

	openStreamer := func(
		ctx context.Context,
		name string,
	) (confluence.Inlet[framer.StreamerRequest], confluence.Outlet[framer.StreamerResponse], io.Closer) {
		var sigCh channel.Channel
		Expect(dist.Channel.NewRetrieve().
			Where(channel.MatchNames(name)).
			Entry(&sigCh).
			Exec(ctx, nil),
		).To(Succeed())
		streamer := MustSucceed(dist.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys: channel.Keys{sigCh.Key()},
		}))
		requests, responses := confluence.Attach(streamer, 2)
		sCtx, cancel := signal.Isolated()
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		// A slight delay guarantees the streamer has started up and is subscribed
		// before the change that should propagate to it is made.
		time.Sleep(5 * time.Millisecond)
		return requests, responses, signal.NewHardShutdown(sCtx, cancel)
	}

	It("Should propagate a group creation to the set channel", func(ctx SpecContext) {
		requests, responses, closeStreamer := openStreamer(ctx, "sy_group_set")
		g := MustSucceed(dist.Group.NewWriter(nil).Create(ctx, "set-"+uuid.NewString(), ontology.RootID))
		var res framer.StreamerResponse
		Eventually(responses.Outlet()).Should(Receive(&res))
		groups := MustSucceed(telem.UnmarshalJSONSeries[group.Group](res.Frame.SeriesAt(0)))
		keys := make([]uuid.UUID, len(groups))
		for i, gg := range groups {
			keys[i] = gg.Key
		}
		Expect(keys).To(ContainElement(g.Key))
		requests.Close()
		Eventually(responses.Outlet()).Should(BeClosed())
		Expect(closeStreamer.Close()).To(Succeed())
	})

	It("Should propagate a group deletion to the delete channel", func(ctx SpecContext) {
		g := MustSucceed(dist.Group.NewWriter(nil).Create(ctx, "delete-"+uuid.NewString(), ontology.RootID))
		requests, responses, closeStreamer := openStreamer(ctx, "sy_group_delete")
		Expect(dist.Group.NewWriter(nil).Delete(ctx, g.Key)).To(Succeed())
		var res framer.StreamerResponse
		Eventually(responses.Outlet()).Should(Receive(&res))
		keys := telem.UnmarshalSeries[uuid.UUID](res.Frame.SeriesAt(0))
		Expect(keys).To(ContainElement(g.Key))
		requests.Close()
		Eventually(responses.Outlet()).Should(BeClosed())
		Expect(closeStreamer.Close()).To(Succeed())
	})
})
