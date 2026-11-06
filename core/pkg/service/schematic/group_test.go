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
	"fmt"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("SetGroup", func() {
	var (
		g      group.Group
		target schematic.Schematic
		other  schematic.Schematic
	)
	BeforeEach(func(ctx SpecContext) {
		g = group.Group{Name: "grouped", Parent: new(project.ResourceID(proj.Key))}
		Expect(groupSvc.NewWriter(nil).Create(ctx, &g)).To(Succeed())
		target = schematic.Schematic{Name: "target", Project: &proj.Key}
		Expect(svc.NewWriter(nil).Create(ctx, &target)).To(Succeed())
		other = schematic.Schematic{Name: "other", Project: &proj.Key}
		Expect(svc.NewWriter(nil).Create(ctx, &other)).To(Succeed())
	})
	retrieve := func(ctx SpecContext, key schematic.Key) schematic.Schematic {
		GinkgoHelper()
		var res schematic.Schematic
		Expect(svc.NewRetrieve().
			Where(schematic.MatchKeys(key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}
	retrieveGroup := func(ctx SpecContext, key schematic.Key) *group.Key {
		GinkgoHelper()
		return retrieve(ctx, key).Group
	}
	createProject := func(ctx SpecContext) project.Project {
		GinkgoHelper()
		p := project.Project{Name: "destination"}
		Expect(projectSvc.NewWriter(nil).Create(ctx, &p)).To(Succeed())
		return p
	}

	It(
		"Should move a schematic into a group and broadcast the move after commit",
		func(ctx SpecContext) {
			rec := &Recorder[schematic.Key, schematic.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			tx := DeferClose(db.OpenTx())
			Expect(svc.NewWriter(tx).SetGroup(ctx, &g.Key, target.Key)).To(Succeed())
			Expect(rec.Snapshot()).To(BeEmpty())
			Expect(tx.Commit(ctx)).To(Succeed())
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Key).To(Equal(target.Key))
			Expect(seen[0].Actions).To(Equal([]schematic.Action{
				schematic.NewSetGroupAction(schematic.SetGroupPayload{Group: &g.Key}),
			}))
			Expect(retrieveGroup(ctx, target.Key)).To(Equal(&g.Key))
		},
	)

	It(
		"Should retrieve schematics by whether they sit in a group",
		func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).SetGroup(ctx, &g.Key, target.Key)).To(Succeed())
			var grouped, ungrouped []schematic.Schematic
			Expect(svc.NewRetrieve().
				Where(schematic.MatchKeys(target.Key, other.Key)).
				Where(schematic.MatchHasGroup(true)).
				Entries(&grouped).
				Exec(ctx, db)).To(Succeed())
			Expect(grouped).To(HaveLen(1))
			Expect(grouped[0].Key).To(Equal(target.Key))
			Expect(svc.NewRetrieve().
				Where(schematic.MatchKeys(target.Key, other.Key)).
				Where(schematic.MatchHasGroup(false)).
				Entries(&ungrouped).
				Exec(ctx, db)).To(Succeed())
			Expect(ungrouped).To(HaveLen(1))
			Expect(ungrouped[0].Key).To(Equal(other.Key))
		},
	)

	It(
		"Should move a schematic into the project that holds the group",
		func(ctx SpecContext) {
			dest := createProject(ctx)
			inDest := group.Group{
				Name:   "in destination",
				Parent: new(project.ResourceID(dest.Key)),
			}
			Expect(groupSvc.NewWriter(nil).Create(ctx, &inDest)).To(Succeed())
			Expect(svc.NewWriter(nil).SetGroup(ctx, &inDest.Key, target.Key)).
				To(Succeed())
			res := retrieve(ctx, target.Key)
			Expect(res.Group).To(Equal(&inDest.Key))
			Expect(res.Project).To(Equal(&dest.Key))
		},
	)

	It("Should reject a group under no project", func(ctx SpecContext) {
		top := group.Group{Name: "top", Scope: resource.TypeProject}
		Expect(groupSvc.NewWriter(nil).Create(ctx, &top)).To(Succeed())
		Expect(svc.NewWriter(nil).SetGroup(ctx, &top.Key, target.Key)).To(MatchError(
			"group: a schematic sits only in a group under a project: validation error",
		))
		Expect(retrieveGroup(ctx, target.Key)).To(BeNil())
	})

	It(
		"Should move a schematic out of its group when it moves into a project",
		func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).SetGroup(ctx, &g.Key, target.Key)).To(Succeed())
			dest := createProject(ctx)
			Expect(svc.NewWriter(nil).SetProject(ctx, dest.Key, target.Key)).
				To(Succeed())
			res := retrieve(ctx, target.Key)
			Expect(res.Group).To(BeNil())
			Expect(res.Project).To(Equal(&dest.Key))
		},
	)

	It(
		"Should move the schematics in a group into the project the group moves under",
		func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).SetGroup(ctx, &g.Key, target.Key)).To(Succeed())
			dest := createProject(ctx)
			Expect(svc.MoveGroups(
				ctx,
				db,
				[]group.Key{g.Key},
				new(project.ResourceID(dest.Key)),
			)).To(Succeed())
			res := retrieve(ctx, target.Key)
			Expect(res.Group).To(Equal(&g.Key))
			Expect(res.Project).To(Equal(&dest.Key))
		},
	)

	It(
		"Should reject a group move that leaves a schematic under no project",
		func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).SetGroup(ctx, &g.Key, target.Key)).To(Succeed())
			Expect(svc.MoveGroups(ctx, db, []group.Key{g.Key}, nil)).To(MatchError(
				"group: a schematic sits only in a group under a project: validation error",
			))
		},
	)

	It("Should reject a group of another scope", func(ctx SpecContext) {
		channels := group.Group{Name: "channels", Scope: resource.TypeChannel}
		Expect(groupSvc.NewWriter(nil).Create(ctx, &channels)).To(Succeed())
		Expect(svc.NewWriter(nil).SetGroup(ctx, &channels.Key, target.Key)).
			To(MatchError(
				"group: a schematic cannot sit in a channel group: validation error",
			))
		Expect(retrieveGroup(ctx, target.Key)).To(BeNil())
	})

	It("Should reject a client dispatch of a group change", func(ctx SpecContext) {
		Expect(svc.Dispatch(ctx, target.Key, "d1", []schematic.Action{
			schematic.NewSetGroupAction(schematic.SetGroupPayload{Group: &g.Key}),
		})).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"only the Core dispatches a schematic group change",
			)),
		))
		Expect(retrieveGroup(ctx, target.Key)).To(BeNil())
	})
})

var _ = Describe("Writer with no transaction", func() {
	It(
		"Should keep the group that a move writes during concurrent creates",
		func(ctx SpecContext) {
			groups := make([]group.Group, 50)
			for i := range groups {
				groups[i] = group.Group{
					Name:   fmt.Sprintf("Concurrent %d", i),
					Parent: new(project.ResourceID(proj.Key)),
				}
				Expect(groupSvc.NewWriter(nil).Create(ctx, &groups[i])).To(Succeed())
			}
			DeferCleanup(func(ctx SpecContext) {
				for _, g := range groups {
					Expect(groupSvc.NewWriter(nil).Delete(ctx, g.Key)).To(Succeed())
				}
			})
			created := schematic.Schematic{Name: "concurrent", Project: &proj.Key}
			Expect(svc.NewWriter(nil).Create(ctx, &created)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(svc.NewWriter(nil).Delete(ctx, created.Key)).To(Succeed())
			})
			retrieveGroup := func() *group.Key {
				GinkgoHelper()
				var res schematic.Schematic
				Expect(svc.NewRetrieve().
					Where(schematic.MatchKeys(created.Key)).
					Entry(&res).
					Exec(ctx, db)).To(Succeed())
				return res.Group
			}
			var (
				wg    sync.WaitGroup
				moved atomic.Bool
			)
			for range 4 {
				wg.Go(func() {
					defer GinkgoRecover()
					for !moved.Load() {
						again := created
						Expect(svc.NewWriter(nil).Create(ctx, &again)).
							To(Succeed())
					}
				})
			}
			wg.Go(func() {
				defer GinkgoRecover()
				defer moved.Store(true)
				for _, g := range groups {
					Expect(svc.NewWriter(nil).SetGroup(ctx, &g.Key, created.Key)).
						To(Succeed())
					Expect(retrieveGroup()).To(Equal(&g.Key))
				}
			})
			wg.Wait()
			Expect(retrieveGroup()).To(Equal(&groups[len(groups)-1].Key))
		},
	)

	It(
		"Should keep a schematic in the project of its group during concurrent moves",
		func(ctx SpecContext) {
			dest := project.Project{Name: "destination"}
			Expect(projectSvc.NewWriter(nil).Create(ctx, &dest)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(projectSvc.NewWriter(nil).Delete(ctx, dest.Key)).To(Succeed())
			})
			projects := []project.Key{proj.Key, dest.Key}
			groups := make([]group.Group, len(projects))
			for i, p := range projects {
				groups[i] = group.Group{
					Name:   "moving",
					Parent: new(project.ResourceID(p)),
				}
				Expect(groupSvc.NewWriter(nil).Create(ctx, &groups[i])).To(Succeed())
			}
			DeferCleanup(func(ctx SpecContext) {
				for _, g := range groups {
					Expect(groupSvc.NewWriter(nil).Delete(ctx, g.Key)).To(Succeed())
				}
			})
			moving := schematic.Schematic{Name: "moving", Project: &proj.Key}
			Expect(svc.NewWriter(nil).Create(ctx, &moving)).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(svc.NewWriter(nil).Delete(ctx, moving.Key)).To(Succeed())
			})
			groupProject := map[group.Key]project.Key{
				groups[0].Key: proj.Key,
				groups[1].Key: dest.Key,
			}
			var (
				wg   sync.WaitGroup
				done atomic.Bool
			)
			wg.Go(func() {
				defer GinkgoRecover()
				for i := 0; !done.Load(); i++ {
					g := groups[i%len(groups)].Key
					Expect(svc.NewWriter(nil).SetGroup(ctx, &g, moving.Key)).
						To(Succeed())
				}
			})
			wg.Go(func() {
				defer GinkgoRecover()
				for i := 0; !done.Load(); i++ {
					p := projects[i%len(projects)]
					Expect(svc.NewWriter(nil).SetProject(ctx, p, moving.Key)).
						To(Succeed())
				}
			})
			wg.Go(func() {
				defer GinkgoRecover()
				defer done.Store(true)
				for range 200 {
					var res schematic.Schematic
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewRetrieve().
							Where(schematic.MatchKeys(moving.Key)).
							Entry(&res).
							Exec(ctx, tx)
					})).To(Succeed())
					if res.Group != nil {
						Expect(res.Project).To(Equal(new(groupProject[*res.Group])))
					}
				}
			})
			wg.Wait()
		},
	)
})
