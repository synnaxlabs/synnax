// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Text that names a channel a built-in displaces", func() {
	It("Should keep naming the channel after a start that declares the built-in", func(
		ctx SpecContext,
	) {
		var (
			held, reader channelv1.Channel
			arcs         []arc.Key
		)
		n := openUpgraded(ctx, func(n testutil.Node) {
			keys := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{
				{Name: "sy_task_cmd", Leaseholder: node.KeyFree, Virtual: true},
				{Name: "reader", Leaseholder: node.KeyFree, Virtual: true},
			}))
			// The built-in holds JSON, so a float channel at its name moves aside.
			held = channelv1.Channel{
				Name:        keys[0].Name,
				Leaseholder: node.KeyFree,
				LocalKey:    keys[0].LocalKey,
				DataType:    telem.Float64T,
				Virtual:     true,
			}
			reader = channelv1.Channel{
				Name:        keys[1].Name,
				Leaseholder: node.KeyFree,
				LocalKey:    keys[1].LocalKey,
				DataType:    telem.Float64T,
				Virtual:     true,
				Expression:  "return sy_task_cmd * 2",
			}
			Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
				Set(ctx, held, reader)).To(Succeed())
			arcs = storeArcs(
				ctx,
				n,
				[]string{"x := sy_task_cmd * 2"},
				[]string{"{ return sy_task_cmd * 3 }"},
			)
		})
		l := MustOpen(svctestutil.OpenLayer(ctx, n))
		var builtIn channel.Channel
		Expect(l.Channel.NewRetrieve().
			Where(channel.MatchNames("sy_task_cmd")).
			Entry(&builtIn).
			Exec(ctx, n.DB)).To(Succeed())
		expectRehomed(ctx, n, l, held.Key(), builtIn)
		Expect(calculationOf(ctx, n, l.Calculation, reader.Key()).Expression).
			To(Equal("return sy_task_cmd_2 * 2"))
		Expect(sourceOf(ctx, n, arcs[0])).To(Equal("x := sy_task_cmd_2 * 2"))
		Expect(sourceOf(ctx, n, arcs[1])).To(Equal("{ return sy_task_cmd_2 * 3 }"))
		Eventually(func(g Gomega) {
			g.Expect(calculationOf(ctx, n, l.Calculation, reader.Key()).Reads).
				To(ConsistOf(held.Key()))
		}).Should(Succeed())
	})
})
