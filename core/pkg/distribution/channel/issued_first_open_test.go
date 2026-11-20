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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("IssuedAtFirstOpen", func() {
	It("Should keep the answer of the first open on a later open", func(
		ctx SpecContext,
	) {
		store := DeferClose(storagetestutil.OpenLayer(ctx))
		host := node.Key(1)
		open := func() *channel.Service {
			GinkgoHelper()
			return MustSucceed(channel.NewService(ctx, channel.ServiceConfig{
				HostKey: host,
				KV:      store.KV,
				TS:      store.TS,
			}))
		}
		out := MustSucceed(open().Allocate(ctx, []channel.Channel{
			{
				Name:        "pressure",
				DataType:    telem.Float32T,
				Leaseholder: node.KeyFree,
				Virtual:     true,
			},
			{
				Name:        "time",
				DataType:    telem.TimestampT,
				IsIndex:     true,
				Leaseholder: host,
			},
		}))
		reopened := open()
		for _, ch := range out {
			Expect(reopened.Issued(ch.Key())).To(BeTrue())
			Expect(reopened.IssuedAtFirstOpen(ch.Key())).To(BeFalse())
		}
	})
})
