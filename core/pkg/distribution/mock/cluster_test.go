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
)

var _ = Describe("Cluster", func() {
	Describe("Name", func() {
		It("Should open a three node memory backed distribution layer", func(ctx SpecContext) {
			cluster := mock.MustOpenCluster(ctx, 3)
			coreOne := cluster.Nodes[node.Key(1)]
			coreTwo := cluster.Nodes[node.Key(2)]
			coreThree := cluster.Nodes[node.Key(3)]

			Expect(coreOne.Cluster.HostKey()).To(Equal(node.Key(1)))
			Expect(coreTwo.Cluster.HostKey()).To(Equal(node.Key(2)))
			Expect(coreThree.Cluster.HostKey()).To(Equal(node.Key(3)))

			ch := channel.Channel{
				Name:        "SG_01",
				DataType:    telem.Float64T,
				Virtual:     true,
				Leaseholder: 1,
			}

			Expect(coreOne.Channel.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			Expect(ch.Key().Leaseholder()).To(Equal(node.Key(1)))

			Eventually(func(g Gomega) {
				var resCh channel.Channel
				g.Expect(coreThree.Channel.NewRetrieve().
					Where(channel.MatchKeys(ch.Key())).
					Entry(&resCh).
					Exec(ctx, nil)).To(Succeed())

				g.Expect(resCh.Key()).To(Equal(ch.Key()))
			}, time.Millisecond*200).Should(Succeed())
		})
	})

})
