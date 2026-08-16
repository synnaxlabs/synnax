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
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

// scopedAction is a short local alias for the Arc action envelope.
type scopedAction = actions.Scoped[arc.Key, arc.Action]

// createArc persists a fresh Arc and returns it with its key populated. Writes
// commit immediately (nil tx) so access-control reads can observe the new
// ontology resource.
func createArc(ctx context.Context, name string) arc.Arc {
	a := arc.Arc{Name: name, Mode: arc.ModeGraph}
	Expect(arcSvc.NewWriter(nil).Create(ctx, &a)).To(Succeed())
	return a
}

var _ = Describe("Service", func() {
	Describe("Retrieve", func() {
		It(
			"Should omit missing keys from a multi-key retrieve instead of failing",
			func(ctx SpecContext) {
				a := createArc(ctx, "partial-survivor")
				grantOn(
					ctx,
					author.OntologyID(),
					access.ActionRetrieve,
					arc.OntologyID(a.Key),
				)
				res := MustSucceed(
					apiSvc.Retrieve(authedCtx(ctx, author), apiarc.RetrieveRequest{
						Keys:                []arc.Key{a.Key, uuid.New()},
						IgnoreNotFoundError: true,
					}),
				)
				Expect(res.Arcs).To(HaveLen(1))
				Expect(res.Arcs[0].Key).To(Equal(a.Key))
			},
		)

		It(
			"Should return an empty result when every requested key is missing",
			func(ctx SpecContext) {
				res := MustSucceed(
					apiSvc.Retrieve(authedCtx(ctx, author), apiarc.RetrieveRequest{
						Keys:                []arc.Key{uuid.New()},
						IgnoreNotFoundError: true,
					}),
				)
				Expect(res.Arcs).To(BeEmpty())
			},
		)
	})

	Describe("Create", func() {
		It(
			"Should materialize the stored document, not echo a client-supplied Raw",
			func(ctx SpecContext) {
				grantOn(
					ctx,
					author.OntologyID(),
					access.ActionCreate,
					ontology.ID{Type: ontology.ResourceTypeArc},
				)
				a := apiarc.Arc{Name: "lying-raw", Mode: arc.ModeText}
				a.Text.Doc = text.Create("a -> b")
				a.Text.Raw = "a -> c"
				res := MustSucceed(
					apiSvc.Create(authedCtx(ctx, author), db, apiarc.CreateRequest{
						Arcs: []apiarc.Arc{a},
					}),
				)
				Expect(res.Arcs[0].Text.Raw).To(Equal("a -> b"))
			},
		)
	})

	Describe("Dispatch", func() {
		Describe("access control", func() {
			It(
				"Should reject the request with access.ErrDenied when the subject has no policy",
				func(ctx SpecContext) {
					a := createArc(ctx, "no-policy")
					Expect(apiSvc.Dispatch(
						authedCtx(ctx, author), db, apiarc.DispatchRequest{
							Key:         a.Key,
							DispatchKey: "sess-1",
							Actions: []arc.Action{
								arc.NewRemoveNodeAction(arc.RemoveNodePayload{
									Key: "n1",
								}),
							},
						},
					)).Error().To(MatchError(access.ErrDenied))
				},
			)

			It(
				"Should accept the request when the subject's policy permits update on the target Arc",
				func(ctx SpecContext) {
					a := createArc(ctx, "with-policy")
					grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
					Expect(apiSvc.Dispatch(
						authedCtx(ctx, author), db, apiarc.DispatchRequest{
							Key:         a.Key,
							DispatchKey: "sess-1",
							Actions: []arc.Action{
								arc.NewSetNodeAction(arc.SetNodePayload{
									Node: graph.Node{
										Key:      "n1",
										Position: spatial.XY{X: 1, Y: 2},
									},
								}),
							},
						},
					)).Error().To(Succeed())
					var res arc.Arc
					Expect(arcSvc.NewRetrieve().
						Where(arc.MatchKeys(a.Key)).
						Entry(&res).Exec(ctx, nil)).To(Succeed())
					Expect(res.Graph.Nodes).To(HaveLen(1))
					Expect(
						res.Graph.Nodes[0].Position,
					).To(Equal(spatial.XY{X: 1, Y: 2}))
				},
			)

			It(
				"Should reject when the subject's policy targets a different Arc",
				func(ctx SpecContext) {
					a := createArc(ctx, "policy-target")
					b := createArc(ctx, "no-policy-target")
					grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
					Expect(apiSvc.Dispatch(
						authedCtx(ctx, author), db, apiarc.DispatchRequest{
							Key:         b.Key,
							DispatchKey: "sess-1",
							Actions: []arc.Action{
								arc.NewRemoveNodeAction(arc.RemoveNodePayload{
									Key: "n1",
								}),
							},
						},
					)).Error().To(MatchError(access.ErrDenied))
				},
			)
		})

		Describe("delegation to Writer.Dispatch", func() {
			It(
				"Should apply a multi-action sequence to the target Arc",
				func(ctx SpecContext) {
					a := createArc(ctx, "multi-action")
					grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
					Expect(apiSvc.Dispatch(
						authedCtx(ctx, author), db, apiarc.DispatchRequest{
							Key:         a.Key,
							DispatchKey: "sess-1",
							Actions: []arc.Action{
								arc.NewSetNodeAction(
									arc.SetNodePayload{Node: graph.Node{Key: "n1"}},
								),
								arc.NewSetNodeAction(
									arc.SetNodePayload{Node: graph.Node{Key: "n2"}},
								),
								arc.NewRenameAction(
									arc.RenamePayload{Name: "renamed-multi"},
								),
							},
						},
					)).Error().To(Succeed())
					var res arc.Arc
					Expect(arcSvc.NewRetrieve().
						Where(arc.MatchKeys(a.Key)).
						Entry(&res).Exec(ctx, nil)).To(Succeed())
					Expect(res.Graph.Nodes).To(HaveLen(2))
					Expect(res.Name).To(Equal("renamed-multi"))
				},
			)

			It(
				"Should bubble up query.ErrNotFound when the target Arc does not exist",
				func(ctx SpecContext) {
					missing := uuid.New()
					grantUpdateOn(ctx, author.OntologyID(), arc.OntologyID(missing))
					Expect(apiSvc.Dispatch(
						authedCtx(ctx, author), db, apiarc.DispatchRequest{
							Key:         missing,
							DispatchKey: "sess-1",
							Actions: []arc.Action{
								arc.NewRenameAction(arc.RenamePayload{
									Name: "ghost",
								}),
							},
						},
					)).Error().To(MatchError(query.ErrNotFound))
				},
			)
		})

		Describe("subject identity propagation", func() {
			It(
				"Should pass the DispatchKey verbatim into the action observer",
				func(ctx SpecContext) {
					a := createArc(ctx, "session-propagation")
					grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
					seen := make(chan scopedAction, 1)
					DeferCleanup(
						arcSvc.OnAction(func(_ context.Context, sa scopedAction) {
							seen <- sa
						}),
					)
					Expect(apiSvc.Dispatch(
						authedCtx(ctx, author), db, apiarc.DispatchRequest{
							Key:         a.Key,
							DispatchKey: "session-marker-xyz",
							Actions: []arc.Action{
								arc.NewSetNodeAction(arc.SetNodePayload{
									Node: graph.Node{Key: "n1"},
								}),
							},
						},
					)).Error().To(Succeed())
					var got scopedAction
					Eventually(seen).Should(Receive(&got))
					Expect(got.Key).To(Equal(a.Key))
					Expect(got.DispatchKey).To(Equal("session-marker-xyz"))
					Expect(got.Seq).To(BeNumerically(">", uint64(0)))
					Expect(got.Actions).To(HaveLen(1))
				},
			)
		})
	})

	Describe("SetRack", func() {
		grantSetRack := func(ctx SpecContext, a arc.Arc) {
			grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
			grantOn(
				ctx,
				author.OntologyID(),
				access.ActionCreate,
				ontology.ID{Type: ontology.ResourceTypeTask},
			)
		}

		It(
			"Should reject the request when the subject has no policy",
			func(ctx SpecContext) {
				a := createArc(ctx, "set-rack-no-policy")
				Expect(apiSvc.SetRack(authedCtx(ctx, author), db, apiarc.SetRackRequest{
					Key:  a.Key,
					Rack: testRack.Key,
				})).Error().To(MatchError(access.ErrDenied))
			},
		)

		It(
			"Should reject when the subject may update the arc but not create tasks",
			func(ctx SpecContext) {
				a := createArc(ctx, "set-rack-no-task-policy")
				grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
				Expect(apiSvc.SetRack(authedCtx(ctx, author), db, apiarc.SetRackRequest{
					Key:  a.Key,
					Rack: testRack.Key,
				})).Error().To(MatchError(access.ErrDenied))
			},
		)

		It("Should bind the rack and return the task", func(ctx SpecContext) {
			a := createArc(ctx, "set-rack-ok")
			grantSetRack(ctx, a)
			res := MustSucceed(
				apiSvc.SetRack(authedCtx(ctx, author), db, apiarc.SetRackRequest{
					Key:  a.Key,
					Rack: testRack.Key,
				}),
			)
			Expect(res.Task).ToNot(BeNil())
			Expect(res.Task.Rack).To(Equal(testRack.Key))
			Expect(res.Task.Config).To(HaveKeyWithValue("arc_key", a.Key.String()))
			Expect(res.Task.Config).To(HaveKey("hash"))
		})

		It("Should clear the rack with task delete permission", func(ctx SpecContext) {
			a := createArc(ctx, "clear-rack-ok")
			grantSetRack(ctx, a)
			grantOn(
				ctx,
				author.OntologyID(),
				access.ActionDelete,
				ontology.ID{Type: ontology.ResourceTypeTask},
			)
			deployed := MustSucceed(
				apiSvc.SetRack(authedCtx(ctx, author), db, apiarc.SetRackRequest{
					Key:  a.Key,
					Rack: testRack.Key,
				}),
			)
			Expect(deployed.Task).ToNot(BeNil())
			res := MustSucceed(
				apiSvc.SetRack(authedCtx(ctx, author), db, apiarc.SetRackRequest{
					Key: a.Key,
				}),
			)
			Expect(res.Task).To(BeNil())
			Expect(arcSvc.NewRetrieve().
				Where(arc.MatchKeys(a.Key)).
				Exec(ctx, nil)).To(Succeed())
		})

		It("Should bubble up not found for a nonexistent arc", func(ctx SpecContext) {
			missing := uuid.New()
			grantUpdateOn(ctx, author.OntologyID(), arc.OntologyID(missing))
			grantOn(
				ctx,
				author.OntologyID(),
				access.ActionCreate,
				ontology.ID{Type: ontology.ResourceTypeTask},
			)
			Expect(apiSvc.SetRack(authedCtx(ctx, author), db, apiarc.SetRackRequest{
				Key:  missing,
				Rack: testRack.Key,
			})).Error().To(MatchError(query.ErrNotFound))
		})
	})
})
