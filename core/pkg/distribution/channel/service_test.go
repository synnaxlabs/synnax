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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/channel"
	. "github.com/synnaxlabs/x/testutil"
)

// validServiceConfig opens an in-memory single-node cluster and returns a fully
// populated ServiceConfig backed by its host resolver, key-value store, time-series
// database, and a fresh in-memory channel transport.
func validServiceConfig(ctx context.Context) channel.ServiceConfig {
	node := mock.NewNode(ctx)
	return channel.ServiceConfig{
		HostResolver: node.Cluster,
		KV:           node.DB,
		TS:           node.Storage.TS,
		Transport:    tmock.NewNetwork().New("mock"),
	}
}

var _ = Describe("ServiceConfig", Ordered, func() {
	var valid channel.ServiceConfig
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		valid = validServiceConfig(ctx)
	})

	Describe("Validate", func() {
		It("Should pass when all required fields are set", func() {
			Expect(valid.Validate()).To(Succeed())
		})
		DescribeTable("Should report each missing required field",
			func(field string) {
				Expect(channel.ServiceConfig{}.Validate()).To(
					MatchError(ContainSubstring(field)),
				)
			},
			Entry("host resolver", "host_resolver: must be non-nil"),
			Entry("kv", "kv: must be non-nil"),
			Entry("time-series database", "ts: must be non-nil"),
			Entry("transport", "transport: must be non-nil"),
		)
	})

	Describe("Override", func() {
		It("Should carry the override's fields onto a zero config", func() {
			Expect(channel.ServiceConfig{}.Override(valid).Validate()).To(Succeed())
		})
	})
})

var _ = Describe("Service", Ordered, func() {
	var valid channel.ServiceConfig
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		valid = validServiceConfig(ctx)
	})

	It("Should open with a valid configuration", func(ctx SpecContext) {
		Expect(channel.NewService(ctx, valid)).ToNot(BeNil())
	})
	It("Should return an error with an invalid configuration", func(ctx SpecContext) {
		Expect(channel.NewService(ctx, channel.ServiceConfig{})).Error().To(
			MatchError(SatisfyAll(
				ContainSubstring("host_resolver: must be non-nil"),
				ContainSubstring("kv: must be non-nil"),
				ContainSubstring("ts: must be non-nil"),
				ContainSubstring("transport: must be non-nil"),
			)),
		)
	})
})
