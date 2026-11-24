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
	"os"
	"path/filepath"

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
	diag = analyzer.AnalyzeSeeded(
		live, "schemas/synnax/"+resource+".oracle", resource, loader, table,
	)
	Expect(diag.Ok()).To(BeTrue(), diag.String())
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

	It("Should name only the table of a type whose key Go promotes", func() {
		resp := MustSucceed(generate("opcua", `
@go output "core/pkg/service/opcua"

Keyed struct {
	key uuid @key

	@go entry
}

ReadConfig struct extends Keyed {
	device string

	@go entry "opc_read_config"
}
`, map[string]string{
			"opcua/v0.oracle": `
Keyed struct {
	key uuid @key

	@go entry
}

ReadConfig struct extends Keyed {
	device string

	@go entry "opc_read_config"
}
`,
		}))
		content := fileAt(resp, "core/pkg/service/opcua/versions/v0/entry.gen.go")
		Expect(content).To(And(
			ContainSubstring("func (k Keyed) GorpKey() uuid.UUID { return k.Key }"),
			ContainSubstring(
				`func (ReadConfig) CustomTypeName() string { return "opc_read_config" }`,
			),
			Not(ContainSubstring("func (r ReadConfig) GorpKey()")),
		))
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
