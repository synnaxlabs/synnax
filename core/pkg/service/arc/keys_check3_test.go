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

var _ = Describe("Compile keys check 3", func() {
	It("Should name a graph Arc as a use of the channel that a node holds as digit "+
		"text", func(ctx SpecContext) {
		in := channel.Channel{Name: "kc3_in", Virtual: true, DataType: telem.Float64T}
		out := channel.Channel{Name: "kc3_out", Virtual: true, DataType: telem.Float64T}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := channelSvc.NewWriter(tx)
			if err := w.Create(ctx, &in); err != nil {
				return err
			}
			return w.Create(ctx, &out)
		})).To(Succeed())
		a := newAutomation(arc.Base{
			Name: "kc3_digits",
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Nodes: graph.Nodes{{Key: "source"}, {Key: "sink"}},
				Edges: graph.Edges{{
					Key:    "edge",
					Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
				}},
				Inputs: map[string]msgpack.EncodedJSON{
					"source": {
						"type":    "on",
						"channel": strconv.FormatUint(uint64(in.Key()), 10),
					},
					"sink": {"type": "write", "channel": float64(out.Key())},
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
		held := MustSucceed(channelSvc.Uses(ctx, db, []channel.Key{in.Key()}))
		holders := make([]gorp.Target, len(held[0]))
		for i, u := range held[0] {
			holders[i] = u.Holder
		}
		Expect(
			holders,
		).To(ContainElement(gorp.NewTarget(string(resource.TypeArc), key)))
	})
})
