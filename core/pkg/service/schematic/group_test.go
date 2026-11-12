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
	"uuid"

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

var _ = Describe("Apply of a move", func() {
	var (
		g      group.Group
		target schematic.Schematic
		other  schematic.Schematic
	)
	BeforeEach(func(ctx SpecContext) {
		g = group.Group{Name: "grouped", Parent: new(project.ResourceID(proj.Key))}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		target = schematic.Schematic{Name: "target", Project: &proj.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &target)
		})).To(Succeed())
		other = schematic.Schematic{Name: "other", Project: &proj.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &other)
		})).To(Succeed())
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
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return projectSvc.NewWriter(tx).Create(ctx, &p)
		})).To(Succeed())
		return p
	}
	apply := func(
		ctx SpecContext,
		key schematic.Key,
		acts ...schematic.Action,
	) ([]schematic.Action, error) {
		var added []schematic.Action
		err := db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			added, err = svc.NewWriter(tx).Apply(ctx, key, acts...)
			return err
		})
		return added, err
	}

	It(
		"Should move a schematic into a group and broadcast the move after commit",
		func(ctx SpecContext) {
			rec := &Recorder[schematic.Key, schematic.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			tx := DeferClose(db.OpenTx())
			Expect(svc.NewWriter(tx).Apply(ctx, target.Key, setGroup(g.Key))).
				To(BeEmpty())
			Expect(rec.Snapshot()).To(BeEmpty())
			Expect(tx.Commit(ctx)).To(Succeed())
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Key).To(Equal(target.Key))
			Expect(seen[0].Actions).To(Equal([]schematic.Action{setGroup(g.Key)}))
			Expect(retrieveGroup(ctx, target.Key)).To(Equal(&g.Key))
		},
	)

	It(
		"Should retrieve schematics by whether they sit in a group",
		func(ctx SpecContext) {
			Expect(apply(ctx, target.Key, setGroup(g.Key))).To(BeEmpty())
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
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Create(ctx, &inDest)
			})).To(Succeed())
			rec := &Recorder[schematic.Key, schematic.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(apply(ctx, target.Key, setGroup(inDest.Key))).
				To(Equal([]schematic.Action{setProject(dest.Key)}))
			res := retrieve(ctx, target.Key)
			Expect(res.Group).To(Equal(&inDest.Key))
			Expect(res.Project).To(Equal(&dest.Key))
			Expect(rec.Snapshot()).To(HaveExactElements(HaveField(
				"Actions",
				[]schematic.Action{setGroup(inDest.Key), setProject(dest.Key)},
			)))
		},
	)

	It("Should move a schematic out of its group", func(ctx SpecContext) {
		Expect(apply(ctx, target.Key, setGroup(g.Key))).To(BeEmpty())
		Expect(apply(ctx, target.Key, removeGroup)).To(BeEmpty())
		res := retrieve(ctx, target.Key)
		Expect(res.Group).To(BeNil())
		Expect(res.Project).To(Equal(&proj.Key))
	})

	It("Should reject a group under no project", func(ctx SpecContext) {
		top := group.Group{Name: "top", Scope: resource.TypeProject}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &top)
		})).To(Succeed())
		Expect(apply(ctx, target.Key, setGroup(top.Key))).Error().To(MatchError(
			"group: a schematic sits only in a group under a project: validation error",
		))
		Expect(retrieveGroup(ctx, target.Key)).To(BeNil())
	})

	It("Should reject a group that does not exist", func(ctx SpecContext) {
		missing := uuid.New()
		Expect(apply(ctx, target.Key, setGroup(missing))).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("group: group %s does not exist", missing)),
		))
		Expect(retrieveGroup(ctx, target.Key)).To(BeNil())
	})

	It(
		"Should move a schematic out of its group when it moves into a project",
		func(ctx SpecContext) {
			Expect(apply(ctx, target.Key, setGroup(g.Key))).To(BeEmpty())
			dest := createProject(ctx)
			rec := &Recorder[schematic.Key, schematic.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(apply(ctx, target.Key, setProject(dest.Key))).
				To(Equal([]schematic.Action{removeGroup}))
			res := retrieve(ctx, target.Key)
			Expect(res.Group).To(BeNil())
			Expect(res.Project).To(Equal(&dest.Key))
			Expect(rec.Snapshot()).To(HaveExactElements(HaveField(
				"Actions",
				[]schematic.Action{setProject(dest.Key), removeGroup},
			)))
		},
	)

	It("Should reject a remove of the project", func(ctx SpecContext) {
		Expect(apply(
			ctx,
			target.Key,
			schematic.NewRemoveAction(schematic.RemovePayload{Key: "project"}),
		)).Error().To(MatchError(ContainSubstring(
			"project: must belong to exactly one of a project and a range",
		)))
		Expect(retrieve(ctx, target.Key).Project).To(Equal(&proj.Key))
	})

	It(
		"Should move the schematics in a group into the project the group moves under",
		func(ctx SpecContext) {
			Expect(apply(ctx, target.Key, setGroup(g.Key))).To(BeEmpty())
			dest := createProject(ctx)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.MoveGroups(
					ctx,
					tx,
					[]group.Key{g.Key},
					new(project.ResourceID(dest.Key)),
				)
			})).To(Succeed())
			res := retrieve(ctx, target.Key)
			Expect(res.Group).To(Equal(&g.Key))
			Expect(res.Project).To(Equal(&dest.Key))
		},
	)

	It(
		"Should reject a group move that leaves a schematic under no project",
		func(ctx SpecContext) {
			Expect(apply(ctx, target.Key, setGroup(g.Key))).To(BeEmpty())
			Expect(svc.MoveGroups(ctx, db, []group.Key{g.Key}, nil)).To(MatchError(
				"group: a schematic sits only in a group under a project: validation error",
			))
		},
	)

	It("Should reject a group of another scope", func(ctx SpecContext) {
		channels := group.Group{Name: "channels", Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &channels)
		})).To(Succeed())
		Expect(apply(ctx, target.Key, setGroup(channels.Key))).Error().To(MatchError(
			"group: a schematic cannot sit in a channel group: validation error",
		))
		Expect(retrieveGroup(ctx, target.Key)).To(BeNil())
	})
})

var _ = Describe("Writer in concurrent transactions", func() {
	It(
		"Should keep the group that a move writes during concurrent applies",
		func(ctx SpecContext) {
			groups := make([]group.Group, 50)
			for i := range groups {
				groups[i] = group.Group{
					Name:   fmt.Sprintf("Concurrent %d", i),
					Parent: new(project.ResourceID(proj.Key)),
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return groupSvc.NewWriter(tx).Create(ctx, &groups[i])
				})).To(Succeed())
			}
			DeferCleanup(func(ctx SpecContext) {
				for _, g := range groups {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return groupSvc.NewWriter(tx).Delete(ctx, g.Key)
					})).To(Succeed())
				}
			})
			created := schematic.Schematic{Name: "concurrent", Project: &proj.Key}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &created)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, created.Key)
				})).To(Succeed())
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
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							_, err := svc.NewWriter(tx).Apply(
								ctx,
								created.Key,
								schematic.NewSetAction(schematic.SetPayload{
									Path:  []string{"name"},
									Value: "renamed",
								}),
							)
							return err
						})).
							To(Succeed())
					}
				})
			}
			wg.Go(func() {
				defer GinkgoRecover()
				defer moved.Store(true)
				for _, g := range groups {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						_, err := svc.NewWriter(tx).
							Apply(ctx, created.Key, setGroup(g.Key))
						return err
					})).
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
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return projectSvc.NewWriter(tx).Create(ctx, &dest)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return projectSvc.NewWriter(tx).Delete(ctx, dest.Key)
				})).To(Succeed())
			})
			projects := []project.Key{proj.Key, dest.Key}
			groups := make([]group.Group, len(projects))
			for i, p := range projects {
				groups[i] = group.Group{
					Name:   "moving",
					Parent: new(project.ResourceID(p)),
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return groupSvc.NewWriter(tx).Create(ctx, &groups[i])
				})).To(Succeed())
			}
			DeferCleanup(func(ctx SpecContext) {
				for _, g := range groups {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return groupSvc.NewWriter(tx).Delete(ctx, g.Key)
					})).To(Succeed())
				}
			})
			moving := schematic.Schematic{Name: "moving", Project: &proj.Key}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &moving)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, moving.Key)
				})).To(Succeed())
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
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						_, err := svc.NewWriter(tx).Apply(ctx, moving.Key, setGroup(g))
						return err
					})).
						To(Succeed())
				}
			})
			wg.Go(func() {
				defer GinkgoRecover()
				for i := 0; !done.Load(); i++ {
					p := projects[i%len(projects)]
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						_, err := svc.NewWriter(tx).
							Apply(ctx, moving.Key, setProject(p))
						return err
					})).
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

// setGroup returns the action that moves a schematic into the group with key g.
func setGroup(g group.Key) schematic.Action {
	return schematic.NewSetAction(
		schematic.SetPayload{Path: []string{"group"}, Value: g},
	)
}

// setProject returns the action that moves a schematic into the project with key p.
func setProject(p project.Key) schematic.Action {
	return schematic.NewSetAction(
		schematic.SetPayload{Path: []string{"project"}, Value: p},
	)
}

// removeGroup is the action that moves a schematic out of its group.
var removeGroup = schematic.NewRemoveAction(schematic.RemovePayload{Key: "group"})
