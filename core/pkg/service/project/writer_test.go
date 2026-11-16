// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
	Describe("Create", func() {
		It("Should create a project", func(ctx SpecContext) {
			proj := project.Project{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
			Expect(proj.Key).ToNot(Equal(uuid.Nil()))
		})
		It(
			"Should return a validation error when the name is empty",
			func(ctx SpecContext) {
				proj := project.Project{}
				Expect(svc.NewWriter(tx).Create(ctx, &proj)).
					To(MatchError(ContainSubstring("name: required")))
			},
		)
		It("Should refuse a project whose key exists", func(ctx SpecContext) {
			proj := project.Project{Name: "stored"}
			Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
			again := project.Project{Key: proj.Key, Name: "refused"}
			Expect(svc.NewWriter(tx).Create(ctx, &again)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("project %s already exists", proj.Key)),
			))
			var res project.Project
			Expect(svc.NewRetrieve().
				Where(project.MatchKeys(proj.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(Equal(proj))
		})
	})
	Describe("CreateMany", func() {
		It("Should create multiple projects", func(ctx SpecContext) {
			projects := []project.Project{
				{Name: "project-1"},
				{Name: "project-2"},
			}
			Expect(svc.NewWriter(tx).CreateMany(ctx, &projects)).To(Succeed())

			var retrieved []project.Project
			Expect(svc.NewRetrieve().Where(project.MatchKeys(
				projects[0].Key,
				projects[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})
	Describe("Update", func() {
		It("Should rename a project", func(ctx SpecContext) {
			proj := project.Project{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, proj.Key, "test2")).To(Succeed())
			var res project.Project
			Expect(
				svc.NewRetrieve().
					Where(project.MatchKeys(proj.Key)).
					Entry(&res).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
		It("Should return a validation error for an empty name", func(ctx SpecContext) {
			proj := project.Project{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, proj.Key, "")).
				To(MatchError(ContainSubstring("name: required")))
		})
	})
	Describe("Delete", func() {
		It("Should delete a project", func(ctx SpecContext) {
			proj := project.Project{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, proj.Key)).To(Succeed())
			var res project.Project
			Expect(
				svc.NewRetrieve().
					Where(project.MatchKeys(proj.Key)).
					Entry(&res).
					Exec(ctx, tx),
			).To(MatchError(query.ErrNotFound))
		})

		It(
			"Should delete the project's visualizations and groups",
			func(ctx SpecContext) {
				proj := project.Project{Name: "doomed"}
				Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
				g := group.Group{Name: "folder", Parent: new(proj.ResourceID())}
				Expect(groupSvc.NewWriter(tx).Create(ctx, &g)).To(Succeed())
				s := schematic.Schematic{Name: "s", Group: &g.Key}
				Expect(schematicSvc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
				lp := defaults.Fill(lineplot.LinePlot{Name: "lp", Project: &proj.Key})
				Expect(lineplotSvc.NewWriter(tx).Create(ctx, &lp)).To(Succeed())
				l := defaults.Fill(log.Log{Name: "l", Project: &proj.Key})
				Expect(logSvc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
				t := table.Table{Name: "t", Project: &proj.Key}
				Expect(tableSvc.NewWriter(tx).Create(ctx, &t)).To(Succeed())
				p := panel.Panel{Name: "p", Project: &proj.Key}
				Expect(panelSvc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
				Expect(svc.NewWriter(tx).Delete(ctx, proj.Key)).To(Succeed())
				Expect(groupSvc.NewRetrieve().
					Where(group.MatchKeys(g.Key)).
					Entry(&group.Group{}).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
				Expect(schematicSvc.NewRetrieve().
					Where(schematic.MatchKeys(s.Key)).
					Entry(&schematic.Schematic{}).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
				Expect(lineplotSvc.NewRetrieve().
					Where(lineplot.MatchKeys(lp.Key)).
					Entry(&lineplot.LinePlot{}).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
				Expect(logSvc.NewRetrieve().
					Where(log.MatchKeys(l.Key)).
					Entry(&log.Log{}).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
				Expect(tableSvc.NewRetrieve().
					Where(table.MatchKeys(t.Key)).
					Entry(&table.Table{}).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
				Expect(panelSvc.NewRetrieve().
					Where(panel.MatchKeys(p.Key)).
					Entry(&panel.Panel{}).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
			},
		)

		It("Should keep the visualizations of other projects", func(ctx SpecContext) {
			doomed := project.Project{Name: "doomed"}
			kept := project.Project{Name: "kept"}
			Expect(svc.NewWriter(tx).Create(ctx, &doomed)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &kept)).To(Succeed())
			l := defaults.Fill(log.Log{Name: "l", Project: &kept.Key})
			Expect(logSvc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, doomed.Key)).To(Succeed())
			Expect(logSvc.NewRetrieve().
				Where(log.MatchKeys(l.Key)).
				Entry(&log.Log{}).
				Exec(ctx, tx)).To(Succeed())
		})
	})

	Describe("Children", func() {
		It(
			"Should return the project's groups and visualizations",
			func(ctx SpecContext) {
				proj := project.Project{Name: "parent"}
				Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
				g := group.Group{Name: "folder", Parent: new(proj.ResourceID())}
				Expect(groupSvc.NewWriter(tx).Create(ctx, &g)).To(Succeed())
				l := defaults.Fill(log.Log{Name: "l", Project: &proj.Key})
				Expect(logSvc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
				p := panel.Panel{Name: "p", Project: &proj.Key}
				Expect(panelSvc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
				Expect(svc.Children(ctx, tx, proj.Key)).To(ConsistOf(
					g.ResourceID(), l.ResourceID(), p.ResourceID(),
				))
			},
		)
	})
	It("Should return the children of a project through a snapshot", func(
		ctx SpecContext,
	) {
		proj := project.Project{Name: "snapshot"}
		Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
		l := defaults.Fill(log.Log{Name: "l", Project: &proj.Key})
		Expect(logSvc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		var children []resource.ID
		Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
			children, err = svc.Children(ctx, snap, proj.Key)
			return err
		})).To(Succeed())
		Expect(children).To(ConsistOf(l.ResourceID()))
	})
	Describe("Group", func() {
		createGroup := func(ctx context.Context, g group.Group) group.Group {
			GinkgoHelper()
			Expect(groupSvc.NewWriter(tx).Create(ctx, &g)).To(Succeed())
			return g
		}
		retrieveGroup := func(ctx context.Context, key project.Key) *group.Key {
			GinkgoHelper()
			var res project.Project
			Expect(svc.NewRetrieve().
				Where(project.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			return res.Group
		}

		It(
			"Should move a project into a top-level group and back out",
			func(ctx SpecContext) {
				g := createGroup(ctx, group.Group{
					Name:  "Top",
					Scope: resource.TypeProject,
				})
				proj := project.Project{Name: "grouped"}
				Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
				Expect(svc.NewWriter(tx).SetGroup(ctx, &g.Key, proj.Key)).
					To(Succeed())
				Expect(retrieveGroup(ctx, proj.Key)).To(Equal(&g.Key))
				Expect(svc.NewWriter(tx).SetGroup(ctx, nil, proj.Key)).To(Succeed())
				Expect(retrieveGroup(ctx, proj.Key)).To(BeNil())
			},
		)

		It("Should reject a group under a project", func(ctx SpecContext) {
			holder := project.Project{Name: "holder"}
			Expect(svc.NewWriter(tx).Create(ctx, &holder)).To(Succeed())
			g := createGroup(ctx, group.Group{
				Name:   "Under project",
				Parent: new(holder.ResourceID()),
			})
			proj := project.Project{Name: "nested"}
			Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
			Expect(svc.NewWriter(tx).SetGroup(ctx, &g.Key, proj.Key)).To(MatchError(
				"group: a project cannot sit in a group under project " +
					holder.Key.String() + ": validation error",
			))
			Expect(retrieveGroup(ctx, proj.Key)).To(BeNil())
		})
	})
})

var _ = Describe("Writer in concurrent transactions", func() {
	retrieveGroup := func(ctx context.Context, key project.Key) *group.Key {
		GinkgoHelper()
		var res project.Project
		Expect(svc.NewRetrieve().
			Where(project.MatchKeys(key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res.Group
	}
	createGroup := func(ctx context.Context, g group.Group) group.Group {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Delete(ctx, g.Key)
			})).To(Succeed())
		})
		return g
	}
	createProject := func(ctx context.Context, name string) project.Project {
		GinkgoHelper()
		p := project.Project{Name: name}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &p)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, p.Key)
			})).To(Succeed())
		})
		return p
	}

	It(
		"Should keep the group that a move writes during concurrent applies",
		func(ctx SpecContext) {
			groups := make([]group.Group, 50)
			for i := range groups {
				groups[i] = createGroup(ctx, group.Group{
					Name:  fmt.Sprintf("Concurrent %d", i),
					Scope: resource.TypeProject,
				})
			}
			proj := createProject(ctx, "concurrent")
			var (
				wg    sync.WaitGroup
				moved atomic.Bool
			)
			for range 4 {
				wg.Go(func() {
					defer GinkgoRecover()
					for !moved.Load() {
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							return svc.NewWriter(tx).Rename(ctx, proj.Key, "renamed")
						})).To(Succeed())
					}
				})
			}
			wg.Go(func() {
				defer GinkgoRecover()
				defer moved.Store(true)
				for _, g := range groups {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).SetGroup(ctx, &g.Key, proj.Key)
					})).
						To(Succeed())
					Expect(retrieveGroup(ctx, proj.Key)).To(Equal(&g.Key))
				}
			})
			wg.Wait()
			Expect(retrieveGroup(ctx, proj.Key)).To(Equal(&groups[len(groups)-1].Key))
		},
	)

	It(
		"Should not put a project in a group that moves under a project at once",
		func(ctx SpecContext) {
			holder := createProject(ctx, "holder")
			for range 30 {
				g := createGroup(ctx, group.Group{
					Name:  "Moving",
					Scope: resource.TypeProject,
				})
				proj := createProject(ctx, "joining")
				var (
					wg               sync.WaitGroup
					joinErr, moveErr error
				)
				wg.Go(func() {
					joinErr = db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).SetGroup(ctx, &g.Key, proj.Key)
					})
				})
				wg.Go(func() {
					moveErr = db.WithTx(ctx, func(tx gorp.Tx) error {
						return groupSvc.NewWriter(tx).
							SetParent(ctx, new(holder.ResourceID()), g.Key)
					})
				})
				wg.Wait()
				Expect([]error{joinErr, moveErr}).To(ContainElement(HaveOccurred()))
			}
		},
	)
})
