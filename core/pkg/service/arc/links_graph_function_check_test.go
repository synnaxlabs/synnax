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
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Channel links of graph functions check", func() {
	var (
		key  channel.Key
		path string
	)
	BeforeEach(func(ctx SpecContext) {
		g := group.Group{
			Name:  "graph_fn_" + uuid.New().String()[:8],
			Scope: resource.TypeChannel,
		}
		ch := channel.Channel{
			Name:     "sensor_" + uuid.New().String()[:8],
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
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(Succeed())
		})
		key, path = ch.Key(), g.Name+"."+ch.Name
	})

	// exportInputs exports a graph Arc with one node that sets param of the graph
	// function fn to key, and returns the node's inputs in the file.
	exportInputs := func(ctx SpecContext, fn, param string) map[string]any {
		GinkgoHelper()
		a := newAutomation(arc.Base{
			Name: "graph_fn_" + uuid.New().String()[:8],
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Functions: []ir.Function{{
					Key: fn,
					Inputs: types.Params{
						{Name: param, Type: types.Chan(types.F32())},
					},
				}},
				Nodes: graph.Nodes{{Key: "node"}},
				Inputs: map[string]msgpack.EncodedJSON{
					"node": {"type": fn, param: key},
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
		env := MustSucceed(svc.Export(ctx, db, arc.ResourceID(a.Base().Key)))
		file := MustSucceed(json.Marshal(env.WithEncodeOptions(
			channelSvc.LinkWriter(ctx, db, func(resource.ID) bool { return true }),
		)))
		var body struct {
			Source struct {
				Graph struct {
					Inputs map[string]map[string]any `json:"inputs"`
				} `json:"graph"`
			} `json:"source"`
		}
		Expect(json.Unmarshal(file, &body)).To(Succeed())
		return body.Source.Graph.Inputs["node"]
	}

	It("Should write the channel input of a graph function as its path", func(
		ctx SpecContext,
	) {
		Expect(exportInputs(ctx, "watch", "source")).
			To(HaveKeyWithValue("source", path))
	})

	It("Should write the path of a graph function that shadows a flow function", func(
		ctx SpecContext,
	) {
		Expect(exportInputs(ctx, "on", "source")).To(HaveKeyWithValue("source", path))
	})
})
