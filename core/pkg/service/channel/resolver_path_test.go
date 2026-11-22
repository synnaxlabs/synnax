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
	"github.com/synnaxlabs/synnax/pkg/service/calculation/compiler"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/resolver"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Arc paths", Ordered, func() {
	var (
		db       *gorp.DB
		svc      *channel.Service
		stand    group.Key
		standDir string
	)
	create := func(ctx SpecContext, name string) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     name,
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    &stand,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	deleteChannel := func(ctx SpecContext, key channel.Key) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, key, false)
		})).To(Succeed())
	}

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		cfg, groups := serviceConfig(ctx, node, node.DB)
		svc = MustOpen(channel.OpenService(ctx, cfg))
		standDir = UniqueChannelName()
		g := group.Group{Name: standDir, Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groups.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		stand = g.Key
	})

	It("Should bind a channel made again under a path at the next lookup", func(
		ctx SpecContext,
	) {
		first := create(ctx, "pt_01")
		path := standDir + ".pt_01"
		Expect(resolver.New(svc, db).Resolve(ctx, path)).
			To(HaveField("ID", int(first.Key())))
		deleteChannel(ctx, first.Key())
		Expect(resolver.New(svc, db).Resolve(ctx, path)).Error().
			To(MatchError(ContainSubstring("channel '" + path + "'")))
		second := create(ctx, "pt_01")
		DeferCleanup(func(ctx SpecContext) { deleteChannel(ctx, second.Key()) })
		Expect(second.Key()).ToNot(Equal(first.Key()))
		Expect(resolver.New(svc, db).Resolve(ctx, path)).
			To(HaveField("ID", int(second.Key())))
	})

	It("Should read a channel by its bare path in a calculation", func(
		ctx SpecContext,
	) {
		read := create(ctx, "tc_01")
		DeferCleanup(func(ctx SpecContext) { deleteChannel(ctx, read.Key()) })
		result := MustSucceed(
			compiler.NewAnalyzer(resolver.New(svc, db)).AnalyzeExpression(
				ctx,
				channel.Channel{Name: UniqueChannelName(), Virtual: true},
				"return "+standDir+".tc_01 * 2",
				false,
			),
		)
		Expect(result.Deps).To(ConsistOf(read.Key()))
	})

	It("Should name a bare path that matches nothing", func(ctx SpecContext) {
		missing := standDir + ".missing"
		res, err := compiler.NewAnalyzer(resolver.New(svc, db)).AnalyzeExpression(
			ctx,
			channel.Channel{Name: UniqueChannelName(), Virtual: true},
			"return "+missing+" * 2",
			false,
		)
		Expect(err).
			To(MatchError(ContainSubstring(missing + " does not exist")))
		Expect(res.Unresolved).To(ConsistOf(missing))
	})
})
