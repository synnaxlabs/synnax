// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Relay stop check", func() {
	It("Should not block the writes of a stored channel once the relay stops", func(
		ctx SpecContext,
	) {
		store := storagetestutil.OpenLayer(GinkgoT().Context())
		DeferCleanup(func() { Expect(store.Close()).To(Succeed()) })
		run, stopRun := context.WithCancel(GinkgoT().Context())
		host := testutil.OpenNode(run, distribution.LayerConfig{Storage: store.Layer})
		DeferCleanup(func() {
			Expect(host.Close()).To(Or(Succeed(), MatchError(context.Canceled)))
		})
		idx := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
			Name:     "relay_stop_time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		}}))[0]
		wCtx, cancel := context.WithCancel(ctx)
		DeferCleanup(cancel)
		w := MustSucceed(host.Framer.OpenWriter(wCtx, writer.Config{
			Keys:  channel.Keys{idx.Key()},
			Start: telem.SecondTS,
		}))
		stopRun()
		// More frames than the relay's write buffer holds.
		const frames = 20000
		written := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(written)
			for i := range telem.TimeStamp(frames) {
				fr := frame.NewUnary(idx.Key(), telem.NewSeriesSecondsTSV(i+1))
				Expect(w.Write(fr)).To(BeTrue())
			}
			Expect(w.Close()).To(Succeed())
		}()
		Eventually(written).WithTimeout(time.Minute).Should(BeClosed())
	})
})
