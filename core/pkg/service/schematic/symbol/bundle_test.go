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
	"context"
	"encoding/json"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/encoding"
	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// errEncode is the failure failEncoder reports.
var errEncode = errors.New("the encoder refused the value")

// failEncoder encodes as JSON but refuses half of a bundle: the manifest when
// onManifest is true, every symbol envelope otherwise.
type failEncoder struct{ onManifest bool }

var _ encoding.FileEncoder = failEncoder{}

func (e failEncoder) refuses(value any) bool {
	_, isManifest := value.(imex.Manifest)
	return isManifest == e.onManifest
}

func (e failEncoder) Encode(ctx context.Context, value any) ([]byte, error) {
	if e.refuses(value) {
		return nil, errEncode
	}
	return xjson.Codec.Encode(ctx, value)
}

func (e failEncoder) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	if e.refuses(value) {
		return errEncode
	}
	return xjson.Codec.EncodeStream(ctx, w, value)
}

func (failEncoder) Extension() string { return xjson.Codec.Extension() }

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
	exportFiles := func(ctx SpecContext, key group.Key) zip.Files {
		GinkgoHelper()
		files, _ := MustSucceed2(svc.ExportGroup(ctx, key, xjson.Codec))
		return files
	}
	fileNames := func(files zip.Files) []string {
		return slices.Collect(maps.Keys(files))
	}
	manifestOf := func(files zip.Files) imex.Manifest {
		GinkgoHelper()
		var m imex.Manifest
		Expect(json.Unmarshal(files["manifest.json"], &m)).To(Succeed())
		return m
	}

	It("Should write one file per symbol beside the manifest", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		createSymbol(ctx, g, "Inlet")
		createSymbol(ctx, g, "Outlet")
		Expect(fileNames(exportFiles(ctx, g.Key))).
			To(ConsistOf("Inlet.json", "Outlet.json", "manifest.json"))
	})
	It("Should stamp the manifest with the group's name", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		createSymbol(ctx, g, "Inlet")
		Expect(manifestOf(exportFiles(ctx, g.Key))).To(Equal(
			imex.Manifest{Version: 2, Type: "symbol_group", Name: "Valves"},
		))
	})
	It("Should write each member as its leaf export envelope", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		sym := createSymbol(ctx, g, "Inlet")
		env := MustSucceed(svc.Export(ctx, symbol.OntologyID(sym.Key)))
		Expect(exportFiles(ctx, g.Key)["Inlet.json"]).
			To(Equal(MustSucceed(json.Marshal(env))))
	})
	It("Should report every exported symbol as a member", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		sym := createSymbol(ctx, g, "Inlet")
		_, members := MustSucceed2(svc.ExportGroup(ctx, g.Key, xjson.Codec))
		Expect(members).To(ConsistOf(symbol.OntologyID(sym.Key)))
	})
	It("Should export an empty group as a manifest alone", func(ctx SpecContext) {
		g := createRoot(ctx, "Empty")
		files, members := MustSucceed2(svc.ExportGroup(ctx, g.Key, xjson.Codec))
		Expect(fileNames(files)).To(ConsistOf("manifest.json"))
		Expect(members).To(BeEmpty())
		Expect(manifestOf(files).Name).To(Equal("Empty"))
	})
	It("Should replace characters a file name cannot hold", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		createSymbol(ctx, g, "in/let:1")
		Expect(exportFiles(ctx, g.Key)).To(HaveKey("in_let_1.json"))
	})
	DescribeTable("Should name a symbol a file name cannot hold with an underscore",
		func(ctx SpecContext, name string) {
			g := createRoot(ctx, "Valves")
			createSymbol(ctx, g, name)
			Expect(exportFiles(ctx, g.Key)).To(HaveKey("_.json"))
		},
		Entry("dots alone", "..."),
		Entry("spaces alone", "   "),
	)
	It("Should push a symbol off a Windows device name", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		createSymbol(ctx, g, "NUL")
		Expect(exportFiles(ctx, g.Key)).To(HaveKey("_NUL.json"))
	})
	It("Should shorten a name a file name cannot hold whole", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		createSymbol(ctx, g, strings.Repeat("a", 400))
		Expect(fileNames(exportFiles(ctx, g.Key))).
			To(ConsistOf("manifest.json", HaveLen(255)))
	})
	It("Should return not found for a missing group", func(ctx SpecContext) {
		Expect(svc.ExportGroup(ctx, uuid.New(), xjson.Codec)).Error().
			To(MatchError(query.ErrNotFound))
	})
	It("Should skip a child that is not a schematic symbol", func(ctx SpecContext) {
		g := createRoot(ctx, "Valves")
		createGroup(ctx, "Nested", g.OntologyID())
		sym := createSymbol(ctx, g, "Ball Valve")
		files, members := MustSucceed2(svc.ExportGroup(ctx, g.Key, xjson.Codec))
		Expect(fileNames(files)).To(ConsistOf("manifest.json", "Ball Valve.json"))
		Expect(members).To(ConsistOf(symbol.OntologyID(sym.Key)))
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
		Entry("differing past the file name limit",
			strings.Repeat("a", 300)+"one", strings.Repeat("a", 300)+"two"),
		Entry("differing only in case", "Inlet", "inlet"),
		Entry("sanitized to the same name", "in/let", `in\let`),
		Entry("sanitized to nothing", "...", "   "),
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
	It("Should return the encoder's error when a symbol fails to encode", func(
		ctx SpecContext,
	) {
		g := createRoot(ctx, "Valves")
		createSymbol(ctx, g, "Inlet")
		Expect(svc.ExportGroup(ctx, g.Key, failEncoder{})).Error().
			To(MatchError(errEncode))
	})
	It("Should return the encoder's error when the manifest fails to encode", func(
		ctx SpecContext,
	) {
		g := createRoot(ctx, "Valves")
		createSymbol(ctx, g, "Inlet")
		Expect(svc.ExportGroup(ctx, g.Key, failEncoder{onManifest: true})).Error().
			To(MatchError(errEncode))
	})
})
