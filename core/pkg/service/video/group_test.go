// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/video"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("SetGroup", func() {
	var (
		g      group.Group
		target video.Video
	)
	BeforeEach(func(ctx SpecContext) {
		g = group.Group{Name: "grouped", Parent: new(project.ResourceID(proj.Key))}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		target = create(ctx, video.Video{Name: "target"})
	})
	createProject := func(ctx SpecContext) project.Project {
		GinkgoHelper()
		p := project.Project{Name: "destination"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return projectSvc.NewWriter(tx).Create(ctx, &p)
		})).To(Succeed())
		return p
	}

	It(
		"Should move a view into a group and broadcast the move after commit",
		func(ctx SpecContext) {
			rec := &Recorder[video.Key, video.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			tx := DeferClose(db.OpenTx())
			Expect(svc.NewWriter(tx).SetGroup(ctx, &g.Key, target.Key)).To(Succeed())
			Expect(rec.Snapshot()).To(BeEmpty())
			Expect(tx.Commit(ctx)).To(Succeed())
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Actions).To(Equal([]video.Action{
				video.NewSetAction(video.SetPayload{
					Path:  []string{"group"},
					Value: g.Key,
				}),
			}))
			Expect(retrieve(ctx, target.Key).Group).To(Equal(&g.Key))
		},
	)

	It("Should match views by whether they sit in a group", func(ctx SpecContext) {
		other := create(ctx, video.Video{Name: "other"})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).SetGroup(ctx, &g.Key, target.Key)
		})).To(Succeed())
		var grouped []video.Video
		Expect(svc.NewRetrieve().
			Where(video.MatchKeys(target.Key, other.Key)).
			Where(video.MatchGrouped(true)).
			Entries(&grouped).
			Exec(ctx, db)).To(Succeed())
		Expect(grouped).To(ConsistOf(HaveField("Key", target.Key)))
	})

	It(
		"Should move a view into the project that holds the group",
		func(ctx SpecContext) {
			dest := createProject(ctx)
			inDest := group.Group{
				Name:   "in destination",
				Parent: new(project.ResourceID(dest.Key)),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Create(ctx, &inDest)
			})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).SetGroup(ctx, &inDest.Key, target.Key)
			})).To(Succeed())
			res := retrieve(ctx, target.Key)
			Expect(res.Group).To(Equal(&inDest.Key))
			Expect(res.Project).To(Equal(&dest.Key))
		},
	)

	It("Should reject a group under no project", func(ctx SpecContext) {
		top := group.Group{Name: "top", Scope: resource.TypeProject}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &top)
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).SetGroup(ctx, &top.Key, target.Key)
		})).To(MatchError(
			"group: a video sits only in a group under a project: validation error",
		))
		Expect(retrieve(ctx, target.Key).Group).To(BeNil())
	})

	It(
		"Should move a view out of its group when it moves into a project",
		func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).SetGroup(ctx, &g.Key, target.Key)
			})).To(Succeed())
			dest := createProject(ctx)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).SetProject(ctx, dest.Key, target.Key)
			})).To(Succeed())
			res := retrieve(ctx, target.Key)
			Expect(res.Group).To(BeNil())
			Expect(res.Project).To(Equal(&dest.Key))
		},
	)

	It(
		"Should move the views in a group into the project the group moves under",
		func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).SetGroup(ctx, &g.Key, target.Key)
			})).To(Succeed())
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

	It("Should reject a client patch of the group", func(ctx SpecContext) {
		Expect(dispatch(ctx, target.Key, "d1", []video.Action{
			video.NewSetAction(video.SetPayload{
				Path:  []string{"group"},
				Value: g.Key,
			}),
		})).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`unknown object member name "group"`)),
		))
		Expect(retrieve(ctx, target.Key).Group).To(BeNil())
	})
})
