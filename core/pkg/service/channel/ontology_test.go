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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Ontology Helpers", func() {
	Describe("ResourceID", func() {
		It("Should return the resource ID for the channel key", func() {
			Expect(channel.ResourceID(distchannel.NewKey(1, 2))).To(Equal(resource.ID{
				Type: "channel",
				Key:  distchannel.NewKey(1, 2).String(),
			}))
		})
	})
	Describe("KeysFromResourceIDs", func() {
		It("Should parse a list of resource IDs into a list of keys", func() {
			ids := []resource.ID{
				{Type: "channel", Key: "1"},
				{Type: "channel", Key: "2"},
			}
			Expect(channel.KeysFromResourceIDs(ids)).To(Equal(channel.Keys{1, 2}))
		})
		It("Should return an error if a key cannot be parsed", func() {
			ids := []resource.ID{
				{Type: "channel", Key: "1"},
				{Type: "channel", Key: "a"},
			}
			Expect(channel.KeysFromResourceIDs(ids)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("a is not a valid channel key")),
			))
		})
	})
	Describe("ResourceIDsFromKeys", func() {
		It("Should return the resource ID for each key", func() {
			keys := channel.Keys{distchannel.NewKey(1, 2), distchannel.NewKey(3, 4)}
			Expect(channel.ResourceIDsFromKeys(keys)).To(Equal([]resource.ID{
				channel.ResourceID(keys[0]),
				channel.ResourceID(keys[1]),
			}))
		})
	})
})

var _ = Describe("Ontology", Ordered, func() {
	Describe("ResourceID", func() {
		It(
			"Should correctly return the resource.ID for the specified channel",
			func(ctx SpecContext) {
				ch := &channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Int64T,
					Virtual:  true,
				}
				Expect(channelWriter.Create(ctx, ch)).To(Succeed())
				Expect(ch.ResourceID()).To(Equal(channel.ResourceID(ch.Key())))
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
				q := search.Query[resource.ID]{Term: "turbopump inlet", Limit: 10}
				Expect(
					MustSucceed(svc.Search(ctx, db, q)).Keys,
				).To(Equal([]resource.ID{inlet.ResourceID()}))
				q = search.Query[resource.ID]{Term: "turbopump", Limit: 10}
				Expect(MustSucceed(svc.Search(ctx, db, q)).Keys).
					To(ConsistOf(inlet.ResourceID(), outlet.ResourceID()))
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
				Expect(v.Key).To(Equal(channel.ResourceID(ch.Key()).String()))
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
