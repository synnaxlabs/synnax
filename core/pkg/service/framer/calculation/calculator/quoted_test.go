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
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/telem/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
)

var _ = Describe("Quoted channel names in a calculation", Ordered, func() {
	var (
		db         *gorp.DB
		channelSvc *channel.Service
		creator    calctestutil.Creator
	)

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: node.DB}))
		statusSvc := MustOpen(
			status.OpenService(ctx, status.ServiceConfig{DB: node.DB}),
		)
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  statusSvc,
		}))
		creator = calctestutil.Creator{DB: db, Channel: channelSvc}
	})

	// doubled computes the calculation expression over base, which takes the samples
	// 10, 20, and 30, and returns the samples that the calculation writes. The compiled
	// calculation imports no channel functions, since a calculator provides none.
	doubled := func(ctx SpecContext, base channel.Channel, expression string) telem.Series {
		GinkgoHelper()
		out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       UniqueChannelName(),
			DataType:   telem.Int64T,
			Expression: expression,
		}))
		mod := MustSucceed(compiler.Compile(ctx, db, compiler.Config{
			ChannelService: channelSvc,
			Channel:        out,
		}))
		r := wazero.NewRuntime(ctx)
		DeferCleanup(r.Close)
		compiled := MustSucceed(r.CompileModule(ctx, mod.WASM))
		DeferCleanup(compiled.Close)
		for _, f := range compiled.ImportedFunctions() {
			module, name, _ := f.Import()
			Expect(module).ToNot(Equal("channels"), name)
		}
		c := MustOpen(calculator.Open(ctx, calculator.Config{Module: mod}))
		fr := frame.NewUnary(base.Key(), telem.NewSeriesV[int64](10, 20, 30))
		written, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
		Expect(changed).To(BeTrue())
		return written.Get(out.Key()).Series[0]
	}
	newBase := func(ctx SpecContext) channel.Channel {
		GinkgoHelper()
		base := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Int64T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &base)
		})).To(Succeed())
		return base
	}

	DescribeTable("Should compute each sample of a calculation that reads a channel",
		func(ctx SpecContext, quoted bool) {
			base := newBase(ctx)
			ref := base.Name
			if quoted {
				ref = "'" + base.Name + "'"
			}
			Expect(doubled(ctx, base, "return "+ref+" * 2")).
				To(MatchSeriesDataV[int64](20, 40, 60))
		},
		Entry("by its bare name", false),
		Entry("by its quoted name", true),
	)

	It("Should compute a calculation that reads one channel bare and quoted", func(
		ctx SpecContext,
	) {
		base := newBase(ctx)
		Expect(doubled(ctx, base, "return "+base.Name+" + '"+base.Name+"'")).
			To(MatchSeriesDataV[int64](20, 40, 60))
	})
})
