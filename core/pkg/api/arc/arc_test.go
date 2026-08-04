// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	arc "github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/encoding/msgpack"
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
		It("Should omit missing keys from a multi-key retrieve instead of failing", func(ctx SpecContext) {
			a := createArc(ctx, "partial-survivor")
			grantOn(ctx, author.OntologyID(), access.ActionRetrieve, arc.OntologyID(a.Key))
			res := MustSucceed(apiSvc.Retrieve(authedCtx(ctx, author), RetrieveRequest{
				Keys:                []arc.Key{a.Key, uuid.New()},
				IgnoreNotFoundError: true,
			}))
			Expect(res.Arcs).To(HaveLen(1))
			Expect(res.Arcs[0].Key).To(Equal(a.Key))
		})

		It("Should return an empty result when every requested key is missing", func(ctx SpecContext) {
			res := MustSucceed(apiSvc.Retrieve(authedCtx(ctx, author), RetrieveRequest{
				Keys:                []arc.Key{uuid.New()},
				IgnoreNotFoundError: true,
			}))
			Expect(res.Arcs).To(BeEmpty())
		})
	})

	Describe("Dispatch", func() {
		Describe("access control", func() {
			It("Should reject the request with access.ErrDenied when the subject has no policy", func(ctx SpecContext) {
				a := createArc(ctx, "no-policy")
				Expect(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
					Key:         a.Key,
					DispatchKey: "sess-1",
					Actions: []arc.Action{arc.NewRemoveNodeAction(arc.RemoveNodePayload{
						Key: "n1",
					})},
				})).Error().To(MatchError(access.ErrDenied))
			})

			It("Should accept the request when the subject's policy permits update on the target Arc", func(ctx SpecContext) {
				a := createArc(ctx, "with-policy")
				grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
				Expect(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
					Key:         a.Key,
					DispatchKey: "sess-1",
					Actions: []arc.Action{arc.NewSetNodeAction(arc.SetNodePayload{
						Node: graph.Node{Key: "n1", Position: spatial.XY{X: 1, Y: 2}},
					})},
				})).Error().To(Succeed())
				var res arc.Arc
				Expect(arcSvc.NewRetrieve().
					Where(arc.MatchKeys(a.Key)).
					Entry(&res).Exec(ctx, nil)).To(Succeed())
				Expect(res.Graph.Nodes).To(HaveLen(1))
				Expect(res.Graph.Nodes[0].Position).To(Equal(spatial.XY{X: 1, Y: 2}))
			})

			It("Should reject when the subject's policy targets a different Arc", func(ctx SpecContext) {
				a := createArc(ctx, "policy-target")
				b := createArc(ctx, "no-policy-target")
				grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
				Expect(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
					Key:         b.Key,
					DispatchKey: "sess-1",
					Actions: []arc.Action{arc.NewRemoveNodeAction(arc.RemoveNodePayload{
						Key: "n1",
					})},
				})).Error().To(MatchError(access.ErrDenied))
			})
		})

		Describe("delegation to Writer.Dispatch", func() {
			It("Should apply a multi-action sequence to the target Arc", func(ctx SpecContext) {
				a := createArc(ctx, "multi-action")
				grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
				Expect(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
					Key:         a.Key,
					DispatchKey: "sess-1",
					Actions: []arc.Action{
						arc.NewSetNodeAction(arc.SetNodePayload{Node: graph.Node{Key: "n1"}}),
						arc.NewSetNodeAction(arc.SetNodePayload{Node: graph.Node{Key: "n2"}}),
						arc.NewRenameAction(arc.RenamePayload{Name: "renamed-multi"}),
					},
				})).Error().To(Succeed())
				var res arc.Arc
				Expect(arcSvc.NewRetrieve().
					Where(arc.MatchKeys(a.Key)).
					Entry(&res).Exec(ctx, nil)).To(Succeed())
				Expect(res.Graph.Nodes).To(HaveLen(2))
				Expect(res.Name).To(Equal("renamed-multi"))
			})

			It("Should bubble up query.ErrNotFound when the target Arc does not exist", func(ctx SpecContext) {
				missing := uuid.New()
				grantUpdateOn(ctx, author.OntologyID(), arc.OntologyID(missing))
				Expect(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
					Key:         missing,
					DispatchKey: "sess-1",
					Actions: []arc.Action{arc.NewRenameAction(arc.RenamePayload{
						Name: "ghost",
					})},
				})).Error().To(MatchError(query.ErrNotFound))
			})
		})

		Describe("semantic hash", func() {
			It("Should return the updated hash after a semantic edit", func(ctx SpecContext) {
				a := createArc(ctx, "hash-echo")
				grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
				before := MustSucceed(arc.Hash(a))
				res := MustSucceed(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
					Key:         a.Key,
					DispatchKey: "sess-1",
					Actions: []arc.Action{arc.NewSetNodeInputsAction(arc.SetNodeInputsPayload{
						Key:    "n1",
						Inputs: msgpack.EncodedJSON{"type": "on", "channel": 1},
					})},
				}))
				Expect(res.Hash).ToNot(BeEmpty())
				Expect(res.Hash).ToNot(Equal(before))
			})

			It("Should return an unchanged hash after a layout-only edit", func(ctx SpecContext) {
				a := createArc(ctx, "hash-layout")
				grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
				placed := MustSucceed(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
					Key:         a.Key,
					DispatchKey: "sess-1",
					Actions: []arc.Action{arc.NewSetNodeAction(arc.SetNodePayload{
						Node: graph.Node{Key: "n1", Position: spatial.XY{X: 0, Y: 0}},
					})},
				}))
				moved := MustSucceed(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
					Key:         a.Key,
					DispatchKey: "sess-2",
					Actions: []arc.Action{arc.NewSetNodeAction(arc.SetNodePayload{
						Node: graph.Node{Key: "n1", Position: spatial.XY{X: 9, Y: 9}},
					})},
				}))
				Expect(moved.Hash).To(Equal(placed.Hash))
			})
		})

		Describe("Retrieve hash", func() {
			It("Should serve the semantic hash on every retrieved arc", func(ctx SpecContext) {
				a := createArc(ctx, "hash-served")
				grantOn(ctx, author.OntologyID(), access.ActionRetrieve, arc.OntologyID(a.Key))
				res := MustSucceed(apiSvc.Retrieve(authedCtx(ctx, author), RetrieveRequest{
					Keys: []arc.Key{a.Key},
				}))
				Expect(res.Arcs).To(HaveLen(1))
				Expect(res.Arcs[0].Hash).ToNot(BeNil())
				Expect(*res.Arcs[0].Hash).To(Equal(MustSucceed(arc.Hash(res.Arcs[0]))))
			})
		})

		Describe("subject identity propagation", func() {
			It("Should pass the DispatchKey verbatim into the action observer", func(ctx SpecContext) {
				a := createArc(ctx, "session-propagation")
				grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
				seen := make(chan scopedAction, 1)
				DeferCleanup(arcSvc.OnAction(func(_ context.Context, sa scopedAction) {
					seen <- sa
				}))
				Expect(apiSvc.Dispatch(authedCtx(ctx, author), db, DispatchRequest{
					Key:         a.Key,
					DispatchKey: "session-marker-xyz",
					Actions: []arc.Action{arc.NewSetNodeAction(arc.SetNodePayload{
						Node: graph.Node{Key: "n1"},
					})},
				})).Error().To(Succeed())
				var got scopedAction
				Eventually(seen).Should(Receive(&got))
				Expect(got.Key).To(Equal(a.Key))
				Expect(got.DispatchKey).To(Equal("session-marker-xyz"))
				Expect(got.Seq).To(BeNumerically(">", uint64(0)))
				Expect(got.Actions).To(HaveLen(1))
			})
		})
	})

	Describe("Deploy", func() {
		grantDeploy := func(ctx SpecContext, a arc.Arc) {
			grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
			grantOn(
				ctx,
				author.OntologyID(),
				access.ActionCreate,
				ontology.ID{Type: ontology.ResourceTypeTask},
			)
		}

		It("Should reject the request when the subject has no policy", func(ctx SpecContext) {
			a := createArc(ctx, "deploy-no-policy")
			Expect(apiSvc.Deploy(authedCtx(ctx, author), db, DeployRequest{
				Key:  a.Key,
				Rack: testRack.Key,
			})).Error().To(MatchError(access.ErrDenied))
		})

		It("Should reject when the subject may update the arc but not create tasks", func(ctx SpecContext) {
			a := createArc(ctx, "deploy-no-task-policy")
			grantUpdateOn(ctx, author.OntologyID(), a.OntologyID())
			Expect(apiSvc.Deploy(authedCtx(ctx, author), db, DeployRequest{
				Key:  a.Key,
				Rack: testRack.Key,
			})).Error().To(MatchError(access.ErrDenied))
		})

		It("Should deploy the arc and return its task", func(ctx SpecContext) {
			a := createArc(ctx, "deploy-ok")
			grantDeploy(ctx, a)
			res := MustSucceed(apiSvc.Deploy(authedCtx(ctx, author), db, DeployRequest{
				Key:  a.Key,
				Rack: testRack.Key,
			}))
			Expect(res.Task).ToNot(BeNil())
			Expect(res.Task.Rack).To(Equal(testRack.Key))
			Expect(res.Task.Config).To(HaveKeyWithValue("arc_key", a.Key.String()))
			Expect(res.Task.Config).To(HaveKey("hash"))
		})

		It("Should undeploy with task delete permission", func(ctx SpecContext) {
			a := createArc(ctx, "undeploy-ok")
			grantDeploy(ctx, a)
			grantOn(
				ctx,
				author.OntologyID(),
				access.ActionDelete,
				ontology.ID{Type: ontology.ResourceTypeTask},
			)
			deployed := MustSucceed(apiSvc.Deploy(authedCtx(ctx, author), db, DeployRequest{
				Key:  a.Key,
				Rack: testRack.Key,
			}))
			Expect(deployed.Task).ToNot(BeNil())
			res := MustSucceed(apiSvc.Deploy(authedCtx(ctx, author), db, DeployRequest{
				Key: a.Key,
			}))
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
			Expect(apiSvc.Deploy(authedCtx(ctx, author), db, DeployRequest{
				Key:  missing,
				Rack: testRack.Key,
			})).Error().To(MatchError(query.ErrNotFound))
		})
	})
})
