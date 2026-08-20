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
	"os"
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
	DescribeTable("Should suffix the second of two symbols taking one file name",
		func(ctx SpecContext, first, second, firstFile, secondFile string) {
			g := createRoot(ctx, "Valves")
			createSymbol(ctx, g, first)
			createSymbol(ctx, g, second)
			files, _ := MustSucceed2(svc.ExportGroup(ctx, g.Key, xjson.Codec))
			Expect(fileNames(files)).
				To(ConsistOf("manifest.json", firstFile, secondFile))
		},
		Entry("identical names", "Inlet", "Inlet", "Inlet.json", "Inlet (1).json"),
		Entry("differing past the file name limit",
			strings.Repeat("a", 300)+"one", strings.Repeat("a", 300)+"two",
			strings.Repeat("a", 250)+".json", strings.Repeat("a", 246)+" (1).json"),
		Entry("differing only in case", "Inlet", "inlet",
			"Inlet.json", "inlet (1).json"),
		Entry("sanitized to the same name", "in/let", `in\let`,
			"in_let.json", "in_let (1).json"),
		Entry("sanitized to nothing", "...", "   ", "_.json", "_ (1).json"),
	)
	DescribeTable("Should suffix a symbol taking a reserved file name",
		func(ctx SpecContext, name, file string) {
			g := createRoot(ctx, "Valves")
			createSymbol(ctx, g, name)
			files, _ := MustSucceed2(svc.ExportGroup(ctx, g.Key, xjson.Codec))
			Expect(fileNames(files)).To(ConsistOf("manifest.json", file))
		},
		Entry("manifest", "manifest", "manifest (1).json"),
		Entry("MANIFEST", "MANIFEST", "MANIFEST (1).json"),
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

// Imports run on the per-spec tx so created rows roll back and the shared DB's counts
// stay intact for the other specs.
var _ = Describe("ImportGroup", func() {
	manifest := func(version int, name string) []byte {
		GinkgoHelper()
		return MustSucceed(json.Marshal(map[string]any{
			"version": version, "type": "symbol_group", "name": name,
		}))
	}
	member := func(name string) []byte {
		GinkgoHelper()
		return MustSucceed(json.Marshal(map[string]any{
			"version": 2,
			"type":    "schematic_symbol",
			"name":    name,
			"data":    map[string]any{"svg": "<svg/>", "variant": "valve"},
		}))
	}
	importGroup := func(ctx SpecContext, files zip.Files) group.Group {
		GinkgoHelper()
		return MustSucceed(svc.ImportGroup(ctx, tx, files))
	}
	childrenOf := func(ctx SpecContext, id ontology.ID) []ontology.Resource {
		GinkgoHelper()
		var children []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(id).
			TraverseTo(ontology.ChildrenTraverser).
			Entries(&children).
			Exec(ctx, tx)).To(Succeed())
		return children
	}
	childNames := func(ctx SpecContext, g group.Group) []string {
		GinkgoHelper()
		children := childrenOf(ctx, g.OntologyID())
		names := make([]string, len(children))
		for i, child := range children {
			Expect(child.ID.Type).To(Equal(ontology.ResourceTypeSchematicSymbol))
			names[i] = child.Name
		}
		return names
	}

	It("Should create a group named by the manifest under the permanent group", func(
		ctx SpecContext,
	) {
		g := importGroup(ctx, zip.Files{"manifest.json": manifest(2, "Valves")})
		Expect(g.Name).To(Equal("Valves"))
		Expect(childrenOf(ctx, svc.Group().OntologyID())).
			To(ContainElement(HaveField("ID", g.OntologyID())))
	})
	It("Should import every member into the created group", func(ctx SpecContext) {
		g := importGroup(ctx, zip.Files{
			"manifest.json": manifest(2, "Valves"),
			"Inlet.json":    member("Inlet"),
			"Outlet.json":   member("Outlet"),
		})
		Expect(childNames(ctx, g)).To(ConsistOf("Inlet", "Outlet"))
	})
	It("Should import an exported bundle back as an equal group", func(
		ctx SpecContext,
	) {
		src := MustSucceed(
			groupSvc.NewWriter(nil).Create(ctx, "Valves", proj.OntologyID()),
		)
		DeferCleanup(func(ctx SpecContext) {
			Expect(groupSvc.NewWriter(nil).Delete(ctx, src.Key)).To(Succeed())
		})
		sym := symbol.Symbol{
			Name: "Inlet",
			Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
		}
		Expect(svc.NewWriter(nil).Create(ctx, &sym, src.OntologyID())).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).Delete(ctx, sym.Key)).To(Succeed())
		})
		files, _ := MustSucceed2(svc.ExportGroup(ctx, src.Key, xjson.Codec))
		g := importGroup(ctx, files)
		Expect(g.Key).ToNot(Equal(src.Key))
		Expect(g.Name).To(Equal("Valves"))
		Expect(childNames(ctx, g)).To(ConsistOf("Inlet"))
	})
	It("Should name a nameless member after its file", func(ctx SpecContext) {
		nameless := MustSucceed(json.Marshal(map[string]any{
			"version": 2,
			"type":    "schematic_symbol",
			"data":    map[string]any{"svg": "<svg/>", "variant": "valve"},
		}))
		g := importGroup(ctx, zip.Files{
			"manifest.json":         manifest(2, "Valves"),
			"Standalone Valve.json": nameless,
		})
		Expect(childNames(ctx, g)).To(ConsistOf("Standalone Valve"))
	})
	It("Should import a legacy Console symbol file", func(ctx SpecContext) {
		legacy := MustSucceed(os.ReadFile("versions/testdata/import_console.json"))
		g := importGroup(ctx, zip.Files{
			"manifest.json": manifest(2, "Valves"),
			"Console.json":  legacy,
		})
		Expect(childNames(ctx, g)).To(ConsistOf("Console Symbol"))
	})
	It("Should ignore files that are not members", func(ctx SpecContext) {
		g := importGroup(ctx, zip.Files{
			"manifest.json": manifest(2, "Valves"),
			"Inlet.json":    member("Inlet"),
			"README.md":     []byte("# Valves"),
			".gitignore":    []byte("*.tmp"),
		})
		Expect(childNames(ctx, g)).To(ConsistOf("Inlet"))
	})
	It("Should import a manifest alone as an empty group", func(ctx SpecContext) {
		g := importGroup(ctx, zip.Files{"manifest.json": manifest(2, "Empty")})
		Expect(childrenOf(ctx, g.OntologyID())).To(BeEmpty())
	})

	Describe("Legacy version 1 manifests", func() {
		v1Manifest := func(name string, memberFiles ...string) []byte {
			GinkgoHelper()
			symbols := make([]map[string]any, len(memberFiles))
			for i, file := range memberFiles {
				symbols[i] = map[string]any{
					"file": file, "key": uuid.NewString(), "name": file,
				}
			}
			return MustSucceed(json.Marshal(map[string]any{
				"version": 1, "type": "symbol_group", "name": name,
				"symbols": symbols,
			}))
		}

		It("Should read membership from the manifest's symbols list", func(
			ctx SpecContext,
		) {
			g := importGroup(ctx, zip.Files{
				"manifest.json": v1Manifest("Valves", "Inlet.json"),
				"Inlet.json":    member("Inlet"),
				"Stray.json":    member("Stray"),
			})
			Expect(g.Name).To(Equal("Valves"))
			Expect(childNames(ctx, g)).To(ConsistOf("Inlet"))
		})
		It("Should reject a listed file the bundle does not hold", func(
			ctx SpecContext,
		) {
			files := zip.Files{"manifest.json": v1Manifest("Valves", "Missing.json")}
			Expect(svc.ImportGroup(ctx, tx, files)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("Missing.json")),
			))
		})
		It("Should reject a manifest that lists itself", func(ctx SpecContext) {
			files := zip.Files{"manifest.json": v1Manifest("Valves", "manifest.json")}
			Expect(svc.ImportGroup(ctx, tx, files)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("lists itself")),
			))
		})
	})

	Describe("Golden bundles", func() {
		loadBundle := func(dir string) zip.Files {
			GinkgoHelper()
			entries := MustSucceed(os.ReadDir(dir))
			files := make(zip.Files, len(entries))
			for _, entry := range entries {
				files[entry.Name()] = MustSucceed(
					os.ReadFile(dir + "/" + entry.Name()),
				)
			}
			return files
		}
		It("Should import the frozen legacy Console bundle", func(ctx SpecContext) {
			g := importGroup(ctx, loadBundle("versions/testdata/import_group_v1"))
			Expect(g.Name).To(Equal("Legacy Group"))
			Expect(childNames(ctx, g)).To(ConsistOf("Inlet", "Outlet"))
		})
		It("Should import a group bundle a shipped Console wrote", func(
			ctx SpecContext,
		) {
			// The manifest declares membership through a symbols list naming each
			// file, and each file's name carries the symbol's key as a suffix.
			g := importGroup(
				ctx, loadBundle("versions/testdata/import_group_console_v1"),
			)
			Expect(g.Name).To(Equal("custom symbols"))
			Expect(childNames(ctx, g)).To(ConsistOf("1801287"))
		})
		It("Should carry a Console symbol's whole spec", func(ctx SpecContext) {
			g := importGroup(
				ctx, loadBundle("versions/testdata/import_group_console_v1"),
			)
			var symbols []symbol.Symbol
			Expect(svc.NewRetrieve().Entries(&symbols).Exec(ctx, tx)).To(Succeed())
			var imported symbol.Symbol
			for _, sym := range symbols {
				if sym.Name == "1801287" {
					imported = sym
				}
			}
			Expect(imported.Data.Variant).To(Equal("static"))
			Expect(imported.Data.SVG).To(HavePrefix("<svg"))
			Expect(imported.Data.States).To(HaveLen(1))
			Expect(imported.Data.PreviewViewport).ToNot(BeNil())
			Expect(g.Name).To(Equal("custom symbols"))
		})
		It("Should import the frozen server bundle", func(ctx SpecContext) {
			g := importGroup(ctx, loadBundle("versions/testdata/import_group_v2"))
			Expect(g.Name).To(Equal("Server Group"))
			Expect(childNames(ctx, g)).To(ConsistOf("Inlet", "Outlet"))
		})
	})

	DescribeTable("Should reject an invalid bundle",
		func(ctx SpecContext, files zip.Files, reason string) {
			Expect(svc.ImportGroup(ctx, tx, files)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(reason)),
			))
		},
		Entry("no manifest",
			zip.Files{"Inlet.json": []byte("{}")}, "bundle holds no manifest.json"),
		Entry("another bundle kind",
			zip.Files{"manifest.json": []byte(
				`{"version":1,"type":"project","name":"Valves"}`,
			)}, `bundle is a "project"`),
		Entry("a version older than supported",
			zip.Files{"manifest.json": []byte(
				`{"version":0,"type":"symbol_group","name":"Valves"}`,
			)}, "unsupported manifest version 0"),
		Entry("a nameless manifest",
			zip.Files{"manifest.json": []byte(
				`{"version":2,"type":"symbol_group","name":""}`,
			)}, "manifest.json names no group"),
	)
	It("Should reject a manifest version newer than supported", func(ctx SpecContext) {
		files := zip.Files{"manifest.json": []byte(
			`{"version":3,"type":"symbol_group","name":"Valves"}`,
		)}
		Expect(svc.ImportGroup(ctx, tx, files)).Error().To(SatisfyAll(
			MatchError(ContainSubstring("symbol_group version 3")),
			MatchError(ContainSubstring("newer than this Core supports")),
		))
	})
	It("Should reject two members whose names compare equal case-folded", func(
		ctx SpecContext,
	) {
		files := zip.Files{
			"manifest.json": manifest(2, "Valves"),
			"Inlet.json":    member("Inlet"),
			"inlet.json":    member("inlet"),
		}
		Expect(svc.ImportGroup(ctx, tx, files)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("have the same file name")),
		))
	})
	It("Should reject a member that is not a schematic symbol", func(ctx SpecContext) {
		files := zip.Files{
			"manifest.json": manifest(2, "Valves"),
			"Plot.json": []byte(
				`{"version":1,"type":"lineplot","name":"Plot"}`,
			),
		}
		Expect(svc.ImportGroup(ctx, tx, files)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("not a schematic symbol")),
		))
	})
	It("Should reject a member that does not decode to an envelope", func(
		ctx SpecContext,
	) {
		files := zip.Files{
			"manifest.json": manifest(2, "Valves"),
			"Inlet.json":    []byte("not json"),
		}
		Expect(svc.ImportGroup(ctx, tx, files)).Error().
			To(MatchError(ContainSubstring("Inlet.json")))
	})
})
