// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock_test

import (
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Builder", func() {
	Describe("New", func() {
		It("Should open a three node memory backed distribution layer", func() {

			builder := mock.NewBuilder()

			coreOne := builder.New()
			coreTwo := builder.New()
			coreThree := builder.New()

			Expect(coreOne.Cluster.HostID()).To(Equal(core.NodeID(1)))
			Expect(coreTwo.Cluster.HostID()).To(Equal(core.NodeID(2)))
			Expect(coreThree.Cluster.HostID()).To(Equal(core.NodeID(3)))

			ch := channel.Channel{
				Name:     "SG_01",
				DataType: telem.Float64T,
				Rate:     25 * telem.Hz,
				NodeID:   1,
			}

			Expect(coreOne.Channel.Create(&ch)).To(Succeed())
			Expect(ch.Key().NodeID()).To(Equal(distribution.NodeID(1)))

			Eventually(func(g Gomega) {
				var resCH channel.Channel
				g.Expect(coreThree.Channel.NewRetrieve().
					WhereKeys(ch.Key()).
					Entry(&resCH).
					Exec(ctx)).To(Succeed())

				g.Expect(resCH.Key()).To(Equal(ch.Key()))
			}).Should(Succeed())

		})
	})

})
