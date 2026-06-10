// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/validate"
)

// scopedAction is a short local alias for the schematic's action envelope so
// the test files don't have to spell out the generic parameters at every use.
type scopedAction = actions.Scoped[schematic.Key, schematic.Action]

// createSchematic persists a fresh schematic owned by the suite project and
// returns it with its key populated. Writes commit immediately (nil tx) so
// access-control reads can observe the new ontology resource.
func createSchematic(ctx context.Context, name string) schematic.Schematic {
	s := schematic.Schematic{Name: name}
	Expect(schematicSvc.NewWriter(nil).Create(ctx, ws.Key, &s)).To(Succeed())
	return s
}

var _ = Describe("api.Service.Dispatch", func() {
	Describe("access control", func() {
		It("Should reject the request with access.ErrDenied when the subject has no policy", func(ctx SpecContext) {
			s := createSchematic(ctx, "no-policy")
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
				Key:         s.Key,
				DispatchKey: "sess-1",
				Actions: []schematic.Action{schematic.NewRemoveNodeAction(schematic.RemoveNodePayload{
					Key: "n1",
				})},
			})).Error().To(MatchError(access.ErrDenied))
		})

		It("Should accept the request when the subject's policy permits update on the target schematic", func(ctx SpecContext) {
			s := createSchematic(ctx, "with-policy")
			grantUpdateOn(ctx, user.OntologyID(author.Key), schematic.OntologyID(s.Key))
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
				Key:         s.Key,
				DispatchKey: "sess-1",
				Actions: []schematic.Action{schematic.NewSetNodeAction(schematic.SetNodePayload{
					Node: schematic.Node{Key: "n1", Position: spatial.XY{X: 1, Y: 2}},
				})},
			})).Error().To(Succeed())
			var res schematic.Schematic
			Expect(schematicSvc.NewRetrieve().
				Where(schematic.MatchKeys(s.Key)).
				Entry(&res).Exec(ctx, nil)).To(Succeed())
			Expect(res.Nodes).To(HaveLen(1))
			Expect(res.Nodes[0].Position).To(Equal(spatial.XY{X: 1, Y: 2}))
		})

		It("Should reject when the subject's policy targets a different schematic", func(ctx SpecContext) {
			a := createSchematic(ctx, "policy-target")
			b := createSchematic(ctx, "no-policy-target")
			grantUpdateOn(ctx, user.OntologyID(author.Key), schematic.OntologyID(a.Key))
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
				Key:         b.Key,
				DispatchKey: "sess-1",
				Actions: []schematic.Action{schematic.NewRemoveNodeAction(schematic.RemoveNodePayload{
					Key: "n1",
				})},
			})).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("delegation to Writer.Dispatch", func() {
		It("Should apply a multi-action sequence to the target schematic", func(ctx SpecContext) {
			s := createSchematic(ctx, "multi-action")
			grantUpdateOn(ctx, user.OntologyID(author.Key), schematic.OntologyID(s.Key))
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
				Key:         s.Key,
				DispatchKey: "sess-1",
				Actions: []schematic.Action{
					schematic.NewSetNodeAction(schematic.SetNodePayload{
						Node: schematic.Node{Key: "n1", Position: spatial.XY{X: 1, Y: 2}},
					}),
					schematic.NewSetNodeAction(schematic.SetNodePayload{
						Node: schematic.Node{Key: "n2", Position: spatial.XY{X: 3, Y: 4}},
					}),
					schematic.NewAddEdgeAction(schematic.AddEdgePayload{Edge: schematic.Edge{
						Key:    "e1",
						Source: schematic.Handle{Node: "n1", Param: "out"},
						Target: schematic.Handle{Node: "n2", Param: "in"},
					}}),
				},
			})).Error().To(Succeed())
			var res schematic.Schematic
			Expect(schematicSvc.NewRetrieve().
				Where(schematic.MatchKeys(s.Key)).
				Entry(&res).Exec(ctx, nil)).To(Succeed())
			Expect(res.Nodes).To(HaveLen(2))
			Expect(res.Edges).To(HaveLen(1))
		})

		It("Should bubble up validate.ErrValidation when dispatching to a snapshot", func(ctx SpecContext) {
			s := createSchematic(ctx, "snap-source")
			var snap schematic.Schematic
			Expect(schematicSvc.NewWriter(nil).Copy(ctx, s.Key, "snap", true, &snap)).To(Succeed())
			grantUpdateOn(ctx, user.OntologyID(author.Key), schematic.OntologyID(snap.Key))
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
				Key:         snap.Key,
				DispatchKey: "sess-1",
				Actions: []schematic.Action{schematic.NewRemoveNodeAction(schematic.RemoveNodePayload{
					Key: "n1",
				})},
			})).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should bubble up query.ErrNotFound when the target schematic does not exist", func(ctx SpecContext) {
			missing := uuid.New()
			grantUpdateOn(ctx, user.OntologyID(author.Key), schematic.OntologyID(missing))
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
				Key:         missing,
				DispatchKey: "sess-1",
				Actions: []schematic.Action{schematic.NewRemoveNodeAction(schematic.RemoveNodePayload{
					Key: "n1",
				})},
			})).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("subject identity propagation", func() {
		It("Should pass the DispatchKey verbatim into the action observer", func(ctx SpecContext) {
			s := createSchematic(ctx, "session-propagation")
			grantUpdateOn(ctx, user.OntologyID(author.Key), schematic.OntologyID(s.Key))
			seen := make(chan scopedAction, 1)
			disconnect := schematicSvc.OnAction(func(_ context.Context, sa scopedAction) {
				seen <- sa
			})
			DeferCleanup(disconnect)
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
				Key:         s.Key,
				DispatchKey: "session-marker-xyz",
				Actions: []schematic.Action{schematic.NewSetNodeAction(schematic.SetNodePayload{
					Node: schematic.Node{Key: "n1"},
				})},
			})).Error().To(Succeed())
			var got scopedAction
			Eventually(seen).Should(Receive(&got))
			Expect(got.Key).To(Equal(s.Key))
			Expect(got.DispatchKey).To(Equal("session-marker-xyz"))
			Expect(got.Seq).To(BeNumerically(">", uint64(0)))
			Expect(got.Actions).To(HaveLen(1))
		})
	})
})
