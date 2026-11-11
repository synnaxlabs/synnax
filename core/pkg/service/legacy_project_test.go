// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// The bundles under testdata/legacy-projects are workspace exports written by shipped
// Console builds, reformatted by the repo formatter. They are the only record of the
// version 0 format, so they run against the whole service layer: the registry a
// released Core assembles, with every leaf importer and task config store wired.
var _ = Describe("Legacy project bundles", func() {
	openLayer := func(ctx SpecContext) (*service.Layer, *gorp.DB) {
		GinkgoHelper()
		node := testutil.NewNode(ctx)
		return MustOpen(svctestutil.OpenLayer(ctx, node)), node.DB
	}

	// importBundle imports the bundle directory at path and returns the layer it landed
	// in, so a spec can read the records it produced.
	importBundle := func(
		ctx SpecContext, path, fileName string,
	) (*service.Layer, *gorp.DB, project.Project) {
		GinkgoHelper()
		l, db := openLayer(ctx)
		b := MustSucceed(l.Bundle.Decode(ctx, LoadBundle(path), fileName))
		tx := db.OpenTx()
		defer func() { Expect(tx.Close()).To(Succeed()) }()
		p := MustSucceed(l.Bundle.Import(ctx, tx, b))
		Expect(tx.Commit(ctx)).To(Succeed())
		return l, db, p
	}

	childrenOf := func(
		ctx SpecContext, db *gorp.DB, l *service.Layer, key project.Key,
	) []resource.Resource {
		GinkgoHelper()
		ids := MustSucceed(l.Project.Children(ctx, db, key))
		return MustSucceed(l.Resource.Retrieve(ctx, db, ids...))
	}

	typesOf := func(children []resource.Resource) []resource.Type {
		types := make([]resource.Type, len(children))
		for i, c := range children {
			types[i] = c.ID.Type
		}
		return types
	}

	Describe("A version 4 layout with key-named members", func() {
		const path = "testdata/legacy-projects/workspace-v4"

		It("Should name the project after the file", func(ctx SpecContext) {
			_, _, p := importBundle(ctx, path, "Test Stand.zip")
			Expect(p.Name).To(Equal("Test Stand"))
		})

		It("Should recreate both schematics and the table", func(ctx SpecContext) {
			l, db, p := importBundle(ctx, path, "Test Stand.zip")
			children := childrenOf(ctx, db, l, p.Key)
			Expect(typesOf(children)).To(ContainElements(
				resource.TypeSchematic,
				resource.TypeSchematic,
				resource.TypeTable,
			))
		})

		It("Should recreate the mosaic as a panel of resource tabs", func(
			ctx SpecContext,
		) {
			l, db, p := importBundle(ctx, path, "Test Stand.zip")
			children := childrenOf(ctx, db, l, p.Key)
			var panels []resource.Resource
			for _, c := range children {
				if c.ID.Type == resource.TypePanel {
					panels = append(panels, c)
				}
			}
			Expect(panels).To(HaveLen(1))
			keys := MustSucceed(panel.KeysFromResourceIDs(
				[]resource.ID{panels[0].ID},
			))
			var pnl panel.Panel
			Expect(l.Panel.NewRetrieve().
				Where(panel.MatchKeys(keys...)).
				Entry(&pnl).
				Exec(ctx, db)).To(Succeed())
			Expect(panel.ResourceRefs(pnl.Root)).ToNot(BeEmpty())
		})
	})

	Describe("A version 5 layout carrying task configs", func() {
		const path = "testdata/legacy-projects/hardware-v5"

		It("Should recreate the schematic, the line plot, and both tasks", func(
			ctx SpecContext,
		) {
			l, db, p := importBundle(ctx, path, "Hardware.zip")
			children := childrenOf(ctx, db, l, p.Key)
			Expect(typesOf(children)).To(ContainElements(
				resource.TypeSchematic,
				resource.TypeLineplot,
			))
			Expect(typesOf(children)).ToNot(ContainElement(resource.TypeLog))
			Expect(lo.Map(children, func(c resource.Resource, _ int) string {
				return c.Name
			})).To(ContainElements("Line Plot", "Schematic"))
			// A task parents under its rack's task group, never the project.
			Expect(typesOf(children)).ToNot(ContainElement(
				resource.TypeTask,
			))
			var tasks []task.Task
			Expect(l.Task.NewRetrieve().
				Where(task.MatchNames("NI Analog Read Task", "LabJack Write Task")).
				Entries(&tasks).
				Exec(ctx, db)).To(Succeed())
			Expect(lo.Map(tasks, func(t task.Task, _ int) string { return t.Type })).To(
				ConsistOf("ni_analog_read", "labjack_write"),
			)
		})
	})

	Describe("A version 5 layout whose task file the export dropped", func() {
		const path = "testdata/legacy-projects/missing-member-v5"

		It("Should reject the bundle, naming the layout it cannot resolve", func(
			ctx SpecContext,
		) {
			// The layout names an ni_analog_write task whose component file the export
			// never wrote. A member the bundle does not hold is a broken bundle, so the
			// import fails rather than dropping it quietly.
			l, _ := openLayer(ctx)
			Expect(l.Bundle.Decode(
				ctx, LoadBundle(path), "Dropped.zip",
			)).Error().To(SatisfyAll(
				MatchError(ContainSubstring("98dbf43a")),
				MatchError(ContainSubstring("not found")),
			))
		})
	})

	Describe("A version 8 layout with a name-named member", func() {
		const path = "testdata/legacy-projects/arc-editor-v8"

		It("Should resolve the member the layout names rather than keys", func(
			ctx SpecContext,
		) {
			l, db, p := importBundle(ctx, path, "Arc Workspace.zip")
			children := childrenOf(ctx, db, l, p.Key)
			Expect(typesOf(children)).To(ContainElement(
				resource.TypeSchematic,
			))
		})
	})
})
