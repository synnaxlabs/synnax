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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
)

var _ = Describe("Logging", func() {
	DescribeTable("New", func(cfg ...distribution.Config) {
		builder := mock.NewCoreBuilder(cfg...)
		coreOne := builder.New()
		coreTwo := builder.New()
		coreThree := builder.New()

		Expect(coreOne.Cluster.HostID()).To(Equal(core.NodeID(1)))
		Expect(coreTwo.Cluster.HostID()).To(Equal(core.NodeID(2)))
		Expect(coreThree.Cluster.HostID()).To(Equal(core.NodeID(3)))

		Expect(coreOne.Storage.KV.Set([]byte("foo"), []byte("bar"))).To(Succeed())

		Eventually(func(g Gomega) {
			v, err := coreOne.Storage.KV.Get([]byte("foo"))
			g.Expect(err).To(Succeed())
			g.Expect(v).To(Equal([]byte("bar")))
		}).Should(Succeed())

		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	},
		Entry("Should open a three node memory backed distribution core"),
		Entry("Should open a three node file-system backed distribution core", distribution.Config{
			Storage: storage.Config{MemBacked: config.BoolPointer(false), Dirname: "./tmp"},
		}),
	)
})
