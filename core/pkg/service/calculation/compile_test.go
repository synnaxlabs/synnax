// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/calculation/compiler"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Compile", func() {
	var (
		db      *gorp.DB
		svc     *calculation.Service
		creator calctestutil.Creator
		base    channel.Channel
		output  channel.Channel
		calc    calculation.Calculation
	)
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		channels := openChannels(ctx, node)
		svc = MustOpen(calculation.OpenService(ctx, calculation.ServiceConfig{
			DB:      db,
			Channel: channels,
		}))
		creator = calctestutil.Creator{DB: db, Channel: channels, Calculation: svc}
		base = channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Create(ctx, &base)
		})).To(Succeed())
		output = MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       UniqueChannelName(),
			DataType:   telem.Int64T,
			Expression: "return " + base.Name + " * 2",
		}))
		calc = MustSucceed(creator.CalculationOf(ctx, output.Key()))
	})

	It("Should compile with the channel service and symbols of the service", func(
		ctx SpecContext,
	) {
		mod := MustSucceed(svc.Compile(ctx, db, compiler.Config{
			Channel:     output,
			Calculation: calc,
		}))
		Expect(mod.Dependencies.Reads.Slice()).To(ConsistOf(base.Key()))
		Expect(mod.Dependencies.Writes.Slice()).To(ContainElement(output.Key()))
	})

	It("Should pre-process with the channel service and symbols of the service", func(
		ctx SpecContext,
	) {
		prog := MustSucceed(svc.PreProcess(ctx, db, compiler.Config{
			Channel:     output,
			Calculation: calc,
		}))
		Expect(prog.Functions).To(HaveLen(1))
		Expect(prog.Functions[0].Channels.Read).To(HaveKey(uint32(base.Key())))
	})

	It("Should analyze an expression with the channels that snap holds", func(
		ctx SpecContext,
	) {
		result := MustSucceed(svc.NewAnalyzer(db).AnalyzeExpression(
			ctx,
			output,
			calc.Expression,
			false,
		))
		Expect(result.Deps).To(ConsistOf(base.Key()))
		Expect(result.ChanDataType).To(Equal(telem.Int64T))
	})
})
