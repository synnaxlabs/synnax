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
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Checker D-401 (1): the text pass of Declare's renames", func() {
	It("Should keep the reads of a calculation on the channel its text names after a "+
		"crash before the pass", func(
		ctx SpecContext,
	) {
		life := GinkgoT().Context()
		store := DeferClose(storagetestutil.OpenLayer(life))
		first := testutil.OpenNode(ctx, distribution.LayerConfig{Storage: store.Layer})
		keys := MustSucceed(first.Channel.Allocate(ctx, []distchannel.Channel{
			{Name: "sy_task_cmd", Leaseholder: node.KeyFree, Virtual: true},
			{Name: "reader", Leaseholder: node.KeyFree, Virtual: true},
		}))
		held := channelv1.Channel{
			Name:        keys[0].Name,
			Leaseholder: node.KeyFree,
			LocalKey:    keys[0].LocalKey,
			DataType:    telem.Float64T,
			Virtual:     true,
		}
		reader := channelv1.Channel{
			Name:        keys[1].Name,
			Leaseholder: node.KeyFree,
			LocalKey:    keys[1].LocalKey,
			DataType:    telem.Float64T,
			Virtual:     true,
			Expression:  "return sy_task_cmd",
		}
		Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](first.DB).
			Set(ctx, held, reader)).To(Succeed())
		host := first.Identity.HostKey.String()
		Expect(first.Close()).To(Succeed())
		for _, counter := range []string{"leasedCounter", "counter.free"} {
			record := host + ".distribution.channel." + counter + ".firstOpen"
			Expect(store.KV.Delete(ctx, []byte(record))).To(Succeed())
		}
		type core struct {
			node  testutil.Node
			layer *service.Layer
		}
		start := func() core {
			GinkgoHelper()
			n := testutil.OpenNode(life, distribution.LayerConfig{Storage: store.Layer})
			return core{node: n, layer: MustSucceed(svctestutil.OpenLayer(ctx, n))}
		}
		stop := func(c core) {
			GinkgoHelper()
			Expect(c.layer.Close()).To(Succeed())
			Expect(c.node.Close()).To(Succeed())
		}
		reads := func(c core) []uint32 {
			GinkgoHelper()
			calc := calculationOf(ctx, c.node, c.layer.Calculation, reader.Key())
			out := make([]uint32, len(calc.Reads))
			for i, k := range calc.Reads {
				out[i] = uint32(k)
			}
			return out
		}

		// The first start declares the built-in at sy_task_cmd, moves held aside, and
		// runs the text pass.
		one := start()
		Eventually(func() []uint32 { return reads(one) }).
			Should(ConsistOf(uint32(held.Key())))
		// A crash after Declare's commit and before the pass leaves the old text and
		// the stored rename. Put both back as that crash leaves them.
		calc := calculationOf(ctx, one.node, one.layer.Calculation, reader.Key())
		calc.Expression = "return sy_task_cmd"
		Expect(gorp.WrapWriter[calculation.Key, calculation.Calculation](one.node.DB).
			Set(ctx, calc)).To(Succeed())
		renameRecord := "sy_channel_text_rename/" +
			strconv.FormatUint(uint64(held.Key()), 10)
		Expect(store.KV.Set(ctx, []byte(renameRecord), []byte("sy_task_cmd"))).
			To(Succeed())
		stop(one)

		// The next start reruns the pass, so the text and the reads name held again.
		two := start()
		DeferCleanup(stop, two)
		Expect(calculationOf(ctx, two.node, two.layer.Calculation, reader.Key()).
			Expression).To(Equal("return sy_task_cmd_2"))
		Eventually(func() []uint32 { return reads(two) }).
			Should(ConsistOf(uint32(held.Key())),
				"the text names sy_task_cmd_2, which is held, so the reads name held")
	})

	It("Should keep an editor's later edit where the editor typed it", func(
		ctx SpecContext,
	) {
		const before = "x := sy_task_cmd * 2"
		var key arc.Key
		n := openUpgraded(ctx, func(n testutil.Node) {
			keys := MustSucceed(n.Channel.Allocate(ctx, []distchannel.Channel{
				{Name: "sy_task_cmd", Leaseholder: node.KeyFree, Virtual: true},
			}))
			Expect(gorp.WrapWriter[channelv1.Key, channelv1.Channel](n.DB).
				Set(ctx, channelv1.Channel{
					Name:        keys[0].Name,
					Leaseholder: node.KeyFree,
					LocalKey:    keys[0].LocalKey,
					DataType:    telem.Float64T,
					Virtual:     true,
				})).To(Succeed())
			key = storeArcs(ctx, n, []string{before}, nil)[0]
		})
		// An editor opened the Arc before the start and types "0" after the "*".
		editor := crdt.New(7)
		doc := text.Create(before)
		editor.Load(doc.Inserts, doc.Deletes)
		appended := editor.Insert(strings.Index(before, "*")+1, "0")
		MustOpen(svctestutil.OpenLayer(ctx, n))
		Expect(sourceOf(ctx, n, key)).To(Equal("x := sy_task_cmd_2 * 2"))
		var stored arc.Arc
		Expect(gorp.NewRetrieve[arc.Key, arc.Arc]().
			Where(gorp.MatchKeys[arc.Key, arc.Arc](key)).
			Entry(&stored).
			Exec(ctx, n.DB)).To(Succeed())
		actions := make([]arc.Action, len(appended))
		for i, ins := range appended {
			actions[i] = arc.NewInsertCharAction(arc.InsertCharPayload{
				ID:     ins.ID,
				Origin: ins.Origin,
				Side:   ins.Side,
				Char:   ins.Char,
			})
		}
		edited := MustSucceed(arc.Reduce(stored, actions...))
		src, ok := edited.Base().Source.Variant.(arc.TextSource)
		Expect(ok).To(BeTrue())
		Expect(src.Text.Materialize().Raw).To(Equal("x := sy_task_cmd_2 *0 2"),
			"the editor typed after the \"*\" that it held")
	})
})
