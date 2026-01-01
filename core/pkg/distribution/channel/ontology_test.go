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
	"iter"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ontology", Ordered, func() {
	var mockCluster *mock.Cluster
	BeforeAll(func() { mockCluster = mock.ProvisionCluster(ctx, 1) })
	AfterAll(func() {
		Expect(mockCluster.Close()).To(Succeed())
	})
	Describe("OpenNexter", func() {
		It("Should correctly iterate over all channels", func() {
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &channel.Channel{Name: "SG01", DataType: telem.Int64T, Virtual: true})).To(Succeed())
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &channel.Channel{Name: "SG02", DataType: telem.Int64T, Virtual: true})).To(Succeed())
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &channel.Channel{Name: "SG03", DataType: telem.Int64T, Virtual: true})).To(Succeed())
			n, closer := MustSucceed2(mockCluster.Nodes[1].Channel.OpenNexter(ctx))
			defer func() {
				GinkgoRecover()
				Expect(closer.Close()).To(Succeed())
			}()
			values := slices.Collect(n)
			Expect(len(values)).To(BeNumerically(">", 4))
			names := lo.Map(values, func(v ontology.Resource, _ int) string { return v.Name })
			Expect(names).To(ContainElements("sy_node_1_control", "SG01", "SG02", "SG03"))
		})
	})
	Describe("OnChange", func() {
		Context("Create", func() {
			It("Should correctly propagate a create change", func() {
				changes := make(chan []ontology.Change, 5)
				dc := mockCluster.Nodes[1].Channel.OnChange(func(ctx context.Context, nexter iter.Seq[ontology.Change]) {
					changesSlice := make([]ontology.Change, 0)
					for ch := range nexter {
						changesSlice = append(changesSlice, ch)
					}
					changes <- changesSlice
				})
				defer dc()
				ch := &channel.Channel{Name: channel.NewRandomName(), DataType: telem.Int64T, Virtual: true}
				Expect(mockCluster.Nodes[1].Channel.Create(ctx, ch))
				Eventually(func(g Gomega) {
					c := <-changes
					g.Expect(c).To(HaveLen(1))
					v := c[0]
					g.Expect(v.Variant).To(Equal(change.Set))
					g.Expect(v.Key.Key).To(Equal(ch.Key().String()))
				}).Should(Succeed())
			})
		})
	})
	Describe("RetrieveResource", func() {
		It("Should correctly retrieve a resource", func() {
			ch := &channel.Channel{Name: channel.NewRandomName(), DataType: telem.Int64T, Virtual: true}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, ch)).To(Succeed())
			r := MustSucceed(mockCluster.Nodes[1].Channel.RetrieveResource(ctx, ch.Key().String(), nil))
			Expect(r.Name).To(Equal(ch.Name))
		})
	})
})
