// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Cluster", func() {
	ShouldNotLeakGoroutinesBeforeEach()
	Describe("Name", func() {
		It("Should open a three node memory backed distribution layer", func() {
			mockCluster := mock.NewCluster()
			coreOne := mockCluster.Provision(ctx)
			coreTwo := mockCluster.Provision(ctx)
			coreThree := mockCluster.Provision(ctx)

			Expect(coreOne.Cluster.HostKey()).To(Equal(cluster.NodeKey(1)))
			Expect(coreTwo.Cluster.HostKey()).To(Equal(cluster.NodeKey(2)))
			Expect(coreThree.Cluster.HostKey()).To(Equal(cluster.NodeKey(3)))

			ch := channel.Channel{
				Name:        "SG_01",
				DataType:    telem.Float64T,
				Virtual:     true,
				Leaseholder: 1,
			}

			Expect(coreOne.Channel.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			Expect(ch.Key().Leaseholder()).To(Equal(cluster.NodeKey(1)))

			Eventually(func(g Gomega) {
				var resCh channel.Channel
				g.Expect(coreThree.Channel.NewRetrieve().
					WhereKeys(ch.Key()).
					Entry(&resCh).
					Exec(ctx, nil)).To(Succeed())

				g.Expect(resCh.Key()).To(Equal(ch.Key()))
			}, "200ms").Should(Succeed())

			Expect(mockCluster.Close()).To(Succeed())
		})
	})

})
