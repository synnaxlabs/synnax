// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a Schematic", func(ctx SpecContext) {
			s := schematic.Schematic{
				Name:  "test",
				Props: map[string]msgpack.EncodedJSON{"key": map[string]any{"data": "data_two"}},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(s.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Update", func() {
		It("Should rename a Schematic", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test", Props: map[string]msgpack.EncodedJSON{
				"key": map[string]any{"data": "data_value"},
			}}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, s.Key, "test2")).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().WhereKeys(s.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("Dispatch", func() {
		It("Should apply a SetNodePosition action", func(ctx SpecContext) {
			s := schematic.Schematic{
				Name: "dispatch-test",
				Nodes: []schematic.Node{
					{Key: "n1", Position: spatial.XY{X: 0, Y: 0}},
				},
				Props: map[string]msgpack.EncodedJSON{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "test-session", []schematic.Action{
				schematic.NewSetNodePositionAction(schematic.SetNodePosition{
					Key:      "n1",
					Position: spatial.XY{X: 100, Y: 200},
				}),
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().WhereKeys(s.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Nodes).To(HaveLen(1))
			Expect(res.Nodes[0].Position).To(Equal(spatial.XY{X: 100, Y: 200}))
		})
		It("Should apply multiple actions in sequence", func(ctx SpecContext) {
			s := schematic.Schematic{
				Name:  "dispatch-multi",
				Props: map[string]msgpack.EncodedJSON{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "test-session", []schematic.Action{
				schematic.NewAddNodeAction(schematic.AddNode{
					Node: schematic.Node{Key: "a", Position: spatial.XY{X: 10, Y: 20}},
				}),
				schematic.NewAddNodeAction(schematic.AddNode{
					Node: schematic.Node{Key: "b", Position: spatial.XY{X: 30, Y: 40}},
				}),
				schematic.NewRemoveNodeAction(schematic.RemoveNode{Key: "a"}),
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().WhereKeys(s.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Nodes).To(HaveLen(1))
			Expect(res.Nodes[0].Key).To(Equal("b"))
		})
		It("Should apply SetEdge and RemoveEdge actions", func(ctx SpecContext) {
			s := schematic.Schematic{
				Name: "dispatch-edges",
				Edges: []schematic.Edge{
					{
						Key:    "e1",
						Source: schematic.Handle{Node: "n1"},
						Target: schematic.Handle{Node: "n2"},
					},
				},
				Props: map[string]msgpack.EncodedJSON{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "test-session", []schematic.Action{
				schematic.NewSetEdgeAction(schematic.SetEdge{
					Edge: schematic.Edge{
						Key:    "e1",
						Source: schematic.Handle{Node: "n1"},
						Target: schematic.Handle{Node: "n3"},
					},
				}),
				schematic.NewSetEdgeAction(schematic.SetEdge{
					Edge: schematic.Edge{
						Key:    "e2",
						Source: schematic.Handle{Node: "n2"},
						Target: schematic.Handle{Node: "n3"},
					},
				}),
				schematic.NewRemoveEdgeAction(schematic.RemoveEdge{Key: "e1"}),
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().WhereKeys(s.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Edges).To(HaveLen(1))
			Expect(res.Edges[0].Key).To(Equal("e2"))
		})
		It("Should return an error when dispatching on a nonexistent key", func(ctx SpecContext) {
			err := svc.NewWriter(tx).Dispatch(ctx, uuid.New(), "test-session", []schematic.Action{
				schematic.NewRemoveNodeAction(schematic.RemoveNode{Key: "n1"}),
			})
			Expect(err).To(HaveOccurred())
		})
	})
	Describe("Copy", func() {
		It("Should copy a Schematic with a new name under the same workspace", func(ctx SpecContext) {
			s := schematic.Schematic{
				Name:  "test",
				Props: map[string]msgpack.EncodedJSON{"key": map[string]any{"data": "data_two"}},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			var cpy schematic.Schematic
			Expect(svc.NewWriter(tx).Copy(ctx, s.Key, "test2", false, &cpy)).To(Succeed())
			Expect(cpy.Key).ToNot(Equal(s.Key))
			Expect(cpy.Name).To(Equal("test2"))
			var res []ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(ws.OntologyID()).TraverseTo(ontology.ChildrenTraverser).Entries(&res).Exec(ctx, tx)).To(Succeed())
			keys := lo.Map(res, func(r ontology.Resource, _ int) string { return r.ID.Key })
			Expect(keys).To(ContainElement(cpy.Key.String()))
		})
	})
})
