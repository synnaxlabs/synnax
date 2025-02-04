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
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Builder", func() {
	Describe("Name", func() {
		It("Should open a three node memory backed distribution layer", func() {

			builder := mock.NewBuilder()

			coreOne := builder.New(ctx)
			coreTwo := builder.New(ctx)
			coreThree := builder.New(ctx)

			Expect(coreOne.Cluster.HostKey()).To(Equal(core.NodeKey(1)))
			Expect(coreTwo.Cluster.HostKey()).To(Equal(core.NodeKey(2)))
			Expect(coreThree.Cluster.HostKey()).To(Equal(core.NodeKey(3)))

			ch := channel.Channel{
				Name:        "SG_01",
				DataType:    telem.Float64T,
				Rate:        25 * telem.Hz,
				Leaseholder: 1,
			}

			Expect(coreOne.Channel.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
			Expect(ch.Key().Leaseholder()).To(Equal(distribution.NodeKey(1)))

			Eventually(func(g Gomega) {
				var resCH channel.Channel
				g.Expect(coreThree.Channel.NewRetrieve().
					WhereKeys(ch.Key()).
					Entry(&resCH).
					Exec(ctx, nil)).To(Succeed())

				g.Expect(resCH.Key()).To(Equal(ch.Key()))
			}, "200ms").Should(Succeed())

			Expect(builder.Close()).To(Succeed())
			Expect(builder.Cleanup()).To(Succeed())
		})
	})

})
