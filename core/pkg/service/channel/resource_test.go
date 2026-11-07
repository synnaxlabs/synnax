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
