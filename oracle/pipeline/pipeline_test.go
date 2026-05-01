// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pipeline_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin"
	. "github.com/synnaxlabs/x/testutil"
)

// stubPlugin is the minimum Plugin needed to exercise pipeline.Run. It
// emits one generated File per top-level type in the resolved table so
// the test can observe deterministic, schema-driven output.
type stubPlugin struct {
	name string
}

func (p *stubPlugin) Name() string                { return p.name }
func (p *stubPlugin) Domains() []string           { return nil }
func (p *stubPlugin) Requires() []string          { return nil }
func (p *stubPlugin) Check(*plugin.Request) error { return nil }
func (p *stubPlugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	files := []plugin.File{}
	for _, t := range req.Resolutions.Types {
		files = append(files, plugin.File{
			Path:    "out/" + t.Namespace + "_" + t.Name + ".gen.go",
			Content: []byte("package out\n// " + t.QualifiedName + "\n"),
		})
	}
	return &plugin.Response{Files: files}, nil
}

var _ = Describe("pipeline.Run", func() {
	var (
		repoRoot string
	)

	BeforeEach(func() {
		repoRoot = MustSucceed(os.MkdirTemp("", "pipeline"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(repoRoot)).To(Succeed())
		})
		Expect(os.MkdirAll(filepath.Join(repoRoot, "schemas"), 0755)).To(Succeed())
	})

	writeSchema := func(name, body string) string {
		rel := "schemas/" + name + ".oracle"
		Expect(os.WriteFile(filepath.Join(repoRoot, rel), []byte(body), 0644)).To(Succeed())
		return rel
	}

	It("runs analyze and generate against a single schema", func(ctx SpecContext) {
		writeSchema("widget", `
@go output "x/go/widget"
Thing struct {
    name string
}
`)
		registry := plugin.NewRegistry()
		Expect(registry.Register(&stubPlugin{name: "stub"})).To(Succeed())

		schemas := MustSucceed(pipeline.DiscoverSchemas(repoRoot))
		result := MustSucceed(pipeline.Run(ctx, pipeline.Options{
			RepoRoot: repoRoot,
			Schemas:  schemas,
			Plugins:  registry,
		}))

		Expect(result.Diagnostics.Ok()).To(BeTrue())
		Expect(result.Resolutions).NotTo(BeNil())
		paths := make(map[string]bool)
		for _, f := range result.Outputs["stub"] {
			paths[f.Path] = true
		}
		Expect(paths).To(HaveKey("out/widget_Thing.gen.go"))
	})

	It("does not double-register types when a schema imports another", func(ctx SpecContext) {
		// Regression test for the original `oracle check` bug: passing
		// every schema as a top-level input AND letting the analyzer
		// transitively resolve imports caused each type to be
		// registered twice.
		writeSchema("base", `
@go output "x/go/base"
Thing struct {
    name string
}
`)
		writeSchema("user", `
import "schemas/base"

@go output "x/go/user"
WithThing struct {
    thing base.Thing
}
`)
		schemas := MustSucceed(pipeline.DiscoverSchemas(repoRoot))
		result := MustSucceed(pipeline.Run(ctx, pipeline.Options{
			RepoRoot: repoRoot,
			Schemas:  schemas,
		}))
		Expect(result.Diagnostics.Ok()).To(BeTrue(),
			"analyzer should not produce duplicate-definition errors for top-level + transitively-imported schemas")
	})

	It("produces byte-identical outputs across runs (determinism)", func(ctx SpecContext) {
		writeSchema("a", `
@go output "x/go/a"
X struct { name string }
Y struct { name string }
Z struct { name string }
`)
		writeSchema("b", `
@go output "x/go/b"
X struct { name string }
`)
		registry := plugin.NewRegistry()
		Expect(registry.Register(&stubPlugin{name: "stub"})).To(Succeed())

		schemas := MustSucceed(pipeline.DiscoverSchemas(repoRoot))
		opts := pipeline.Options{RepoRoot: repoRoot, Schemas: schemas, Plugins: registry}
		first := MustSucceed(pipeline.Run(ctx, opts))
		second := MustSucceed(pipeline.Run(ctx, opts))

		Expect(first.Outputs["stub"]).To(HaveLen(len(second.Outputs["stub"])))
		firstByPath := make(map[string][]byte)
		for _, f := range first.Outputs["stub"] {
			firstByPath[f.Path] = f.Content
		}
		for _, f := range second.Outputs["stub"] {
			Expect(string(f.Content)).To(Equal(string(firstByPath[f.Path])),
				"output for %s diverged between runs", f.Path)
		}
	})

	It("rejects empty schema set", func(ctx SpecContext) {
		_, err := pipeline.Run(ctx, pipeline.Options{
			RepoRoot: repoRoot,
			Schemas:  nil,
		})
		Expect(err).To(MatchError(ContainSubstring("at least one schema")))
	})

	It("rejects empty repo root", func(ctx SpecContext) {
		_, err := pipeline.Run(ctx, pipeline.Options{
			Schemas: []string{"schemas/x.oracle"},
		})
		Expect(err).To(MatchError(ContainSubstring("RepoRoot is required")))
	})
})

var _ = Describe("pipeline.DiscoverSchemas", func() {
	It("returns repo-relative paths in sorted order", func() {
		repoRoot := MustSucceed(os.MkdirTemp("", "discover"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(repoRoot)).To(Succeed())
		})
		Expect(os.MkdirAll(filepath.Join(repoRoot, "schemas"), 0755)).To(Succeed())
		for _, name := range []string{"c.oracle", "a.oracle", "b.oracle"} {
			Expect(os.WriteFile(filepath.Join(repoRoot, "schemas", name), []byte(""), 0644)).To(Succeed())
		}
		got := MustSucceed(pipeline.DiscoverSchemas(repoRoot))
		Expect(got).To(Equal([]string{"schemas/a.oracle", "schemas/b.oracle", "schemas/c.oracle"}))
	})
})
