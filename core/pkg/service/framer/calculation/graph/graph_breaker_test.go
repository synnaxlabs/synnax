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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/graph"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Graph breaker", func() {
	// saveAndAdd creates the channels, saves the calculations in order through the
	// calculation writer, and adds the output of the last one to a new graph.
	saveAndAdd := func(
		ctx SpecContext,
		chs []channel.Channel,
		calcs ...func(chs []channel.Channel) calculation.Calculation,
	) error {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).CreateMany(ctx, &chs)
		})).To(Succeed())
		var last calculation.Calculation
		for _, f := range calcs {
			last = f(chs)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return creator.Calculation.NewWriter(tx).Create(ctx, &last)
			})).To(Succeed(), "the save accepts the calculation")
		}
		var output channel.Channel
		Expect(channelSvc.NewRetrieve().
			Where(channel.MatchKeys(last.Output)).
			Entry(&output).
			Exec(ctx, db)).To(Succeed())
		g := MustSucceed(graph.New(graph.Config{
			Channel:     channelSvc,
			Calculation: creator.Calculation,
			DB:          db,
		}))
		return g.Add(ctx, output)
	}
	maxResetOn := func(reset channel.Key) []calculation.Operation {
		return []calculation.Operation{{Variant: calculation.MaxOperation{
			Window: calculation.Window{Reset: new(reset)},
		}}}
	}

	It(
		"Should run a saved calculation whose operation resets on a channel that "+
			"is not a boolean",
		func(ctx SpecContext) {
			Expect(saveAndAdd(
				ctx,
				[]channel.Channel{
					{Name: "brk_type_base", DataType: telem.Int64T, Virtual: true},
					{Name: "brk_type_flag", DataType: telem.Int64T, Virtual: true},
					{Name: "brk_type_out", DataType: telem.Int64T, Virtual: true},
				},
				func(chs []channel.Channel) calculation.Calculation {
					return calculation.Calculation{
						Output:     chs[2].Key(),
						Expression: "return brk_type_base * 2",
						Operations: maxResetOn(chs[1].Key()),
					}
				},
			)).To(Succeed())
		},
	)

	It(
		"Should run a saved calculation that resets on a calculation reading its "+
			"output",
		func(ctx SpecContext) {
			Expect(saveAndAdd(
				ctx,
				[]channel.Channel{
					{Name: "brk_cyc_base", DataType: telem.Int64T, Virtual: true},
					{Name: "brk_cyc_first", DataType: telem.Int64T, Virtual: true},
					{Name: "brk_cyc_second", DataType: telem.Int64T, Virtual: true},
				},
				func(chs []channel.Channel) calculation.Calculation {
					return calculation.Calculation{
						Output:     chs[2].Key(),
						Expression: "return brk_cyc_first > 0",
					}
				},
				func(chs []channel.Channel) calculation.Calculation {
					return calculation.Calculation{
						Output:     chs[1].Key(),
						Expression: "return brk_cyc_base * 2",
						Operations: maxResetOn(chs[2].Key()),
					}
				},
			)).To(Succeed())
		},
	)
})
