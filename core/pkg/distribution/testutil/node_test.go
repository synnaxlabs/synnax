// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Node", func() {
	It("Should open a node with host key 1", func(ctx SpecContext) {
		n := DeferClose(testutil.OpenNode(ctx))
		Expect(n.Identity.HostKey).To(Equal(node.KeyBootstrapper))
		Expect(n.Host.HostKey()).To(Equal(node.KeyBootstrapper))
	})

	It("Should create channels in the storage layer", func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		ch := MustSucceed(n.Channel.Create(ctx, []channel.Channel{{
			Name:        "SG_01",
			DataType:    telem.Float64T,
			Virtual:     true,
			Leaseholder: 1,
		}}))[0]
		Expect(ch.Key().Lease()).To(Equal(node.Key(1)))
		chs := MustSucceed(n.Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey()))
		Expect(chs).To(HaveLen(1))
	})

	It(
		"Should create channels while a transaction holds the write lock",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			tx := DeferClose(n.DB.OpenTx())
			Expect(n.Channel.Create(ctx, []channel.Channel{{
				Name:        "SG_02",
				DataType:    telem.Float64T,
				Virtual:     true,
				Leaseholder: 1,
			}})).To(HaveLen(1))
			Expect(tx.Commit(ctx)).To(Succeed())
		},
		NodeTimeout(10*time.Second),
	)

	It("Should close the distribution and storage layers", func(ctx SpecContext) {
		Expect(testutil.OpenNode(ctx).Close()).To(Succeed())
	})
})
