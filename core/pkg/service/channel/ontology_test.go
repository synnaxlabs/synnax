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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Ontology Helpers", func() {
	Describe("OntologyID", func() {
		It("Should return the ontology ID for the channel key", func() {
			Expect(channel.OntologyID(distchannel.NewKey(1, 2))).To(Equal(ontology.ID{
				Type: "channel",
				Key:  distchannel.NewKey(1, 2).String(),
			}))
		})
	})
	Describe("KeysFromOntologyIDs", func() {
		It("Should parse a list of ontology IDs into a list of keys", func() {
			ids := []ontology.ID{
				{Type: "channel", Key: "1"},
				{Type: "channel", Key: "2"},
			}
			Expect(channel.KeysFromOntologyIDs(ids)).To(Equal(channel.Keys{1, 2}))
		})
		It("Should return an error if a key cannot be parsed", func() {
			ids := []ontology.ID{
				{Type: "channel", Key: "1"},
				{Type: "channel", Key: "a"},
			}
			Expect(channel.KeysFromOntologyIDs(ids)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("a is not a valid channel key")),
			))
		})
	})
	Describe("OntologyIDsFromKeys", func() {
		It("Should return the ontology ID for each key", func() {
			keys := channel.Keys{distchannel.NewKey(1, 2), distchannel.NewKey(3, 4)}
			Expect(channel.OntologyIDsFromKeys(keys)).To(Equal([]ontology.ID{
				channel.OntologyID(keys[0]),
				channel.OntologyID(keys[1]),
			}))
		})
	})
})

var _ = Describe("Ontology", Ordered, func() {
	Describe("OntologyID", func() {
		It(
			"Should correctly return the ontology.ID for the specified channel",
			func(ctx SpecContext) {
				ch := &channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Int64T,
					Virtual:  true,
				}
				Expect(channelWriter.Create(ctx, ch)).To(Succeed())
				Expect(ch.OntologyID()).To(Equal(channel.OntologyID(ch.Key())))
			},
		)
	})
	Describe("ToPayload", func() {
		It("Should include operations when the channel has them", func() {
			ch := channel.Channel{
				Name:       "with_ops",
				DataType:   telem.Float64T,
				Operations: []channel.Operation{{Type: channel.OperationTypeAvg}},
			}
			Expect(ch.ToPayload()).To(HaveKeyWithValue("operations", ch.Operations))
		})
		It("Should omit the operations key when the channel has no operations", func() {
			ch := channel.Channel{Name: "no_ops", DataType: telem.Float64T}
			Expect(ch.ToPayload()).ToNot(HaveKey("operations"))
		})
	})
	Describe("Search", func() {
		It(
			"Should return the channels whose name matches the term",
			func(ctx SpecContext) {
				inlet := channel.Channel{
					Name:     "turbopump_inlet_pressure",
					DataType: telem.Int64T,
					Virtual:  true,
				}
				outlet := channel.Channel{
					Name:     "turbopump_outlet_pressure",
					DataType: telem.Int64T,
					Virtual:  true,
				}
				Expect(channelWriter.Create(ctx, &inlet)).To(Succeed())
				Expect(channelWriter.Create(ctx, &outlet)).To(Succeed())
				Expect(MustSucceed(svc.Search(ctx, db, "turbopump inlet", 10)).Keys).
					To(Equal([]ontology.ID{inlet.OntologyID()}))
				Expect(MustSucceed(svc.Search(ctx, db, "turbopump", 10)).Keys).
					To(ConsistOf(inlet.OntologyID(), outlet.OntologyID()))
			},
		)
	})
	Describe("OnChange", func() {
		Context("Create", func() {
			It("Should correctly propagate a create change", func(ctx SpecContext) {
				changes := make(chan []ontology.Change, 5)
				dc := svc.OnChange(
					func(_ context.Context, nexter iter.Seq[ontology.Change]) {
						changesSlice := make([]ontology.Change, 0)
						for ch := range nexter {
							changesSlice = append(changesSlice, ch)
						}
						changes <- changesSlice
					},
				)
				defer dc()
				ch := &channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Int64T,
					Virtual:  true,
				}
				Expect(channelWriter.Create(ctx, ch)).To(Succeed())
				var c []ontology.Change
				Eventually(changes).Should(Receive(&c))
				Expect(c).To(HaveLen(1))
				v := c[0]
				Expect(v.Variant).To(Equal(change.VariantSet))
				Expect(v.Key).To(Equal(channel.OntologyID(ch.Key()).String()))
			})
		})
	})
	Describe("RetrieveResource", func() {
		It("Should correctly retrieve a resource", func(ctx SpecContext) {
			ch := &channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Int64T,
				Virtual:  true,
			}
			Expect(channelWriter.Create(ctx, ch)).To(Succeed())
			r := MustSucceed(svc.RetrieveResource(ctx, ch.Key().String(), nil))
			Expect(r.Name).To(Equal(ch.Name))
		})
	})
})
