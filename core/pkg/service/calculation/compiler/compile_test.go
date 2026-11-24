// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/stl/join"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/calculation/compiler"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	db         *gorp.DB
	channelSvc *channel.Service
	creator    calctestutil.Creator
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: node.DB,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: node.DB,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		TS:      node.Storage.TS,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	creator = calctestutil.Creator{
		DB:      db,
		Channel: channelSvc,
		Calculation: MustOpen(calculation.OpenService(ctx, calculation.ServiceConfig{
			DB:      db,
			Channel: channelSvc,
			Framer:  node.Framer,
		})),
	}
})

var _ = Describe("Compile", func() {
	It("Should compile simple expression", func(ctx SpecContext) {
		base := channel.Channel{Name: "base", DataType: telem.Int64T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &base)
		})).To(Succeed())
		calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       "calc",
			DataType:   telem.Int64T,
			Expression: "return base * 2",
		}))
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
			Calculation:    MustSucceed(creator.CalculationOf(ctx, calc.Key())),
		}))
		Expect(mod.Channel.Key()).To(Equal(calc.Key()))
		Expect(mod.Dependencies.Reads.Slice()).To(ContainElement(base.Key()))
		Expect(mod.Dependencies.Writes.Slice()).To(ContainElement(calc.Key()))
	})

	It("Should compile expression with operations", func(ctx SpecContext) {
		base := channel.Channel{Name: "base2", DataType: telem.Int64T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &base)
		})).To(Succeed())
		calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       "calc2",
			DataType:   telem.Int64T,
			Expression: "return base2 + 1",
			Operations: []calculation.Operation{{Variant: calculation.AvgOperation{
				Window: calculation.Window{Duration: 5 * telem.Second},
			}}},
		}))
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
			Calculation:    MustSucceed(creator.CalculationOf(ctx, calc.Key())),
		}))
		Expect(mod.Channel.Key()).To(Equal(calc.Key()))
		Expect(mod.Dependencies.Reads.Slice()).To(ContainElement(base.Key()))
	})

	It("Should compile with multiple dependencies", func(ctx SpecContext) {
		channels := []channel.Channel{
			{Name: "base3", DataType: telem.Int64T, Virtual: true},
			{Name: "base4", DataType: telem.Int64T, Virtual: true},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).CreateMany(ctx, &channels)
		})).To(Succeed())
		calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       "calc3",
			DataType:   telem.Int64T,
			Expression: "return base3 + base4",
		}))
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
			Calculation:    MustSucceed(creator.CalculationOf(ctx, calc.Key())),
		}))
		Expect(
			mod.Dependencies.Reads.Slice(),
		).To(ContainElements(channel.KeysFromChannels(channels)))
		Expect(mod.Dependencies.Writes.Slice()).To(ContainElement(calc.Key()))
	})

	It("Should compile expression with derivative operation", func(ctx SpecContext) {
		base := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float64T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &base)
		})).To(Succeed())
		calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       UniqueChannelName(),
			DataType:   telem.Float64T,
			Expression: "return " + base.Name,
			Operations: []calculation.Operation{
				{Variant: calculation.DerivativeOperation{}},
			},
		}))
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
			Calculation:    MustSucceed(creator.CalculationOf(ctx, calc.Key())),
		}))
		Expect(mod.Channel.Key()).To(Equal(calc.Key()))
		Expect(mod.Dependencies.Reads.Slice()).To(ContainElement(base.Key()))
	})

	It("Should fail with invalid expression", func(ctx SpecContext) {
		calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       "calc4",
			DataType:   telem.Int64T,
			Expression: "return 1",
		}))
		c := MustSucceed(creator.CalculationOf(ctx, calc.Key()))
		c.Expression = "return invalid_syntax {{"
		Expect(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
			Calculation:    c,
		})).Error().To(ContainSubstring("extraneous input '{'"))
	})
})

var _ = Describe("Join", func() {
	virtual := func(ctx SpecContext, name string) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{Name: name, DataType: telem.Float64T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	stored := func(ctx SpecContext, name string) channel.Channel {
		GinkgoHelper()
		index := channel.Channel{
			Name:     name + "_time",
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		ch := channel.Channel{Name: name, DataType: telem.Float64T}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := channelSvc.NewWriter(tx)
			if err := w.Create(ctx, &index); err != nil {
				return err
			}
			ch.LocalIndex = index.LocalKey
			return w.Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	compile := func(
		ctx SpecContext,
		expression string,
		live bool,
	) (compiler.Module, error) {
		calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       UniqueChannelName(),
			DataType:   telem.Float64T,
			Expression: expression,
		}))
		return compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
			Calculation:    MustSucceed(creator.CalculationOf(ctx, calc.Key())),
			Live:           live,
		})
	}
	joinOf := func(mod compiler.Module) (ir.Node, bool) {
		for _, n := range mod.IR.Nodes {
			if n.Type == compiler.JoinType {
				return n, true
			}
		}
		return ir.Node{}, false
	}

	It("Should pair the reads of a live program through a join", func(ctx SpecContext) {
		stored(ctx, "join_a")
		stored(ctx, "join_b")
		mod := MustSucceed(compile(ctx, "return join_a + join_b", true))
		Expect(mod.Joined).To(BeTrue())
		n, found := joinOf(mod)
		Expect(found).To(BeTrue())
		Expect(n.Inputs).ToNot(ContainElement(
			HaveField("Name", HavePrefix("arrival_")),
		))
	})

	It("Should not join the reads of a history program", func(ctx SpecContext) {
		stored(ctx, "history_a")
		stored(ctx, "history_b")
		mod := MustSucceed(compile(ctx, "return history_a + history_b", false))
		Expect(mod.Joined).To(BeFalse())
		_, found := joinOf(mod)
		Expect(found).To(BeFalse())
	})

	It("Should not join reads that are all virtual", func(ctx SpecContext) {
		virtual(ctx, "virtual_a")
		virtual(ctx, "virtual_b")
		mod := MustSucceed(compile(ctx, "return virtual_a + virtual_b", true))
		Expect(mod.Joined).To(BeFalse())
		_, found := joinOf(mod)
		Expect(found).To(BeFalse())
	})

	It("Should pair a virtual read beside a stored one by arrival", func(
		ctx SpecContext,
	) {
		s := stored(ctx, "mixed_stored")
		v := virtual(ctx, "mixed_virtual")
		mod := MustSucceed(compile(ctx, "return mixed_stored + mixed_virtual", true))
		n, found := joinOf(mod)
		Expect(found).To(BeTrue())
		// The inputs take the order of their keys.
		arrival := 1
		if v.Key() < s.Key() {
			arrival = 0
		}
		Expect(n.Inputs).To(ContainElement(
			HaveField("Name", join.ArrivalParam(arrival)),
		))
		Expect(n.Inputs).ToNot(ContainElement(
			HaveField("Name", join.ArrivalParam(1-arrival)),
		))
	})

	It("Should fail a joined program whose primary is not a stored read", func(
		ctx SpecContext,
	) {
		stored(ctx, "primary_stored")
		v := virtual(ctx, "primary_virtual")
		calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       UniqueChannelName(),
			DataType:   telem.Float64T,
			Expression: "return primary_stored + primary_virtual",
		}))
		c := MustSucceed(creator.CalculationOf(ctx, calc.Key()))
		c.Primary = new(v.Key())
		Expect(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
			Calculation:    c,
			Live:           true,
		})).Error().To(MatchError(ContainSubstring(
			"of a joined calculation is not a stored read",
		)))
	})

	It("Should bind a channel named join in the expression", func(ctx SpecContext) {
		named := stored(ctx, "join")
		stored(ctx, "join_other")
		mod := MustSucceed(compile(ctx, "return join + join_other", true))
		Expect(mod.Joined).To(BeTrue())
		Expect(mod.Dependencies.Reads.Slice()).To(ContainElement(named.Key()))
	})
})
