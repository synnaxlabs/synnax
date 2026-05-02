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
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a Schematic", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test", Authority: 1}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(s.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Update", func() {
		It("Should rename a Schematic", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test", Authority: 1}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, s.Key, "test2")).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().Where(schematic.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("SetData", func() {
		It("Should replace every body field on the Schematic while preserving Key and Name", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test", Authority: 1}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).SetData(ctx, s.Key, schematic.Schematic{
				Name:      "ignored-name",
				Authority: 5,
				Nodes:     []schematic.Node{{Key: "n1"}},
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().
				Where(schematic.MatchKeys(s.Key)).
				Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test"))
			Expect(res.Authority).To(BeEquivalentTo(5))
			Expect(res.Nodes).To(HaveLen(1))
		})
		It("Should preserve the Snapshot flag against caller overrides", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test", Authority: 1}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).SetData(ctx, s.Key, schematic.Schematic{
				Authority: 5,
				Snapshot:  true,
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().
				Where(schematic.MatchKeys(s.Key)).
				Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Snapshot).To(BeFalse())
		})
	})

	Describe("Dispatch", func() {
		It("Should apply a single SetNodePosition action", func(ctx SpecContext) {
			s := schematic.Schematic{
				Name:      "test",
				Authority: 1,
				Nodes: []schematic.Node{
					{Key: "n1", Position: spatial.XY{X: 0, Y: 0}},
				},
			}
			MustSucceed(uuid.Nil, svc.NewWriter(tx).Create(ctx, ws.Key, &s))
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "session-1", []schematic.Action{
				schematic.NewSetNodePositionAction(schematic.SetNodePosition{
					Key:      "n1",
					Position: spatial.XY{X: 100, Y: 200},
				}),
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().Where(schematic.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Nodes).To(HaveLen(1))
			Expect(res.Nodes[0].Position).To(Equal(spatial.XY{X: 100, Y: 200}))
		})
		It("Should apply a sequence of mixed actions atomically", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test", Authority: 1}
			MustSucceed(uuid.Nil, svc.NewWriter(tx).Create(ctx, ws.Key, &s))
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "session-1", []schematic.Action{
				schematic.NewAddNodeAction(schematic.AddNode{
					Node: schematic.Node{Key: "n1", Position: spatial.XY{X: 1, Y: 2}},
				}),
				schematic.NewAddNodeAction(schematic.AddNode{
					Node: schematic.Node{Key: "n2", Position: spatial.XY{X: 3, Y: 4}},
				}),
				schematic.NewSetEdgeAction(schematic.SetEdge{
					Edge: schematic.Edge{
						Key:    "e1",
						Source: schematic.Handle{Node: "n1"},
						Target: schematic.Handle{Node: "n2"},
					},
				}),
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().Where(schematic.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Nodes).To(HaveLen(2))
			Expect(res.Edges).To(HaveLen(1))
		})
		It("Should reject Dispatch on a snapshot schematic", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test", Authority: 1}
			MustSucceed(uuid.Nil, svc.NewWriter(tx).Create(ctx, ws.Key, &s))
			var snap schematic.Schematic
			Expect(svc.NewWriter(tx).Copy(ctx, s.Key, "snap", true, &snap)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, snap.Key, "session-1", []schematic.Action{
				schematic.NewRemoveNodeAction(schematic.RemoveNode{Key: "n1"}),
			})).To(MatchError(validate.ErrValidation))
		})
		It("Should be a no-op when actions reference non-existent keys", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test", Authority: 1}
			MustSucceed(uuid.Nil, svc.NewWriter(tx).Create(ctx, ws.Key, &s))
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "session-1", []schematic.Action{
				schematic.NewRemoveNodeAction(schematic.RemoveNode{Key: "ghost"}),
				schematic.NewRemoveEdgeAction(schematic.RemoveEdge{Key: "ghost-edge"}),
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().Where(schematic.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Nodes).To(BeEmpty())
			Expect(res.Edges).To(BeEmpty())
		})
	})

	Describe("Copy", func() {
		It("Should copy a Schematic with a new name under the same workspace", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test", Authority: 1}
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
		It("Should copy a Schematic into a snapshot that cannot be modified", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "test", Authority: 1}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			var cpy schematic.Schematic
			Expect(svc.NewWriter(tx).Copy(ctx, s.Key, "test2", true, &cpy)).To(Succeed())
			Expect(svc.NewWriter(tx).SetData(ctx, cpy.Key, schematic.Schematic{Authority: 2})).To(MatchError(validate.ErrValidation))
		})
	})
})
