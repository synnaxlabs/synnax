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
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Service", Ordered, func() {
	var (
		mockCluster *mock.Cluster
		services    map[node.Key]*channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		mockCluster = mock.MustOpenCluster(ctx, 1)
		services = make(map[node.Key]*channel.Service)
		for k, n := range mockCluster.Nodes {
			services[k] = openService(ctx, n)
		}
	})

	Describe("Observe", func() {
		It("Should notify when a channel is created", func(ctx SpecContext) {
			var called atomic.Bool
			disconnect := services[1].Observe().OnChange(func(ctx context.Context, _ gorp.TxReader[channel.Key, channel.Channel]) {
				called.Store(true)
			})
			DeferCleanup(disconnect)
			ch := channel.Channel{
				Name:        channel.NewRandomName(),
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(services[1].Create(ctx, &ch)).To(Succeed())
			Eventually(called.Load).Should(BeTrue())
		})
	})
})

var _ = Describe("Service Passthrough", func() {
	Describe("Group", func() {
		It("Should return a valid group", func(ctx SpecContext) {
			g := svc.Group()
			Expect(g.Key).ToNot(BeZero())
		})
	})

	Describe("NewRetrieve", func() {
		It("Should retrieve a channel created through the service", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(svc.Create(ctx, &ch)).To(Succeed())
			var retrieved channel.Channel
			Expect(svc.NewRetrieve().Where(channel.MatchKeys(ch.Key())).Entry(&retrieved).Exec(ctx, nil)).To(Succeed())
			Expect(retrieved.Name).To(Equal(ch.Name))
		})
	})

	Describe("Observe", func() {
		It("Should return a non-nil observable", func(ctx SpecContext) {
			obs := svc.Observe()
			Expect(obs).ToNot(BeNil())
		})
	})
})
