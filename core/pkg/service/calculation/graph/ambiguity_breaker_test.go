// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// A v0.62 store can hold one name in two folders. A calculated channel that reads the
// name is invalid at open, and heals once one holder leaves the name, as a calculated
// channel with an unknown name heals once a channel takes the name.
var _ = Describe("Ambiguity breaker", func() {
	// ambiguousCalc stores a calculated channel that reads dep, then a second holder
	// of dep in another folder, and opens the graph.
	ambiguousCalc := func(
		ctx context.Context,
		name, dep string,
		key channel.LocalKey,
	) (channel.Channel, channel.Channel) {
		GinkgoHelper()
		createDep(ctx, dep)
		calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       name,
			DataType:   telem.Int64T,
			Expression: "return " + dep + " + 1",
		}))
		folder := uuid.New()
		second := channel.Channel{
			Name:        dep,
			Leaseholder: node.KeyFree,
			LocalKey:    key,
			DataType:    telem.Int64T,
			Virtual:     true,
			Group:       &folder,
		}
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, second)).
			To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).
				Delete(ctx, second.Key())).To(Succeed())
		})
		openGraph(ctx)
		Expect(expectStatus(ctx, calc.Key()).Description).
			To(ContainSubstring("2 channels are named '" + dep + "'"))
		return calc, second
	}

	It("Should heal a calculated channel that reads an ambiguous name once one holder "+
		"is deleted", func(ctx SpecContext) {
		const dep = "amb_heal_del_dep"
		calc, second := ambiguousCalc(ctx, "amb_heal_del_calc", dep, 1_000_501)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Delete(ctx, second.Key(), false)
		})).To(Succeed())
		// One channel holds the name now, so a fresh analysis resolves it.
		Expect(channelSvc.NewArcSymbolResolver(db).Resolve(ctx, dep)).
			Error().ToNot(HaveOccurred())
		eventuallyExpectNoStatus(ctx, calc.Key())
	})

	It("Should heal a calculated channel that reads an ambiguous name once one holder "+
		"is renamed", func(ctx SpecContext) {
		const dep = "amb_heal_ren_dep"
		calc, second := ambiguousCalc(ctx, "amb_heal_ren_calc", dep, 1_000_502)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).
				Rename(ctx, second.Key(), "amb_heal_ren_2", false)
		})).To(Succeed())
		// One channel holds the name now, so a fresh analysis resolves it.
		Expect(channelSvc.NewArcSymbolResolver(db).Resolve(ctx, dep)).
			Error().ToNot(HaveOccurred())
		eventuallyExpectNoStatus(ctx, calc.Key())
	})
})
