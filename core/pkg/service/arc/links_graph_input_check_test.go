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
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Channel links of graph inputs that hold no key check", func() {
	var key channel.Key
	BeforeEach(func(ctx SpecContext) {
		suffix := uuid.New().String()[:8]
		g := group.Group{Name: "graph_input_" + suffix, Scope: resource.TypeChannel}
		ch := channel.Channel{
			Name:     "graph_input_pt_" + suffix,
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := groupSvc.NewWriter(tx).Create(ctx, &g); err != nil {
				return err
			}
			ch.Group = new(g.Key)
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		key = ch.Key()
	})

	DescribeTable("Should export a graph Arc whose channel input is no key", func(
		ctx SpecContext,
		stored func() any,
	) {
		a := newAutomation(arc.Base{
			Name: "graph_input_" + uuid.New().String()[:8],
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Nodes: graph.Nodes{{Key: "n"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"n": {"type": "on", "channel": stored()},
				},
			}}},
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &a)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, a.Base().Key)
			})).To(Succeed())
		})
		Expect(svc.Export(ctx, db, arc.ResourceID(a.Base().Key))).Error().
			ToNot(HaveOccurred())
	},
		Entry("a key as text", func() any { return fmt.Sprint(key) }),
		Entry("a channel name", func() any { return "valve_cmd" }),
		Entry("null", func() any { return nil }),
	)
})
