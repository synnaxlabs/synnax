// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bundle_test

import (
	"encoding/json/v2"
	"maps"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	text "github.com/synnaxlabs/arc/text/versions/v1"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/defaults"
	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

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
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return projectSvc.NewWriter(tx).Create(ctx, &p)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return projectSvc.NewWriter(tx).Delete(ctx, p.Key)
		})).To(Succeed())
	})
	return p
}

func createLog(ctx SpecContext, proj project.Key, name string) log.Log {
	GinkgoHelper()
	l := defaults.Fill(log.Log{Project: &proj, Name: name})
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return logSvc.NewWriter(tx).Create(ctx, &l)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return logSvc.NewWriter(tx).Delete(ctx, l.Key)
		})).To(Succeed())
	})
	return l
}

func createLinePlot(ctx SpecContext, proj project.Key, name string) lineplot.LinePlot {
	GinkgoHelper()
	lp := defaults.Fill(lineplot.LinePlot{Project: &proj, Name: name})
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return lineplotSvc.NewWriter(tx).Create(ctx, &lp)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return lineplotSvc.NewWriter(tx).Delete(ctx, lp.Key)
		})).To(Succeed())
	})
	return lp
}

func createGroup(ctx SpecContext, name string, parent resource.ID) group.Group {
	GinkgoHelper()
	g := group.Group{Name: name, Parent: &parent}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return groupSvc.NewWriter(tx).Create(ctx, &g)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Delete(ctx, g.Key)
		})).To(Succeed())
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
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return taskSvc.NewWriter(tx).Create(ctx, &t)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return taskSvc.NewWriter(tx).Delete(ctx, t.Key, false)
		})).To(Succeed())
	})
	return t
}

func createArc(ctx SpecContext, name, source string) arc.Arc {
	GinkgoHelper()
	a := arc.Arc{Variant: arc.AutomationArc{
		Base: arc.Base{
			Name:   name,
			Source: arc.Source{Variant: arc.TextSource{Text: text.Text{Raw: source}}},
		},
	}}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return arcSvc.NewWriter(tx).Create(ctx, &a)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return arcSvc.NewWriter(tx).Delete(ctx, a.Base().Key)
		})).To(Succeed())
	})
	return a
}

func createPanel(
	ctx SpecContext, name string, proj project.Key, root panel.Node,
) panel.Panel {
	GinkgoHelper()
	p := panel.Panel{Name: name, Root: root, Project: &proj}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return panelSvc.NewWriter(tx).Create(ctx, &p)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return panelSvc.NewWriter(tx).Delete(ctx, p.Key)
		})).To(Succeed())
	})
	return p
}

func moveToGroup(ctx SpecContext, l log.Log, g group.Group) {
	GinkgoHelper()
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		_, err := logSvc.NewWriter(tx).Apply(ctx, l.Key, log.NewSetAction(
			log.SetPayload{Path: []string{"group"}, Value: g.Key},
		))
		return err
	})).To(Succeed())
}

func resourceTab(id resource.ID) panel.Tab {
	return panel.Tab{Variant: panel.ResourceTab{Key: uuid.New(), Resource: id}}
}

func leaf(tabs ...panel.Tab) panel.Node {
	return panel.Node{Variant: panel.LeafNode{
		Tabs: append([]panel.Tab{}, tabs...),
	}}
}

// childrenOf returns what the project tree shows under id: the groups and ungrouped
// members of a project, or the groups and members of a group.
func childrenOf(ctx SpecContext, id resource.ID) []resource.Resource {
	GinkgoHelper()
	var ids []resource.ID
	if id.Type == resource.TypeProject {
		key := MustSucceed(project.KeyFromResourceID(id))
		deps := MustSucceed(projectSvc.Children(ctx, tx, key))
		grouped := set.New[resource.ID]()
		for _, d := range deps {
			if d.Type == resource.TypeGroup {
				inner := MustSucceed(group.KeyFromResourceID(d))
				grouped.Add(descendants(ctx, inner)...)
			}
		}
		for _, d := range deps {
			if !grouped.Contains(d) {
				ids = append(ids, d)
			}
		}
	} else {
		key := MustSucceed(group.KeyFromResourceID(id))
		ids = MustSucceed(groupSvc.Children(ctx, tx, key))
	}
	children := MustSucceed(resourceSvc.Retrieve(ctx, tx, ids...))
	Expect(children).To(HaveLen(len(ids)))
	return children
}

// descendants returns every group and member that sits under the group with key.
func descendants(ctx SpecContext, key group.Key) []resource.ID {
	GinkgoHelper()
	children := MustSucceed(groupSvc.Children(ctx, tx, key))
	out := slices.Clone(children)
	for _, c := range children {
		if c.Type == resource.TypeGroup {
			inner := MustSucceed(group.KeyFromResourceID(c))
			out = append(out, descendants(ctx, inner)...)
		}
	}
	return out
}

var _ = Describe("Export", func() {
	It("Should return NotFound for a missing project", func(ctx SpecContext) {
		Expect(svc.Export(ctx, db, uuid.New(), xjson.Codec)).Error().
			To(MatchError(query.ErrNotFound))
	})

	It("Should export an empty project as a manifest alone", func(ctx SpecContext) {
		proj := createProject(ctx, "Empty")
		files, members := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
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
			ctx, "Controls", proj.Key, leaf(resourceTab(l.ResourceID())),
		)
		files, members := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveLen(3))
		Expect(decode(files["Metrics Log.json"])).To(HaveKeyWithValue("type", "log"))
		panelBody := decode(files["Controls.json"])
		Expect(panelBody).To(HaveKeyWithValue("type", "panel"))
		root := panelBody["root"].(map[string]any)
		Expect(root["tabs"]).To(ConsistOf(
			HaveKeyWithValue("resource", "Metrics Log.json"),
		))
		Expect(members).To(ConsistOf(l.ResourceID(), p.ResourceID()))
	})

	It("Should place grouped members in nested directories", func(ctx SpecContext) {
		proj := createProject(ctx, "Grouped")
		outer := createGroup(ctx, "Propulsion", proj.ResourceID())
		inner := createGroup(ctx, "Tanks", outer.ResourceID())
		l := createLog(ctx, proj.Key, "Pressure")
		moveToGroup(ctx, l, inner)
		createPanel(
			ctx, "Controls", proj.Key, leaf(resourceTab(l.ResourceID())),
		)
		files, members := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("Propulsion/Tanks/Pressure.json"))
		root := decode(files["Controls.json"])["root"].(map[string]any)
		Expect(root["tabs"]).To(ConsistOf(
			HaveKeyWithValue("resource", "Propulsion/Tanks/Pressure.json"),
		))
		Expect(members).To(ContainElements(outer.ResourceID(), inner.ResourceID()))
	})

	It("Should drop a group with no exported members", func(ctx SpecContext) {
		proj := createProject(ctx, "Sparse")
		createGroup(ctx, "Empty Group", proj.ResourceID())
		files, members := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveLen(1))
		Expect(members).To(BeEmpty())
	})

	It("Should leave a draft out of the export", func(ctx SpecContext) {
		proj := createProject(ctx, "Drafts")
		draft := panel.Panel{Name: "Scratch", Root: leaf(), Project: &proj.Key}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			r := role.Role{Name: "drafter"}
			if err := roleSvc.NewWriter(tx, false).Create(ctx, &r); err != nil {
				return err
			}
			u, err := userSvc.NewWriter(tx).Create(ctx, user.User{
				Key:  uuid.New(),
				Name: "drafter",
				Role: r.Key,
			})
			if err != nil {
				return err
			}
			draft.Owner = &u.Key
			return panelSvc.NewWriter(tx).Create(ctx, &draft)
		})).To(Succeed())
		files, members := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
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
			proj.Key,
			leaf(resourceTab(l.ResourceID()), resourceTab(lp.ResourceID())),
		)
		files, members := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveLen(3))
		root := decode(files["Controls.json"])["root"].(map[string]any)
		Expect(root["tabs"]).To(ConsistOf(HaveKeyWithValue("resource", "Kept.json")))
		Expect(members).ToNot(ContainElement(lp.ResourceID()))
	})

	It("Should export the tasks a panel's tabs reference beside it", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "Panel Tasks")
		t := createTask(ctx, "Sequence")
		p := createPanel(
			ctx, "Controls", proj.Key, leaf(resourceTab(t.ResourceID())),
		)
		files, members := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("Sequence.json"))
		root := decode(files["Controls.json"])["root"].(map[string]any)
		Expect(root["tabs"]).To(ConsistOf(
			HaveKeyWithValue("resource", "Sequence.json"),
		))
		Expect(members).To(ConsistOf(t.ResourceID(), p.ResourceID()))
	})

	It("Should export the Arcs a panel's tabs reference beside it", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "Panel Arcs")
		a := createArc(ctx, "Startup", "x := 1")
		p := createPanel(
			ctx, "Controls", proj.Key, leaf(resourceTab(a.ResourceID())),
		)
		files, members := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("Startup.json"))
		Expect(decode(files["Startup.json"])).To(SatisfyAll(
			HaveKeyWithValue("type", "arc-automation"),
			HaveKeyWithValue("source", map[string]any{
				"mode": "text",
				"text": map[string]any{"raw": "x := 1"},
			}),
		))
		root := decode(files["Controls.json"])["root"].(map[string]any)
		Expect(root["tabs"]).To(ConsistOf(
			HaveKeyWithValue("resource", "Startup.json"),
		))
		Expect(members).To(ConsistOf(a.ResourceID(), p.ResourceID()))
	})

	It("Should place a task referenced by two panels once", func(ctx SpecContext) {
		proj := createProject(ctx, "Shared Task")
		t := createTask(ctx, "Sequence")
		createPanel(
			ctx, "First", proj.Key, leaf(resourceTab(t.ResourceID())),
		)
		createPanel(
			ctx, "Second", proj.Key, leaf(resourceTab(t.ResourceID())),
		)
		files, members := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveLen(4))
		Expect(files).To(HaveKey("Sequence.json"))
		for _, name := range []string{"First.json", "Second.json"} {
			root := decode(files[name])["root"].(map[string]any)
			Expect(root["tabs"]).To(ConsistOf(
				HaveKeyWithValue("resource", "Sequence.json"),
			))
		}
		Expect(members).To(ContainElement(t.ResourceID()))
	})

	It("Should skip a task tab whose task no longer exists", func(ctx SpecContext) {
		proj := createProject(ctx, "Dangling Task")
		dangling := resource.ID{
			Type: resource.TypeTask,
			Key:  uuid.New().String(),
		}
		createPanel(ctx, "Controls", proj.Key, leaf(resourceTab(dangling)))
		files, _ := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveLen(2))
		root := decode(files["Controls.json"])["root"].(map[string]any)
		Expect(root["tabs"]).To(BeEmpty())
	})

	It("Should suffix the second of two members in one directory that collide", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "Colliding")
		createLog(ctx, proj.Key, "Pressure")
		createLog(ctx, proj.Key, "pressure")
		files, _ := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("Pressure.json"))
		Expect(files).To(HaveKey("pressure (1).json"))
	})

	It("Should suffix the second of two colliding groups", func(ctx SpecContext) {
		proj := createProject(ctx, "Twin Groups")
		g1 := createGroup(ctx, "Valves", proj.ResourceID())
		g2 := createGroup(ctx, "valves", proj.ResourceID())
		l1 := createLog(ctx, proj.Key, "A")
		l2 := createLog(ctx, proj.Key, "B")
		moveToGroup(ctx, l1, g1)
		moveToGroup(ctx, l2, g2)
		files, _ := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("Valves/A.json"))
		Expect(files).To(HaveKey("valves (1)/B.json"))
	})

	It("Should allow equal names in different directories", func(ctx SpecContext) {
		proj := createProject(ctx, "Shadowed")
		g := createGroup(ctx, "Propulsion", proj.ResourceID())
		outerLog := createLog(ctx, proj.Key, "Pressure")
		innerLog := createLog(ctx, proj.Key, "Pressure")
		moveToGroup(ctx, innerLog, g)
		files, _ := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("Pressure.json"))
		Expect(files).To(HaveKey("Propulsion/Pressure.json"))
		Expect(outerLog.Key).ToNot(Equal(innerLog.Key))
	})

	It("Should suffix a member claiming a reserved file name", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "Reserved")
		createLog(ctx, proj.Key, "manifest")
		files, _ := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("manifest.json"))
		Expect(files).To(HaveKey("manifest (1).json"))
	})

	It("Should export a root member named LAYOUT without a suffix", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "Layout Member")
		createLog(ctx, proj.Key, "LAYOUT")
		files, _ := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("LAYOUT.json"))
	})

	It("Should allow a reserved name inside a group directory", func(ctx SpecContext) {
		proj := createProject(ctx, "Nested Reserved")
		g := createGroup(ctx, "Configs", proj.ResourceID())
		l := createLog(ctx, proj.Key, "manifest")
		moveToGroup(ctx, l, g)
		files, _ := MustSucceed2(svc.Export(ctx, db, proj.Key, xjson.Codec))
		Expect(files).To(HaveKey("Configs/manifest.json"))
	})
})

func bundleManifest(name string) []byte {
	GinkgoHelper()
	return MustSucceed(json.Marshal(map[string]any{
		"version": 1, "type": "project", "name": name,
	}))
}

func bundleLog(name string) []byte {
	GinkgoHelper()
	return MustSucceed(json.Marshal(map[string]any{
		"version": 2, "type": "log", "name": name, "channels": []any{},
	}))
}

func bundlePanel(name string, paths ...string) []byte {
	GinkgoHelper()
	tabs := make([]any, len(paths))
	for i, p := range paths {
		tabs[i] = map[string]any{
			"key": uuid.New().String(), "variant": "resource", "resource": p,
		}
	}
	return MustSucceed(json.Marshal(map[string]any{
		"version": 0,
		"type":    "panel",
		"name":    name,
		"root":    map[string]any{"variant": "leaf", "tabs": tabs},
	}))
}

func retrievePanel(ctx SpecContext, id resource.ID) panel.Panel {
	GinkgoHelper()
	keys := MustSucceed(panel.KeysFromResourceIDs([]resource.ID{id}))
	var p panel.Panel
	Expect(panelSvc.NewRetrieve().
		Where(panel.MatchKeys(keys...)).
		Entry(&p).
		Exec(ctx, tx)).To(Succeed())
	return p
}

func firstTabResource(p panel.Panel) resource.ID {
	GinkgoHelper()
	l, ok := p.Root.Variant.(panel.LeafNode)
	Expect(ok).To(BeTrue())
	Expect(l.Tabs).ToNot(BeEmpty())
	r, ok := l.Tabs[0].Variant.(panel.ResourceTab)
	Expect(ok).To(BeTrue())
	return r.Resource
}

// tabResources returns the resource ID behind every resource tab in the leaf, in tab
// order.
func tabResources(l panel.LeafNode) []resource.ID {
	GinkgoHelper()
	ids := make([]resource.ID, 0, len(l.Tabs))
	for _, t := range l.Tabs {
		if r, ok := t.Variant.(panel.ResourceTab); ok {
			ids = append(ids, r.Resource)
		}
	}
	return ids
}

func childOfType(
	children []resource.Resource, t resource.Type,
) resource.Resource {
	GinkgoHelper()
	for _, c := range children {
		if c.ID.Type == t {
			return c
		}
	}
	Fail(string("no child of type " + t))
	return resource.Resource{}
}

// importFiles decodes files and imports them on the per-spec tx, so created rows roll
// back and the shared DB's counts stay intact for the other specs.
func importFiles(
	ctx SpecContext,
	files zip.Files,
	fileName string,
) (project.Project, error) {
	b, err := svc.Decode(ctx, files, fileName)
	if err != nil {
		return project.Project{}, err
	}
	return svc.Import(ctx, tx, b)
}

var _ = Describe("Import", func() {
	importProject := func(ctx SpecContext, files zip.Files) project.Project {
		GinkgoHelper()
		return MustSucceed(importFiles(ctx, files, "Fallback.zip"))
	}

	Describe("A whole bundle directory", func() {
		BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
		// The fixture is a bundle as the Console writes one to disk: nested group
		// directories, panels referencing members across them, an Arc and a task that
		// only a panel tab pulls in, suffixed name collisions, and a stray file.
		bundle := func() zip.Files {
			GinkgoHelper()
			return LoadBundle("testdata/test-stand-alpha")
		}

		named := func(
			children []resource.Resource, name string,
		) resource.Resource {
			GinkgoHelper()
			for _, c := range children {
				if c.Name == name {
					return c
				}
			}
			Fail("no child named " + name)
			return resource.Resource{}
		}

		It("Should name the project after the manifest", func(ctx SpecContext) {
			Expect(importProject(ctx, bundle()).Name).To(Equal("Test Stand Alpha"))
		})

		It("Should replace a lone surrogate in the manifest with U+FFFD", func(
			ctx SpecContext,
		) {
			Expect(importProject(ctx, zip.Files{
				"manifest.json": []byte(
					`{"version":1,"type":"project","name":"Tank \ud83d"}`,
				),
			}).Name).To(Equal("Tank \uFFFD"))
		})

		It("Should recreate the root members and the group directories", func(
			ctx SpecContext,
		) {
			proj := importProject(ctx, bundle())
			children := childrenOf(ctx, project.ResourceID(proj.Key))
			Expect(children).To(ConsistOf(
				HaveField("Name", "Event Log"),
				HaveField("Name", "Duplicate"),
				HaveField("Name", "manifest"),
				HaveField("Name", "Overview"),
				HaveField("Name", "Subsystems"),
			))
			Expect(named(children, "Subsystems").ID.Type).
				To(Equal(resource.TypeGroup))
		})

		It("Should recreate every nested directory as a group", func(
			ctx SpecContext,
		) {
			proj := importProject(ctx, bundle())
			subsystems := named(
				childrenOf(ctx, project.ResourceID(proj.Key)), "Subsystems",
			)
			inner := childrenOf(ctx, subsystems.ID)
			Expect(inner).To(ConsistOf(
				HaveField("Name", "Propulsion"),
				HaveField("Name", "Avionics"),
			))
			Expect(childrenOf(ctx, named(inner, "Propulsion").ID)).To(ConsistOf(
				HaveField("Name", "Chamber Log"),
				HaveField("Name", "Duplicate"),
				HaveField("Name", "Propulsion View"),
			))
			Expect(childrenOf(ctx, named(inner, "Avionics").ID)).To(ConsistOf(
				HaveField("Name", "Telemetry Log"),
			))
		})

		It("Should resolve a panel tab that crosses a directory", func(
			ctx SpecContext,
		) {
			proj := importProject(ctx, bundle())
			children := childrenOf(ctx, project.ResourceID(proj.Key))
			overview := retrievePanel(ctx, named(children, "Overview").ID)
			first, ok := overview.Root.Variant.(panel.SplitNode)
			Expect(ok).To(BeTrue())
			leafNode, ok := first.First.Variant.(panel.LeafNode)
			Expect(ok).To(BeTrue())
			Expect(leafNode.Tabs).To(HaveLen(3))
			chamber := childrenOf(ctx, named(childrenOf(
				ctx, named(children, "Subsystems").ID,
			), "Propulsion").ID)
			Expect(tabResources(leafNode)).To(ContainElements(
				named(children, "Event Log").ID,
				named(chamber, "Chamber Log").ID,
			))
		})

		It("Should keep a view tab's type", func(ctx SpecContext) {
			proj := importProject(ctx, bundle())
			overview := retrievePanel(ctx, named(
				childrenOf(ctx, project.ResourceID(proj.Key)), "Overview",
			).ID)
			split, ok := overview.Root.Variant.(panel.SplitNode)
			Expect(ok).To(BeTrue())
			leafNode, ok := split.First.Variant.(panel.LeafNode)
			Expect(ok).To(BeTrue())
			v, ok := leafNode.Tabs[2].Variant.(panel.ViewTab)
			Expect(ok).To(BeTrue())
			Expect(v.Type).To(Equal("docs"))
		})

		It("Should create the Arc a panel tab pulls in, unparented", func(
			ctx SpecContext,
		) {
			proj := importProject(ctx, bundle())
			overview := retrievePanel(ctx, named(
				childrenOf(ctx, project.ResourceID(proj.Key)), "Overview",
			).ID)
			split, ok := overview.Root.Variant.(panel.SplitNode)
			Expect(ok).To(BeTrue())
			last, ok := split.Last.Variant.(panel.LeafNode)
			Expect(ok).To(BeTrue())
			refs := tabResources(last)
			Expect(refs).To(HaveLen(2))
			arcID := refs[0]
			Expect(arcID.Type).To(Equal(resource.TypeArc))
			var a arc.Arc
			Expect(arcSvc.NewRetrieve().
				Where(arc.MatchKeys(MustSucceed(uuid.Parse(arcID.Key)))).
				Entry(&a).
				Exec(ctx, tx)).To(Succeed())
			Expect(a.Base().Name).To(Equal("Startup Sequence"))
			Expect(textOf(a.Base()).Materialize().Raw).
				To(Equal("stage chamber_pressure -> ok\n"))
			Expect(refs[1].Type).To(Equal(resource.TypeTask))
		})

		It("Should ignore the stray non-member file", func(ctx SpecContext) {
			files := bundle()
			Expect(files).To(HaveKey("README.md"))
			proj := importProject(ctx, files)
			Expect(childrenOf(ctx, project.ResourceID(proj.Key))).To(HaveLen(5))
		})
	})

	It("Should import an exported bundle back as an equal project", func(
		ctx SpecContext,
	) {
		src := createProject(ctx, "Test Stand 12")
		g := createGroup(ctx, "Propulsion", src.ResourceID())
		grouped := createLog(ctx, src.Key, "Pressure")
		moveToGroup(ctx, grouped, g)
		createLog(ctx, src.Key, "Metrics")
		createPanel(
			ctx,
			"Controls",
			src.Key,
			leaf(resourceTab(grouped.ResourceID())),
		)
		files, _ := MustSucceed2(svc.Export(ctx, db, src.Key, xjson.Codec))
		tx = DeferClose(db.OpenTx())
		proj := importProject(ctx, files)
		Expect(proj.Key).ToNot(Equal(src.Key))
		Expect(proj.Name).To(Equal("Test Stand 12"))
		children := childrenOf(ctx, project.ResourceID(proj.Key))
		Expect(children).To(HaveLen(3))
		newGroup := childOfType(children, resource.TypeGroup)
		Expect(newGroup.Name).To(Equal("Propulsion"))
		Expect(newGroup.ID).ToNot(Equal(g.ResourceID()))
		rootLog := childOfType(children, resource.TypeLog)
		Expect(rootLog.Name).To(Equal("Metrics"))
		groupChildren := childrenOf(ctx, newGroup.ID)
		Expect(groupChildren).To(HaveLen(1))
		Expect(groupChildren[0].Name).To(Equal("Pressure"))
		Expect(groupChildren[0].ID).ToNot(Equal(grouped.ResourceID()))
		p := retrievePanel(ctx, childOfType(children, resource.TypePanel).ID)
		Expect(p.Name).To(Equal("Controls"))
		Expect(firstTabResource(p)).To(Equal(groupChildren[0].ID))
	})

	It("Should name the project from the manifest", func(ctx SpecContext) {
		tx = DeferClose(db.OpenTx())
		proj := importProject(ctx, zip.Files{"manifest.json": bundleManifest("Named")})
		Expect(proj.Name).To(Equal("Named"))
		Expect(childrenOf(ctx, project.ResourceID(proj.Key))).To(BeEmpty())
	})

	It("Should fall back to the file name when the manifest is nameless", func(
		ctx SpecContext,
	) {
		tx = DeferClose(db.OpenTx())
		proj := MustSucceed(importFiles(
			ctx,
			zip.Files{"manifest.json": bundleManifest("")},
			"My Project.zip",
		))
		Expect(proj.Name).To(Equal("My Project"))
	})

	It("Should recreate nested directories as nested groups", func(ctx SpecContext) {
		tx = DeferClose(db.OpenTx())
		proj := importProject(ctx, zip.Files{
			"manifest.json":     bundleManifest("Grouped"),
			"A/B/Pressure.json": bundleLog("Pressure"),
		})
		children := childrenOf(ctx, project.ResourceID(proj.Key))
		Expect(children).To(HaveLen(1))
		outer := children[0]
		Expect(outer.ID.Type).To(Equal(resource.TypeGroup))
		Expect(outer.Name).To(Equal("A"))
		inners := childrenOf(ctx, outer.ID)
		Expect(inners).To(HaveLen(1))
		Expect(inners[0].Name).To(Equal("B"))
		logs := childrenOf(ctx, inners[0].ID)
		Expect(logs).To(HaveLen(1))
		Expect(logs[0].Name).To(Equal("Pressure"))
		Expect(logs[0].ID.Type).To(Equal(resource.TypeLog))
	})

	It("Should place a panel under its directory's group", func(ctx SpecContext) {
		tx = DeferClose(db.OpenTx())
		proj := importProject(ctx, zip.Files{
			"manifest.json":      bundleManifest("Paneled"),
			"Deck/Pressure.json": bundleLog("Pressure"),
			"Deck/Controls.json": bundlePanel("Controls", "Deck/Pressure.json"),
		})
		children := childrenOf(ctx, project.ResourceID(proj.Key))
		Expect(children).To(HaveLen(1))
		deck := childrenOf(ctx, children[0].ID)
		Expect(deck).To(HaveLen(2))
		p := retrievePanel(ctx, childOfType(deck, resource.TypePanel).ID)
		Expect(firstTabResource(p)).
			To(Equal(childOfType(deck, resource.TypeLog).ID))
	})

	It("Should leave imported tasks under the rack", func(ctx SpecContext) {
		src := createProject(ctx, "Panel Tasks")
		t := createTask(ctx, "Sequence")
		createPanel(
			ctx, "Controls", src.Key, leaf(resourceTab(t.ResourceID())),
		)
		files, _ := MustSucceed2(svc.Export(ctx, db, src.Key, xjson.Codec))
		tx = DeferClose(db.OpenTx())
		proj := importProject(ctx, files)
		children := childrenOf(ctx, project.ResourceID(proj.Key))
		Expect(children).To(HaveLen(1))
		Expect(children[0].ID.Type).To(Equal(resource.TypePanel))
		newTaskID := firstTabResource(retrievePanel(ctx, children[0].ID))
		Expect(newTaskID.Type).To(Equal(resource.TypeTask))
		Expect(newTaskID).ToNot(Equal(t.ResourceID()))
		Expect(resourceSvc.Retrieve(ctx, tx, newTaskID)).To(ConsistOf(
			resource.Resource{ID: newTaskID, Name: "Sequence"},
		))
	})

	It("Should parent imported Arcs under Arcs", func(ctx SpecContext) {
		src := createProject(ctx, "Panel Arcs")
		a := createArc(ctx, "Startup", "x := 1")
		createPanel(
			ctx, "Controls", src.Key, leaf(resourceTab(a.ResourceID())),
		)
		files, _ := MustSucceed2(svc.Export(ctx, db, src.Key, xjson.Codec))
		// The import lands beside the original, where the name is taken.
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return arcSvc.NewWriter(tx).Apply(
				ctx,
				a.Base().Key,
				arc.NewSetAction(
					arc.SetPayload{Path: []string{"name"}, Value: "Startup Source"},
				),
			)
		})).To(Succeed())
		tx = DeferClose(db.OpenTx())
		proj := importProject(ctx, files)
		children := childrenOf(ctx, project.ResourceID(proj.Key))
		Expect(children).To(HaveLen(1))
		Expect(children[0].ID.Type).To(Equal(resource.TypePanel))
		newArcID := firstTabResource(retrievePanel(ctx, children[0].ID))
		Expect(newArcID.Type).To(Equal(resource.TypeArc))
		Expect(newArcID).ToNot(Equal(a.ResourceID()))
		key := MustSucceed(uuid.Parse(newArcID.Key))
		var res arc.Arc
		Expect(arcSvc.NewRetrieve().
			Where(arc.MatchKeys(key)).
			Entry(&res).
			Exec(ctx, tx)).To(Succeed())
		Expect(res.Base().Name).To(Equal("Startup"))
		Expect(textOf(res.Base()).Materialize().Raw).To(Equal("x := 1"))
		var root directory.Directory
		Expect(arcSvc.NewDirectoryRetrieve().
			Where(directory.MatchNested(false)).
			Entry(&root).
			Exec(ctx, tx)).To(Succeed())
		Expect(root.Name).To(Equal("Arcs"))
		Expect(res.Base().Directory).To(HaveValue(Equal(root.Key)))
	})

	It("Should reject a bundle whose Arc name is taken", func(ctx SpecContext) {
		src := createProject(ctx, "Taken Arcs")
		a := createArc(ctx, "Startup", "x := 1")
		createPanel(
			ctx, "Controls", src.Key, leaf(resourceTab(a.ResourceID())),
		)
		files, _ := MustSucceed2(svc.Export(ctx, db, src.Key, xjson.Codec))
		tx = DeferClose(db.OpenTx())
		Expect(importFiles(ctx, files, "Fallback.zip")).Error().
			To(MatchError(ContainSubstring(`name "Startup" is already used`)))
	})

	It("Should ignore files that are not members", func(ctx SpecContext) {
		tx = DeferClose(db.OpenTx())
		proj := importProject(ctx, zip.Files{
			"manifest.json": bundleManifest("Tidy"),
			"Metrics.json":  bundleLog("Metrics"),
			"README.md":     []byte("# Tidy"),
			".gitignore":    []byte("*.tmp"),
		})
		Expect(childrenOf(ctx, project.ResourceID(proj.Key))).To(HaveLen(1))
	})

	It("Should allow equal member names in different directories", func(
		ctx SpecContext,
	) {
		tx = DeferClose(db.OpenTx())
		proj := importProject(ctx, zip.Files{
			"manifest.json":   bundleManifest("Shadowed"),
			"Pressure.json":   bundleLog("Pressure"),
			"A/Pressure.json": bundleLog("Pressure"),
		})
		Expect(childrenOf(ctx, project.ResourceID(proj.Key))).To(HaveLen(2))
	})

	It("Should reject a member with no registered importer", func(ctx SpecContext) {
		tx = DeferClose(db.OpenTx())
		files := zip.Files{
			"manifest.json": bundleManifest("Plots"),
			"Plot.json":     []byte(`{"version":1,"type":"lineplot","name":"Plot"}`),
		}
		// PathedError breaks the errors.Is chain, so the assertion matches on text.
		Expect(importFiles(ctx, files, "Fallback.zip")).Error().To(SatisfyAll(
			MatchError(ContainSubstring("Plot.json")),
			MatchError(ContainSubstring("no importer registered")),
		))
	})

	It("Should reject a panel referencing a file outside the bundle", func(
		ctx SpecContext,
	) {
		tx = DeferClose(db.OpenTx())
		files := zip.Files{
			"manifest.json": bundleManifest("Dangling"),
			"Controls.json": bundlePanel("Controls", "Missing.json"),
		}
		Expect(importFiles(ctx, files, "Fallback.zip")).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("Controls.json")),
			MatchError(ContainSubstring(`"Missing.json"`)),
		))
	})
})

var _ = Describe("Decode", func() {
	DescribeTable("Should reject an invalid bundle",
		func(ctx SpecContext, files zip.Files, reason string) {
			Expect(svc.Decode(ctx, files, "Fallback.zip")).Error().To(SatisfyAll(
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

	It("Should reject a manifest version newer than supported", func(ctx SpecContext) {
		files := zip.Files{"manifest.json": []byte(
			`{"version":2,"type":"project","name":"Future"}`,
		)}
		Expect(svc.Decode(ctx, files, "Fallback.zip")).Error().To(SatisfyAll(
			MatchError(ContainSubstring("project version 2")),
			MatchError(ContainSubstring("newer than this Core supports")),
		))
	})

	It("Should reject a member that does not decode to an envelope", func(
		ctx SpecContext,
	) {
		files := zip.Files{
			"manifest.json": bundleManifest("Broken"),
			"Bad.json":      []byte("not json"),
		}
		Expect(svc.Decode(ctx, files, "Fallback.zip")).Error().
			To(MatchError(ContainSubstring("Bad.json")))
	})

	It("Should reject a layout whose component file is missing", func(
		ctx SpecContext,
	) {
		files := zip.Files{"LAYOUT.json": legacyLayoutFile(map[string]map[string]any{
			"k1": {"key": "k1", "type": "log", "name": "Metrics"},
		})}
		Expect(svc.Decode(ctx, files, "Legacy.zip")).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`data for layout "k1" not found`)),
		))
	})

	It("Should reject a member at a version newer than supported", func(
		ctx SpecContext,
	) {
		files := zip.Files{
			"manifest.json": bundleManifest("Future"),
			"Metrics.json": []byte(
				`{"version":99,"type":"log","name":"Metrics","channels":[]}`,
			),
		}
		Expect(svc.Decode(ctx, files, "Fallback.zip")).Error().To(SatisfyAll(
			MatchError(ContainSubstring("Metrics.json")),
			MatchError(ContainSubstring("newer than this Core supports")),
		))
	})

	It("Should reject a legacy member at a version newer than supported", func(
		ctx SpecContext,
	) {
		files := zip.Files{
			"LAYOUT.json": legacyLayoutFile(map[string]map[string]any{
				"k1": {"key": "k1", "type": "log", "name": "Metrics"},
			}),
			"Metrics.json": legacyLogState(map[string]any{"version": 99}),
		}
		Expect(svc.Decode(ctx, files, "Legacy.zip")).Error().To(SatisfyAll(
			MatchError(ContainSubstring("Metrics.json")),
			MatchError(ContainSubstring("newer than this Core supports")),
		))
	})
})

func legacyLayoutFile(layouts map[string]map[string]any) []byte {
	GinkgoHelper()
	return MustSucceed(json.Marshal(map[string]any{"layouts": layouts}))
}

// legacyLogState is typeless, recognized as a log by its channels array.
func legacyLogState(extra map[string]any) []byte {
	GinkgoHelper()
	body := map[string]any{
		"version":       "0.0.0",
		"channels":      []any{4, 5},
		"remoteCreated": false,
	}
	maps.Copy(body, extra)
	return MustSucceed(json.Marshal(body))
}

var _ = Describe("Legacy import", func() {
	importLegacy := func(ctx SpecContext, files zip.Files) project.Project {
		GinkgoHelper()
		return MustSucceed(importFiles(ctx, files, "Legacy Project.zip"))
	}
	logLayout := map[string]map[string]any{
		"k1": {"key": "k1", "type": "log", "name": "Metrics"},
	}

	It("Should name the project after the file name", func(ctx SpecContext) {
		tx = DeferClose(db.OpenTx())
		proj := importLegacy(ctx, zip.Files{
			"LAYOUT.json": legacyLayoutFile(nil),
		})
		Expect(proj.Name).To(Equal("Legacy Project"))
		Expect(childrenOf(ctx, project.ResourceID(proj.Key))).To(BeEmpty())
	})

	DescribeTable("Should name a layout's component file after the layout",
		func(ctx SpecContext, fileName string, extra map[string]any) {
			tx = DeferClose(db.OpenTx())
			proj := importLegacy(ctx, zip.Files{
				"LAYOUT.json": legacyLayoutFile(logLayout),
				fileName:      legacyLogState(extra),
			})
			children := childrenOf(ctx, project.ResourceID(proj.Key))
			Expect(children).To(HaveLen(1))
			Expect(children[0].ID.Type).To(Equal(resource.TypeLog))
			Expect(children[0].Name).To(Equal("Metrics"))
		},
		Entry("named after the layout", "Metrics.json", nil),
		Entry("named after the layout key", "k1.json", nil),
		Entry("matched by its body key", "State.json", map[string]any{"key": "k1"}),
		Entry("matched by its body name", "State.json",
			map[string]any{"name": "Metrics"}),
	)

	DescribeTable("Should import a lone surrogate escape as U+FFFD",
		func(ctx SpecContext, layout, member []byte, name string) {
			tx = DeferClose(db.OpenTx())
			proj := importLegacy(
				ctx,
				zip.Files{"LAYOUT.json": layout, "k1.json": member},
			)
			children := childrenOf(ctx, project.ResourceID(proj.Key))
			Expect(children).To(HaveLen(1))
			Expect(children[0].ID.Type).To(Equal(resource.TypeLog))
			Expect(children[0].Name).To(Equal(name))
		},
		Entry("in the layout file",
			[]byte(`{"layouts":{"k1":{"key":"k1","type":"log","name":"Tank \ud83d"}}}`),
			legacyLogState(nil),
			"Tank \uFFFD",
		),
		Entry("in a member file",
			legacyLayoutFile(logLayout),
			[]byte(`{"version":"0.0.0","channels":[4],"name":"Tank \ud83d"}`),
			"Tank \uFFFD",
		),
	)

	It("Should skip layouts whose type is not a frozen legacy type", func(
		ctx SpecContext,
	) {
		tx = DeferClose(db.OpenTx())
		proj := importLegacy(ctx, zip.Files{
			"LAYOUT.json": legacyLayoutFile(map[string]map[string]any{
				"m1": {"key": "m1", "type": "mosaic", "name": "Main"},
			}),
		})
		Expect(childrenOf(ctx, project.ResourceID(proj.Key))).To(BeEmpty())
	})

	It("Should import legacy task files through the leaf machinery", func(
		ctx SpecContext,
	) {
		t := createTask(ctx, "Sequence")
		env := MustSucceed(imexSvc.Export(ctx, db, task.ResourceID(t.Key)))
		tx = DeferClose(db.OpenTx())
		proj := importLegacy(ctx, zip.Files{
			"LAYOUT.json": legacyLayoutFile(map[string]map[string]any{
				"t1": {"key": "t1", "type": "pagerduty_alert", "name": "Sequence"},
			}),
			"Sequence.json": MustSucceed(json.Marshal(env)),
		})
		// The recreated task parents under the rack, so the project stays empty.
		Expect(childrenOf(ctx, project.ResourceID(proj.Key))).To(BeEmpty())
		var tasks []task.Task
		Expect(taskSvc.NewRetrieve().
			Where(task.MatchNames("Sequence")).
			Entries(&tasks).
			Exec(ctx, tx)).To(Succeed())
		Expect(tasks).To(HaveLen(2))
	})
})

func legacyTilingFile(
	layouts map[string]map[string]any,
	mosaics map[string]any,
) []byte {
	GinkgoHelper()
	return MustSucceed(json.Marshal(map[string]any{
		"layouts": layouts, "mosaics": mosaics,
	}))
}

func legacyLeaf(keys ...string) map[string]any {
	tabs := make([]any, len(keys))
	for i, k := range keys {
		tabs[i] = map[string]any{"tabKey": k}
	}
	return map[string]any{"key": 1, "tabs": tabs}
}

var _ = Describe("Legacy tiling", func() {
	BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
	importTiling := func(ctx SpecContext, files zip.Files) project.Project {
		GinkgoHelper()
		return MustSucceed(importFiles(ctx, files, "Legacy Project.zip"))
	}
	tiledLayouts := map[string]map[string]any{
		"main": {"key": "main", "type": "main", "name": "Main"},
		"k1":   {"key": "k1", "type": "log", "name": "Metrics"},
		"k2":   {"key": "k2", "type": "log", "name": "Pressure"},
	}
	twoLogFiles := func(extra zip.Files) zip.Files {
		GinkgoHelper()
		files := zip.Files{
			"Metrics.json":  legacyLogState(nil),
			"Pressure.json": legacyLogState(nil),
		}
		maps.Copy(files, extra)
		return files
	}

	It("Should recreate the main window's mosaic as a panel", func(ctx SpecContext) {
		proj := importTiling(ctx, twoLogFiles(zip.Files{
			"LAYOUT.json": legacyTilingFile(tiledLayouts, map[string]any{
				"main": map[string]any{"root": legacyLeaf("k1", "k2")},
			}),
		}))
		children := childrenOf(ctx, project.ResourceID(proj.Key))
		Expect(children).To(HaveLen(3))
		p := retrievePanel(ctx, childOfType(children, resource.TypePanel).ID)
		Expect(p.Name).To(Equal("Main"))
		l, ok := p.Root.Variant.(panel.LeafNode)
		Expect(ok).To(BeTrue())
		Expect(l.Tabs).To(HaveLen(2))
		for _, t := range l.Tabs {
			r, ok := t.Variant.(panel.ResourceTab)
			Expect(ok).To(BeTrue())
			Expect(children).To(ContainElement(HaveField("ID", r.Resource)))
		}
	})

	It("Should preserve split direction and fractional size", func(ctx SpecContext) {
		proj := importTiling(ctx, twoLogFiles(zip.Files{
			"LAYOUT.json": legacyTilingFile(tiledLayouts, map[string]any{
				"main": map[string]any{"root": map[string]any{
					"key":       1,
					"direction": "y",
					"size":      0.3,
					"first":     legacyLeaf("k1"),
					"last":      legacyLeaf("k2"),
				}},
			}),
		}))
		children := childrenOf(ctx, project.ResourceID(proj.Key))
		p := retrievePanel(ctx, childOfType(children, resource.TypePanel).ID)
		split, ok := p.Root.Variant.(panel.SplitNode)
		Expect(ok).To(BeTrue())
		Expect(split.Direction).To(Equal(spatial.DirectionY))
		Expect(float64(split.Size)).To(BeNumerically("~", 0.3, 1e-9))
	})

	It("Should fall back to an even split for a non-fractional size", func(
		ctx SpecContext,
	) {
		proj := importTiling(ctx, twoLogFiles(zip.Files{
			"LAYOUT.json": legacyTilingFile(tiledLayouts, map[string]any{
				"main": map[string]any{"root": map[string]any{
					"key":   1,
					"size":  250,
					"first": legacyLeaf("k1"),
					"last":  legacyLeaf("k2"),
				}},
			}),
		}))
		children := childrenOf(ctx, project.ResourceID(proj.Key))
		p := retrievePanel(ctx, childOfType(children, resource.TypePanel).ID)
		split, ok := p.Root.Variant.(panel.SplitNode)
		Expect(ok).To(BeTrue())
		Expect(split.Direction).To(Equal(spatial.DirectionX))
		Expect(float64(split.Size)).To(BeNumerically("~", 0.5, 1e-9))
	})

	It("Should drop unresolvable tabs and collapse emptied splits", func(
		ctx SpecContext,
	) {
		proj := importTiling(ctx, zip.Files{
			"LAYOUT.json": legacyTilingFile(tiledLayouts, map[string]any{
				"main": map[string]any{"root": map[string]any{
					"key":   1,
					"first": legacyLeaf("k1", "docs"),
					"last":  legacyLeaf("missing"),
				}},
			}),
			"Metrics.json":  legacyLogState(nil),
			"Pressure.json": legacyLogState(nil),
		})
		children := childrenOf(ctx, project.ResourceID(proj.Key))
		p := retrievePanel(ctx, childOfType(children, resource.TypePanel).ID)
		l, ok := p.Root.Variant.(panel.LeafNode)
		Expect(ok).To(BeTrue())
		Expect(l.Tabs).To(HaveLen(1))
	})

	It("Should create one panel per window named after its layout entry", func(
		ctx SpecContext,
	) {
		layouts := map[string]map[string]any{
			"main": {"key": "main", "type": "main", "name": "Main"},
			"w1":   {"key": "w1", "type": "mosaicWindow", "name": "Aux"},
			"k1":   {"key": "k1", "type": "log", "name": "Metrics"},
			"k2":   {"key": "k2", "type": "log", "name": "Pressure"},
		}
		proj := importTiling(ctx, twoLogFiles(zip.Files{
			"LAYOUT.json": legacyTilingFile(layouts, map[string]any{
				"main": map[string]any{"root": legacyLeaf("k1")},
				"w1":   map[string]any{"root": legacyLeaf("k2")},
			}),
		}))
		children := childrenOf(ctx, project.ResourceID(proj.Key))
		names := make([]string, 0, 2)
		for _, c := range children {
			if c.ID.Type == resource.TypePanel {
				names = append(names, c.Name)
			}
		}
		Expect(names).To(ConsistOf("Main", "Aux"))
	})

	It("Should import the documents alone when the tiling does not parse", func(
		ctx SpecContext,
	) {
		proj := importTiling(ctx, zip.Files{
			"LAYOUT.json": legacyTilingFile(map[string]map[string]any{
				"k1": {"key": "k1", "type": "log", "name": "Metrics"},
			}, map[string]any{"main": "garbage"}),
			"Metrics.json": legacyLogState(nil),
		})
		children := childrenOf(ctx, project.ResourceID(proj.Key))
		Expect(children).To(HaveLen(1))
		Expect(children[0].ID.Type).To(Equal(resource.TypeLog))
	})
})
