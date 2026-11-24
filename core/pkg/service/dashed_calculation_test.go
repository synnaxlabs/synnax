// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/telem/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Dashed calculation upgrade", func() {
	It("Should compute a v0.62 calculation that reads a dashed channel", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		stored := storeV062(ctx, n, []string{"pt-01", "doubled"}, func(
			cs []channelv1.Channel,
		) {
			cs[1].Expression = "return pt-01 * 2"
		})
		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		streamer := MustSucceed(l.Framer.NewStreamer(ctx, framer.StreamerConfig{
			Keys:        channel.Keys{stored[1].Key()},
			SendOpenAck: true,
		}))
		requests, responses := confluence.Attach(streamer, 2)
		sCtx, cancel := signal.Isolated()
		shutdown := signal.NewHardShutdown(sCtx, cancel)
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
		DeferCleanup(func() {
			requests.Close()
			confluence.Drain(responses)
			Expect(shutdown.Close()).To(Succeed())
		})
		Eventually(responses.Outlet(), 5*time.Second).Should(Receive())
		in := channel.Keys{stored[0].Key()}
		w := MustSucceed(l.Framer.OpenWriter(ctx, framer.WriterConfig{
			Start: telem.Now(),
			Keys:  in,
		}))
		Expect(w.Write(frame.NewMulti(in, []telem.Series{
			telem.NewSeriesV[float32](3),
		}))).To(BeTrue())
		Expect(w.Close()).To(Succeed())
		var res framer.StreamerResponse
		Eventually(responses.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.SeriesAt(0)).To(MatchSeriesDataV[float32](6))
	})
})
