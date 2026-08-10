// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"encoding/json"
	"maps"
	"slices"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// ExportGroup reads committed data, so every fixture below is created outside the
// per-spec tx and deleted afterwards to keep the shared DB's counts intact.
var _ = Describe("ExportGroup", func() {
	createGroup := func(ctx SpecContext, name string, parent ontology.ID) group.Group {
		GinkgoHelper()
		g := MustSucceed(groupSvc.NewWriter(nil).Create(ctx, name, parent))
		DeferCleanup(func(ctx SpecContext) {
			Expect(groupSvc.NewWriter(nil).Delete(ctx, g.Key)).To(Succeed())
		})
		return g
	}
	createRoot := func(ctx SpecContext, name string) group.Group {
		GinkgoHelper()
		return createGroup(ctx, name, proj.OntologyID())
	}
	createSymbol := func(ctx SpecContext, g group.Group, name string) symbol.Symbol {
		GinkgoHelper()
		sym := symbol.Symbol{
			Name: name,
			Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
		}
		Expect(svc.NewWriter(nil).Create(ctx, &sym, g.OntologyID())).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).Delete(ctx, sym.Key)).To(Succeed())
		})
		return sym
	}
	fileNames := func(bundle symbol.GroupBundle) []string {
		return slices.Collect(maps.Keys(bundle.Files))
	}
	manifestOf := func(bundle symbol.GroupBundle) symbol.GroupManifest {
		GinkgoHelper()
		var m symbol.GroupManifest
		Expect(json.Unmarshal(bundle.Files["manifest.json"], &m)).To(Succeed())
		return m
	}

	It("Should write one file per symbol beside the manifest", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		createSymbol(ctx, g, "Inlet")
		createSymbol(ctx, g, "Outlet")
		Expect(fileNames(MustSucceed(svc.ExportGroup(ctx, g.Key, xjson.Codec)))).
			To(ConsistOf("Inlet.json", "Outlet.json", "manifest.json"))
	})
	It("Should stamp the manifest with the group's name", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		createSymbol(ctx, g, "Inlet")
		Expect(
			manifestOf(MustSucceed(svc.ExportGroup(ctx, g.Key, xjson.Codec))),
		).To(Equal(
			symbol.GroupManifest{Version: 2, Type: "symbol_group", Name: "Valves"},
		))
	})
	It("Should write each member as its leaf export envelope", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		sym := createSymbol(ctx, g, "Inlet")
		bundle := MustSucceed(svc.ExportGroup(ctx, g.Key, xjson.Codec))
		env := MustSucceed(svc.Export(ctx, symbol.OntologyID(sym.Key)))
		Expect(bundle.Files["Inlet.json"]).To(Equal(MustSucceed(json.Marshal(env))))
	})
	It("Should report every exported symbol as a member", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		sym := createSymbol(ctx, g, "Inlet")
		Expect(MustSucceed(svc.ExportGroup(ctx, g.Key, xjson.Codec)).Members).
			To(ConsistOf(symbol.OntologyID(sym.Key)))
	})
	It("Should export an empty group as a manifest alone", func(ctx SpecContext) {
		g := createRoot(ctx, "Empty")
		bundle := MustSucceed(svc.ExportGroup(ctx, g.Key, xjson.Codec))
		Expect(fileNames(bundle)).To(ConsistOf("manifest.json"))
		Expect(bundle.Members).To(BeEmpty())
		Expect(manifestOf(bundle).Name).To(Equal("Empty"))
	})
	It("Should replace characters a file name cannot hold", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		createSymbol(ctx, g, "in/let:1")
		Expect(MustSucceed(svc.ExportGroup(ctx, g.Key, xjson.Codec)).Files).
			To(HaveKey("in_let_1.json"))
	})
	DescribeTable("Should reject a symbol a file name cannot hold",
		func(ctx SpecContext, name string) {
			g := createRoot(ctx, "Valves")
			createSymbol(ctx, g, name)
			Expect(svc.ExportGroup(ctx, g.Key, xjson.Codec)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("holds no character a file name can keep")),
			))
		},
		Entry("dots alone", "..."),
		Entry("spaces alone", "   "),
	)
	It("Should push a symbol off a Windows device name", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		createSymbol(ctx, g, "NUL")
		Expect(MustSucceed(svc.ExportGroup(ctx, g.Key, xjson.Codec)).Files).
			To(HaveKey("_NUL.json"))
	})
	It("Should return not found for a missing group", func(ctx SpecContext) {
		Expect(svc.ExportGroup(ctx, uuid.New(), xjson.Codec)).Error().
			To(MatchError(query.ErrNotFound))
	})
	It("Should reject a group holding a non-symbol child", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		createGroup(ctx, "Nested", g.OntologyID())
		Expect(svc.ExportGroup(ctx, g.Key, xjson.Codec)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("not a schematic symbol")),
		))
	})
	DescribeTable("Should reject two symbols that take one file name",
		func(ctx SpecContext, first, second string) {
			g := createRoot(ctx, "Valves")
			createSymbol(ctx, g, first)
			createSymbol(ctx, g, second)
			Expect(svc.ExportGroup(ctx, g.Key, xjson.Codec)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("both export to")),
			))
		},
		Entry("identical names", "Inlet", "Inlet"),
		Entry("differing only in case", "Inlet", "inlet"),
		Entry("sanitized to the same name", "in/let", `in\let`),
	)
	DescribeTable("Should reject a symbol taking a reserved file name",
		func(ctx SpecContext, name string) {
			g := createRoot(ctx, "Valves")
			createSymbol(ctx, g, name)
			Expect(svc.ExportGroup(ctx, g.Key, xjson.Codec)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("reserved file name")),
			))
		},
		Entry("manifest", "manifest"),
		Entry("MANIFEST", "MANIFEST"),
	)
})

var _ = Describe("GroupBundle", func() {
	It("Should marshal to the bundle's files", func() {
		files := zip.Files{"manifest.json": []byte(`{"version":2}`)}
		Expect(symbol.GroupBundle{Files: files}.MarshalZIP()).To(Equal(files))
	})
})
