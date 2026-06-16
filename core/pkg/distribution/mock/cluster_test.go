// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Cluster", func() {
	ShouldNotLeakGoroutinesPerSpec()
	Describe("Name", func() {
		It("Should open a three node memory backed distribution layer", func(ctx SpecContext) {
			mockCluster := mock.NewCluster()
			coreOne := mockCluster.Provision(ctx)
			coreTwo := mockCluster.Provision(ctx)
			coreThree := mockCluster.Provision(ctx)

			Expect(coreOne.Cluster.HostKey()).To(Equal(node.Key(1)))
			Expect(coreTwo.Cluster.HostKey()).To(Equal(node.Key(2)))
			Expect(coreThree.Cluster.HostKey()).To(Equal(node.Key(3)))

			ch := channel.Channel{
				Name:        "SG_01",
				DataType:    telem.Float64T,
				Virtual:     true,
				Leaseholder: 1,
			}

			Expect(coreOne.CreateChannel(ctx, &ch)).To(Succeed())
			Expect(ch.Key().Leaseholder()).To(Equal(node.Key(1)))

			Eventually(func(g Gomega) {
				var resChs []channel.Channel
				g.Expect(coreThree.RetrieveChannelsInto(ctx, &resChs, ch.Key())).To(Succeed())
				g.Expect(resChs).To(HaveLen(1))
				g.Expect(resChs[0].Key()).To(Equal(ch.Key()))
			}, time.Millisecond*200).Should(Succeed())

			Expect(mockCluster.Close()).To(Succeed())
		})
	})

})
