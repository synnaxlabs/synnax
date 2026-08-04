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
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

// stubPlugin is the minimum Plugin needed to exercise pipeline.Run. It
// emits one generated File per top-level type in the resolved table so
// the test can observe deterministic, schema-driven output.
type stubPlugin struct {
	name string
}

func (p *stubPlugin) Name() string              { return p.name }
func (*stubPlugin) Domains() []string           { return nil }
func (*stubPlugin) Requires() []string          { return nil }
func (*stubPlugin) Check(*plugin.Request) error { return nil }
func (*stubPlugin) Generate(req *plugin.Request) (*plugin.Response, error) {
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
	var repoRoot string

	BeforeEach(func() {
		repoRoot = MustSucceed(os.MkdirTemp("", "pipeline"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(repoRoot)).To(Succeed())
		})
		Expect(os.MkdirAll(filepath.Join(repoRoot, "schemas"), 0o755)).To(Succeed())
	})

	writeSchema := func(name, body string) string {
		rel := "schemas/" + name + ".oracle"
		Expect(
			os.WriteFile(filepath.Join(repoRoot, rel), []byte(body), 0o644),
		).To(Succeed())
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
		paths := set.New[string]()
		for _, f := range result.Outputs["stub"] {
			paths.Add(f.Path)
		}
		Expect(paths.Contains("out/widget_Thing.gen.go")).To(BeTrue())
	})

	It(
		"supplies snapshots so versioned packages alias their predecessor",
		func(ctx SpecContext) {
			writeSchema("thing", `
@go output "x/go/thing"
Stable struct {
    @go version 1
    name string
}
Grown struct {
    @go version 1
    value int32
    extra string
}
`)
			snapDir := filepath.Join(repoRoot, "schemas", "snapshots", "v1")
			Expect(os.MkdirAll(snapDir, 0o755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(snapDir, "thing.oracle"), []byte(`
@go output "x/go/thing"
Stable struct {
    @go version 0
    name string
}
Grown struct {
    @go version 0
    value int32
}
`), 0o644)).To(Succeed())

			registry := plugin.NewRegistry()
			Expect(
				registry.Register(gotypes.New(gotypes.DefaultOptions())),
			).To(Succeed())
			schemas := MustSucceed(pipeline.DiscoverSchemas(repoRoot))
			result := MustSucceed(pipeline.Run(ctx, pipeline.Options{
				RepoRoot: repoRoot,
				Schemas:  schemas,
				Plugins:  registry,
			}))
			Expect(result.Diagnostics.Ok()).To(BeTrue())

			var current string
			for _, f := range result.Outputs["go/types"] {
				if f.Path == "x/go/thing/versions/v1/types.gen.go" {
					current = string(f.Content)
				}
			}
			Expect(current).To(ContainSubstring("type Stable = v0.Stable"))
			Expect(current).To(ContainSubstring("type Grown struct"))
			Expect(current).NotTo(ContainSubstring("type Stable struct"))
		},
	)

	It(
		"does not double-register types when a schema imports another",
		func(ctx SpecContext) {
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
		},
	)

	It("resolves imports across nested schema folders", func(ctx SpecContext) {
		writeNested := func(rel, body string) {
			abs := filepath.Join(repoRoot, rel)
			Expect(os.MkdirAll(filepath.Dir(abs), 0o755)).To(Succeed())
			Expect(os.WriteFile(abs, []byte(body), 0o644)).To(Succeed())
		}
		writeNested("schemas/x/telem.oracle", `
@go output "x/go/telem"
Rate struct {
    hz float64
}
`)
		writeNested("schemas/synnax/channel.oracle", `
import "schemas/x/telem"

@go output "core/pkg/distribution/channel"
Channel struct {
    rate telem.Rate
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

		Expect(result.Diagnostics.Ok()).To(BeTrue(),
			"a synnax schema importing a nested x schema should resolve cleanly")
		paths := set.New[string]()
		for _, f := range result.Outputs["stub"] {
			paths.Add(f.Path)
		}
		Expect(paths.Contains("out/telem_Rate.gen.go")).To(BeTrue())
		Expect(paths.Contains("out/channel_Channel.gen.go")).To(BeTrue())
	})

	It(
		"produces byte-identical outputs across runs (determinism)",
		func(ctx SpecContext) {
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
			opts := pipeline.Options{
				RepoRoot: repoRoot,
				Schemas:  schemas,
				Plugins:  registry,
			}
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
		},
	)

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
		Expect(os.MkdirAll(filepath.Join(repoRoot, "schemas"), 0o755)).To(Succeed())
		for _, name := range []string{"c.oracle", "a.oracle", "b.oracle"} {
			Expect(
				os.WriteFile(
					filepath.Join(repoRoot, "schemas", name),
					[]byte(""),
					0o644,
				),
			).To(Succeed())
		}
		got := MustSucceed(pipeline.DiscoverSchemas(repoRoot))
		Expect(
			got,
		).To(Equal([]string{"schemas/a.oracle", "schemas/b.oracle", "schemas/c.oracle"}))
	})

	It("recurses into subdirectories", func() {
		repoRoot := MustSucceed(os.MkdirTemp("", "discover"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(repoRoot)).To(Succeed())
		})
		write := func(rel string) {
			abs := filepath.Join(repoRoot, rel)
			Expect(os.MkdirAll(filepath.Dir(abs), 0o755)).To(Succeed())
			Expect(os.WriteFile(abs, []byte(""), 0o644)).To(Succeed())
		}
		write("schemas/x/telem.oracle")
		write("schemas/synnax/channel.oracle")
		write("schemas/arc/ir.oracle")

		Expect(pipeline.DiscoverSchemas(repoRoot)).To(Equal([]string{
			"schemas/arc/ir.oracle",
			"schemas/synnax/channel.oracle",
			"schemas/x/telem.oracle",
		}))
	})

	It("excludes the snapshots directory", func() {
		repoRoot := MustSucceed(os.MkdirTemp("", "discover"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(repoRoot)).To(Succeed())
		})
		write := func(rel string) {
			abs := filepath.Join(repoRoot, rel)
			Expect(os.MkdirAll(filepath.Dir(abs), 0o755)).To(Succeed())
			Expect(os.WriteFile(abs, []byte(""), 0o644)).To(Succeed())
		}
		write("schemas/synnax/channel.oracle")
		write("schemas/snapshots/v56/channel.oracle")
		write("schemas/snapshots/v56/arc/ir.oracle")
		Expect(pipeline.DiscoverSchemas(repoRoot)).
			To(Equal([]string{"schemas/synnax/channel.oracle"}))
	})

	It("returns empty when the schemas directory does not exist", func() {
		repoRoot := MustSucceed(os.MkdirTemp("", "discover"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(repoRoot)).To(Succeed())
		})
		Expect(pipeline.DiscoverSchemas(repoRoot)).To(BeEmpty())
	})

	It("returns a wrapped error when a schema subdirectory cannot be read", func() {
		if os.Geteuid() == 0 {
			Skip("filesystem permissions are bypassed when running as root")
		}
		repoRoot := MustSucceed(os.MkdirTemp("", "discover"))
		locked := filepath.Join(repoRoot, "schemas", "locked")
		Expect(os.MkdirAll(locked, 0o755)).To(Succeed())
		Expect(os.Chmod(locked, 0o000)).To(Succeed())
		DeferCleanup(func() {
			Expect(os.Chmod(locked, 0o755)).To(Succeed())
			Expect(os.RemoveAll(repoRoot)).To(Succeed())
		})

		Expect(pipeline.DiscoverSchemas(repoRoot)).Error().
			To(MatchError(ContainSubstring("walk schema directory")))
	})
})
