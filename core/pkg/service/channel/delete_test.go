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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Planned delete", Ordered, func() {
	var (
		db  *gorp.DB
		svc *channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		svc = openService(ctx, node)
	})
	create := func(ctx SpecContext, ch channel.Channel) channel.Channel {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	exists := func(ctx SpecContext, keys ...channel.Key) bool {
		GinkgoHelper()
		return MustSucceed(svc.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			Exists(ctx, db))
	}

	It("Should plan and delete channels that nothing holds", func(ctx SpecContext) {
		keys := MustSucceed(CreateVirtual(ctx, db, svc, 2))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			removed, effects := MustSucceed2(svc.PlanDelete(ctx, tx, keys))
			Expect(removed).To(ConsistOf(keys))
			Expect(effects.Restricts).To(BeEmpty())
			return svc.NewWriter(tx).DeletePlanned(ctx, removed, effects)
		})).To(Succeed())
		Expect(exists(ctx, keys...)).To(BeFalse())
	})

	It("Should find the keys of named channels and skip an unknown name", func(
		ctx SpecContext,
	) {
		a := create(ctx, channel.Channel{
			Name: UniqueChannelName(), DataType: telem.Float32T, Virtual: true,
		})
		b := create(ctx, channel.Channel{
			Name: UniqueChannelName(), DataType: telem.Float32T, Virtual: true,
		})
		Expect(svc.KeysByNames(ctx, db, a.Name, UniqueChannelName(), b.Name)).
			To(ConsistOf(a.Key(), b.Key()))
	})

	It("Should refuse to plan the delete of a built-in channel", func(ctx SpecContext) {
		builtIn := channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.TimestampT,
			IsIndex:     true,
			Leaseholder: 1,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Declare(ctx, &builtIn)
		})).To(Succeed())
		Expect(svc.PlanDelete(ctx, db, channel.Keys{builtIn.Key()})).
			Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("can't delete built-in channel")),
		))
	})

	It("Should remove the index of a deleted calculated channel", func(
		ctx SpecContext,
	) {
		calc := create(ctx, channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float64T,
			Expression: "return 1 + 1",
			Virtual:    true,
		})
		removed, _ := MustSucceed2(svc.PlanDelete(ctx, db, channel.Keys{calc.Key()}))
		Expect(removed).To(ConsistOf(calc.Key(), calc.Index()))
	})

	It("Should refuse a planned delete while a restrict holder still exists", func(
		ctx SpecContext,
	) {
		idx := create(ctx, channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.TimestampT,
			IsIndex:     true,
			Leaseholder: 1,
		})
		data := create(ctx, channel.Channel{
			Name:        UniqueChannelName(),
			DataType:    telem.Float64T,
			LocalIndex:  idx.LocalKey,
			Leaseholder: 1,
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			removed, effects := MustSucceed2(
				svc.PlanDelete(ctx, tx, channel.Keys{idx.Key()}),
			)
			Expect(effects.Restricts).To(HaveLen(1))
			return svc.NewWriter(tx).DeletePlanned(ctx, removed, effects)
		})).To(MatchError(gorp.RestrictError{
			Target:    gorp.NewTarget("channel", idx.Key()),
			Holder:    gorp.NewTarget("channel", data.Key()),
			Reference: "index",
		}))
		Expect(exists(ctx, idx.Key())).To(BeTrue())
	})
})
