// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/iter"
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
			names := []string{RandomName(), RandomName(), RandomName()}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &channel.Channel{Name: names[0], DataType: telem.Int64T, Virtual: true})).To(Succeed())
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &channel.Channel{Name: names[1], DataType: telem.Int64T, Virtual: true})).To(Succeed())
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, &channel.Channel{Name: names[2], DataType: telem.Int64T, Virtual: true})).To(Succeed())
			n := MustSucceed(mockCluster.Nodes[1].Channel.OpenNexter())
			v := MustBeOk(n.Next(ctx))
			Expect(v.Name).To(Equal("sy_node_1_control"))
			v = MustBeOk(n.Next(ctx))
			Expect(v.Name).To(Equal(names[0]))
			v = MustBeOk(n.Next(ctx))
			Expect(v.Name).To(Equal(names[1]))
			v = MustBeOk(n.Next(ctx))
			Expect(v.Name).To(Equal(names[2]))
			Expect(n.Close()).To(Succeed())
		})
	})
	Describe("OnChange", func() {
		Context("Create", func() {
			It("Should correctly propagate a create change", func() {
				changes := make(chan []ontology.Change, 5)
				dc := mockCluster.Nodes[1].Channel.OnChange(func(ctx context.Context, nexter iter.Nexter[ontology.Change]) {
					changesSlice := make([]ontology.Change, 0)
					for {
						v, ok := nexter.Next(ctx)
						if !ok {
							break
						}
						changesSlice = append(changesSlice, v)
					}
					changes <- changesSlice
				})
				defer dc()
				ch := &channel.Channel{Name: RandomName(), DataType: telem.Int64T, Virtual: true}
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
			ch := &channel.Channel{Name: RandomName(), DataType: telem.Int64T, Virtual: true}
			Expect(mockCluster.Nodes[1].Channel.Create(ctx, ch)).To(Succeed())
			r := MustSucceed(mockCluster.Nodes[1].Channel.RetrieveResource(ctx, ch.Key().String(), nil))
			Expect(r.Name).To(Equal(ch.Name))
		})
	})
})
