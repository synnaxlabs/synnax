// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apipanel "github.com/synnaxlabs/synnax/pkg/api/panel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// createPanel persists a panel in the suite project and returns it with its key
// populated. Writes commit immediately (nil tx) so the api enforcer can observe the new
// panel.
func createPanel(ctx context.Context, name string) panel.Panel {
	GinkgoHelper()
	p := panel.Panel{Name: name, Project: &proj.Key}
	Expect(panelSvc.NewWriter(nil).Create(ctx, &p)).To(Succeed())
	return p
}

var _ = Describe("Service", func() {
	Describe("Create", func() {
		It(
			"Should reject the request when the subject has no create policy",
			func(ctx SpecContext) {
				u := newUser(ctx)
				p := panel.Panel{Key: uuid.New(), Name: "no-policy", Project: &proj.Key}
				Expect(apiSvc.Create(AuthedCtx(ctx, u), nil, apipanel.CreateRequest{
					Panels: []panel.Panel{p},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)

		It("Should create the panels in the provided project", func(ctx SpecContext) {
			u := newUser(ctx)
			p := panel.Panel{Key: uuid.New(), Name: "with-parent", Project: &proj.Key}
			grantCreate(ctx, u.ResourceID())
			res := MustSucceed(
				apiSvc.Create(AuthedCtx(ctx, u), nil, apipanel.CreateRequest{
					Panels: []panel.Panel{p},
				}),
			)
			Expect(res.Panels).To(HaveLen(1))
			Expect(res.Panels[0].Key).To(Equal(p.Key))
			var got panel.Panel
			Expect(panelSvc.NewRetrieve().Where(panel.MatchKeys(p.Key)).
				Entry(&got).Exec(ctx, nil)).To(Succeed())
			Expect(got.Name).To(Equal("with-parent"))
			Expect(got.Project).To(HaveValue(Equal(proj.Key)))
		})

		It("Should stamp the dispatch key onto the create broadcast", func(
			ctx SpecContext,
		) {
			u := newUser(ctx)
			grantCreate(ctx, u.ResourceID())
			rec := &Recorder[panel.Key, panel.Action]{}
			DeferCleanup(panelSvc.OnAction(rec.Record))
			p := panel.Panel{Key: uuid.New(), Name: "broadcast", Project: &proj.Key}
			MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil, apipanel.CreateRequest{
				Panels:      []panel.Panel{p},
				DispatchKey: "dk-create",
			}))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].Key).To(Equal(p.Key))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		})

		It(
			"Should reject the request without update on the owner",
			func(ctx SpecContext) {
				u := newUser(ctx)
				grant(ctx, u.ResourceID(), access.ActionCreate,
					resource.ID{Type: resource.TypePanel})
				p := panel.Panel{Key: uuid.New(), Name: "denied", Project: &proj.Key}
				Expect(apiSvc.Create(AuthedCtx(ctx, u), nil, apipanel.CreateRequest{
					Panels: []panel.Panel{p},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)

		It(
			"Should reject a panel with neither a project nor a range",
			func(ctx SpecContext) {
				u := newUser(ctx)
				p := panel.Panel{Key: uuid.New(), Name: "no-owner"}
				grant(ctx, u.ResourceID(), access.ActionCreate,
					resource.ID{Type: resource.TypePanel})
				Expect(apiSvc.Create(AuthedCtx(ctx, u), nil, apipanel.CreateRequest{
					Panels: []panel.Panel{p},
				})).Error().To(MatchError(ContainSubstring(
					"must belong to exactly one of a project and a range",
				)))
			},
		)

		It(
			"Should reject the request when the subject cannot create in the project",
			func(ctx SpecContext) {
				u := newUser(ctx)
				p := panel.Panel{Key: uuid.New(), Name: "denied", Project: &proj.Key}
				grant(ctx, u.ResourceID(), access.ActionCreate,
					resource.ID{Type: resource.TypePanel})
				Expect(apiSvc.Create(AuthedCtx(ctx, u), nil, apipanel.CreateRequest{
					Panels: []panel.Panel{p},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
	})

	Describe("Retrieve", func() {
		It(
			"Should return the panels matching the requested keys",
			func(ctx SpecContext) {
				p := createPanel(ctx, "retrievable")
				grant(ctx, author.ResourceID(), access.ActionRetrieve, p.ResourceID())
				res := MustSucceed(
					apiSvc.Retrieve(AuthedCtx(ctx, author), apipanel.RetrieveRequest{
						Where: &panel.Where{
							Key: &filter.Set[panel.Key]{In: []panel.Key{p.Key}},
						},
					}),
				)
				Expect(res.Panels).To(HaveLen(1))
				Expect(res.Panels[0].Key).To(Equal(p.Key))
				Expect(res.Panels[0].Name).To(Equal("retrievable"))
			},
		)

		It(
			"Should reject the request when the subject cannot retrieve a matched panel",
			func(ctx SpecContext) {
				p := createPanel(ctx, "forbidden")
				Expect(apiSvc.Retrieve(AuthedCtx(ctx, author), apipanel.RetrieveRequest{
					Where: &panel.Where{
						Key: &filter.Set[panel.Key]{In: []panel.Key{p.Key}},
					},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)

		It(
			"Should omit missing keys from a multi-key retrieve instead of failing",
			func(ctx SpecContext) {
				p := createPanel(ctx, "partial-survivor")
				missing := uuid.New()
				grant(ctx, author.ResourceID(), access.ActionRetrieve,
					panel.ResourceID(p.Key), panel.ResourceID(missing))
				res := MustSucceed(
					apiSvc.Retrieve(AuthedCtx(ctx, author), apipanel.RetrieveRequest{
						Where: &panel.Where{
							Key: &filter.Set[panel.Key]{
								In: []panel.Key{p.Key, missing},
							},
						},
					}),
				)
				Expect(res.Panels).To(HaveLen(1))
				Expect(res.Panels[0].Key).To(Equal(p.Key))
			},
		)

		It(
			"Should return an empty result when every requested key is missing",
			func(ctx SpecContext) {
				res := MustSucceed(
					apiSvc.Retrieve(AuthedCtx(ctx, author), apipanel.RetrieveRequest{
						Where: &panel.Where{
							Key: &filter.Set[panel.Key]{In: []panel.Key{uuid.New()}},
						},
					}),
				)
				Expect(res.Panels).To(BeEmpty())
			},
		)

		It("Should return the panels of the requested projects", func(ctx SpecContext) {
			p := createPanel(ctx, "in-project")
			grant(ctx, author.ResourceID(), access.ActionRetrieve, p.ResourceID())
			res := MustSucceed(
				apiSvc.Retrieve(AuthedCtx(ctx, author), apipanel.RetrieveRequest{
					Where: &panel.Where{
						Key:      &filter.Set[panel.Key]{Eq: &p.Key},
						Projects: []project.Key{proj.Key},
					},
				}),
			)
			Expect(res.Panels).To(HaveLen(1))
			res = MustSucceed(
				apiSvc.Retrieve(AuthedCtx(ctx, author), apipanel.RetrieveRequest{
					Where: &panel.Where{
						Key:      &filter.Set[panel.Key]{Eq: &p.Key},
						Projects: []project.Key{uuid.New()},
					},
				}),
			)
			Expect(res.Panels).To(BeEmpty())
		})

		It("Should honor the limit and offset bounds", func(ctx SpecContext) {
			a := createPanel(ctx, "page-a")
			b := createPanel(ctx, "page-b")
			grant(ctx, author.ResourceID(), access.ActionRetrieve,
				a.ResourceID(), b.ResourceID())
			res := MustSucceed(
				apiSvc.Retrieve(AuthedCtx(ctx, author), apipanel.RetrieveRequest{
					Where: &panel.Where{
						Key: &filter.Set[panel.Key]{In: []panel.Key{a.Key, b.Key}},
					},
					Limit:  new(uint32(1)),
					Offset: 1,
				}),
			)
			Expect(res.Panels).To(HaveLen(1))
		})
	})

	Describe("SetProject", func() {
		It(
			"Should reject the request when the subject has no update policy",
			func(ctx SpecContext) {
				p := createPanel(ctx, "move-denied")
				Expect(apiSvc.SetProject(
					AuthedCtx(ctx, author),
					nil,
					apipanel.SetProjectRequest{
						Project: proj.Key,
						Keys:    []panel.Key{p.Key},
					},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)

		It(
			"Should reject the request without update on the destination project",
			func(ctx SpecContext) {
				u := newUser(ctx)
				dest := project.Project{Name: "denied destination"}
				Expect(projectSvc.NewWriter(nil).Create(ctx, &dest)).To(Succeed())
				p := createPanel(ctx, "move-denied")
				grant(ctx, u.ResourceID(), access.ActionUpdate,
					p.ResourceID(), proj.ResourceID())
				Expect(apiSvc.SetProject(
					AuthedCtx(ctx, u),
					nil,
					apipanel.SetProjectRequest{
						Project: dest.Key,
						Keys:    []panel.Key{p.Key},
					},
				)).Error().To(MatchError(access.ErrDenied))
				Expect(retrievePanel(ctx, p.Key).Project).To(Equal(&proj.Key))
			},
		)

		It("Should move the panels into the project", func(ctx SpecContext) {
			u := newUser(ctx)
			dest := project.Project{Name: "destination"}
			Expect(projectSvc.NewWriter(nil).Create(ctx, &dest)).To(Succeed())
			p := createPanel(ctx, "moved")
			grant(ctx, u.ResourceID(), access.ActionUpdate,
				p.ResourceID(), proj.ResourceID(), dest.ResourceID())
			Expect(apiSvc.SetProject(
				AuthedCtx(ctx, u),
				nil,
				apipanel.SetProjectRequest{Project: dest.Key, Keys: []panel.Key{p.Key}},
			)).Error().To(Succeed())
			Expect(retrievePanel(ctx, p.Key).Project).To(Equal(&dest.Key))
		})
	})

	Describe("Dispatch", func() {
		It(
			"Should reject the request when the subject has no update policy",
			func(ctx SpecContext) {
				p := createPanel(ctx, "dispatch-denied")
				Expect(
					apiSvc.Dispatch(
						AuthedCtx(ctx, author),
						apipanel.DispatchRequest{
							Key:         p.Key,
							DispatchKey: "sess-1",
							Actions: []panel.Action{
								panel.NewRenameAction(panel.RenamePayload{Name: "x"}),
							},
						},
					),
				).Error().
					To(MatchError(access.ErrDenied))
			},
		)

		It(
			"Should apply the action sequence to the target panel",
			func(ctx SpecContext) {
				p := createPanel(ctx, "dispatch-ok")
				grant(ctx, author.ResourceID(), access.ActionUpdate, p.ResourceID())
				Expect(
					apiSvc.Dispatch(
						AuthedCtx(ctx, author),
						apipanel.DispatchRequest{
							Key:         p.Key,
							DispatchKey: "sess-1",
							Actions: []panel.Action{
								panel.NewRenameAction(
									panel.RenamePayload{Name: "renamed"},
								),
							},
						},
					),
				).Error().
					To(Succeed())
				var got panel.Panel
				Expect(panelSvc.NewRetrieve().Where(panel.MatchKeys(p.Key)).
					Entry(&got).Exec(ctx, nil)).To(Succeed())
				Expect(got.Name).To(Equal("renamed"))
			},
		)
	})

	Describe("Delete", func() {
		It(
			"Should reject the request when the subject has no delete policy",
			func(ctx SpecContext) {
				p := createPanel(ctx, "delete-denied")
				Expect(
					apiSvc.Delete(AuthedCtx(ctx, author), nil, apipanel.DeleteRequest{
						Keys: []panel.Key{p.Key},
					}),
				).Error().
					To(MatchError(access.ErrDenied))
			},
		)

		It("Should delete the panels with the given keys", func(ctx SpecContext) {
			p := createPanel(ctx, "deletable")
			grant(ctx, author.ResourceID(), access.ActionDelete, p.ResourceID())
			Expect(apiSvc.Delete(AuthedCtx(ctx, author), nil, apipanel.DeleteRequest{
				Keys: []panel.Key{p.Key},
			})).Error().To(Succeed())
			var got panel.Panel
			Expect(panelSvc.NewRetrieve().Where(panel.MatchKeys(p.Key)).
				Entry(&got).Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})
	})
})

// createGroup creates a group in the suite project.
func createGroup(ctx SpecContext, name string) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Parent: new(project.ResourceID(proj.Key))}
	Expect(groupSvc.NewWriter(nil).Create(ctx, &g)).To(Succeed())
	return g
}

func retrievePanel(ctx SpecContext, key panel.Key) panel.Panel {
	GinkgoHelper()
	var p panel.Panel
	Expect(panelSvc.NewRetrieve().
		Where(panel.MatchKeys(key)).
		Entry(&p).
		Exec(ctx, nil)).To(Succeed())
	return p
}

var _ = Describe("SetGroup", func() {
	It(
		"Should refuse the move without update on the destination group",
		func(ctx SpecContext) {
			u := newUser(ctx)
			p := createPanel(ctx, "Set group denied")
			g := createGroup(ctx, "Set group denied")
			grant(ctx, u.ResourceID(), access.ActionUpdate, panel.ResourceID(p.Key))
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, u), db,
				apipanel.SetGroupRequest{Group: &g.Key, Keys: []panel.Key{p.Key}},
			)).Error().To(MatchError(access.ErrDenied))
			Expect(retrievePanel(ctx, p.Key).Group).To(BeNil())
		},
	)
	It(
		"Should move the panel with update on the destination group",
		func(ctx SpecContext) {
			u := newUser(ctx)
			p := createPanel(ctx, "Set group granted")
			g := createGroup(ctx, "Set group granted")
			grant(
				ctx,
				u.ResourceID(),
				access.ActionUpdate,
				panel.ResourceID(p.Key),
				g.ResourceID(),
			)
			Expect(apiSvc.SetGroup(
				AuthedCtx(ctx, u), db,
				apipanel.SetGroupRequest{Group: &g.Key, Keys: []panel.Key{p.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(retrievePanel(ctx, p.Key).Group).To(Equal(&g.Key))
		},
	)
	Context("Into a group under another project", func() {
		var (
			u    user.User
			x    panel.Panel
			dest project.Project
			g    group.Group
		)
		BeforeEach(func(ctx SpecContext) {
			u = newUser(ctx)
			x = createPanel(ctx, "Cross project")
			dest = project.Project{Name: "Destination"}
			Expect(projectSvc.NewWriter(nil).Create(ctx, &dest)).To(Succeed())
			g = group.Group{Name: "Destination", Parent: new(dest.ResourceID())}
			Expect(groupSvc.NewWriter(nil).Create(ctx, &g)).To(Succeed())
		})
		move := func(ctx SpecContext) (struct{}, error) {
			return apiSvc.SetGroup(
				AuthedCtx(ctx, u),
				db,
				apipanel.SetGroupRequest{Group: &g.Key, Keys: []panel.Key{x.Key}},
			)
		}
		DescribeTable(
			"Should refuse the move without update on both projects",
			func(ctx SpecContext, projects func() []resource.ID) {
				objects := projects()
				grant(ctx, u.ResourceID(), access.ActionUpdate, append(
					objects,
					panel.ResourceID(x.Key),
					g.ResourceID(),
				)...)
				Expect(move(ctx)).Error().To(MatchError(access.ErrDenied))
				Expect(retrievePanel(ctx, x.Key).Project).To(Equal(&proj.Key))
			},
			Entry("without the destination", func() []resource.ID {
				return []resource.ID{project.ResourceID(proj.Key)}
			}),
			Entry("without the source", func() []resource.ID {
				return []resource.ID{dest.ResourceID()}
			}),
		)
		It("Should move the panel with update on both projects", func(ctx SpecContext) {
			objects := []resource.ID{project.ResourceID(proj.Key), dest.ResourceID()}
			grant(ctx, u.ResourceID(), access.ActionUpdate, append(
				objects,
				panel.ResourceID(x.Key),
				g.ResourceID(),
			)...)
			MustSucceed(move(ctx))
			Expect(retrievePanel(ctx, x.Key).Project).To(Equal(&dest.Key))
		})
	})
})

var _ = Describe("Create in a group", func() {
	newPanel := func(name string, g group.Group) panel.Panel {
		return panel.Panel{
			Key:     uuid.New(),
			Name:    name,
			Project: &proj.Key,
			Group:   &g.Key,
		}
	}
	It(
		"Should refuse the create without update on the group",
		func(ctx SpecContext) {
			u := newUser(ctx)
			g := createGroup(ctx, "Create denied")
			grantCreate(ctx, u.ResourceID())
			Expect(apiSvc.Create(AuthedCtx(ctx, u), db, apipanel.CreateRequest{
				Panels: []panel.Panel{newPanel("Denied", g)},
			})).Error().To(MatchError(access.ErrDenied))
		},
	)
	It("Should create the panel with update on the group", func(ctx SpecContext) {
		u := newUser(ctx)
		g := createGroup(ctx, "Create granted")
		grantCreate(ctx, u.ResourceID())
		grant(ctx, u.ResourceID(), access.ActionUpdate, g.ResourceID())
		p := newPanel("Granted", g)
		Expect(apiSvc.Create(AuthedCtx(ctx, u), db, apipanel.CreateRequest{
			Panels: []panel.Panel{p},
		})).Error().ToNot(HaveOccurred())
		Expect(retrievePanel(ctx, p.Key).Group).To(Equal(&g.Key))
	})
	Context("Under another project", func() {
		var (
			u    user.User
			dest project.Project
			g    group.Group
		)
		BeforeEach(func(ctx SpecContext) {
			u = newUser(ctx)
			grantCreate(ctx, u.ResourceID())
			dest = project.Project{Name: "Create destination"}
			Expect(projectSvc.NewWriter(nil).Create(ctx, &dest)).To(Succeed())
			g = group.Group{Name: "Create destination", Parent: new(dest.ResourceID())}
			Expect(groupSvc.NewWriter(nil).Create(ctx, &g)).To(Succeed())
		})
		create := func(ctx SpecContext, carried *project.Key) (apipanel.CreateResponse, error) {
			return apiSvc.Create(AuthedCtx(ctx, u), db, apipanel.CreateRequest{
				Panels: []panel.Panel{
					{
						Key:     uuid.New(),
						Project: carried,
						Name:    "Elsewhere",
						Group:   &g.Key,
					},
				},
			})
		}
		DescribeTable(
			"Should refuse a create in a group without update on the group's project",
			func(ctx SpecContext, carried func() *project.Key) {
				grant(ctx, u.ResourceID(), access.ActionUpdate, g.ResourceID())
				Expect(create(ctx, carried())).Error().To(MatchError(access.ErrDenied))
			},
			Entry(
				"with the project it carries",
				func() *project.Key { return &proj.Key },
			),
			Entry("with no project", func() *project.Key { return nil }),
		)
		It(
			"Should create the panel with update on the group's project",
			func(ctx SpecContext) {
				grant(
					ctx,
					u.ResourceID(),
					access.ActionUpdate,
					g.ResourceID(),
					dest.ResourceID(),
				)
				res := MustSucceed(create(ctx, nil))
				Expect(
					retrievePanel(ctx, res.Panels[0].Key).Project,
				).To(Equal(&dest.Key))
			},
		)
	})
})
