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
	"maps"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// legacyLayoutFile builds a LAYOUT.json holding one layout record per entry.
func legacyLayoutFile(layouts map[string]map[string]any) []byte {
	GinkgoHelper()
	return MustSucceed(json.Marshal(map[string]any{"layouts": layouts}))
}

// legacyLogState is a frozen legacy Console log state: typeless, recognized by its
// channels array.
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
		return MustSucceed(svc.Import(ctx, tx, files, "Legacy Project.zip"))
	}
	logLayout := map[string]map[string]any{
		"k1": {"key": "k1", "type": "log", "name": "Metrics"},
	}

	It("Should name the project after the file name", func(ctx SpecContext) {
		proj := importLegacy(ctx, zip.Files{
			"LAYOUT.json": legacyLayoutFile(nil),
		})
		Expect(proj.Name).To(Equal("Legacy Project"))
		Expect(childrenOf(ctx, project.OntologyID(proj.Key))).To(BeEmpty())
	})

	DescribeTable("Should locate a layout's component file",
		func(ctx SpecContext, fileName string, extra map[string]any, logName string) {
			proj := importLegacy(ctx, zip.Files{
				"LAYOUT.json": legacyLayoutFile(logLayout),
				fileName:      legacyLogState(extra),
			})
			children := childrenOf(ctx, project.OntologyID(proj.Key))
			Expect(children).To(HaveLen(1))
			Expect(children[0].ID.Type).To(Equal(ontology.ResourceTypeLog))
			Expect(children[0].Name).To(Equal(logName))
		},
		Entry("named after the layout", "Metrics.json", nil, "Metrics"),
		Entry("named after the layout key", "k1.json", nil, "k1"),
		Entry("matched by its body key", "State.json",
			map[string]any{"key": "k1"}, "State"),
		Entry("matched by its body name", "State.json",
			map[string]any{"name": "Metrics"}, "Metrics"),
	)

	It("Should skip layouts whose type is not a frozen legacy type", func(
		ctx SpecContext,
	) {
		proj := importLegacy(ctx, zip.Files{
			"LAYOUT.json": legacyLayoutFile(map[string]map[string]any{
				"m1": {"key": "m1", "type": "mosaic", "name": "Main"},
			}),
		})
		Expect(childrenOf(ctx, project.OntologyID(proj.Key))).To(BeEmpty())
	})

	It("Should import legacy task files through the leaf machinery", func(
		ctx SpecContext,
	) {
		t := createTask(ctx, "Sequence")
		env := MustSucceed(imexSvc.Export(ctx, task.OntologyID(t.Key)))
		proj := importLegacy(ctx, zip.Files{
			"LAYOUT.json": legacyLayoutFile(map[string]map[string]any{
				"t1": {"key": "t1", "type": "pagerduty_alert", "name": "Sequence"},
			}),
			"Sequence.json": MustSucceed(json.Marshal(env)),
		})
		// The recreated task parents under the rack, so the project stays empty.
		Expect(childrenOf(ctx, project.OntologyID(proj.Key))).To(BeEmpty())
		var tasks []task.Task
		Expect(taskSvc.NewRetrieve().
			Where(task.MatchNames("Sequence")).
			Entries(&tasks).
			Exec(ctx, tx)).To(Succeed())
		Expect(tasks).To(HaveLen(2))
	})

	It("Should reject a layout whose component file is missing", func(
		ctx SpecContext,
	) {
		files := zip.Files{"LAYOUT.json": legacyLayoutFile(logLayout)}
		Expect(svc.Import(ctx, tx, files, "Legacy.zip")).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`data for layout "k1" not found`)),
		))
	})

	It("Should list the legacy members' types for access checks", func(
		ctx SpecContext,
	) {
		objects := MustSucceed(svc.ImportObjects(ctx, zip.Files{
			"LAYOUT.json":  legacyLayoutFile(logLayout),
			"Metrics.json": legacyLogState(nil),
		}))
		Expect(objects).To(ConsistOf(
			ontology.ID{Type: ontology.ResourceTypeProject},
			ontology.ID{Type: ontology.ResourceTypeLog},
		))
	})
})

// legacyTilingFile builds a LAYOUT.json holding layout records beside per-window
// mosaic trees.
func legacyTilingFile(
	layouts map[string]map[string]any,
	mosaics map[string]any,
) []byte {
	GinkgoHelper()
	return MustSucceed(json.Marshal(map[string]any{
		"layouts": layouts, "mosaics": mosaics,
	}))
}

// legacyLeaf builds a legacy mosaic leaf node holding one tab per layout key.
func legacyLeaf(keys ...string) map[string]any {
	tabs := make([]any, len(keys))
	for i, k := range keys {
		tabs[i] = map[string]any{"tabKey": k}
	}
	return map[string]any{"key": 1, "tabs": tabs}
}

var _ = Describe("Legacy tiling", func() {
	importTiling := func(ctx SpecContext, files zip.Files) project.Project {
		GinkgoHelper()
		return MustSucceed(svc.Import(ctx, tx, files, "Legacy Project.zip"))
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
		children := childrenOf(ctx, project.OntologyID(proj.Key))
		Expect(children).To(HaveLen(3))
		p := retrievePanel(ctx, childOfType(children, ontology.ResourceTypePanel).ID)
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
		children := childrenOf(ctx, project.OntologyID(proj.Key))
		p := retrievePanel(ctx, childOfType(children, ontology.ResourceTypePanel).ID)
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
		children := childrenOf(ctx, project.OntologyID(proj.Key))
		p := retrievePanel(ctx, childOfType(children, ontology.ResourceTypePanel).ID)
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
		children := childrenOf(ctx, project.OntologyID(proj.Key))
		p := retrievePanel(ctx, childOfType(children, ontology.ResourceTypePanel).ID)
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
		children := childrenOf(ctx, project.OntologyID(proj.Key))
		names := make([]string, 0, 2)
		for _, c := range children {
			if c.ID.Type == ontology.ResourceTypePanel {
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
		children := childrenOf(ctx, project.OntologyID(proj.Key))
		Expect(children).To(HaveLen(1))
		Expect(children[0].ID.Type).To(Equal(ontology.ResourceTypeLog))
	})

	It("Should list the panel type when the tiling converts", func(ctx SpecContext) {
		objects := MustSucceed(svc.ImportObjects(ctx, twoLogFiles(zip.Files{
			"LAYOUT.json": legacyTilingFile(tiledLayouts, map[string]any{
				"main": map[string]any{"root": legacyLeaf("k1")},
			}),
		})))
		Expect(objects).To(ConsistOf(
			ontology.ID{Type: ontology.ResourceTypeProject},
			ontology.ID{Type: ontology.ResourceTypeLog},
			ontology.ID{Type: ontology.ResourceTypePanel},
		))
	})
})
