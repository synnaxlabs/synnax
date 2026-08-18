// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	"encoding/json"
	"maps"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/project/legacy"
	"github.com/synnaxlabs/x/encoding/zip"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func layoutFile(layouts map[string]map[string]any, mosaics map[string]any) []byte {
	GinkgoHelper()
	body := map[string]any{"layouts": layouts}
	if mosaics != nil {
		body["mosaics"] = mosaics
	}
	return MustSucceed(json.Marshal(body))
}

// logState is typeless, recognized as a log by its channels array.
func logState(extra map[string]any) []byte {
	GinkgoHelper()
	body := map[string]any{
		"version":       "0.0.0",
		"channels":      []any{4, 5},
		"remoteCreated": false,
	}
	maps.Copy(body, extra)
	return MustSucceed(json.Marshal(body))
}

func leaf(keys ...string) map[string]any {
	tabs := make([]any, len(keys))
	for i, k := range keys {
		tabs[i] = map[string]any{"tabKey": k}
	}
	return map[string]any{"key": 1, "tabs": tabs}
}

var logLayouts = map[string]map[string]any{
	"k1": {"key": "k1", "type": "log", "name": "Metrics"},
	"k2": {"key": "k2", "type": "log", "name": "Pressure"},
}

func logFiles() zip.Files {
	GinkgoHelper()
	return zip.Files{
		"Metrics.json":  logState(nil),
		"Pressure.json": logState(nil),
	}
}

var _ = Describe("Members", func() {
	It("Should decode importable members in layout-key order", func(ctx SpecContext) {
		members := MustSucceed(legacy.Members(
			ctx, imexSvc, layoutFile(logLayouts, nil), logFiles(),
		))
		Expect(members).To(HaveLen(2))
		Expect(members[0].LayoutKey).To(Equal("k1"))
		Expect(members[0].Path).To(Equal("Metrics.json"))
		Expect(members[0].Env.Type).To(Equal(string(ontology.ResourceTypeLog)))
		Expect(members[1].LayoutKey).To(Equal("k2"))
	})

	It("Should skip layouts whose type is not a frozen legacy type", func(
		ctx SpecContext,
	) {
		layouts := map[string]map[string]any{
			"m1": {"key": "m1", "type": "mosaic", "name": "Main"},
		}
		Expect(legacy.Members(
			ctx, imexSvc, layoutFile(layouts, nil), zip.Files{},
		)).To(BeEmpty())
	})

	It("Should reject a layout whose component file is missing", func(
		ctx SpecContext,
	) {
		Expect(legacy.Members(
			ctx, imexSvc, layoutFile(logLayouts, nil), zip.Files{},
		)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`data for layout "k1" not found`)),
		))
	})

	It("Should reject an undecodable layout slice", func(ctx SpecContext) {
		Expect(legacy.Members(ctx, imexSvc, []byte("garbage"), zip.Files{})).
			Error().To(MatchError(ContainSubstring(legacy.LayoutFileName)))
	})
})

var _ = Describe("HasPanels", func() {
	members := []legacy.Member{{LayoutKey: "k1"}}

	It("Should report a tiling whose tabs resolve to members", func(ctx SpecContext) {
		data := layoutFile(logLayouts, map[string]any{
			"main": map[string]any{"root": leaf("k1")},
		})
		Expect(legacy.HasPanels(ctx, data, members)).To(BeTrue())
	})

	It("Should ignore a tiling with no resolvable tabs", func(ctx SpecContext) {
		data := layoutFile(logLayouts, map[string]any{
			"main": map[string]any{"root": leaf("missing")},
		})
		Expect(legacy.HasPanels(ctx, data, members)).To(BeFalse())
	})

	It("Should ignore a tiling that does not parse", func(ctx SpecContext) {
		data := layoutFile(logLayouts, map[string]any{"main": "garbage"})
		Expect(legacy.HasPanels(ctx, data, members)).To(BeFalse())
	})
})

var _ = Describe("CreatePanels", func() {
	var (
		parent ontology.ID
		refs   map[string]ontology.ID
	)
	BeforeEach(func(ctx SpecContext) {
		g := MustSucceed(
			groupSvc.NewWriter(tx).Create(ctx, "Legacy Parent", ontology.RootID),
		)
		parent = g.OntologyID()
		l := log.Log{Name: "Metrics"}
		Expect(logSvc.NewWriter(tx).Create(ctx, project.Key{}, &l)).To(Succeed())
		refs = map[string]ontology.ID{"k1": log.OntologyID(l.Key)}
	})
	panelsUnder := func(ctx SpecContext, parent ontology.ID) []panel.Panel {
		GinkgoHelper()
		var children []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(parent).
			TraverseTo(ontology.ChildrenTraverser).
			Entries(&children).
			Exec(ctx, tx)).To(Succeed())
		var ids []ontology.ID
		for _, c := range children {
			if c.ID.Type == ontology.ResourceTypePanel {
				ids = append(ids, c.ID)
			}
		}
		if len(ids) == 0 {
			return nil
		}
		keys := MustSucceed(panel.KeysFromOntologyIDs(ids))
		var panels []panel.Panel
		Expect(panelSvc.NewRetrieve().
			Where(panel.MatchKeys(keys...)).
			Entries(&panels).
			Exec(ctx, tx)).To(Succeed())
		return panels
	}

	It("Should recreate a window's mosaic as a panel named by its layout", func(
		ctx SpecContext,
	) {
		layouts := map[string]map[string]any{
			"main": {"key": "main", "type": "main", "name": "Main"},
		}
		data := layoutFile(layouts, map[string]any{
			"main": map[string]any{"root": leaf("k1")},
		})
		Expect(legacy.CreatePanels(
			ctx, tx, panelSvc, parent, data, refs,
		)).To(Succeed())
		panels := panelsUnder(ctx, parent)
		Expect(panels).To(HaveLen(1))
		Expect(panels[0].Name).To(Equal("Main"))
		l, ok := panels[0].Root.Variant.(panel.LeafNode)
		Expect(ok).To(BeTrue())
		Expect(l.Tabs).To(HaveLen(1))
	})

	It("Should create no panels when the tiling does not parse", func(
		ctx SpecContext,
	) {
		data := layoutFile(nil, map[string]any{"main": "garbage"})
		Expect(legacy.CreatePanels(
			ctx, tx, panelSvc, parent, data, refs,
		)).To(Succeed())
		Expect(panelsUnder(ctx, parent)).To(BeEmpty())
	})

	It("Should skip windows with no resolvable tabs", func(ctx SpecContext) {
		data := layoutFile(nil, map[string]any{
			"w1": map[string]any{"root": leaf("missing")},
		})
		Expect(legacy.CreatePanels(
			ctx, tx, panelSvc, parent, data, refs,
		)).To(Succeed())
		Expect(panelsUnder(ctx, parent)).To(BeEmpty())
	})
})
