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
	"encoding/json"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// decode unpacks a bundle file's bytes into the generic map assertions inspect.
func decode(b []byte) map[string]any {
	GinkgoHelper()
	var m map[string]any
	Expect(json.Unmarshal(b, &m)).To(Succeed())
	return m
}

// Export reads committed data, so every fixture below is created outside the per-spec
// tx and deleted afterwards to keep the shared DB's counts intact.
var _ = Describe("Export", func() {
	createProject := func(ctx SpecContext, name string) project.Project {
		GinkgoHelper()
		p := project.Project{Name: name}
		Expect(writer.Create(ctx, &p)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(writer.Delete(ctx, p.Key)).To(Succeed())
		})
		return p
	}
	createLog := func(ctx SpecContext, proj project.Key, name string) log.Log {
		GinkgoHelper()
		l := log.Log{Name: name}
		Expect(logSvc.NewWriter(nil).Create(ctx, proj, &l)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(logSvc.NewWriter(nil).Delete(ctx, l.Key)).To(Succeed())
		})
		return l
	}
	createLinePlot := func(
		ctx SpecContext, proj project.Key, name string,
	) lineplot.LinePlot {
		GinkgoHelper()
		lp := lineplot.LinePlot{Name: name}
		Expect(lineplotSvc.NewWriter(nil).Create(ctx, proj, &lp)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(lineplotSvc.NewWriter(nil).Delete(ctx, lp.Key)).To(Succeed())
		})
		return lp
	}
	createGroup := func(ctx SpecContext, name string, parent ontology.ID) group.Group {
		GinkgoHelper()
		g := MustSucceed(groupSvc.NewWriter(nil).Create(ctx, name, parent))
		DeferCleanup(func(ctx SpecContext) {
			Expect(groupSvc.NewWriter(nil).Delete(ctx, g.Key)).To(Succeed())
		})
		return g
	}
	createPanel := func(
		ctx SpecContext, name string, parent ontology.ID, root panel.Node,
	) panel.Panel {
		GinkgoHelper()
		p := panel.Panel{Name: name, Root: root, Parent: &parent}
		Expect(panelSvc.NewWriter(nil).Create(ctx, &p)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(panelSvc.NewWriter(nil).Delete(ctx, p.Key)).To(Succeed())
		})
		return p
	}
	moveToGroup := func(ctx SpecContext, child, from, to ontology.ID) {
		GinkgoHelper()
		w := otg.NewWriter(nil)
		Expect(w.DeleteRelationships(ctx, ontology.Relationship{
			From: from, Type: ontology.RelationshipTypeParentOf, To: child,
		})).To(Succeed())
		Expect(w.DefineRelationships(
			ctx, to, ontology.RelationshipTypeParentOf, child,
		)).To(Succeed())
	}
	resourceTab := func(id ontology.ID) panel.Tab {
		return panel.Tab{Variant: panel.TabResource{
			TabBase:  panel.TabBase{Key: uuid.New()},
			Resource: id,
		}}
	}
	leaf := func(tabs ...panel.Tab) panel.Node {
		return panel.Node{Variant: panel.NodeLeaf{
			Leaf: panel.Leaf{Tabs: append([]panel.Tab{}, tabs...)},
		}}
	}

	It("Should return NotFound for a missing project", func(ctx SpecContext) {
		Expect(svc.Export(ctx, uuid.New(), xjson.Codec)).Error().
			To(MatchError(query.ErrNotFound))
	})

	It("Should export an empty project as a manifest alone", func(ctx SpecContext) {
		proj := createProject(ctx, "Empty")
		files, members := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveLen(1))
		manifest := decode(files["manifest.json"])
		Expect(manifest).To(HaveKeyWithValue("version", BeNumerically("==", 1)))
		Expect(manifest).To(HaveKeyWithValue("type", "project"))
		Expect(manifest).To(HaveKeyWithValue("name", "Empty"))
		Expect(members).To(BeEmpty())
	})

	It("Should export documents and panels at the root", func(ctx SpecContext) {
		proj := createProject(ctx, "Test Stand 12")
		l := createLog(ctx, proj.Key, "Metrics Log")
		p := createPanel(
			ctx, "Controls", proj.OntologyID(), leaf(resourceTab(l.OntologyID())),
		)
		files, members := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveLen(3))
		Expect(decode(files["Metrics Log.json"])).To(HaveKeyWithValue("type", "log"))
		panelBody := decode(files["Controls.json"])
		Expect(panelBody).To(HaveKeyWithValue("type", "panel"))
		root := panelBody["root"].(map[string]any)
		Expect(root["tabs"]).To(ConsistOf(
			HaveKeyWithValue("resource", "Metrics Log.json"),
		))
		Expect(members).To(ConsistOf(l.OntologyID(), p.OntologyID()))
	})

	It("Should place grouped members in nested directories", func(ctx SpecContext) {
		proj := createProject(ctx, "Grouped")
		outer := createGroup(ctx, "Propulsion", proj.OntologyID())
		inner := createGroup(ctx, "Tanks", outer.OntologyID())
		l := createLog(ctx, proj.Key, "Pressure")
		moveToGroup(ctx, l.OntologyID(), proj.OntologyID(), inner.OntologyID())
		createPanel(
			ctx, "Controls", proj.OntologyID(), leaf(resourceTab(l.OntologyID())),
		)
		files, members := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("Propulsion/Tanks/Pressure.json"))
		root := decode(files["Controls.json"])["root"].(map[string]any)
		Expect(root["tabs"]).To(ConsistOf(
			HaveKeyWithValue("resource", "Propulsion/Tanks/Pressure.json"),
		))
		Expect(members).To(ContainElements(outer.OntologyID(), inner.OntologyID()))
	})

	It("Should drop a group with no exported members", func(ctx SpecContext) {
		proj := createProject(ctx, "Sparse")
		createGroup(ctx, "Empty Group", proj.OntologyID())
		files, members := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveLen(1))
		Expect(members).To(BeEmpty())
	})

	It("Should skip children the registry cannot export", func(ctx SpecContext) {
		proj := createProject(ctx, "Skips")
		lp := createLinePlot(ctx, proj.Key, "Unexportable")
		l := createLog(ctx, proj.Key, "Kept")
		createPanel(
			ctx,
			"Controls",
			proj.OntologyID(),
			leaf(resourceTab(l.OntologyID()), resourceTab(lp.OntologyID())),
		)
		files, members := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveLen(3))
		root := decode(files["Controls.json"])["root"].(map[string]any)
		Expect(root["tabs"]).To(ConsistOf(HaveKeyWithValue("resource", "Kept.json")))
		Expect(members).ToNot(ContainElement(lp.OntologyID()))
	})

	It("Should error when two members in one directory collide", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "Colliding")
		createLog(ctx, proj.Key, "Pressure")
		createLog(ctx, proj.Key, "pressure")
		Expect(svc.Export(ctx, proj.Key, xjson.Codec)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("both export to")),
		))
	})

	It("Should allow equal names in different directories", func(ctx SpecContext) {
		proj := createProject(ctx, "Shadowed")
		g := createGroup(ctx, "Propulsion", proj.OntologyID())
		outerLog := createLog(ctx, proj.Key, "Pressure")
		innerLog := createLog(ctx, proj.Key, "Pressure")
		moveToGroup(ctx, innerLog.OntologyID(), proj.OntologyID(), g.OntologyID())
		files, _ := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("Pressure.json"))
		Expect(files).To(HaveKey("Propulsion/Pressure.json"))
		Expect(outerLog.Key).ToNot(Equal(innerLog.Key))
	})

	It("Should error when a member claims a reserved file name", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "Reserved")
		createLog(ctx, proj.Key, "manifest")
		Expect(svc.Export(ctx, proj.Key, xjson.Codec)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("reserved file name")),
		))
	})

	It("Should allow a reserved name inside a group directory", func(ctx SpecContext) {
		proj := createProject(ctx, "Nested Reserved")
		g := createGroup(ctx, "Configs", proj.OntologyID())
		l := createLog(ctx, proj.Key, "manifest")
		moveToGroup(ctx, l.OntologyID(), proj.OntologyID(), g.OntologyID())
		files, _ := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("Configs/manifest.json"))
	})
})
