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
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/validate"
)

// createSchematic persists a fresh schematic owned by the suite workspace and
// returns it with its key populated. Writes commit immediately (nil tx) so
// access-control reads can observe the new ontology resource.
func createSchematic(ctx context.Context, name string) schematic.Schematic {
	s := schematic.Schematic{Name: name, Authority: 1}
	Expect(schemSvc.NewWriter(nil).Create(ctx, ws.Key, &s)).To(Succeed())
	return s
}

var _ = Describe("api.Service.Dispatch", func() {
	Describe("access control", func() {
		It("Should reject the request with access.ErrDenied when the subject has no policy", func(ctx SpecContext) {
			s := createSchematic(ctx, "no-policy")
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), DispatchRequest{
				Key:        s.Key,
				SessionKey: "sess-1",
				Actions: []schematic.Action{schematic.NewSetAuthorityAction(schematic.SetAuthority{
					Value: 9,
				})},
			})).Error().To(MatchError(access.ErrDenied))
		})

		It("Should accept the request when the subject's policy permits update on the target schematic", func(ctx SpecContext) {
			s := createSchematic(ctx, "with-policy")
			grantUpdateOn(ctx, user.OntologyID(author.Key), schematic.OntologyID(s.Key))
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), DispatchRequest{
				Key:        s.Key,
				SessionKey: "sess-1",
				Actions: []schematic.Action{schematic.NewSetAuthorityAction(schematic.SetAuthority{
					Value: 7,
				})},
			})).Error().To(Succeed())
			var res schematic.Schematic
			Expect(schemSvc.NewRetrieve().
				Where(schematic.MatchKeys(s.Key)).
				Entry(&res).Exec(ctx, nil)).To(Succeed())
			Expect(res.Authority).To(BeEquivalentTo(7))
		})

		It("Should reject when the subject's policy targets a different schematic", func(ctx SpecContext) {
			a := createSchematic(ctx, "policy-target")
			b := createSchematic(ctx, "no-policy-target")
			grantUpdateOn(ctx, user.OntologyID(author.Key), schematic.OntologyID(a.Key))
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), DispatchRequest{
				Key:        b.Key,
				SessionKey: "sess-1",
				Actions: []schematic.Action{schematic.NewSetAuthorityAction(schematic.SetAuthority{
					Value: 9,
				})},
			})).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("delegation to Writer.Dispatch", func() {
		It("Should apply a multi-action sequence to the target schematic", func(ctx SpecContext) {
			s := createSchematic(ctx, "multi-action")
			grantUpdateOn(ctx, user.OntologyID(author.Key), schematic.OntologyID(s.Key))
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), DispatchRequest{
				Key:        s.Key,
				SessionKey: "sess-1",
				Actions: []schematic.Action{
					schematic.NewAddNodeAction(schematic.AddNode{
						Node: schematic.Node{Key: "n1", Position: spatial.XY{X: 1, Y: 2}},
					}),
					schematic.NewAddNodeAction(schematic.AddNode{
						Node: schematic.Node{Key: "n2", Position: spatial.XY{X: 3, Y: 4}},
					}),
					schematic.NewSetEdgeAction(schematic.SetEdge{Edge: schematic.Edge{
						Key:    "e1",
						Source: schematic.Handle{Node: "n1", Param: "out"},
						Target: schematic.Handle{Node: "n2", Param: "in"},
					}}),
				},
			})).Error().To(Succeed())
			var res schematic.Schematic
			Expect(schemSvc.NewRetrieve().
				Where(schematic.MatchKeys(s.Key)).
				Entry(&res).Exec(ctx, nil)).To(Succeed())
			Expect(res.Nodes).To(HaveLen(2))
			Expect(res.Edges).To(HaveLen(1))
		})

		It("Should bubble up validate.ErrValidation when dispatching to a snapshot", func(ctx SpecContext) {
			s := createSchematic(ctx, "snap-source")
			var snap schematic.Schematic
			Expect(schemSvc.NewWriter(nil).Copy(ctx, s.Key, "snap", true, &snap)).To(Succeed())
			grantUpdateOn(ctx, user.OntologyID(author.Key), schematic.OntologyID(snap.Key))
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), DispatchRequest{
				Key:        snap.Key,
				SessionKey: "sess-1",
				Actions: []schematic.Action{schematic.NewSetAuthorityAction(schematic.SetAuthority{
					Value: 9,
				})},
			})).Error().To(MatchError(validate.ErrValidation))
		})

		It("Should bubble up query.ErrNotFound when the target schematic does not exist", func(ctx SpecContext) {
			missing := uuid.New()
			grantUpdateOn(ctx, user.OntologyID(author.Key), schematic.OntologyID(missing))
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), DispatchRequest{
				Key:        missing,
				SessionKey: "sess-1",
				Actions: []schematic.Action{schematic.NewSetAuthorityAction(schematic.SetAuthority{
					Value: 9,
				})},
			})).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("subject identity propagation", func() {
		It("Should pass the SessionKey verbatim into the action observer", func(ctx SpecContext) {
			s := createSchematic(ctx, "session-propagation")
			grantUpdateOn(ctx, user.OntologyID(author.Key), schematic.OntologyID(s.Key))
			seen := make(chan schematic.ScopedAction, 1)
			disconnect := schemSvc.OnAction(func(_ context.Context, sa schematic.ScopedAction) {
				seen <- sa
			})
			DeferCleanup(disconnect)
			Expect(apiSvc.Dispatch(authedCtx(ctx, author), DispatchRequest{
				Key:        s.Key,
				SessionKey: "session-marker-xyz",
				Actions: []schematic.Action{schematic.NewSetAuthorityAction(schematic.SetAuthority{
					Value: 12,
				})},
			})).Error().To(Succeed())
			var got schematic.ScopedAction
			Eventually(seen).Should(Receive(&got))
			Expect(got.Key).To(Equal(s.Key))
			Expect(got.SessionKey).To(Equal("session-marker-xyz"))
			Expect(got.Actions).To(HaveLen(1))
		})
	})
})
