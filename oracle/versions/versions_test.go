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
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

// writeRepo materializes a schema tree in a temp directory and returns its
// root.
func writeRepo(files map[string]string) string {
	GinkgoHelper()
	root := GinkgoT().TempDir()
	for p, content := range files {
		full := filepath.Join(root, p)
		Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
		Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
	}
	return root
}

func resolverFor(files map[string]string) *versions.Resolver {
	GinkgoHelper()
	root := writeRepo(files)
	chains := MustSucceed(versions.Discover(root))
	return versions.NewResolver(chains, analyzer.NewStandardFileLoader(root))
}

var _ = Describe("Chain", func() {
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

	It("Should skip a chain directory marked removed", func() {
		root := writeRepo(map[string]string{
			"schemas/synnax/versions/channel/v0.oracle": "Key = uuid\n",
			"schemas/synnax/versions/alias/removed":     "",
		})
		Expect(MustSucceed(versions.Discover(root))).
			To(SatisfyAll(HaveLen(1), HaveKey("schemas/synnax/channel")))
	})
})

var _ = Describe("Removed", func() {
	It("Should return the chains marked removed", func() {
		root := writeRepo(map[string]string{
			"schemas/synnax/versions/channel/v0.oracle": "Key = uuid\n",
			"schemas/synnax/versions/alias/removed":     "",
		})
		Expect(MustSucceed(versions.Removed(root))).
			To(Equal(set.New("schemas/synnax/alias")))
	})

	It("Should return no chains when no schemas directory exists", func() {
		Expect(MustSucceed(versions.Removed(GinkgoT().TempDir()))).To(BeEmpty())
	})

	It("Should reject a removed marker next to version files", func() {
		root := writeRepo(map[string]string{
			"schemas/synnax/versions/alias/v0.oracle": "Key = uuid\n",
			"schemas/synnax/versions/alias/removed":   "",
		})
		Expect(versions.Removed(root)).Error().To(MatchError(
			filepath.Join(root, "schemas/synnax/versions/alias") +
				" must contain only its removed marker",
		))
	})

	It("Should reject a removed chain whose live file exists", func() {
		root := writeRepo(map[string]string{
			"schemas/synnax/alias.oracle":           "Key = uuid\n",
			"schemas/synnax/versions/alias/removed": "",
		})
		Expect(versions.Removed(root)).Error().To(MatchError(
			"schemas/synnax/alias is marked removed, but schemas/synnax/alias.oracle " +
				"exists",
		))
	})
})

var _ = Describe("Resolver", func() {
	Describe("File", func() {
		It("Should partition declarations into defined and aliased", func() {
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
			f := MustSucceed(r.File("schemas/synnax/channel", 1))
			Expect(f.Aliases).To(HaveKeyWithValue(
				"Key", versions.Alias{Version: 0, Name: "Key"},
			))
			Expect(f.Defined).To(HaveLen(1))
			Expect(f.Defined[0].QualifiedName).To(Equal("v1.Channel"))
		})

		DescribeTable("Should record dependency pins from imports", func(
			importPath string,
		) {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
				"schemas/synnax/versions/channel/v0.oracle": `
import "` + importPath + `"

Channel struct {
	key uuid @key
	created telem.TimeStamp
}
`,
			})
			f := MustSucceed(r.File("schemas/synnax/channel", 0))
			Expect(f.Pins).To(HaveKeyWithValue("schemas/x/telem", 0))
			ch := MustBeOk(f.Table.Get("v0.Channel"))
			Expect(ch.Namespace).To(Equal("v0"))
		},
			Entry("without an extension", "schemas/x/versions/telem/v0"),
			Entry("with an extension", "schemas/x/versions/telem/v0.oracle"),
		)

		It("Should resolve the live schema of a versioned resource", func() {
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
			f := MustSucceed(r.File("schemas/synnax/channel", 0))
			Expect(f.Live).To(ContainElement("telem"))
			Expect(f.LivePaths).To(ContainElement("schemas/x/telem"))
			Expect(f.Pins).To(BeEmpty())
		})

		It("Should resolve an unversioned schema through a transitive pin", func() {
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
			f := MustSucceed(r.File("schemas/synnax/arc", 0))
			arc := MustBeOk(f.Table.Get("v0.Arc"))
			form, ok := arc.Form.(resolution.StructForm)
			Expect(ok).To(BeTrue())
			field := form.Fields[2]
			Expect(MustBeOk(field.Type.Resolve(f.Table)).QualifiedName).
				To(Equal("program.Program"))
		})

		It("Should resolve the live schema of an unversioned resource", func() {
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
			f := MustSucceed(r.File("schemas/synnax/arc", 0))
			Expect(f.Pins).ToNot(HaveKey("schemas/arc/program"))
			arc := MustBeOk(f.Table.Get("v0.Arc"))
			Expect(arc.Namespace).To(Equal("v0"))
			form, ok := arc.Form.(resolution.StructForm)
			Expect(ok).To(BeTrue())
			field := form.Fields[1]
			Expect(MustBeOk(field.Type.Resolve(f.Table)).QualifiedName).
				To(Equal("program.Program"))
		})

		It("Should resolve transitive pins under internal namespaces", func() {
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
			f := MustSucceed(r.File("schemas/synnax/channel", 0))
			node := MustBeOk(f.Table.Get("x/node@v0.Node"))
			Expect(node.Namespace).To(Equal("x/node@v0"))
			Expect(MustBeOk(f.Table.Get("x/telem@v0.TimeStamp")).Namespace).
				To(Equal("x/telem@v0"))
		})

		It("Should resolve each name of two pins that share a namespace", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/status/v0.oracle":      "Code = int64\n",
				"schemas/synnax/versions/status/v0.oracle": "Level = int32\n",
				"schemas/synnax/versions/channel/v0.oracle": `
import "schemas/synnax/versions/status/v0"
import "schemas/x/versions/status/v0"

Channel struct {
	key uuid @key
	code status.Code
	level status.Level
}
`,
				"schemas/synnax/versions/channel/v1.oracle": `
import "schemas/synnax/versions/status/v0"
import "schemas/x/versions/status/v0"

Channel struct extends v0.Channel {
	name string
}
`,
			})
			f := MustSucceed(r.File("schemas/synnax/channel", 1))
			form, ok := MustBeOk(f.Table.Get("v0.Channel")).Form.(resolution.StructForm)
			Expect(ok).To(BeTrue())
			Expect(form.Fields[1].Type.Name).To(Equal("x/status@v0.Code"))
			Expect(form.Fields[2].Type.Name).To(Equal("synnax/status@v0.Level"))
		})

		It("Should resolve two pins of one namespace through a transitive pin", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/status/v0.oracle":      "Code = int64\n",
				"schemas/synnax/versions/status/v0.oracle": "Level = int32\n",
				"schemas/synnax/versions/channel/v0.oracle": `
import "schemas/synnax/versions/status/v0"
import "schemas/x/versions/status/v0"

Payload union on kind {
	coded {
		code status.Code
	}
	leveled {
		level status.Level
	}
}

Channel struct {
	key uuid @key
	code status.Code
	payload Payload
}
`,
				"schemas/synnax/versions/ranger/v0.oracle": `
import "schemas/synnax/versions/channel/v0"

Range struct {
	key uuid @key
	channel channel.Channel
}
`,
			})
			f := MustSucceed(r.File("schemas/synnax/ranger", 0))
			ch, ok := MustBeOk(f.Table.Get("synnax/channel@v0.Channel")).
				Form.(resolution.StructForm)
			Expect(ok).To(BeTrue())
			Expect(ch.Fields[1].Type.Name).To(Equal("x/status@v0.Code"))
			payload, ok := MustBeOk(f.Table.Get("synnax/channel@v0.Payload")).
				Form.(resolution.UnionForm)
			Expect(ok).To(BeTrue())
			fields := make([]string, 0, len(payload.Variants))
			for _, v := range payload.Variants {
				p, ok := MustBeOk(f.Table.Get(v.Type.Name)).Form.(resolution.StructForm)
				Expect(ok).To(BeTrue())
				fields = append(fields, p.Fields[0].Type.Name)
			}
			Expect(fields).To(ConsistOf("x/status@v0.Code", "synnax/status@v0.Level"))
		})

		It("Should keep a live name that shares its namespace with a pin", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/status/v0.oracle": "Code = int64\n",
				"schemas/synnax/status.oracle":        "Level = int32\n",
				"schemas/synnax/versions/channel/v0.oracle": `
import "schemas/synnax/status"
import "schemas/x/versions/status/v0"

Channel struct {
	key uuid @key
	code status.Code
	level status.Level
}
`,
				"schemas/synnax/versions/channel/v1.oracle": `
import "schemas/synnax/status"
import "schemas/x/versions/status/v0"

Channel struct extends v0.Channel {
	name string
}
`,
			})
			f := MustSucceed(r.File("schemas/synnax/channel", 1))
			form, ok := MustBeOk(f.Table.Get("v0.Channel")).Form.(resolution.StructForm)
			Expect(ok).To(BeTrue())
			Expect(form.Fields[1].Type.Name).To(Equal("x/status@v0.Code"))
			Expect(form.Fields[2].Type.Name).To(Equal("status.Level"))
		})

		It("Should keep a pin's namespace free of a dependency's live names", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/status/v0.oracle": "Code = int64\n",
				"schemas/synnax/status.oracle":        "Level = int32\n",
				"schemas/synnax/versions/rack/v0.oracle": `
import "schemas/synnax/status"

Rack struct {
	key uuid @key
	level status.Level
}
`,
				"schemas/synnax/versions/channel/v0.oracle": `
import "schemas/x/versions/status/v0"
import "schemas/synnax/versions/rack/v0"

Channel struct {
	key uuid @key
	rack rack.Rack
	level status.Level
}
`,
			})
			Expect(r.File("schemas/synnax/channel", 0)).Error().
				To(MatchError(ContainSubstring("unresolved type: status.Level")))
		})

		It("Should reject two pins of one namespace that declare one name", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/status/v0.oracle":      "Code = int64\n",
				"schemas/synnax/versions/status/v0.oracle": "Code = int32\n",
				"schemas/synnax/versions/channel/v0.oracle": `
import "schemas/synnax/versions/status/v0"
import "schemas/x/versions/status/v0"

Channel struct {
	key uuid @key
	code status.Code
}
`,
			})
			Expect(r.File("schemas/synnax/channel", 0)).Error().
				To(MatchError(ContainSubstring("duplicate type: status.Code")))
		})

		It("Should reject a pinned name that the version file does not import", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/notation/v0.oracle": `
Notation enum {
	standard = "standard"
}
`,
				"schemas/synnax/versions/schematic/v0.oracle": `
import "schemas/x/versions/notation/v0"

Schematic struct {
	key uuid @key
	notation notation.Notation
}
`,
				"schemas/synnax/versions/schematic/v1.oracle": `
Schematic struct {
	key uuid @key
	notation Notation
}
`,
			})
			Expect(
				r.File("schemas/synnax/schematic", 1),
			).Error().
				To(MatchError(And(
					ContainSubstring("schemas/synnax/versions/schematic/v1.oracle"),
					ContainSubstring("unresolved type: Notation"),
				)))
		})

		It("Should error on a pin cycle between resources", func() {
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
			Expect(r.File("schemas/synnax/a", 0)).Error().
				To(MatchError(ContainSubstring("version dependency cycle")))
		})

		It("Should error on a live path with no chain", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
			})
			Expect(r.File("schemas/x/node", 0)).Error().
				To(MatchError(ContainSubstring("no version chain for schemas/x/node")))
		})

		It("Should error on a version outside the chain's range", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
			})
			Expect(r.File("schemas/x/telem", 3)).Error().
				To(MatchError(ContainSubstring("schemas/x/telem has no version 3")))
		})

		It("Should error on an unparsable version file", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp struct {{{\n",
			})
			Expect(r.File("schemas/x/telem", 0)).Error().
				To(MatchError(ContainSubstring("failed to parse")))
		})

		It("Should error when analysis of a version file fails", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "Key = uuid\nKey = string\n",
			})
			Expect(r.File("schemas/x/telem", 0)).Error().
				To(MatchError(ContainSubstring("analysis of")))
		})

		It("Should error when a version file is missing on disk", func() {
			root := writeRepo(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
			})
			chains := MustSucceed(versions.Discover(root))
			r := versions.NewResolver(chains, analyzer.NewStandardFileLoader(root))
			Expect(os.Remove(filepath.Join(
				root, "schemas/x/versions/telem/v0.oracle",
			))).To(Succeed())
			Expect(r.File("schemas/x/telem", 0)).Error().
				To(MatchError(ContainSubstring("failed to load")))
		})

		It("Should copy a pinned definer's live shapes into the table", func() {
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
			f := MustSucceed(r.File("schemas/synnax/channel", 0))
			Expect(MustBeOk(f.Table.Get("telem.TimeStamp")).Namespace).
				To(Equal("telem"))
		})
	})

	Describe("Pinned surface copies", func() {
		It("Should copy every declaration form into the pin namespace", func() {
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
			f := MustSucceed(r.File("schemas/synnax/channel", 0))
			for _, name := range []string{
				"Kind", "Leaf", "Node", "Span", "Ref", "Wrapper",
			} {
				t := MustBeOk(f.Table.Get("x/shape@v0." + name))
				Expect(t.Namespace).To(Equal("x/shape@v0"), name)
			}
			node := MustBeOk(f.Table.Get("shape.Node"))
			form, ok := node.Form.(resolution.UnionForm)
			Expect(ok).To(BeTrue())
			variants := form.Variants
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
			GinkgoHelper()
			return resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": "Key = uuid\n",
				"schemas/synnax/versions/channel/v1.oracle": `
Key = v0.Key

Name = string
`,
			})
		}

		It("Should locate a name defined at the requested version", func() {
			r := definerFixture()
			ns, name, ok := r.Definer("schemas/synnax/channel", 1, "Name")
			Expect(ok).To(BeTrue())
			Expect(ns).To(Equal("synnax/channel@v1"))
			Expect(name).To(Equal("Name"))
		})

		It("Should follow alias lines to the defining version", func() {
			r := definerFixture()
			ns, name, ok := r.Definer("schemas/synnax/channel", 1, "Key")
			Expect(ok).To(BeTrue())
			Expect(ns).To(Equal("synnax/channel@v0"))
			Expect(name).To(Equal("Key"))
		})

		It("Should report false for a name the version does not carry", func() {
			r := definerFixture()
			_, _, ok := r.Definer("schemas/synnax/channel", 0, "Name")
			Expect(ok).To(BeFalse())
		})

		It("Should report false for a live path with no chain", func() {
			r := definerFixture()
			_, _, ok := r.Definer("schemas/x/telem", 0, "TimeStamp")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Ended", func() {
		It("Should report a chain ended by an empty tombstone file", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
				"schemas/x/versions/telem/v1.oracle": "// gone\n",
			})
			Expect(r.Ended("schemas/x/telem")).To(BeTrue())
		})

		It("Should report a chain with a declaring last version as live", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
			})
			Expect(r.Ended("schemas/x/telem")).To(BeFalse())
		})

		It("Should reject a tombstone before the chain's last version", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
				"schemas/x/versions/telem/v1.oracle": "// gone\n",
				"schemas/x/versions/telem/v2.oracle": "TimeStamp = int64\n",
			})
			Expect(r.File("schemas/x/telem", 1)).Error().To(MatchError(
				ContainSubstring("only the last version file may be a tombstone"),
			))
		})

		It("Should reject a chain whose only version is a tombstone", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "// gone\n",
			})
			Expect(r.Ended("schemas/x/telem")).Error().To(MatchError(
				ContainSubstring("delete the chain directory"),
			))
		})

		It("Should error on a live path with no chain", func() {
			r := resolverFor(map[string]string{
				"schemas/x/versions/telem/v0.oracle": "TimeStamp = int64\n",
			})
			Expect(r.Ended("schemas/x/node")).Error().
				To(MatchError(ContainSubstring("no version chain for schemas/x/node")))
		})
	})

	Describe("Surface", func() {
		It("Should map every name to its defining version", func() {
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
			surf := MustSucceed(r.Surface("schemas/synnax/channel", 1))
			Expect(surf).To(HaveLen(2))
			Expect(surf["Key"].Version).To(Equal(0))
			Expect(surf["Channel"].Version).To(Equal(1))
			Expect(surf["Channel"].Type.QualifiedName).To(Equal("v1.Channel"))
		})

		It("Should drop names absent from a version", func() {
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
			surf := MustSucceed(r.Surface("schemas/synnax/channel", 1))
			Expect(surf).To(HaveKey("Key"))
			Expect(surf).ToNot(HaveKey("Legacy"))
		})

		It("Should follow an alias to a non-defining version transitively", func() {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": "Key = uuid\n",
				"schemas/synnax/versions/channel/v1.oracle": "Key = v0.Key\n",
				"schemas/synnax/versions/channel/v2.oracle": "Key = v1.Key\n",
			})
			surf := MustSucceed(r.Surface("schemas/synnax/channel", 2))
			Expect(surf).To(HaveKey("Key"))
			Expect(surf["Key"].Version).To(Equal(0))
		})

		It("Should reject an alias to a version that never carried the name", func() {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": "Key = uuid\n",
				"schemas/synnax/versions/channel/v1.oracle": "Key = v0.Key\nName = v0.Name\n",
			})
			Expect(r.Surface("schemas/synnax/channel", 1)).Error().
				To(MatchError(ContainSubstring("unresolved type: v0.Name")))
		})

		It("Should inherit docs from the predecessor and honor overrides", func() {
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
			Expect(MustSucceed(r.Surface(live, 0))["Key"].Doc).
				To(Equal("unique channel identifier"))
			Expect(MustSucceed(r.Surface(live, 1))["Key"].Doc).
				To(Equal("unique channel identifier"))
			Expect(MustSucceed(r.Surface(live, 2))["Key"].Doc).
				To(Equal("now numeric"))
		})
	})

	Describe("SurfaceInto", func() {
		It("Should stamp the chain-resolved doc onto a docless redeclaration", func() {
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
				table, "schemas/synnax/channel", 1, "channel@v1",
			)).To(Succeed())
			t := MustBeOk(table.Get("channel@v1.Key"))
			Expect(doc.Get(t.Domains)).To(Equal("unique channel identifier"))
		})

		It("Should not re-register a namespace already in the table", func() {
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
				table, "schemas/synnax/channel", 0, "channel@v0",
			)).To(Succeed())
			_, ok := table.Get("channel@v0.Key")
			Expect(ok).To(BeFalse())
		})

		It("Should carry the enum a union borrows into the surface", func() {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/auth/v0.oracle": `
Method enum {
	password = "password"
}
Credentials struct { username string }
Proof union on method Method {
	password Credentials
}
`,
			})
			table := resolution.NewTable()
			Expect(r.SurfaceInto(
				table, "schemas/synnax/auth", 0, "auth@v0",
			)).To(Succeed())
			form, ok := MustBeOk(table.Get("auth@v0.Proof")).Form.(resolution.UnionForm)
			Expect(ok).To(BeTrue())
			Expect(form.Enum.Name).To(Equal("auth@v0.Method"))
		})

		It("Should take patched variant payloads from the patching version", func() {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": `
Key = uuid
Node union on variant {
	box {
		width int32 = 4
	}
}
`,
				"schemas/synnax/versions/channel/v1.oracle": `
Key = v0.Key
Node union on variant extends v0.Node {
	box {
		width int32?
	}
}
`,
			})
			table := resolution.NewTable()
			Expect(r.SurfaceInto(
				table, "schemas/synnax/channel", 1, "channel@v1",
			)).To(Succeed())
			box := MustBeOk(table.Get("channel@v1.NodeBoxPayload"))
			form, ok := box.Form.(resolution.StructForm)
			Expect(ok).To(BeTrue())
			width := MustBeOk(lo.Find(
				form.Fields,
				func(f resolution.Field) bool { return f.Name == "width" },
			))
			Expect(width.Optional).To(BeTrue())
		})

		It("Should propagate resolution failures", func() {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": "Key struct {{{\n",
			})
			Expect(r.SurfaceInto(
				resolution.NewTable(), "schemas/synnax/channel", 0, "channel@v0",
			)).Error().To(MatchError(ContainSubstring("failed to parse")))
		})
	})
})
