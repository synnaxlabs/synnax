// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package distribution_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("LayerConfig", func() {
	It("Should reject a storage layer with no Gorp DB", func() {
		cfg := distribution.LayerConfig{
			Storage:          &storage.Layer{},
			AdvertiseAddress: "localhost:0",
		}
		Expect(cfg.Validate()).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("gorp")),
		))
	})
})

var _ = Describe("Layer", func() {
	var layer *distribution.Layer
	BeforeEach(func(ctx SpecContext) {
		store := DeferClose(storagetestutil.OpenLayer(GinkgoT().Context()))
		layer = DeferClose(MustSucceed(distribution.OpenLayer(
			ctx,
			distribution.LayerConfig{
				Storage:          store.Layer,
				AdvertiseAddress: "localhost:0",
			},
		)))
	})

	Describe("Run", func() {
		It("Should return nil when its context ends", func() {
			StartRun(layer.Run)()
		})

		It("Should fail while another Run runs", func(ctx SpecContext) {
			StartRun(layer.Run)
			Expect(layer.Run(ctx)).To(MatchError(ContainSubstring("run called twice")))
		})

		It("Should fail after an earlier Run returns", func(ctx SpecContext) {
			StartRun(layer.Run)()
			Expect(layer.Run(ctx)).To(MatchError(ContainSubstring("run called twice")))
		})

		It("Should fail after Close", func(ctx SpecContext) {
			Expect(layer.Close()).To(Succeed())
			Expect(layer.Run(ctx)).
				To(MatchError(ContainSubstring("run called after close")))
		})
	})

	Describe("Close", func() {
		It("Should fail while Run runs", func() {
			StartRun(layer.Run)
			Expect(layer.Close()).
				To(MatchError(ContainSubstring("close called while Run runs")))
		})

		It("Should fail streamers that wait for a Run", func(ctx SpecContext) {
			s := MustSucceed(layer.Framer.NewStreamer(framer.StreamerConfig{
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
			Expect(layer.Close()).To(Succeed())
			Expect(sCtx.Wait()).To(MatchError(ContainSubstring("relay stopped")))
		})
	})
})
