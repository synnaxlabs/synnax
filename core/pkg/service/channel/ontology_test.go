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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Ontology Helpers", func() {
	Describe("OntologyID", func() {
		It("Should return the ontology ID for the channel key", func() {
			Expect(channel.OntologyID(channel.NewKey(1, 2))).To(Equal(ontology.ID{
				Type: "channel",
				Key:  channel.NewKey(1, 2).String(),
			}))
		})
	})
	Describe("KeysFromOntologyIDs", func() {
		It("Should parse a list of ontology IDs into a list of keys", func() {
			ids := []ontology.ID{{Type: "channel", Key: "1"}, {Type: "channel", Key: "2"}}
			Expect(channel.KeysFromOntologyIDs(ids)).To(Equal(channel.Keys{1, 2}))
		})
		It("Should return an error if a key cannot be parsed", func() {
			ids := []ontology.ID{{Type: "channel", Key: "1"}, {Type: "channel", Key: "a"}}
			Expect(channel.KeysFromOntologyIDs(ids)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("a is not a valid channel key")),
			))
		})
	})
	Describe("OntologyIDsFromKeys", func() {
		It("Should return the ontology ID for each key", func() {
			keys := channel.Keys{channel.NewKey(1, 2), channel.NewKey(3, 4)}
			Expect(channel.OntologyIDsFromKeys(keys)).To(Equal([]ontology.ID{
				channel.OntologyID(keys[0]),
				channel.OntologyID(keys[1]),
			}))
		})
	})
})

var _ = Describe("Ontology", Ordered, func() {
	Describe("OntologyID", func() {
		It("Should correctly return the ontology.ID for the specified channel", func(ctx SpecContext) {
			ch := &channel.Channel{Name: channel.NewRandomName(), DataType: telem.Int64T, Virtual: true}
			Expect(svc.Create(ctx, ch)).To(Succeed())
			Expect(ch.OntologyID()).To(Equal(channel.OntologyID(ch.Key())))
		})
	})
	Describe("OpenNexter", func() {
		It("Should correctly iterate over all channels", func(ctx SpecContext) {
			Expect(svc.Create(ctx, &channel.Channel{Name: "SG01", DataType: telem.Int64T, Virtual: true})).To(Succeed())
			Expect(svc.Create(ctx, &channel.Channel{Name: "SG02", DataType: telem.Int64T, Virtual: true})).To(Succeed())
			Expect(svc.Create(ctx, &channel.Channel{Name: "SG03", DataType: telem.Int64T, Virtual: true})).To(Succeed())
			n, closer := MustSucceed2(svc.OpenNexter(ctx))
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
			It("Should correctly propagate a create change", func(ctx SpecContext) {
				changes := make(chan []ontology.Change, 5)
				dc := svc.OnChange(func(ctx context.Context, nexter iter.Seq[ontology.Change]) {
					changesSlice := make([]ontology.Change, 0)
					for ch := range nexter {
						changesSlice = append(changesSlice, ch)
					}
					changes <- changesSlice
				})
				defer dc()
				ch := &channel.Channel{Name: channel.NewRandomName(), DataType: telem.Int64T, Virtual: true}
				Expect(svc.Create(ctx, ch)).To(Succeed())
				Eventually(func(g Gomega) {
					c := <-changes
					g.Expect(c).To(HaveLen(1))
					v := c[0]
					g.Expect(v.Variant).To(Equal(change.VariantSet))
					g.Expect(v.Key).To(Equal(channel.OntologyID(ch.Key()).String()))
				}).Should(Succeed())
			})
		})
	})
	Describe("RetrieveResource", func() {
		It("Should correctly retrieve a resource", func(ctx SpecContext) {
			ch := &channel.Channel{Name: channel.NewRandomName(), DataType: telem.Int64T, Virtual: true}
			Expect(svc.Create(ctx, ch)).To(Succeed())
			r := MustSucceed(svc.RetrieveResource(ctx, ch.Key().String(), nil))
			Expect(r.Name).To(Equal(ch.Name))
		})
	})
})
