// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

// writeRepo materializes a schema tree in a temp directory and returns its
// root.
func writeRepo(files map[string]string) string {
	root := GinkgoT().TempDir()
	for p, content := range files {
		full := filepath.Join(root, p)
		Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
		Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
	}
	return root
}

func resolverFor(files map[string]string) *versions.Resolver {
	root := writeRepo(files)
	chains := MustSucceed(versions.Discover(root))
	return versions.NewResolver(chains, analyzer.NewStandardFileLoader(root))
}

var _ = Describe("Chain", func() {
	Describe("LiveFromFilePath", func() {
		It("Should convert a version file path to its live path", func() {
			Expect(versions.LiveFromFilePath("schemas/x/versions/telem/v0")).
				To(Equal("schemas/x/telem"))
			Expect(versions.LiveFromFilePath("schemas/x/versions/telem/v0.oracle")).
				To(Equal("schemas/x/telem"))
		})

		It("Should reject non-version paths", func() {
			Expect(versions.LiveFromFilePath("schemas/x/telem")).Error().
				To(MatchError(ContainSubstring("not a schema version file")))
			Expect(versions.LiveFromFilePath("schemas/x/versions/telem/extra/v0")).
				Error().To(MatchError(ContainSubstring("not a schema version file")))
			Expect(versions.LiveFromFilePath("schemas/x/versions/telem/foo")).
				Error().To(MatchError(ContainSubstring("not a schema version file")))
		})
	})

	Describe("DepNS", func() {
		It("Should build the internal pinned-surface namespace", func() {
			Expect(versions.DepNS("schemas/x/telem", 3)).To(Equal("x/telem@v3"))
		})
	})

	DescribeTable(
		"ParseDepNS",
		func(ns, wantLive string, wantN int, wantOk bool) {
			live, n, ok := versions.ParseDepNS(ns)
			Expect(ok).To(Equal(wantOk))
			Expect(live).To(Equal(wantLive))
			Expect(n).To(Equal(wantN))
		},
		Entry(
			"round-trips a DepNS namespace",
			"x/telem@v3",
			"schemas/x/telem",
			3,
			true,
		),
		Entry("rejects a namespace without a version tag", "x/telem", "", 0, false),
		Entry("rejects a non-numeric version", "x/telem@vabc", "", 0, false),
	)
})

var _ = Describe("Discover", func() {
	It("Should find chains across domains", func() {
		root := writeRepo(map[string]string{
			"schemas/synnax/versions/channel/v0.oracle": "Key = uuid\n",
			"schemas/synnax/versions/channel/v1.oracle": "Key = v0.Key\n",
			"schemas/x/versions/telem/v0.oracle":        "TimeStamp = int64\n",
		})
		chains := MustSucceed(versions.Discover(root))
		Expect(chains).To(HaveLen(2))
		channel := chains["schemas/synnax/channel"]
		Expect(channel.Numbers).To(Equal([]int{0, 1}))
		Expect(channel.Current()).To(Equal(1))
		Expect(channel.Dir()).To(Equal("schemas/synnax/versions/channel"))
		Expect(channel.FilePath(1)).
			To(Equal("schemas/synnax/versions/channel/v1"))
		Expect(chains["schemas/x/telem"].Resource).To(Equal("telem"))
	})

	It("Should return no chains when no versions directories exist", func() {
		root := writeRepo(map[string]string{
			"schemas/synnax/channel.oracle": "Key = uuid\n",
		})
		Expect(MustSucceed(versions.Discover(root))).To(BeEmpty())
	})

	It("Should accept a chain that skips a version", func() {
		root := writeRepo(map[string]string{
			"schemas/synnax/versions/channel/v0.oracle": "Key = uuid\n",
			"schemas/synnax/versions/channel/v2.oracle": "Key = v0.Key\n",
		})
		chain := MustSucceed(versions.Discover(root))["schemas/synnax/channel"]
		Expect(chain.Numbers).To(Equal([]int{0, 2}))
		Expect(chain.Current()).To(Equal(2))
		pred, ok := chain.Predecessor(2)
		Expect(ok).To(BeTrue())
		Expect(pred).To(Equal(0))
	})

	It("Should reject stray files in a chain directory", func() {
		root := writeRepo(map[string]string{
			"schemas/synnax/versions/channel/v0.oracle":    "Key = uuid\n",
			"schemas/synnax/versions/channel/notes.oracle": "Key = uuid\n",
		})
		Expect(versions.Discover(root)).Error().
			To(MatchError(ContainSubstring("only vN.oracle files")))
	})

	It("Should reject a subdirectory inside a chain directory", func() {
		root := writeRepo(map[string]string{
			"schemas/synnax/versions/channel/v0.oracle":       "Key = uuid\n",
			"schemas/synnax/versions/channel/notes/v0.oracle": "Key = uuid\n",
		})
		Expect(versions.Discover(root)).Error().
			To(MatchError(ContainSubstring("only vN.oracle files")))
	})

	It("Should reject a stray file in a domain's versions directory", func() {
		root := writeRepo(map[string]string{
			"schemas/synnax/versions/notes.oracle": "Key = uuid\n",
		})
		Expect(versions.Discover(root)).Error().To(MatchError(
			ContainSubstring("only per-resource chain directories"),
		))
	})

	It("Should reject an empty chain directory", func() {
		root := writeRepo(map[string]string{
			"schemas/synnax/versions/channel/v0.oracle": "Key = uuid\n",
		})
		Expect(os.MkdirAll(
			filepath.Join(root, "schemas/synnax/versions/label"), 0o755,
		)).To(Succeed())
		Expect(versions.Discover(root)).Error().
			To(MatchError(ContainSubstring("declares no versions")))
	})
})

var _ = Describe("Resolver", func() {
	Describe("File", func() {
		It("Should partition declarations into defined and aliased", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": `
Key = uuid

Channel struct {
	key Key @key
	name string
}
`,
				"schemas/synnax/versions/channel/v1.oracle": `
Key = v0.Key

Channel struct {
	key Key @key
	name string
	virtual bool
}
`,
			})
			f := MustSucceed(r.File(ctx, "schemas/synnax/channel", 1))
			Expect(f.Aliases).To(HaveKeyWithValue(
				"Key", versions.Alias{Version: 0, Name: "Key"},
			))
			Expect(f.Defined).To(HaveLen(1))
			Expect(f.Defined[0].QualifiedName).To(Equal("v1.Channel"))
		})

		It("Should record dependency pins from imports", func(ctx SpecContext) {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
				"schemas/synnax/versions/channel/v0.oracle": `
import "schemas/x/versions/telem/v0"

Channel struct {
	key uuid @key
	created telem.TimeStamp
}
`,
			})
			f := MustSucceed(r.File(ctx, "schemas/synnax/channel", 0))
			Expect(f.Pins).To(HaveKeyWithValue("schemas/x/telem", 0))
			ch := MustBeOk(f.Table.Get("v0.Channel"))
			Expect(ch.Namespace).To(Equal("v0"))
		})

		It("Should resolve the live schema of a versioned resource", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/x/telem.oracle":             "TimeStamp = int64\n",
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
				"schemas/synnax/versions/channel/v0.oracle": `
import "schemas/x/telem"

Channel struct {
	key uuid @key
	created telem.TimeStamp {
		@go marshal omit
	}
}
`,
			})
			f := MustSucceed(r.File(ctx, "schemas/synnax/channel", 0))
			Expect(f.Live).To(ContainElement("telem"))
			Expect(f.LivePaths).To(ContainElement("schemas/x/telem"))
			Expect(f.Pins).To(BeEmpty())
		})

		It("Should resolve an unversioned schema through a transitive pin", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/x/telem.oracle":             "TimeStamp = int64\n",
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
				"schemas/arc/program.oracle": `
import "schemas/x/telem"

Program struct {
	created telem.TimeStamp
}
`,
				"schemas/synnax/versions/label/v0.oracle": `
import "schemas/x/versions/telem/v0"

Label struct {
	key uuid @key
	created telem.TimeStamp
}
`,
				"schemas/synnax/versions/arc/v0.oracle": `
import "schemas/arc/program"
import "schemas/synnax/versions/label/v0"

Arc struct {
	key uuid @key
	label label.Label
	compiled program.Program
}
`,
			})
			f := MustSucceed(r.File(ctx, "schemas/synnax/arc", 0))
			arc := MustBeOk(f.Table.Get("v0.Arc"))
			field := arc.Form.(resolution.StructForm).Fields[2]
			Expect(MustBeOk(field.Type.Resolve(f.Table)).QualifiedName).
				To(Equal("program.Program"))
		})

		It("Should resolve the live schema of an unversioned resource", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/arc/program.oracle": "Program struct {\n\twasm bytes\n}\n",
				"schemas/synnax/versions/arc/v0.oracle": `
import "schemas/arc/program"

Arc struct {
	key uuid @key
	compiled program.Program
}
`,
			})
			f := MustSucceed(r.File(ctx, "schemas/synnax/arc", 0))
			Expect(f.Pins).ToNot(HaveKey("schemas/arc/program"))
			arc := MustBeOk(f.Table.Get("v0.Arc"))
			Expect(arc.Namespace).To(Equal("v0"))
			field := arc.Form.(resolution.StructForm).Fields[1]
			Expect(MustBeOk(field.Type.Resolve(f.Table)).QualifiedName).
				To(Equal("program.Program"))
		})

		It("Should resolve transitive pins under internal namespaces", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
				"schemas/x/versions/telem/v1.oracle": "TimeStamp = int32\n",
				"schemas/x/versions/node/v0.oracle": `
import "schemas/x/versions/telem/v0"

Node struct {
	key uuid @key
	ts telem.TimeStamp
}
`,
				"schemas/synnax/versions/channel/v0.oracle": `
import "schemas/x/versions/telem/v1"
import "schemas/x/versions/node/v0"

Channel struct {
	key uuid @key
	ts telem.TimeStamp
	node node.Node
}
`,
			})
			f := MustSucceed(r.File(ctx, "schemas/synnax/channel", 0))
			node := MustBeOk(f.Table.Get("x/node@v0.Node"))
			Expect(node.Namespace).To(Equal("x/node@v0"))
			Expect(MustBeOk(f.Table.Get("x/telem@v0.TimeStamp")).Namespace).
				To(Equal("x/telem@v0"))
		})

		It("Should error on a pin cycle between resources", func(ctx SpecContext) {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/a/v0.oracle": `
import "schemas/synnax/versions/b/v0"

A struct {
	key uuid @key
	b b.B
}
`,
				"schemas/synnax/versions/b/v0.oracle": `
import "schemas/synnax/versions/a/v0"

B struct {
	key uuid @key
	a a.A
}
`,
			})
			Expect(r.File(ctx, "schemas/synnax/a", 0)).Error().
				To(MatchError(ContainSubstring("version dependency cycle")))
		})

		It("Should error on a live path with no chain", func(ctx SpecContext) {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
			})
			Expect(r.File(ctx, "schemas/x/node", 0)).Error().
				To(MatchError(ContainSubstring("no version chain for schemas/x/node")))
		})

		It("Should error on a version outside the chain's range", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
			})
			Expect(r.File(ctx, "schemas/x/telem", 3)).Error().
				To(MatchError(ContainSubstring("schemas/x/telem has no version 3")))
		})

		It("Should error on an unparsable version file", func(ctx SpecContext) {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp struct {{{\n",
			})
			Expect(r.File(ctx, "schemas/x/telem", 0)).Error().
				To(MatchError(ContainSubstring("failed to parse")))
		})

		It("Should error when analysis of a version file fails", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "Key = uuid\nKey = string\n",
			})
			Expect(r.File(ctx, "schemas/x/telem", 0)).Error().
				To(MatchError(ContainSubstring("analysis of")))
		})

		It("Should error when a version file is missing on disk", func(
			ctx SpecContext,
		) {
			root := writeRepo(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
			})
			chains := MustSucceed(versions.Discover(root))
			r := versions.NewResolver(chains, analyzer.NewStandardFileLoader(root))
			Expect(os.Remove(filepath.Join(
				root, "schemas/x/versions/telem/v0.oracle",
			))).To(Succeed())
			Expect(r.File(ctx, "schemas/x/telem", 0)).Error().
				To(MatchError(ContainSubstring("failed to load")))
		})

		It("Should copy a pinned definer's live shapes into the table", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/x/telem.oracle": "TimeStamp = int64\n",
				"schemas/synnax/versions/label/v0.oracle": `
import "schemas/x/telem"

Label struct {
	key uuid @key
	seen telem.TimeStamp {
		@go marshal omit
	}

	@go marshal
}
`,
				"schemas/synnax/versions/channel/v0.oracle": `
import "schemas/synnax/versions/label/v0"

Channel struct {
	key uuid @key
	label label.Label

	@go marshal
}
`,
			})
			f := MustSucceed(r.File(ctx, "schemas/synnax/channel", 0))
			Expect(MustBeOk(f.Table.Get("telem.TimeStamp")).Namespace).
				To(Equal("telem"))
		})
	})

	Describe("Pinned surface copies", func() {
		It("Should copy every declaration form into the pin namespace", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/x/versions/shape/v0.oracle": `
Kind enum {
	linear = "linear"
}

Leaf struct {
	name string
}

Node union on variant {
	leaf Leaf
	inline {
		depth int64
	}
}

Span int64

Ref = Leaf

Wrapper struct<D extends record = record> {
	details D
}
`,
				"schemas/synnax/versions/channel/v0.oracle": `
import "schemas/x/versions/shape/v0"

Channel struct {
	key uuid @key
	kind shape.Kind
	node shape.Node
	span shape.Span
	ref shape.Ref
	wrapped shape.Wrapper

	@go marshal
}
`,
			})
			f := MustSucceed(r.File(ctx, "schemas/synnax/channel", 0))
			for _, name := range []string{
				"Kind", "Leaf", "Node", "Span", "Ref", "Wrapper",
			} {
				t := MustBeOk(f.Table.Get("x/shape@v0." + name))
				Expect(t.Namespace).To(Equal("x/shape@v0"), name)
			}
			node := MustBeOk(f.Table.Get("shape.Node"))
			variants := node.Form.(resolution.UnionForm).Variants
			Expect(variants).To(HaveLen(2))
			Expect(variants[0].Type.Name).To(Equal("shape.Leaf"))
		})
	})

	Describe("Chains", func() {
		It("Should return the discovered chains", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
			})
			Expect(r.Chains()).To(HaveKey("schemas/x/telem"))
		})
	})

	Describe("Definer", func() {
		definerFixture := func() *versions.Resolver {
			return resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": "Key = uuid\n",
				"schemas/synnax/versions/channel/v1.oracle": `
Key = v0.Key

Name = string
`,
			})
		}

		It("Should locate a name defined at the requested version", func(
			ctx SpecContext,
		) {
			r := definerFixture()
			ns, name, ok := r.Definer(ctx, "schemas/synnax/channel", 1, "Name")
			Expect(ok).To(BeTrue())
			Expect(ns).To(Equal("synnax/channel@v1"))
			Expect(name).To(Equal("Name"))
		})

		It("Should follow alias lines to the defining version", func(
			ctx SpecContext,
		) {
			r := definerFixture()
			ns, name, ok := r.Definer(ctx, "schemas/synnax/channel", 1, "Key")
			Expect(ok).To(BeTrue())
			Expect(ns).To(Equal("synnax/channel@v0"))
			Expect(name).To(Equal("Key"))
		})

		It("Should report false for a name the version does not carry", func(
			ctx SpecContext,
		) {
			r := definerFixture()
			_, _, ok := r.Definer(ctx, "schemas/synnax/channel", 0, "Name")
			Expect(ok).To(BeFalse())
		})

		It("Should report false for a live path with no chain", func(
			ctx SpecContext,
		) {
			r := definerFixture()
			_, _, ok := r.Definer(ctx, "schemas/x/telem", 0, "TimeStamp")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Ended", func() {
		It("Should report a chain ended by an empty tombstone file", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
				"schemas/x/versions/telem/v1.oracle": "// gone\n",
			})
			Expect(r.Ended(ctx, "schemas/x/telem")).To(BeTrue())
		})

		It("Should report a chain with a declaring last version as live", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
			})
			Expect(r.Ended(ctx, "schemas/x/telem")).To(BeFalse())
		})

		It("Should reject a tombstone before the chain's last version", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
				"schemas/x/versions/telem/v1.oracle": "// gone\n",
				"schemas/x/versions/telem/v2.oracle": "TimeStamp = int64\n",
			})
			Expect(r.File(ctx, "schemas/x/telem", 1)).Error().To(MatchError(
				ContainSubstring("only the last version file may be a tombstone"),
			))
		})

		It("Should reject a chain whose only version is a tombstone", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "// gone\n",
			})
			Expect(r.Ended(ctx, "schemas/x/telem")).Error().To(MatchError(
				ContainSubstring("delete the chain directory"),
			))
		})

		It("Should error on a live path with no chain", func(ctx SpecContext) {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
			})
			Expect(r.Ended(ctx, "schemas/x/node")).Error().
				To(MatchError(ContainSubstring("no version chain for schemas/x/node")))
		})
	})

	Describe("Surface", func() {
		It("Should map every name to its defining version", func(ctx SpecContext) {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": `
Key = uuid

Channel struct {
	key Key @key
	name string
}
`,
				"schemas/synnax/versions/channel/v1.oracle": `
Key = v0.Key

Channel struct {
	key Key @key
	name string
	virtual bool
}
`,
			})
			surf := MustSucceed(r.Surface(ctx, "schemas/synnax/channel", 1))
			Expect(surf).To(HaveLen(2))
			Expect(surf["Key"].Version).To(Equal(0))
			Expect(surf["Channel"].Version).To(Equal(1))
			Expect(surf["Channel"].Type.QualifiedName).To(Equal("v1.Channel"))
		})

		It("Should drop names absent from a version", func(ctx SpecContext) {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": `
Key = uuid

Legacy struct {
	key Key @key
}
`,
				"schemas/synnax/versions/channel/v1.oracle": `
Key = v0.Key
`,
			})
			surf := MustSucceed(r.Surface(ctx, "schemas/synnax/channel", 1))
			Expect(surf).To(HaveKey("Key"))
			Expect(surf).ToNot(HaveKey("Legacy"))
		})

		It("Should follow an alias to a non-defining version transitively", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": "Key = uuid\n",
				"schemas/synnax/versions/channel/v1.oracle": "Key = v0.Key\n",
				"schemas/synnax/versions/channel/v2.oracle": "Key = v1.Key\n",
			})
			surf := MustSucceed(r.Surface(ctx, "schemas/synnax/channel", 2))
			Expect(surf).To(HaveKey("Key"))
			Expect(surf["Key"].Version).To(Equal(0))
		})

		It("Should reject an alias to a version that never carried the name", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": "Key = uuid\n",
				"schemas/synnax/versions/channel/v1.oracle": "Key = v0.Key\nName = v0.Name\n",
			})
			Expect(r.Surface(ctx, "schemas/synnax/channel", 1)).Error().
				To(MatchError(ContainSubstring("does not carry it")))
		})

		It("Should inherit docs from the predecessor and honor overrides", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": `
Key = uuid {
	@doc value "unique channel identifier"
}
`,
				"schemas/synnax/versions/channel/v1.oracle": `
Key = string
`,
				"schemas/synnax/versions/channel/v2.oracle": `
Key = int64 {
	@doc value "now numeric"
}
`,
			})
			live := "schemas/synnax/channel"
			Expect(MustSucceed(r.Surface(ctx, live, 0))["Key"].Doc).
				To(Equal("unique channel identifier"))
			Expect(MustSucceed(r.Surface(ctx, live, 1))["Key"].Doc).
				To(Equal("unique channel identifier"))
			Expect(MustSucceed(r.Surface(ctx, live, 2))["Key"].Doc).
				To(Equal("now numeric"))
		})
	})

	Describe("SurfaceInto", func() {
		It("Should stamp the chain-resolved doc onto a docless redeclaration", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": `
Key = uuid {
	@doc value "unique channel identifier"
}
`,
				"schemas/synnax/versions/channel/v1.oracle": `
Key = string
`,
			})
			table := resolution.NewTable()
			Expect(r.SurfaceInto(
				ctx, table, "schemas/synnax/channel", 1, "channel@v1",
			)).To(Succeed())
			t := MustBeOk(table.Get("channel@v1.Key"))
			Expect(doc.Get(t.Domains)).To(Equal("unique channel identifier"))
		})

		It("Should not re-register a namespace already in the table", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": "Key = uuid\n",
			})
			table := resolution.NewTable()
			Expect(table.Add(resolution.Type{
				Name:          "Existing",
				Namespace:     "channel@v0",
				QualifiedName: "channel@v0.Existing",
			})).To(Succeed())
			Expect(r.SurfaceInto(
				ctx, table, "schemas/synnax/channel", 0, "channel@v0",
			)).To(Succeed())
			_, ok := table.Get("channel@v0.Key")
			Expect(ok).To(BeFalse())
		})

		It("Should propagate resolution failures", func(ctx SpecContext) {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": "Key struct {{{\n",
			})
			Expect(r.SurfaceInto(
				ctx, resolution.NewTable(), "schemas/synnax/channel", 0, "channel@v0",
			)).Error().To(MatchError(ContainSubstring("failed to parse")))
		})
	})
})
