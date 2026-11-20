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
			Operations: []channel.Operation{{Type: "avg", Duration: 5 * telem.Second}},
		}))
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
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
			Operations: []channel.Operation{{Type: "derivative"}},
		}))
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
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
		calc.Expression = "return invalid_syntax {{"
		Expect(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
		})).Error().To(ContainSubstring("extraneous input '{'"))
	})
})
