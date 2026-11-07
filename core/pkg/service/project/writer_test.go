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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
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
	Describe("DeleteChannel", func() {
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
			owner := project.Project{Name: "owner"}
			Expect(svc.NewWriter(tx).Create(ctx, &owner)).To(Succeed())
			g := createGroup(ctx, group.Group{
				Name:   "Under project",
				Parent: new(owner.ResourceID()),
			})
			proj := project.Project{Name: "nested"}
			Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
			Expect(svc.NewWriter(tx).SetGroup(ctx, &g.Key, proj.Key)).To(MatchError(
				"group: a project cannot sit in a group under project " +
					owner.Key.String() + ": validation error",
			))
			Expect(retrieveGroup(ctx, proj.Key)).To(BeNil())
		})

		It(
			"Should keep the stored group when a project is created again",
			func(ctx SpecContext) {
				g := createGroup(ctx, group.Group{
					Name:  "Kept",
					Scope: resource.TypeProject,
				})
				proj := project.Project{Name: "kept", Group: &g.Key}
				Expect(svc.NewWriter(tx).Create(ctx, &proj)).To(Succeed())
				again := project.Project{Key: proj.Key, Name: "kept again"}
				Expect(svc.NewWriter(tx).Create(ctx, &again)).To(Succeed())
				Expect(retrieveGroup(ctx, proj.Key)).To(Equal(&g.Key))
			},
		)
	})
})

var _ = Describe("Writer with no transaction", func() {
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
		Expect(groupSvc.NewWriter(nil).Create(ctx, &g)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(groupSvc.NewWriter(nil).Delete(ctx, g.Key)).To(Succeed())
		})
		return g
	}
	createProject := func(ctx context.Context, name string) project.Project {
		GinkgoHelper()
		p := project.Project{Name: name}
		Expect(svc.NewWriter(nil).Create(ctx, &p)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).Delete(ctx, p.Key)).To(Succeed())
		})
		return p
	}

	It(
		"Should keep the group that a move writes during concurrent creates",
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
						again := project.Project{Key: proj.Key, Name: proj.Name}
						Expect(svc.NewWriter(nil).Create(ctx, &again)).To(Succeed())
					}
				})
			}
			wg.Go(func() {
				defer GinkgoRecover()
				defer moved.Store(true)
				for _, g := range groups {
					Expect(svc.NewWriter(nil).SetGroup(ctx, &g.Key, proj.Key)).
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
			owner := createProject(ctx, "owner")
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
					joinErr = svc.NewWriter(nil).SetGroup(ctx, &g.Key, proj.Key)
				})
				wg.Go(func() {
					moveErr = groupSvc.NewWriter(nil).
						SetParent(ctx, new(owner.ResourceID()), g.Key)
				})
				wg.Wait()
				Expect([]error{joinErr, moveErr}).To(ContainElement(HaveOccurred()))
			}
		},
	)
})
