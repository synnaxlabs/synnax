// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package clustermock_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/clustermock"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/x/signal"
	"time"
)

var _ = Describe("Clustermock", func() {
	Describe("Builder", func() {
		It("Should provision a set of cluster ClusterAPIs correctly", func() {
			cfg := cluster.Config{Gossip: gossip.Config{Interval: 50 * time.Millisecond}}
			builder := clustermock.NewBuilder(cfg)
			ctx, cancel := signal.TODO()
			defer cancel()
			c1, err := builder.New(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(c1.HostID()).To(Equal(node.ID(1)))
			c2, err := builder.New(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(c2.HostID()).To(Equal(node.ID(2)))
			Expect(c2.Nodes()).To(HaveLen(2))
		})
	})
})
