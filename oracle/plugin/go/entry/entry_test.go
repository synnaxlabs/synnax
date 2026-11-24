// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package entry_test

import (
	"maps"
	"os"
	"path/filepath"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/entry"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

const resourceSchema = `
@go output "core/pkg/service/resource"

Type enum {
	label = "label"
	ni_device = "ni_device"
	icd = "icd"
	status = "status"
}
`

// generate writes the version files and creates the package directories under a
// temporary repository, analyzes the live source and the resource schema into one
// table, and runs the plugin.
func generate(
	resource, live string,
	versionFiles map[string]string,
	packageDirs ...string,
) (*plugin.Response, error) {
	GinkgoHelper()
	return generateWith(map[string]string{resource: live}, versionFiles, packageDirs...)
}

// generateWith is generate for several live sources, keyed by resource.
func generateWith(
	lives map[string]string,
	versionFiles map[string]string,
	packageDirs ...string,
) (*plugin.Response, error) {
	GinkgoHelper()
	root := GinkgoT().TempDir()
	for _, dir := range packageDirs {
		Expect(os.MkdirAll(filepath.Join(root, dir), 0o755)).To(Succeed())
	}
	for rel, content := range versionFiles {
		full := filepath.Join(root, "schemas/synnax/versions", rel)
		Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
		Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
	}
	chains := MustSucceed(versions.Discover(root))
	resolver := versions.NewResolver(chains, analyzer.NewStandardFileLoader(root))
	table := resolution.NewTable()
	loader := analyzer.NewStandardFileLoader(root)
	diag := analyzer.AnalyzeSeeded(
		resourceSchema, "schemas/synnax/resource.oracle", "resource", loader, table,
	)
	Expect(diag.Ok()).To(BeTrue(), diag.String())
	for _, resource := range slices.Sorted(maps.Keys(lives)) {
		diag = analyzer.AnalyzeSeeded(
			lives[resource],
			"schemas/synnax/"+resource+".oracle",
			resource,
			loader,
			table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
	}
	return entry.New(entry.DefaultOptions()).Generate(&plugin.Request{
		Resolutions: table, RepoRoot: root, Versions: resolver,
	})
}

func fileAt(resp *plugin.Response, path string) string {
	GinkgoHelper()
	for _, f := range resp.Files {
		if f.Path == path {
			return string(f.Content)
		}
	}
	Fail("no file at " + path)
	return ""
}

var _ = Describe("Go Entry Plugin", func() {
	It("Should generate the methods in every version that declares the entry", func() {
		resp := MustSucceed(generate("label", `
@go output "core/pkg/service/label"

Key = uuid

Label struct {
	key Key @key
	name string
	color string

	@go entry
	@resource type "label"
}
`, map[string]string{
			"label/v0.oracle": `
Key = uuid

Label struct {
	key Key @key
	name string

	@go entry
	@resource type "label"
}
`,
			"label/v1.oracle": `
Key = v0.Key

Label struct extends v0.Label {
	color string
}
`,
		}))
		for _, v := range []string{"v0", "v1"} {
			Expect(fileAt(resp, "core/pkg/service/label/versions/"+v+"/entry.gen.go")).
				To(And(
					ContainSubstring("package "+v),
					ContainSubstring(
						`"github.com/synnaxlabs/synnax/pkg/service/resource"`,
					),
					ContainSubstring("func (l Label) GorpKey() Key { return l.Key }"),
					ContainSubstring(
						"return resource.ID{Type: resource.TypeLabel, Key: l.Key.String()}",
					),
					Not(ContainSubstring("CustomTypeName")),
				))
		}
	})

	It("Should skip a type the version aliases to its predecessor", func() {
		resp := MustSucceed(generate("label", `
@go output "core/pkg/service/label"

Key = uuid

Label struct {
	key Key @key

	@go entry
}
`, map[string]string{
			"label/v0.oracle": `
Key = uuid

Label struct {
	key Key @key

	@go entry
	@go marshal
}
`,
			"label/v1.oracle": `
Key = v0.Key

Label = v0.Label
`,
		},
			"core/pkg/service/label/versions/v0",
			"core/pkg/service/label/versions/v1",
		))
		Expect(fileAt(resp, "core/pkg/service/label/versions/v0/entry.gen.go")).
			To(ContainSubstring("GorpKey"))
		Expect(resp.Deletions).
			To(Equal([]string{"core/pkg/service/label/versions/v1/entry.gen.go"}))
	})

	It("Should name the table, and convert an initialism and a uuid key", func() {
		resp := MustSucceed(generate("ni_device", `
@go output "core/pkg/service/ni/device"

Device struct {
	key uuid @key

	@go entry "ni_device"
	@resource type "ni_device"
}
`, map[string]string{
			"ni_device/v0.oracle": `
Device struct {
	key uuid @key

	@go entry "ni_device"
	@resource type "ni_device"
}
`,
		}))
		Expect(fileAt(resp, "core/pkg/service/ni/device/versions/v0/entry.gen.go")).
			To(And(
				ContainSubstring(`"github.com/google/uuid"`),
				ContainSubstring(
					`func (Device) CustomTypeName() string { return "ni_device" }`,
				),
				ContainSubstring(
					"func (d Device) GorpKey() uuid.UUID { return d.Key }",
				),
				ContainSubstring("Type: resource.TypeNiDevice, Key: d.Key.String()"),
			))
	})

	It("Should read a union's key through its base", func() {
		resp := MustSucceed(generate("icd", `
@go output "core/pkg/service/icd"

Key = uuid

Base struct {
	key Key @key
}

ICD union on type extends Base {
	a { size int32 }
	b { name string }

	@go entry
	@resource type "icd"
}
`, map[string]string{
			"icd/v0.oracle": `
Key = uuid

Base struct {
	key Key @key
}

ICD union on type extends Base {
	a { size int32 }
	b { name string }

	@go entry
	@resource type "icd"
}
`,
		}))
		Expect(fileAt(resp, "core/pkg/service/icd/versions/v0/entry.gen.go")).
			To(And(
				ContainSubstring("func (i ICD) GorpKey() Key { return i.Base().Key }"),
				ContainSubstring("Type: resource.TypeICD, Key: i.Base().Key.String()"),
			))
	})

	It("Should use a string key as is, with the type parameters of a generic", func() {
		resp := MustSucceed(generate("status", `
@go output "core/pkg/service/status"

Key = string

Status struct<Details?> {
	key Key @key
	details Details?

	@go entry "Status"
	@resource type "status"
}
`, map[string]string{
			"status/v0.oracle": `
Key = string

Status struct<Details?> {
	key Key @key
	details Details?

	@go entry "Status"
	@resource type "status"
}
`,
		}))
		Expect(fileAt(resp, "core/pkg/service/status/versions/v0/entry.gen.go")).
			To(And(
				ContainSubstring(
					`func (Status[Details]) CustomTypeName() string { return "Status" }`,
				),
				ContainSubstring(
					"func (s Status[Details]) GorpKey() Key { return s.Key }",
				),
				ContainSubstring("Type: resource.TypeStatus, Key: s.Key}"),
			))
	})

	It("Should read an inherited key", func() {
		resp := MustSucceed(generate("opcua", `
@go output "core/pkg/service/opcua"

Keyed struct {
	key uuid @key
}

ReadConfig struct extends Keyed {
	device string

	@go entry "opc_read_config"
}
`, map[string]string{
			"opcua/v0.oracle": `
Keyed struct {
	key uuid @key
}

ReadConfig struct extends Keyed {
	device string

	@go entry "opc_read_config"
}
`,
		}))
		Expect(fileAt(resp, "core/pkg/service/opcua/versions/v0/entry.gen.go")).
			To(And(
				ContainSubstring(
					`func (ReadConfig) CustomTypeName() string { return "opc_read_config" }`,
				),
				ContainSubstring(
					"func (r ReadConfig) GorpKey() uuid.UUID { return r.Key }",
				),
				Not(ContainSubstring("func (k Keyed)")),
			))
	})

	It("Should read a key typed in an earlier version through an aliased base", func() {
		resp := MustSucceed(generate("view", `
@go output "core/pkg/service/view"

Key = uuid

Base struct {
	key Key @key
}

View union on type extends Base {
	first { size int32 }
	second { name string }
}
`, map[string]string{
			"view/v0.oracle": `
Key = uuid

Base struct {
	key Key @key
}

View union on type extends Base {
	first { size int32 }

	@go entry
}
`,
			"view/v1.oracle": `
Key = v0.Key

Base = v0.Base

View union on type extends Base {
	first { size int32 }
	second { name string }

	@go entry
}
`,
		}))
		Expect(fileAt(resp, "core/pkg/service/view/versions/v1/entry.gen.go")).
			To(ContainSubstring("func (v View) GorpKey() Key { return v.Base().Key }"))
	})

	DescribeTable("Should convert the key to the string a resource ID holds",
		func(decls, expected string) {
			source := decls + `
Label struct {
	key Key @key

	@go entry
	@resource type "label"
}
`
			resp := MustSucceed(generate("label",
				"@go output \"core/pkg/service/label\"\n"+source,
				map[string]string{"label/v0.oracle": source},
			))
			Expect(fileAt(resp, "core/pkg/service/label/versions/v0/entry.gen.go")).
				To(ContainSubstring("Key: " + expected + "}"))
		},
		Entry("a distinct string", "Key string", "string(l.Key)"),
		Entry("an unsigned integer", "Key uint32",
			"strconv.FormatUint(uint64(l.Key), 10)"),
		Entry("a signed integer alias", "Key = int64",
			"strconv.FormatInt(int64(l.Key), 10)"),
		Entry("a distinct uuid", "Key uuid", "uuid.UUID(l.Key).String()"),
	)

	DescribeTable("Should fail on a type it cannot generate methods for",
		func(source, message string) {
			Expect(generate("label",
				"@go output \"core/pkg/service/label\"\n"+source,
				map[string]string{"label/v0.oracle": source},
			)).Error().To(MatchError(ContainSubstring(message)))
		},
		Entry("a resource type without a key", `
Label struct {
	name string

	@go entry
	@resource type "label"
}
`, "@resource type needs a @key field"),
		Entry("an unsupported key type", `
Label struct {
	key float64 @key

	@go entry
}
`, "unsupported key type float64"),
		Entry("a union without a shared base", `
A struct {
	key uuid @key
}

B struct {
	name string
}

Label union on type extends A, B {
	first { size int32 }

	@go entry
}
`, "needs one base that every variant inherits"),
		Entry("an enum", `
Label enum {
	a = "a"

	@go entry
}
`, "applies only to a struct or union"),
	)

	It("Should fail on a key type from another package", func() {
		// The live files only name the outputs; the version files drive generation.
		Expect(generateWith(map[string]string{
			"label": `
@go output "core/pkg/service/label"

Label struct {
	key uint32 @key
}
`,
			"other": `
@go output "core/pkg/service/other"

Key uint32
`,
		}, map[string]string{
			"other/v0.oracle": "Key uint32\n",
			"label/v0.oracle": `
import "schemas/synnax/versions/other/v0"

Label struct {
	key other.Key @key

	@go entry
}
`,
		})).Error().To(MatchError(ContainSubstring("must be declared in the same version")))
	})

	It("Should fail on a resource type the enum does not declare", func() {
		Expect(generate("label", `
@go output "core/pkg/service/label"

Label struct {
	key uuid @key

	@go entry
	@resource type "missing"
}
`, map[string]string{
			"label/v0.oracle": `
Label struct {
	key uuid @key

	@go entry
	@resource type "missing"
}
`,
		})).Error().To(MatchError(ContainSubstring(`no member "missing"`)))
	})
})
