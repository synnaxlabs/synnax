// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package clustermock_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/clustermock"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/node"
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("Cluster Mock", func() {
	Describe("Builder", func() {
		It("Should provision a set of cluster ClusterAPIs correctly", func() {
			cfg := cluster.Config{Gossip: gossip.Config{Interval: 50 * time.Millisecond}}
			builder := clustermock.NewBuilder(cfg)
			ctx, cancel := signal.Isolated()
			defer cancel()
			c1, err := builder.New(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(c1.HostKey()).To(Equal(node.Key(1)))
			c2, err := builder.New(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(c2.HostKey()).To(Equal(node.Key(2)))
			Expect(c2.Nodes()).To(HaveLen(2))
		})
	})
})
