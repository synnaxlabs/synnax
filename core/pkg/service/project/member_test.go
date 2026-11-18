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
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// createProject creates a project and returns it.
func createProject(ctx SpecContext) project.Project {
	GinkgoHelper()
	p := project.Project{Name: "Holder"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.NewWriter(tx).Create(ctx, &p)
	})).To(Succeed())
	return p
}

// createGroup creates a group under parent with scope and returns it.
func createGroup(
	ctx SpecContext,
	parent *resource.ID,
	scope resource.Type,
) group.Group {
	GinkgoHelper()
	g := group.Group{Name: "Holder", Parent: parent, Scope: scope}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return groupSvc.NewWriter(tx).Create(ctx, &g)
	})).To(Succeed())
	return g
}

var _ = Describe("CheckMember", func() {
	It("Should accept a project", func() {
		Expect(project.CheckMember(new(uuid.New()))).To(Succeed())
	})

	It("Should reject no project", func() {
		Expect(project.CheckMember(nil)).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("project: must belong to a project")),
		))
	})
})

var _ = Describe("GroupProject", func() {
	It("Should return nil for a nil key", func(ctx SpecContext) {
		Expect(project.GroupProject(ctx, db, nil, resource.TypeSchematic)).To(BeNil())
	})

	It("Should return the project of a group under a project", func(ctx SpecContext) {
		p := createProject(ctx)
		g := createGroup(ctx, new(p.ResourceID()), resource.TypeProject)
		Expect(project.GroupProject(ctx, db, &g.Key, resource.TypeSchematic)).
			To(Equal(&p.Key))
	})

	It("Should return the top project of a nested group", func(ctx SpecContext) {
		p := createProject(ctx)
		top := createGroup(ctx, new(p.ResourceID()), resource.TypeProject)
		g := createGroup(ctx, new(top.ResourceID()), resource.TypeProject)
		Expect(project.GroupProject(ctx, db, &g.Key, resource.TypeSchematic)).
			To(Equal(&p.Key))
	})

	It("Should refuse a group under no project", func(ctx SpecContext) {
		g := createGroup(ctx, nil, resource.TypeProject)
		Expect(project.GroupProject(ctx, db, &g.Key, resource.TypeSchematic)).Error().
			To(MatchError(ContainSubstring(
				"group: a schematic sits only in a group under a project",
			)))
	})

	It("Should refuse a group whose scope cannot hold the type", func(ctx SpecContext) {
		g := createGroup(ctx, nil, resource.TypeChannel)
		Expect(project.GroupProject(ctx, db, &g.Key, resource.TypeSchematic)).Error().
			To(MatchError(ContainSubstring(
				"group: a schematic cannot sit in a channel group",
			)))
	})
})

var _ = Describe("ValidateGroup", func() {
	It("Should accept a nil group", func(ctx SpecContext) {
		p := createProject(ctx)
		Expect(project.ValidateGroup(ctx, db, p.Key, nil, resource.TypeLineplot)).
			To(Succeed())
	})

	It("Should accept a group in the project", func(ctx SpecContext) {
		p := createProject(ctx)
		g := createGroup(ctx, new(p.ResourceID()), resource.TypeProject)
		Expect(project.ValidateGroup(ctx, db, p.Key, &g.Key, resource.TypeLineplot)).
			To(Succeed())
	})

	It("Should refuse a group in another project", func(ctx SpecContext) {
		p := createProject(ctx)
		other := createProject(ctx)
		g := createGroup(ctx, new(other.ResourceID()), resource.TypeProject)
		Expect(project.ValidateGroup(ctx, db, p.Key, &g.Key, resource.TypeLineplot)).
			To(MatchError(ContainSubstring(fmt.Sprintf(
				"group: group %s is not in project %s",
				g.Key,
				p.Key,
			))))
	})
})

var _ = Describe("TopParentProject", func() {
	It("Should return the project at the top", func() {
		k := uuid.New()
		Expect(project.TopParentProject(
			new(project.ResourceID(k)),
			resource.TypeSchematic,
		)).To(Equal(&k))
	})

	DescribeTable(
		"Should refuse a top parent that is not a project",
		func(top *resource.ID) {
			Expect(project.TopParentProject(top, resource.TypeSchematic)).Error().
				To(MatchError(ContainSubstring(
					"group: a schematic sits only in a group under a project",
				)))
		},
		Entry("no top parent", nil),
		Entry("a range", new(ranger.ResourceID(uuid.New()))),
	)

	It("Should refuse a project key that is not a UUID", func() {
		Expect(project.TopParentProject(
			&resource.ID{Type: resource.TypeProject, Key: "bad"},
			resource.TypeSchematic,
		)).Error().To(MatchError(validate.ErrValidation))
	})
})

var _ = Describe("Place", func() {
	var (
		p, other   project.Project
		g, otherG  group.Group
		placements = func(
			ctx SpecContext,
			prev, next project.Placement,
		) (project.Placement, []patch.Action, error) {
			return project.Place(
				ctx,
				db,
				resource.TypeLineplot,
				prev,
				next,
				func(s patch.Set) patch.Action { return s },
				func(r patch.Remove) patch.Action { return r },
			)
		}
	)
	BeforeEach(func(ctx SpecContext) {
		p, other = createProject(ctx), createProject(ctx)
		g = createGroup(ctx, new(p.ResourceID()), resource.TypeProject)
		otherG = createGroup(ctx, new(other.ResourceID()), resource.TypeProject)
	})

	It("Should keep a placement that the edit does not change", func(ctx SpecContext) {
		at := project.Placement{Group: &g.Key, Project: &p.Key}
		placed, added := MustSucceed2(placements(ctx, at, at))
		Expect(placed).To(Equal(at))
		Expect(added).To(BeEmpty())
	})

	It("Should keep the project of a move into a group in it", func(ctx SpecContext) {
		next := project.Placement{Group: &g.Key, Project: &p.Key}
		placed, added := MustSucceed2(
			placements(ctx, project.Placement{Project: &p.Key}, next),
		)
		Expect(placed).To(Equal(next))
		Expect(added).To(BeEmpty())
	})

	It(
		"Should set the project of a move into a group of another project",
		func(ctx SpecContext) {
			placed, added := MustSucceed2(placements(
				ctx,
				project.Placement{Group: &g.Key, Project: &p.Key},
				project.Placement{Group: &otherG.Key, Project: &p.Key},
			))
			Expect(placed).To(Equal(
				project.Placement{Group: &otherG.Key, Project: &other.Key},
			))
			Expect(added).To(Equal([]patch.Action{
				patch.Set{Path: []string{"project"}, Value: other.Key},
			}))
		},
	)

	It(
		"Should set the project of the group when an edit moves both",
		func(ctx SpecContext) {
			placed, added := MustSucceed2(placements(
				ctx,
				project.Placement{Project: &p.Key},
				project.Placement{Group: &otherG.Key, Project: &p.Key},
			))
			Expect(placed.Project).To(Equal(&other.Key))
			Expect(added).To(Equal([]patch.Action{
				patch.Set{Path: []string{"project"}, Value: other.Key},
			}))
		},
	)

	It("Should keep the project of a move out of a group", func(ctx SpecContext) {
		next := project.Placement{Project: &p.Key}
		placed, added := MustSucceed2(placements(
			ctx,
			project.Placement{Group: &g.Key, Project: &p.Key},
			next,
		))
		Expect(placed).To(Equal(next))
		Expect(added).To(BeEmpty())
	})

	It(
		"Should remove the group of a move into another project",
		func(ctx SpecContext) {
			placed, added := MustSucceed2(placements(
				ctx,
				project.Placement{Group: &g.Key, Project: &p.Key},
				project.Placement{Group: &g.Key, Project: &other.Key},
			))
			Expect(placed).To(Equal(project.Placement{Project: &other.Key}))
			Expect(added).To(Equal([]patch.Action{patch.Remove{Key: "group"}}))
		},
	)

	It(
		"Should add nothing to a move into another project with no group",
		func(ctx SpecContext) {
			next := project.Placement{Project: &other.Key}
			placed, added := MustSucceed2(
				placements(ctx, project.Placement{Project: &p.Key}, next),
			)
			Expect(placed).To(Equal(next))
			Expect(added).To(BeEmpty())
		},
	)

	It("Should refuse a group that does not exist", func(ctx SpecContext) {
		missing := uuid.New()
		Expect(placements(
			ctx,
			project.Placement{Project: &p.Key},
			project.Placement{Group: &missing, Project: &p.Key},
		)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("group: group %s does not exist", missing)),
		))
	})

	It("Should refuse a move into the zero group", func(ctx SpecContext) {
		Expect(placements(
			ctx,
			project.Placement{Project: &p.Key},
			project.Placement{Group: new(uuid.Nil()), Project: &p.Key},
		)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("group: group %s does not exist", uuid.Nil())),
		))
	})
})
