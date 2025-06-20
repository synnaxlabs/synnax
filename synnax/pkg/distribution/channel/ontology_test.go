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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ontology", Ordered, func() {
	var (
		services map[core.NodeKey]channel.Service
		builder  *mock.CoreBuilder
	)
	BeforeAll(func() { builder, services, _ = provisionServices() })
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	Describe("OpenNexter", func() {
		It("Should correctly iterate over all channels", func() {
			Expect(services[1].Create(ctx, &channel.Channel{Name: "SG01", DataType: telem.Int64T, Virtual: true})).To(Succeed())
			Expect(services[1].Create(ctx, &channel.Channel{Name: "SG02", DataType: telem.Int64T, Virtual: true})).To(Succeed())
			Expect(services[1].Create(ctx, &channel.Channel{Name: "SG03", DataType: telem.Int64T, Virtual: true})).To(Succeed())
			n := testutil.MustSucceed(services[1].OpenNexter())
			v, ok := n.Next(ctx)
			Expect(ok).To(BeTrue())
			Expect(v.Name).To(Equal("SG01"))
			v, ok = n.Next(ctx)
			Expect(ok).To(BeTrue())
			Expect(v.Name).To(Equal("SG02"))
			v, ok = n.Next(ctx)
			Expect(ok).To(BeTrue())
			Expect(v.Name).To(Equal("SG03"))
			Expect(n.Close()).To(Succeed())
		})
	})
	Describe("OnChange", func() {
		Context("Create", func() {
			It("Should correctly propagate a create change", func() {
				changes := make(chan []ontology.Change, 5)
				dc := services[1].OnChange(func(ctx context.Context, nexter iter.Nexter[ontology.Change]) {
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
				ch := &channel.Channel{Name: "SG01", DataType: telem.Int64T, Virtual: true}
				Expect(services[1].Create(ctx, ch))
				Eventually(func(g Gomega) {
					c := <-changes
					g.Expect(c).To(HaveLen(1))
					v := c[0]
					g.Expect(v.Variant).To(Equal(change.Set))
					g.Expect(v.Key.Key).To(Equal(ch.Key().String()))
				}, 1*time.Second).Should(Succeed())
			})
		})
	})
	Describe("RetrieveResource", func() {
		It("Should correctly retrieve a resource", func() {
			ch := &channel.Channel{Name: "SG01", DataType: telem.Int64T, Virtual: true}
			Expect(services[1].Create(ctx, ch))
			r, err := services[1].RetrieveResource(ctx, ch.Key().String(), nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(r).ToNot(BeNil())
			Expect(r.Name).To(Equal(ch.Name))
		})
	})
})
