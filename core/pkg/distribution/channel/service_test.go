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
)

// stubTransport is a no-op channel.Transport used to satisfy the non-nil requirement on
// ServiceConfig.Transport in tests that exercise configuration validation only. Its
// endpoint accessors are never invoked, so they return nil.
type stubTransport struct{}

var _ channel.Transport = stubTransport{}

func (stubTransport) CreateClient() channel.CreateTransportClient { return nil }
func (stubTransport) CreateServer() channel.CreateTransportServer { return nil }
func (stubTransport) DeleteClient() channel.DeleteTransportClient { return nil }
func (stubTransport) DeleteServer() channel.DeleteTransportServer { return nil }
func (stubTransport) RenameClient() channel.RenameTransportClient { return nil }
func (stubTransport) RenameServer() channel.RenameTransportServer { return nil }

var _ = Describe("Service", func() {
	Describe("ServiceConfig", func() {
		DescribeTable("Should report each missing required field",
			func(field string) {
				Expect(channel.ServiceConfig{}.Validate()).To(MatchError(ContainSubstring(field)))
			},
			Entry("host resolver", "host_resolver"),
			Entry("kv read writer", "kv_read_writer"),
			Entry("time-series database", "ts"),
			Entry("transport", "transport"),
		)

		Context("With all required dependencies", Ordered, func() {
			var (
				builder *mock.Cluster
				valid   channel.ServiceConfig
			)
			BeforeAll(func(ctx SpecContext) {
				builder = mock.MustOpenCluster(ctx, 1)
				n := builder.Nodes[node.KeyBootstrapper]
				valid = channel.ServiceConfig{
					HostResolver: n.Cluster,
					KVReadWriter: n.DB,
					TS:           n.Storage.TS,
					Transport:    stubTransport{},
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
