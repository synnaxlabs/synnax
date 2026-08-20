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
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/security"
	secmock "github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/synnax/pkg/service"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
)

// The bundles under testdata/legacy-projects are workspace exports written by shipped
// Console builds. They are the only record of the version 0 format, so they run against
// the whole service layer: the registry a released Core assembles, with every leaf
// importer and task config store wired.
var _ = Describe("Legacy project bundles", func() {
	openLayer := func(ctx SpecContext) (*service.Layer, *gorp.DB) {
		GinkgoHelper()
		node := mock.NewNode(ctx)
		sec := MustSucceed(security.NewProvider(security.ProviderConfig{
			Insecure: new(true),
			KeySize:  secmock.SmallKeySize,
		}))
		return MustOpen(service.OpenLayer(ctx, service.LayerConfig{
			Distribution: node.Layer,
			Security:     sec,
			Storage:      node.Storage,
		})), node.DB
	}

	// importBundle imports the bundle directory at path and returns the layer it landed
	// in, so a spec can walk the ontology it produced.
	importBundle := func(
		ctx SpecContext, path, fileName string,
	) (*service.Layer, project.Project) {
		GinkgoHelper()
		l, db := openLayer(ctx)
		tx := db.OpenTx()
		defer func() { Expect(tx.Close()).To(Succeed()) }()
		p := MustSucceed(l.Project.Import(ctx, tx, LoadBundle(path), fileName))
		Expect(tx.Commit(ctx)).To(Succeed())
		return l, p
	}

	childrenOf := func(
		ctx SpecContext, l *service.Layer, id ontology.ID,
	) []ontology.Resource {
		GinkgoHelper()
		var children []ontology.Resource
		Expect(l.Ontology.NewRetrieve().
			WhereIDs(id).
			TraverseTo(ontology.ChildrenTraverser).
			Entries(&children).
			Exec(ctx, nil)).To(Succeed())
		return children
	}

	typesOf := func(children []ontology.Resource) []ontology.ResourceType {
		types := make([]ontology.ResourceType, len(children))
		for i, c := range children {
			types[i] = c.ID.Type
		}
		return types
	}

	Describe("A version 4 layout with key-named members", func() {
		const path = "testdata/legacy-projects/workspace-v4"

		It("Should name the project after the file", func(ctx SpecContext) {
			_, p := importBundle(ctx, path, "Test Stand.zip")
			Expect(p.Name).To(Equal("Test Stand"))
		})

		It("Should recreate both schematics and the table", func(ctx SpecContext) {
			l, p := importBundle(ctx, path, "Test Stand.zip")
			children := childrenOf(ctx, l, project.OntologyID(p.Key))
			Expect(typesOf(children)).To(ContainElements(
				ontology.ResourceTypeSchematic,
				ontology.ResourceTypeSchematic,
				ontology.ResourceTypeTable,
			))
		})

		It("Should recreate the mosaic as a panel of resource tabs", func(
			ctx SpecContext,
		) {
			l, p := importBundle(ctx, path, "Test Stand.zip")
			children := childrenOf(ctx, l, project.OntologyID(p.Key))
			var panels []ontology.Resource
			for _, c := range children {
				if c.ID.Type == ontology.ResourceTypePanel {
					panels = append(panels, c)
				}
			}
			Expect(panels).To(HaveLen(1))
			keys := MustSucceed(panel.KeysFromOntologyIDs(
				[]ontology.ID{panels[0].ID},
			))
			var pnl panel.Panel
			Expect(l.Panel.NewRetrieve().
				Where(panel.MatchKeys(keys...)).
				Entry(&pnl).
				Exec(ctx, nil)).To(Succeed())
			Expect(panel.ResourceRefs(pnl.Root)).ToNot(BeEmpty())
		})
	})

	Describe("A version 5 layout carrying task configs", func() {
		const path = "testdata/legacy-projects/hardware-v5"

		It("Should recreate the schematic, the line plot, and both tasks", func(
			ctx SpecContext,
		) {
			l, p := importBundle(ctx, path, "Hardware.zip")
			children := childrenOf(ctx, l, project.OntologyID(p.Key))
			Expect(typesOf(children)).To(ContainElements(
				ontology.ResourceTypeSchematic,
				ontology.ResourceTypeLineplot,
			))
			// A task parents under its rack's task group, never the project.
			Expect(typesOf(children)).ToNot(ContainElement(
				ontology.ResourceTypeTask,
			))
		})
	})

	Describe("A version 5 layout whose task file the export dropped", func() {
		const path = "testdata/legacy-projects/missing-member-v5"

		It("Should import the members the bundle does hold", func(ctx SpecContext) {
			// The layout names an ni_analog_write task whose component file the
			// export never wrote. The task drops; the rest of the bundle lands.
			l, p := importBundle(ctx, path, "Dropped.zip")
			children := childrenOf(ctx, l, project.OntologyID(p.Key))
			Expect(typesOf(children)).To(ContainElements(
				ontology.ResourceTypeSchematic,
				ontology.ResourceTypeLineplot,
			))
		})

		It("Should leave the dropped member out of the panel's tabs", func(
			ctx SpecContext,
		) {
			l, p := importBundle(ctx, path, "Dropped.zip")
			children := childrenOf(ctx, l, project.OntologyID(p.Key))
			var panelID ontology.ID
			for _, c := range children {
				if c.ID.Type == ontology.ResourceTypePanel {
					panelID = c.ID
				}
			}
			Expect(panelID.Key).ToNot(BeEmpty())
			keys := MustSucceed(panel.KeysFromOntologyIDs([]ontology.ID{panelID}))
			var pnl panel.Panel
			Expect(l.Panel.NewRetrieve().
				Where(panel.MatchKeys(keys...)).
				Entry(&pnl).
				Exec(ctx, nil)).To(Succeed())
			refs := panel.ResourceRefs(pnl.Root)
			Expect(refs).ToNot(BeEmpty())
			Expect(typesOf(children)).ToNot(ContainElement(
				ontology.ResourceTypeTask,
			))
		})
	})

	Describe("A version 8 layout with a name-named member", func() {
		const path = "testdata/legacy-projects/arc-editor-v8"

		It("Should resolve the member the layout names rather than keys", func(
			ctx SpecContext,
		) {
			l, p := importBundle(ctx, path, "Arc Workspace.zip")
			children := childrenOf(ctx, l, project.OntologyID(p.Key))
			Expect(typesOf(children)).To(ContainElement(
				ontology.ResourceTypeSchematic,
			))
		})
	})
})
