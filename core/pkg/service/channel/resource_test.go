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
	"github.com/samber/lo"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	create := func(ctx SpecContext, name string, internal bool) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     name,
			DataType: telem.Float32T,
			Virtual:  true,
			Internal: internal,
		}
		Expect(channelWriter.Create(ctx, &ch)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(channelWriter.Delete(ctx, ch.Key(), true)).To(Succeed())
		})
		return ch
	}

	It("Should leave internal channels out of a search", func(ctx SpecContext) {
		create(ctx, "hydrazine", true)
		tank := create(ctx, "hydrazine_tank", false)
		Expect(MustSucceed(svc.Resources().Search(
			ctx,
			db,
			search.Query[resource.ID]{Term: "hydrazine", Limit: 1},
		)).Keys).To(Equal([]resource.ID{tank.ResourceID()}))
	})

	It("Should leave internal channels out of a list", func(ctx SpecContext) {
		hidden := create(ctx, "hydrazine", true)
		tank := create(ctx, "hydrazine_tank", false)
		ids := lo.Map(
			MustSucceed(svc.Resources().List(ctx, db, 10000, nil)),
			func(r resource.Resource, _ int) resource.ID { return r.ID },
		)
		Expect(ids).To(ContainElement(tank.ResourceID()))
		Expect(ids).ToNot(ContainElement(hidden.ResourceID()))
	})

	It("Should name each channel by its name", func(ctx SpecContext) {
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(channelWriter.Create(ctx, &ch)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(channelWriter.Delete(ctx, ch.Key(), false)).To(Succeed())
		})
		id := channel.ResourceID(ch.Key())
		Expect(svc.Resources().Retrieve(ctx, db, []resource.ID{id})).
			To(Equal([]resource.Resource{{ID: id, Name: ch.Name}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypeChannel, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("pump is not a valid channel key")),
		))
	})

	It("Should return a validation error for a key with a leading zero", func(
		ctx SpecContext,
	) {
		ch := create(ctx, "fuel", false)
		key := "0" + ch.ResourceID().Key
		Expect(svc.Resources().Retrieve(
			ctx,
			db,
			[]resource.ID{{Type: resource.TypeChannel, Key: key}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid channel key "%s"`, key)),
		))
	})

	It("Should leave out an ID that names no channel", func(ctx SpecContext) {
		ch := create(ctx, "oxidizer", false)
		missing := resource.ID{Type: resource.TypeChannel, Key: "4294967295"}
		Expect(svc.Resources().Retrieve(
			ctx,
			db,
			[]resource.ID{missing, ch.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: ch.ResourceID(), Name: ch.Name}}))
	})

	It("Should list each channel by its name", func(ctx SpecContext) {
		ch := create(ctx, "oxidizer", false)
		Expect(svc.Resources().List(ctx, db, 10000, nil)).To(ContainElement(
			resource.Resource{ID: ch.ResourceID(), Name: ch.Name},
		))
	})
})

var _ = Describe("Resource helpers", func() {
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

var _ = Describe("Resource", Ordered, func() {
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
					MustSucceed(svc.Resources().Search(ctx, db, q)).Keys,
				).To(Equal([]resource.ID{inlet.ResourceID()}))
				q = search.Query[resource.ID]{Term: "turbopump", Limit: 10}
				Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).
					To(ConsistOf(inlet.ResourceID(), outlet.ResourceID()))
			},
		)
	})
})
