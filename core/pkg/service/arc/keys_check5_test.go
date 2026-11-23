// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Compile keys check 5", Ordered, func() {
	var in, ctl, out channel.Channel
	createGraph := func(ctx SpecContext, name string, source, sink any) arc.Key {
		GinkgoHelper()
		a := newAutomation(arc.Base{
			Name: name,
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Nodes: graph.Nodes{{Key: "source"}, {Key: "sink"}},
				Edges: graph.Edges{{
					Key:    "edge",
					Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
				}},
				Inputs: map[string]msgpack.EncodedJSON{
					"source": {"type": "on", "channel": source},
					"sink":   {"type": "write", "channel": sink},
				},
			}}},
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &a)
		})).To(Succeed())
		key := a.Base().Key
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, key)
			})).To(Succeed())
		})
		return key
	}
	digits := func(ch channel.Channel) string {
		return strconv.FormatUint(uint64(ch.Key()), 10)
	}

	BeforeAll(func(ctx SpecContext) {
		chs := []*channel.Channel{
			{Name: "kc5_in", Virtual: true, DataType: telem.Float64T},
			{Name: "kc5_ctl", Virtual: true, DataType: telem.Float64T},
			{Name: "kc5_out", Virtual: true, DataType: telem.Float64T},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := channelSvc.NewWriter(tx)
			for _, ch := range chs {
				if err := w.Create(ctx, ch); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
		in, ctl, out = *chs[0], *chs[1], *chs[2]
	})

	It("Should name no use of the channel that a graph node holds as digit text", func(
		ctx SpecContext,
	) {
		key := createGraph(ctx, "kc5_refs", digits(in), float64(out.Key()))
		held := MustSucceed(channelSvc.Uses(ctx, db, []channel.Key{in.Key()}))
		holders := make([]gorp.Target, len(held[0]))
		for i, u := range held[0] {
			holders[i] = u.Holder
		}
		Expect(holders).
			ToNot(ContainElement(gorp.NewTarget(string(resource.TypeArc), key)))
	})

	It("Should leave a node's digit text in place when the upgrade clears the "+
		"channels the Core never issued", func(ctx SpecContext) {
		key := createGraph(ctx, "kc5_clear", digits(in), float64(ctl.Key()))
		unissued := map[string]bool{
			gorp.FormatKey(in.Key()):  true,
			gorp.FormatKey(ctl.Key()): true,
		}
		clear := svc.DanglingMigration("kc5_clear", func(t gorp.Target) (bool, error) {
			return !unissued[t.Key], nil
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return clear.Run(ctx, tx, alamos.Instrumentation{})
		})).To(Succeed())
		var a arc.Arc
		Expect(svc.NewRetrieve().Where(arc.MatchKeys(key)).Entry(&a).Exec(ctx, db)).
			To(Succeed())
		src, ok := a.Base().Source.Variant.(arc.GraphSource)
		Expect(ok).To(BeTrue())
		Expect(src.Graph.Inputs["source"]["channel"]).To(Equal(digits(in)))
		// The control: the clear removes a number input of an unissued channel.
		Expect(src.Graph.Inputs["sink"]).ToNot(HaveKey("channel"))
	})
})
