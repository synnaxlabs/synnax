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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/validate"
)

type vis struct {
	Key     uuid.UUID
	Group   *group.Key
	Project *project.Key
}

func (v vis) GorpKey() uuid.UUID { return v.Key }

func groupOf(v vis) *group.Key { return v.Group }

func ownerOf(v vis) *resource.ID { return project.OwnerID(v.Project, nil) }

// createProject creates a project and returns it.
func createProject(ctx SpecContext) project.Project {
	GinkgoHelper()
	p := project.Project{Name: "Owner"}
	Expect(writer.Create(ctx, &p)).To(Succeed())
	return p
}

// createGroup creates a group under parent with scope and returns it.
func createGroup(
	ctx SpecContext,
	parent *resource.ID,
	scope resource.Type,
) group.Group {
	GinkgoHelper()
	g := group.Group{Name: "Owner", Parent: parent, Scope: scope}
	Expect(groupSvc.NewWriter(nil).Create(ctx, &g)).To(Succeed())
	return g
}

var _ = Describe("ValidateOwner", func() {
	It("Should accept a project alone", func() {
		Expect(project.ValidateOwner(new(uuid.New()), nil)).To(Succeed())
	})

	It("Should accept a range alone", func() {
		Expect(project.ValidateOwner(nil, new(ranger.Key(uuid.New())))).To(Succeed())
	})

	DescribeTable(
		"Should reject anything but exactly one owner",
		func(proj *project.Key, rng *ranger.Key) {
			Expect(project.ValidateOwner(proj, rng)).To(MatchError(ContainSubstring(
				"project: must belong to exactly one of a project and a range",
			)))
		},
		Entry("neither", nil, nil),
		Entry("both", new(uuid.New()), new(ranger.Key(uuid.New()))),
	)
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

var _ = Describe("OwnerID", func() {
	It("Should return the project", func() {
		k := uuid.New()
		Expect(project.OwnerID(&k, nil)).To(Equal(new(project.ResourceID(k))))
	})

	It("Should return the range", func() {
		k := ranger.Key(uuid.New())
		Expect(project.OwnerID(nil, &k)).To(Equal(new(ranger.ResourceID(k))))
	})

	It("Should return nil with no owner", func() {
		Expect(project.OwnerID(nil, nil)).To(BeNil())
	})
})

var _ = Describe("CreateObjects", func() {
	It("Should return the owner a new visualization carries", func(ctx SpecContext) {
		k := uuid.New()
		Expect(project.CreateObjects(
			ctx,
			db,
			resource.TypeSchematic,
			[]vis{{Key: uuid.New(), Project: &k}},
			nil,
			groupOf,
			ownerOf,
		)).To(Equal([]resource.ID{project.ResourceID(k)}))
	})

	It(
		"Should return the project of the group a new visualization joins",
		func(ctx SpecContext) {
			p := createProject(ctx)
			g := createGroup(ctx, new(p.ResourceID()), resource.TypeProject)
			Expect(project.CreateObjects(
				ctx,
				db,
				resource.TypeSchematic,
				[]vis{{Key: uuid.New(), Project: new(uuid.New()), Group: &g.Key}},
				nil,
				groupOf,
				ownerOf,
			)).To(Equal([]resource.ID{p.ResourceID()}))
		},
	)

	It("Should add nothing for a stored visualization", func(ctx SpecContext) {
		v := vis{Key: uuid.New(), Project: new(uuid.New())}
		Expect(project.CreateObjects(
			ctx,
			db,
			resource.TypeSchematic,
			[]vis{v},
			[]vis{v},
			groupOf,
			ownerOf,
		)).To(BeEmpty())
	})

	It("Should add nothing for a visualization with no owner", func(ctx SpecContext) {
		Expect(project.CreateObjects(
			ctx,
			db,
			resource.TypeSchematic,
			[]vis{{Key: uuid.New()}},
			nil,
			groupOf,
			ownerOf,
		)).To(BeEmpty())
	})

	It("Should return each owner once", func(ctx SpecContext) {
		k := uuid.New()
		Expect(project.CreateObjects(
			ctx,
			db,
			resource.TypeSchematic,
			[]vis{{Key: uuid.New(), Project: &k}, {Key: uuid.New(), Project: &k}},
			nil,
			groupOf,
			ownerOf,
		)).To(Equal([]resource.ID{project.ResourceID(k)}))
	})

	It("Should return the error of a group that cannot hold the type", func(
		ctx SpecContext,
	) {
		g := createGroup(ctx, nil, resource.TypeChannel)
		Expect(project.CreateObjects(
			ctx,
			db,
			resource.TypeSchematic,
			[]vis{{Key: uuid.New(), Group: &g.Key}},
			nil,
			groupOf,
			ownerOf,
		)).Error().To(MatchError(ContainSubstring(
			"group: a schematic cannot sit in a channel group",
		)))
	})
})

var _ = Describe("SetGroupObjects", func() {
	It("Should return nothing for a nil group", func(ctx SpecContext) {
		Expect(project.SetGroupObjects(
			ctx,
			db,
			nil,
			resource.TypeSchematic,
			[]*resource.ID{new(project.ResourceID(uuid.New()))},
		)).To(BeEmpty())
	})

	It("Should return nothing for a move in the same project", func(ctx SpecContext) {
		p := createProject(ctx)
		g := createGroup(ctx, new(p.ResourceID()), resource.TypeProject)
		Expect(project.SetGroupObjects(
			ctx,
			db,
			&g.Key,
			resource.TypeSchematic,
			[]*resource.ID{new(p.ResourceID())},
		)).To(BeEmpty())
	})

	It(
		"Should return the owners left and the project joined",
		func(ctx SpecContext) {
			p := createProject(ctx)
			g := createGroup(ctx, new(p.ResourceID()), resource.TypeProject)
			from := project.ResourceID(uuid.New())
			Expect(project.SetGroupObjects(
				ctx,
				db,
				&g.Key,
				resource.TypeSchematic,
				[]*resource.ID{&from, new(p.ResourceID()), &from},
			)).To(Equal([]resource.ID{from, p.ResourceID()}))
		},
	)

	It("Should return the error of a group under no project", func(ctx SpecContext) {
		g := createGroup(ctx, nil, resource.TypeProject)
		Expect(project.SetGroupObjects(
			ctx,
			db,
			&g.Key,
			resource.TypeSchematic,
			nil,
		)).Error().To(MatchError(ContainSubstring(
			"group: a schematic sits only in a group under a project",
		)))
	})
})

var _ = Describe("MoveObjects", func() {
	It("Should return the IDs, the owners left, and the owners joined", func() {
		ids := []resource.ID{{Type: resource.TypeSchematic, Key: "a"}}
		from, to := project.ResourceID(uuid.New()), project.ResourceID(uuid.New())
		Expect(project.MoveObjects(ids, []*resource.ID{&from}, &to)).
			To(Equal([]resource.ID{ids[0], from, to}))
	})

	It("Should skip a nil owner", func() {
		to := project.ResourceID(uuid.New())
		Expect(project.MoveObjects(nil, []*resource.ID{nil}, &to, nil)).
			To(Equal([]resource.ID{to}))
	})

	It("Should return each owner once", func() {
		to := project.ResourceID(uuid.New())
		Expect(project.MoveObjects(nil, []*resource.ID{&to}, &to)).
			To(Equal([]resource.ID{to}))
	})
})
