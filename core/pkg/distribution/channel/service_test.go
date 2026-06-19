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
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
)

var _ = Describe("Service", func() {
	Describe("ServiceConfig", func() {
		DescribeTable("Should report each missing required field",
			func(field string) {
				Expect(channel.ServiceConfig{}.Validate()).To(MatchError(ContainSubstring(field)))
			},
			Entry("host resolver", "host_resolver"),
			Entry("kv read writer", "kv_read_writer"),
			Entry("ts db", "ts_db"),
			Entry("transport", "transport"),
		)

		Context("With all required dependencies", Ordered, func() {
			var (
				builder *mock.Cluster
				valid   channel.ServiceConfig
			)
			BeforeAll(func(ctx SpecContext) {
				builder = mock.NewCluster(ctx, 1)
				n := builder.Nodes[node.KeyBootstrapper]
				valid = channel.ServiceConfig{
					HostResolver: n.Cluster,
					KVReadWriter: n.DB,
					TSDB:         n.Storage.TS,
					Transport:    tmock.NewChannelNetwork().New("mock"),
				}
			})

			It("Should pass validation when all required fields are set", func() {
				Expect(valid.Validate()).To(Succeed())
			})
			It("Should carry the override's fields onto a zero config", func() {
				Expect(channel.ServiceConfig{}.Override(valid).Validate()).To(Succeed())
			})
		})
	})

	Describe("NewService", func() {
		It("Should return an error when required configuration is missing", func(ctx SpecContext) {
			Expect(channel.NewService(ctx, channel.ServiceConfig{})).Error().To(
				MatchError(ContainSubstring("host_resolver")),
			)
		})
	})
})
