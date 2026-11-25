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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Description breaker", func() {
	It("Should send the description of a created channel on the set channel", func(
		ctx SpecContext,
	) {
		responses := openStreamer(ctx, "sy_channel_set")
		ch := channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.Float32T,
			Virtual:     true,
			Description: "Pressure at the inlet",
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		var payloads []map[string]any
		receiveSeries(responses, func(g Gomega, s telem.Series) {
			g.Expect(s.DecodeJSON[map[string]any]()).To(ContainElement(
				HaveKeyWithValue("name", ch.Name),
				&payloads,
			))
		})
		Expect(payloads).To(ConsistOf(
			HaveKeyWithValue("description", "Pressure at the inlet"),
		))
	})
})
