// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("UniqueChannelName", func() {
	It("Should generate a name with the test channel prefix", func() {
		Expect(UniqueChannelName()).To(HavePrefix("test_ch_"))
	})
	It("Should generate unique channel names", func() {
		count := 100
		existingNames := make(set.Set[string], count)
		for range count {
			nextName := UniqueChannelName()
			Expect(existingNames.Contains(nextName)).To(BeFalse())
			existingNames.Add(nextName)
		}
	})
})

var _ = Describe("CreateVirtual", func() {
	var svc *channel.Service
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: node.DB}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       node.DB,
			Ontology: otg,
		}))
		svc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel:      node.Channel,
			DB:           node.DB,
			HostProvider: node.Cluster,
			Ontology:     otg,
			Status:       statusSvc,
		}))
	})

	It("Should create virtual channels with unique names", func(ctx SpecContext) {
		keys := MustSucceed(CreateVirtual(ctx, svc, 3))
		Expect(keys).To(HaveLen(3))
		var channels []channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			Entries(&channels).
			Exec(ctx, nil)).To(Succeed())
		Expect(channel.KeysFromChannels(channels)).To(ConsistOf(keys))
		Expect(channels).To(HaveEach(HaveField("Virtual", BeTrue())))
		Expect(set.New(channel.Names(channels)...)).To(HaveLen(3))
	})

	It("Should create no channels for a zero count", func(ctx SpecContext) {
		Expect(CreateVirtual(ctx, svc, 0)).To(BeEmpty())
	})
})
