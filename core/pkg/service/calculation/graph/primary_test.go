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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/calculation/graph"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Primary", func() {
	virtual := func(ctx SpecContext, name string) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	// store replaces the stored primary and reads of the calculation that writes
	// output, as an upgrade or a user leaves them.
	store := func(ctx SpecContext, output channel.Key, primary *channel.Key) {
		GinkgoHelper()
		c := MustSucceed(creator.CalculationOf(ctx, output))
		Expect(gorp.NewUpdate[calculation.Key, calculation.Calculation]().
			Where(gorp.MatchKeys[calculation.Key, calculation.Calculation](c.Key)).
			Change(func(
				_ gorp.Context,
				c calculation.Calculation,
			) calculation.Calculation {
				c.Primary = primary
				if primary == nil {
					c.Reads = nil
				}
				return c
			}).
			Exec(ctx, db)).To(Succeed())
	}
	primaryOf := func(ctx context.Context, output channel.Key) *channel.Key {
		GinkgoHelper()
		return MustSucceed(creator.CalculationOf(ctx, output)).Primary
	}

	It("Should store the primary of an upgraded record as it opens, before any "+
		"calculator runs, and keep it on the next open", func(ctx SpecContext) {
		first := virtual(ctx, "pr_open_first")
		second := virtual(ctx, "pr_open_second")
		out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       "pr_open_out",
			Expression: "return pr_open_second + pr_open_first",
		}))
		store(ctx, out.Key(), nil)
		g := MustSucceed(graph.Open(ctx, graph.Config{
			DB:          db,
			Channel:     channelSvc,
			Calculation: creator.Calculation,
			Status:      statusSvc,
		}))
		Expect(primaryOf(ctx, out.Key())).To(HaveValue(Equal(first.Key())))
		Expect(g.Close()).To(Succeed())
		store(ctx, out.Key(), new(second.Key()))
		openGraph(ctx)
		Expect(primaryOf(ctx, out.Key())).To(HaveValue(Equal(second.Key())))
	})
})
