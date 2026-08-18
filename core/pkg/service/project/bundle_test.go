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
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/zip"
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
func createProject(ctx SpecContext, name string) project.Project {
	GinkgoHelper()
	p := project.Project{Name: name}
	Expect(writer.Create(ctx, &p)).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(writer.Delete(ctx, p.Key)).To(Succeed())
	})
	return p
}

func createLog(ctx SpecContext, proj project.Key, name string) log.Log {
	GinkgoHelper()
	l := log.Log{Name: name}
	Expect(logSvc.NewWriter(nil).Create(ctx, proj, &l)).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(logSvc.NewWriter(nil).Delete(ctx, l.Key)).To(Succeed())
	})
	return l
}

func createLinePlot(ctx SpecContext, proj project.Key, name string) lineplot.LinePlot {
	GinkgoHelper()
	lp := lineplot.LinePlot{Name: name}
	Expect(lineplotSvc.NewWriter(nil).Create(ctx, proj, &lp)).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(lineplotSvc.NewWriter(nil).Delete(ctx, lp.Key)).To(Succeed())
	})
	return lp
}

func createGroup(ctx SpecContext, name string, parent ontology.ID) group.Group {
	GinkgoHelper()
	g := MustSucceed(groupSvc.NewWriter(nil).Create(ctx, name, parent))
	DeferCleanup(func(ctx SpecContext) {
		Expect(groupSvc.NewWriter(nil).Delete(ctx, g.Key)).To(Succeed())
	})
	return g
}

func createTask(ctx SpecContext, name string) task.Task {
	GinkgoHelper()
	t := task.Task{
		Rack: testRack.Key,
		Name: name,
		Type: pagerduty.AlertTaskType,
	}
	Expect(taskSvc.NewWriter(nil).Create(ctx, &t)).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(taskSvc.NewWriter(nil).Delete(ctx, t.Key, false)).To(Succeed())
	})
	return t
}

func createPanel(
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

func moveToGroup(ctx SpecContext, child, from, to ontology.ID) {
	GinkgoHelper()
	w := otg.NewWriter(nil)
	Expect(w.DeleteRelationships(ctx, ontology.Relationship{
		From: from, Type: ontology.RelationshipTypeParentOf, To: child,
	})).To(Succeed())
	Expect(w.DefineRelationships(
		ctx, to, ontology.RelationshipTypeParentOf, child,
	)).To(Succeed())
}

func addToGroup(ctx SpecContext, child, to ontology.ID) {
	GinkgoHelper()
	Expect(otg.NewWriter(nil).DefineRelationships(
		ctx, to, ontology.RelationshipTypeParentOf, child,
	)).To(Succeed())
}

func resourceTab(id ontology.ID) panel.Tab {
	return panel.Tab{Variant: panel.ResourceTab{
		TabBase:  panel.TabBase{Key: uuid.New()},
		Resource: id,
	}}
}

func leaf(tabs ...panel.Tab) panel.Node {
	return panel.Node{Variant: panel.LeafNode{
		Tabs: append([]panel.Tab{}, tabs...),
	}}
}

// childrenOf reads the ontology children of id on the per-spec tx.
func childrenOf(ctx SpecContext, id ontology.ID) []ontology.Resource {
	GinkgoHelper()
	var children []ontology.Resource
	Expect(otg.NewRetrieve().
		WhereIDs(id).
		TraverseTo(ontology.ChildrenTraverser).
		Entries(&children).
		Exec(ctx, tx)).To(Succeed())
	return children
}

var _ = Describe("Export", func() {
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

	It("Should export the tasks a panel's tabs reference beside it", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "Panel Tasks")
		t := createTask(ctx, "Sequence")
		p := createPanel(
			ctx, "Controls", proj.OntologyID(), leaf(resourceTab(t.OntologyID())),
		)
		files, members := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("Sequence.json"))
		root := decode(files["Controls.json"])["root"].(map[string]any)
		Expect(root["tabs"]).To(ConsistOf(
			HaveKeyWithValue("resource", "Sequence.json"),
		))
		Expect(members).To(ConsistOf(t.OntologyID(), p.OntologyID()))
	})

	It("Should place a task referenced by two panels once", func(ctx SpecContext) {
		proj := createProject(ctx, "Shared Task")
		t := createTask(ctx, "Sequence")
		createPanel(
			ctx, "First", proj.OntologyID(), leaf(resourceTab(t.OntologyID())),
		)
		createPanel(
			ctx, "Second", proj.OntologyID(), leaf(resourceTab(t.OntologyID())),
		)
		files, members := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveLen(4))
		Expect(files).To(HaveKey("Sequence.json"))
		for _, name := range []string{"First.json", "Second.json"} {
			root := decode(files[name])["root"].(map[string]any)
			Expect(root["tabs"]).To(ConsistOf(
				HaveKeyWithValue("resource", "Sequence.json"),
			))
		}
		Expect(members).To(ContainElement(t.OntologyID()))
	})

	It("Should skip a task tab whose task no longer exists", func(ctx SpecContext) {
		proj := createProject(ctx, "Dangling Task")
		dangling := ontology.ID{
			Type: ontology.ResourceTypeTask,
			Key:  uuid.NewString(),
		}
		createPanel(ctx, "Controls", proj.OntologyID(), leaf(resourceTab(dangling)))
		files, _ := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveLen(2))
		root := decode(files["Controls.json"])["root"].(map[string]any)
		Expect(root["tabs"]).To(BeEmpty())
	})

	It("Should place a document with two parent groups once", func(ctx SpecContext) {
		proj := createProject(ctx, "Dual Parent")
		first := createGroup(ctx, "First", proj.OntologyID())
		second := createGroup(ctx, "Second", proj.OntologyID())
		l := createLog(ctx, proj.Key, "Shared")
		moveToGroup(ctx, l.OntologyID(), proj.OntologyID(), first.OntologyID())
		addToGroup(ctx, l.OntologyID(), second.OntologyID())
		files, members := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		// The manifest plus one placement: the first parent keeps the document, and the
		// other group drops as empty.
		Expect(files).To(HaveLen(2))
		Expect(members).To(HaveLen(2))
		Expect(members).To(ContainElement(l.OntologyID()))
	})

	It("Should suffix the second of two members in one directory that collide", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "Colliding")
		createLog(ctx, proj.Key, "Pressure")
		createLog(ctx, proj.Key, "pressure")
		files, _ := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("Pressure.json"))
		Expect(files).To(HaveKey("pressure (1).json"))
	})

	It("Should suffix the second of two colliding groups", func(ctx SpecContext) {
		proj := createProject(ctx, "Twin Groups")
		g1 := createGroup(ctx, "Valves", proj.OntologyID())
		g2 := createGroup(ctx, "valves", proj.OntologyID())
		l1 := createLog(ctx, proj.Key, "A")
		l2 := createLog(ctx, proj.Key, "B")
		moveToGroup(ctx, l1.OntologyID(), proj.OntologyID(), g1.OntologyID())
		moveToGroup(ctx, l2.OntologyID(), proj.OntologyID(), g2.OntologyID())
		files, _ := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("Valves/A.json"))
		Expect(files).To(HaveKey("valves (1)/B.json"))
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

	It("Should suffix a member claiming a reserved file name", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "Reserved")
		createLog(ctx, proj.Key, "manifest")
		files, _ := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("manifest.json"))
		Expect(files).To(HaveKey("manifest (1).json"))
	})

	It("Should export a root member named LAYOUT without a suffix", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "Layout Member")
		createLog(ctx, proj.Key, "LAYOUT")
		files, _ := MustSucceed2(svc.Export(ctx, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("LAYOUT.json"))
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

// bundleManifest builds a version 1 project manifest file.
func bundleManifest(name string) []byte {
	GinkgoHelper()
	return MustSucceed(json.Marshal(map[string]any{
		"version": 1, "type": "project", "name": name,
	}))
}

// bundleLog builds a member file holding a current-format log envelope.
func bundleLog(name string) []byte {
	GinkgoHelper()
	return MustSucceed(json.Marshal(map[string]any{
		"version": 2, "type": "log", "name": name, "channels": []any{},
	}))
}

// bundlePanel builds a member file holding a panel envelope whose leaf carries one
// resource tab per path.
func bundlePanel(name string, paths ...string) []byte {
	GinkgoHelper()
	tabs := make([]any, len(paths))
	for i, p := range paths {
		tabs[i] = map[string]any{
			"key": uuid.NewString(), "variant": "resource", "resource": p,
		}
	}
	return MustSucceed(json.Marshal(map[string]any{
		"version": 0,
		"type":    "panel",
		"name":    name,
		"root":    map[string]any{"variant": "leaf", "tabs": tabs},
	}))
}

// retrievePanel reads the panel behind id on the per-spec tx.
func retrievePanel(ctx SpecContext, id ontology.ID) panel.Panel {
	GinkgoHelper()
	keys := MustSucceed(panel.KeysFromOntologyIDs([]ontology.ID{id}))
	var p panel.Panel
	Expect(panelSvc.NewRetrieve().
		Where(panel.MatchKeys(keys...)).
		Entry(&p).
		Exec(ctx, tx)).To(Succeed())
	return p
}

// firstTabResource returns the ontology ID the panel's first resource tab displays.
func firstTabResource(p panel.Panel) ontology.ID {
	GinkgoHelper()
	l, ok := p.Root.Variant.(panel.LeafNode)
	Expect(ok).To(BeTrue())
	Expect(l.Tabs).ToNot(BeEmpty())
	r, ok := l.Tabs[0].Variant.(panel.ResourceTab)
	Expect(ok).To(BeTrue())
	return r.Resource
}

// childOfType returns the single child of the given type, failing when none matches.
func childOfType(
	children []ontology.Resource, t ontology.ResourceType,
) ontology.Resource {
	GinkgoHelper()
	for _, c := range children {
		if c.ID.Type == t {
			return c
		}
	}
	Fail(string("no child of type " + t))
	return ontology.Resource{}
}

// Imports run on the per-spec tx so created rows roll back and the shared DB's counts
// stay intact for the other specs.
var _ = Describe("Import", func() {
	importProject := func(ctx SpecContext, files zip.Files) project.Project {
		GinkgoHelper()
		return MustSucceed(svc.Import(ctx, tx, files, "Fallback.zip"))
	}

	It("Should import an exported bundle back as an equal project", func(
		ctx SpecContext,
	) {
		src := createProject(ctx, "Test Stand 12")
		g := createGroup(ctx, "Propulsion", src.OntologyID())
		grouped := createLog(ctx, src.Key, "Pressure")
		moveToGroup(ctx, grouped.OntologyID(), src.OntologyID(), g.OntologyID())
		createLog(ctx, src.Key, "Metrics")
		createPanel(
			ctx,
			"Controls",
			src.OntologyID(),
			leaf(resourceTab(grouped.OntologyID())),
		)
		files, _ := MustSucceed2(svc.Export(ctx, src.Key, xjson.Codec))
		proj := importProject(ctx, files)
		Expect(proj.Key).ToNot(Equal(src.Key))
		Expect(proj.Name).To(Equal("Test Stand 12"))
		children := childrenOf(ctx, project.OntologyID(proj.Key))
		Expect(children).To(HaveLen(3))
		newGroup := childOfType(children, ontology.ResourceTypeGroup)
		Expect(newGroup.Name).To(Equal("Propulsion"))
		Expect(newGroup.ID).ToNot(Equal(g.OntologyID()))
		rootLog := childOfType(children, ontology.ResourceTypeLog)
		Expect(rootLog.Name).To(Equal("Metrics"))
		groupChildren := childrenOf(ctx, newGroup.ID)
		Expect(groupChildren).To(HaveLen(1))
		Expect(groupChildren[0].Name).To(Equal("Pressure"))
		Expect(groupChildren[0].ID).ToNot(Equal(grouped.OntologyID()))
		p := retrievePanel(ctx, childOfType(children, ontology.ResourceTypePanel).ID)
		Expect(p.Name).To(Equal("Controls"))
		Expect(firstTabResource(p)).To(Equal(groupChildren[0].ID))
	})

	It("Should name the project from the manifest", func(ctx SpecContext) {
		proj := importProject(ctx, zip.Files{"manifest.json": bundleManifest("Named")})
		Expect(proj.Name).To(Equal("Named"))
		Expect(childrenOf(ctx, project.OntologyID(proj.Key))).To(BeEmpty())
	})

	It("Should fall back to the file name when the manifest is nameless", func(
		ctx SpecContext,
	) {
		proj := MustSucceed(svc.Import(
			ctx,
			tx,
			zip.Files{"manifest.json": bundleManifest("")},
			"My Project.zip",
		))
		Expect(proj.Name).To(Equal("My Project"))
	})

	It("Should recreate nested directories as nested groups", func(ctx SpecContext) {
		proj := importProject(ctx, zip.Files{
			"manifest.json":     bundleManifest("Grouped"),
			"A/B/Pressure.json": bundleLog("Pressure"),
		})
		children := childrenOf(ctx, project.OntologyID(proj.Key))
		Expect(children).To(HaveLen(1))
		outer := children[0]
		Expect(outer.ID.Type).To(Equal(ontology.ResourceTypeGroup))
		Expect(outer.Name).To(Equal("A"))
		inners := childrenOf(ctx, outer.ID)
		Expect(inners).To(HaveLen(1))
		Expect(inners[0].Name).To(Equal("B"))
		logs := childrenOf(ctx, inners[0].ID)
		Expect(logs).To(HaveLen(1))
		Expect(logs[0].Name).To(Equal("Pressure"))
		Expect(logs[0].ID.Type).To(Equal(ontology.ResourceTypeLog))
	})

	It("Should place a panel under its directory's group", func(ctx SpecContext) {
		proj := importProject(ctx, zip.Files{
			"manifest.json":      bundleManifest("Paneled"),
			"Deck/Pressure.json": bundleLog("Pressure"),
			"Deck/Controls.json": bundlePanel("Controls", "Deck/Pressure.json"),
		})
		children := childrenOf(ctx, project.OntologyID(proj.Key))
		Expect(children).To(HaveLen(1))
		deck := childrenOf(ctx, children[0].ID)
		Expect(deck).To(HaveLen(2))
		p := retrievePanel(ctx, childOfType(deck, ontology.ResourceTypePanel).ID)
		Expect(firstTabResource(p)).
			To(Equal(childOfType(deck, ontology.ResourceTypeLog).ID))
	})

	It("Should leave imported tasks under the rack", func(ctx SpecContext) {
		src := createProject(ctx, "Panel Tasks")
		t := createTask(ctx, "Sequence")
		createPanel(
			ctx, "Controls", src.OntologyID(), leaf(resourceTab(t.OntologyID())),
		)
		files, _ := MustSucceed2(svc.Export(ctx, src.Key, xjson.Codec))
		proj := importProject(ctx, files)
		children := childrenOf(ctx, project.OntologyID(proj.Key))
		Expect(children).To(HaveLen(1))
		Expect(children[0].ID.Type).To(Equal(ontology.ResourceTypePanel))
		newTaskID := firstTabResource(retrievePanel(ctx, children[0].ID))
		Expect(newTaskID.Type).To(Equal(ontology.ResourceTypeTask))
		Expect(newTaskID).ToNot(Equal(t.OntologyID()))
		var res ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(newTaskID).
			Entry(&res).
			Exec(ctx, tx)).To(Succeed())
		Expect(res.Name).To(Equal("Sequence"))
	})

	It("Should ignore files that are not members", func(ctx SpecContext) {
		proj := importProject(ctx, zip.Files{
			"manifest.json": bundleManifest("Tidy"),
			"Metrics.json":  bundleLog("Metrics"),
			"README.md":     []byte("# Tidy"),
			".gitignore":    []byte("*.tmp"),
		})
		Expect(childrenOf(ctx, project.OntologyID(proj.Key))).To(HaveLen(1))
	})

	It("Should allow equal member names in different directories", func(
		ctx SpecContext,
	) {
		proj := importProject(ctx, zip.Files{
			"manifest.json":   bundleManifest("Shadowed"),
			"Pressure.json":   bundleLog("Pressure"),
			"A/Pressure.json": bundleLog("Pressure"),
		})
		Expect(childrenOf(ctx, project.OntologyID(proj.Key))).To(HaveLen(2))
	})

	DescribeTable("Should reject an invalid bundle",
		func(ctx SpecContext, files zip.Files, reason string) {
			Expect(svc.Import(ctx, tx, files, "Fallback.zip")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(reason)),
			))
		},
		Entry("no manifest or legacy layout",
			zip.Files{"Metrics.json": []byte("{}")}, "bundle holds no manifest.json"),
		Entry("another bundle kind",
			zip.Files{"manifest.json": []byte(
				`{"version":2,"type":"symbol_group","name":"Valves"}`,
			)}, `bundle is a "symbol_group"`),
		Entry("a manifest at version zero",
			zip.Files{"manifest.json": []byte(
				`{"version":0,"type":"project","name":"Old"}`,
			)}, "unsupported manifest version 0"),
		Entry("two members whose names compare equal case-folded",
			zip.Files{
				"manifest.json": []byte(
					`{"version":1,"type":"project","name":"Colliding"}`,
				),
				"Pressure.json": []byte(`{"version":2,"type":"log","name":"A"}`),
				"pressure.json": []byte(`{"version":2,"type":"log","name":"B"}`),
			}, "have the same file name"),
		Entry("a file and a directory sharing a name",
			zip.Files{
				"manifest.json": []byte(
					`{"version":1,"type":"project","name":"Colliding"}`,
				),
				"A.json":        []byte(`{"version":2,"type":"log","name":"A"}`),
				"a.json/B.json": []byte(`{"version":2,"type":"log","name":"B"}`),
			}, "have the same file name"),
	)

	It("Should reject a member with no registered importer", func(ctx SpecContext) {
		files := zip.Files{
			"manifest.json": bundleManifest("Plots"),
			"Plot.json":     []byte(`{"version":1,"type":"lineplot","name":"Plot"}`),
		}
		// PathedError breaks the errors.Is chain, so the assertion matches on text.
		Expect(svc.Import(ctx, tx, files, "Fallback.zip")).Error().To(SatisfyAll(
			MatchError(ContainSubstring("Plot.json")),
			MatchError(ContainSubstring("no importer registered")),
		))
	})

	It("Should reject a manifest version newer than supported", func(ctx SpecContext) {
		files := zip.Files{"manifest.json": []byte(
			`{"version":2,"type":"project","name":"Future"}`,
		)}
		Expect(svc.Import(ctx, tx, files, "Fallback.zip")).Error().To(SatisfyAll(
			MatchError(ContainSubstring("project version 2")),
			MatchError(ContainSubstring("newer than this Core supports")),
		))
	})

	It("Should reject a panel referencing a file outside the bundle", func(
		ctx SpecContext,
	) {
		files := zip.Files{
			"manifest.json": bundleManifest("Dangling"),
			"Controls.json": bundlePanel("Controls", "Missing.json"),
		}
		Expect(svc.Import(ctx, tx, files, "Fallback.zip")).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("Controls.json")),
			MatchError(ContainSubstring(`"Missing.json"`)),
		))
	})

	It("Should reject a member that does not decode to an envelope", func(
		ctx SpecContext,
	) {
		files := zip.Files{
			"manifest.json": bundleManifest("Broken"),
			"Bad.json":      []byte("not json"),
		}
		Expect(svc.Import(ctx, tx, files, "Fallback.zip")).Error().
			To(MatchError(ContainSubstring("Bad.json")))
	})
})

var _ = Describe("ImportObjects", func() {
	It("Should list the project and each distinct member type", func(ctx SpecContext) {
		objects := MustSucceed(svc.ImportObjects(ctx, zip.Files{
			"manifest.json": bundleManifest("Valves"),
			"Metrics.json":  bundleLog("Metrics"),
			"Flow.json":     bundleLog("Flow"),
		}))
		Expect(objects).To(ConsistOf(
			ontology.ID{Type: ontology.ResourceTypeProject},
			ontology.ID{Type: ontology.ResourceTypeLog},
		))
	})

	It("Should include the group and panel types when the bundle carries them", func(
		ctx SpecContext,
	) {
		objects := MustSucceed(svc.ImportObjects(ctx, zip.Files{
			"manifest.json":  bundleManifest("Valves"),
			"A/Metrics.json": bundleLog("Metrics"),
			"Controls.json":  bundlePanel("Controls", "A/Metrics.json"),
		}))
		Expect(objects).To(ConsistOf(
			ontology.ID{Type: ontology.ResourceTypeProject},
			ontology.ID{Type: ontology.ResourceTypeLog},
			ontology.ID{Type: ontology.ResourceTypePanel},
			ontology.ID{Type: ontology.ResourceTypeGroup},
		))
	})

	It("Should reject an invalid bundle with the import's error", func(
		ctx SpecContext,
	) {
		Expect(svc.ImportObjects(ctx, zip.Files{"Metrics.json": []byte("{}")})).
			Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("bundle holds no manifest.json")),
		))
	})
})
