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
		})
	})
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

	It("Should reject a chain with a version gap", func() {
		root := writeRepo(map[string]string{
			"schemas/synnax/versions/channel/v0.oracle": "Key = uuid\n",
			"schemas/synnax/versions/channel/v2.oracle": "Key = v0.Key\n",
		})
		Expect(versions.Discover(root)).Error().
			To(MatchError(ContainSubstring("missing v1")))
	})

	It("Should reject stray files in a chain directory", func() {
		root := writeRepo(map[string]string{
			"schemas/synnax/versions/channel/v0.oracle":    "Key = uuid\n",
			"schemas/synnax/versions/channel/notes.oracle": "Key = uuid\n",
		})
		Expect(versions.Discover(root)).Error().
			To(MatchError(ContainSubstring("only vN.oracle files")))
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

		It("Should reject imports of live schemas", func(ctx SpecContext) {
			r := resolverFor(map[string]string{
				"schemas/x/telem.oracle": "TimeStamp = int64\n",
				"schemas/synnax/versions/channel/v0.oracle": `
import "schemas/x/telem"

Channel struct {
	key uuid @key
	created telem.TimeStamp
}
`,
			})
			Expect(r.File(ctx, "schemas/synnax/channel", 0)).Error().
				To(MatchError(ContainSubstring(
					"may only import other version files",
				)))
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

		It("Should reject an alias to a non-defining version", func(
			ctx SpecContext,
		) {
			r := resolverFor(map[string]string{
				"schemas/synnax/versions/channel/v0.oracle": "Key = uuid\n",
				"schemas/synnax/versions/channel/v1.oracle": "Key = v0.Key\n",
				"schemas/synnax/versions/channel/v2.oracle": "Key = v1.Key\n",
			})
			Expect(r.Surface(ctx, "schemas/synnax/channel", 2)).Error().
				To(MatchError(ContainSubstring("does not define it")))
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
})
