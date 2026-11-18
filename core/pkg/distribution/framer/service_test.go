// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	Describe("Run", func() {
		var svc *framer.Service
		BeforeEach(func() {
			store := DeferClose(storagetestutil.OpenLayer(GinkgoT().Context()))
			svc = MustSucceed(framer.OpenService(framer.ServiceConfig{TS: store.TS}))
		})
		stream := func(
			ctx SpecContext,
		) (signal.Context, confluence.Outlet[framer.StreamerResponse]) {
			GinkgoHelper()
			s := MustSucceed(svc.NewStreamer(framer.StreamerConfig{
				Keys:        channel.Keys{channel.NewKey(node.KeyFree, 1)},
				SendOpenAck: new(true),
			}))
			sCtx, cancel := signal.WithCancel(ctx)
			req, res := confluence.Attach(s, 1)
			s.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			DeferCleanup(func() {
				req.Close()
				cancel()
				confluence.Drain(res)
			})
			return sCtx, res
		}

		It("Should serve streamers until its context ends", func(ctx SpecContext) {
			stop := StartRun(svc.Run)
			_, res := stream(ctx)
			Eventually(res.Outlet()).Should(Receive())
			stop()
		})

		It("Should fail streamers that start after it returns", func(ctx SpecContext) {
			StartRun(svc.Run)()
			sCtx, _ := stream(ctx)
			Expect(sCtx.Wait()).To(MatchError(ContainSubstring("relay stopped")))
		})

		It("Should fail a second Run", func(ctx SpecContext) {
			StartRun(svc.Run)
			Expect(svc.Run(ctx)).To(MatchError(ContainSubstring("run called twice")))
		})
	})
})
