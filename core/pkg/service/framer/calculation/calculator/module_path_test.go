// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculator_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation/compiler"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Module member names in a calculation", Ordered, func() {
	var (
		db       *gorp.DB
		channels *channel.Service
		creator  calctestutil.Creator
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		groups := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		channels = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      db,
			HostKey: node.Identity.HostKey,
			Status:  statuses,
		}))
		creator = calctestutil.Creator{DB: db, Channel: channels}
		g := group.Group{Name: "math", Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groups.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		inf := channel.Channel{
			Name:     "inf",
			DataType: telem.Float64T,
			Virtual:  true,
			Group:    &g.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Create(ctx, &inf)
		})).To(Succeed())
	})

	It("Should save, compile, and compute a calculation that reads the channel at a "+
		"module member's path", func(ctx SpecContext) {
		var inf channel.Channel
		Expect(channels.NewRetrieve().
			Where(channel.MatchPaths("math.inf")).
			Entry(&inf).
			Exec(ctx, db)).To(Succeed())
		out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       UniqueChannelName(),
			DataType:   telem.Float64T,
			Expression: "return math.inf * 2.0",
		}))
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channels,
			Channel:        out,
		}))
		c := MustOpen(calculator.Open(ctx, calculator.Config{Module: mod}))
		fr := frame.NewUnary(inf.Key(), telem.NewSeriesV(1.5, 2.5))
		written, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
		Expect(changed).To(BeTrue())
		Expect(written.Get(out.Key()).Series[0]).
			To(telem.MatchSeriesDataV(3.0, 5.0))
	})

	It("Should fail a module member that no channel holds the same way at save and at "+
		"compile", func(ctx SpecContext) {
		calc := channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float64T,
			Virtual:    true,
			Expression: "return math.nan * 2.0",
		}
		_, saveErr := channel.NewCalculationAnalyzer(channels.NewArcSymbolResolver(db)).
			Analyze(ctx, calc)
		Expect(saveErr).To(MatchError(ContainSubstring("math.nan does not exist")))
		Expect(compiler.PreProcess(ctx, db, compiler.Config{
			ChannelService: channels,
			Channel:        calc,
		})).Error().To(MatchError(saveErr.Error()))
	})
})
