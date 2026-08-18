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
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
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

// depPlugin is a stub plugin with configurable dependencies. It records the order
// plugins ran in, so specs can assert topological scheduling.
type depPlugin struct {
	name     string
	requires []string
	err      error
	nilResp  bool
	mu       *sync.Mutex
	order    *[]string
}

func (p *depPlugin) Name() string                { return p.name }
func (p *depPlugin) Domains() []string           { return nil }
func (p *depPlugin) Requires() []string          { return p.requires }
func (p *depPlugin) Check(*plugin.Request) error { return nil }
func (p *depPlugin) Generate(*plugin.Request) (*plugin.Response, error) {
	p.mu.Lock()
	*p.order = append(*p.order, p.name)
	p.mu.Unlock()
	if p.err != nil {
		return nil, p.err
	}
	if p.nilResp {
		return nil, nil
	}
	return &plugin.Response{
		Files:     []plugin.File{{Path: "out/" + p.name, Content: []byte(p.name)}},
		Deletions: []string{"out/" + p.name + ".old"},
	}, nil
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

	Describe("plugin scheduling", func() {
		var (
			mu    sync.Mutex
			order []string
		)

		BeforeEach(func() {
			order = nil
			writeSchema("widget", `
@go output "x/go/widget"
Thing struct {
    name string
}
`)
		})

		newDep := func(name string, requires ...string) *depPlugin {
			return &depPlugin{
				name: name, requires: requires, mu: &mu, order: &order,
			}
		}

		run := func(ctx SpecContext, plugins ...plugin.Plugin) *pipeline.Result {
			GinkgoHelper()
			registry := plugin.NewRegistry()
			for _, p := range plugins {
				Expect(registry.Register(p)).To(Succeed())
			}
			return MustSucceed(pipeline.Run(ctx, pipeline.Options{
				RepoRoot: repoRoot,
				Schemas:  MustSucceed(pipeline.DiscoverSchemas(repoRoot)),
				Plugins:  registry,
			}))
		}

		It("runs a plugin's dependencies before the plugin", func(ctx SpecContext) {
			result := run(ctx, newDep("late", "early"), newDep("early"))
			Expect(result.Diagnostics.Ok()).To(BeTrue())
			Expect(order).To(Equal([]string{"early", "late"}))
			Expect(result.Outputs).To(HaveKey("late"))
			Expect(result.Deletions["early"]).To(Equal([]string{"out/early.old"}))
		})

		It("fails when a plugin requires an unregistered plugin", func(
			ctx SpecContext,
		) {
			result := run(ctx, newDep("lonely", "ghost"))
			Expect(result.Diagnostics.Ok()).To(BeFalse())
			Expect(result.Diagnostics.String()).To(
				ContainSubstring(`plugin "lonely" requires unknown plugin "ghost"`),
			)
		})

		It("still runs cyclically dependent plugins", func(ctx SpecContext) {
			result := run(ctx, newDep("a", "b"), newDep("b", "a"))
			Expect(result.Diagnostics.Ok()).To(BeTrue())
			Expect(order).To(ConsistOf("a", "b"))
		})

		It("surfaces a plugin generation error as a diagnostic", func(
			ctx SpecContext,
		) {
			failing := newDep("broken")
			failing.err = errors.New("kaboom")
			result := run(ctx, failing)
			Expect(result.Diagnostics.Ok()).To(BeFalse())
			Expect(result.Diagnostics.String()).To(SatisfyAll(
				ContainSubstring("plugin broken"),
				ContainSubstring("kaboom"),
			))
		})

		It("tolerates a nil plugin response", func(ctx SpecContext) {
			quiet := newDep("quiet")
			quiet.nilResp = true
			result := run(ctx, quiet)
			Expect(result.Diagnostics.Ok()).To(BeTrue())
			Expect(result.Outputs).NotTo(HaveKey("quiet"))
		})
	})

	Describe("version chains", func() {
		var write func(rel, content string)

		BeforeEach(func() {
			write = func(rel, content string) {
				abs := filepath.Join(repoRoot, rel)
				Expect(os.MkdirAll(filepath.Dir(abs), 0o755)).To(Succeed())
				Expect(os.WriteFile(abs, []byte(content), 0o644)).To(Succeed())
			}
			write("licenses/headers/template.txt",
				"Copyright {{YEAR}} Synnax Labs, Inc.\n")
			write("schemas/synnax/versions/channel/v0.oracle", `
Channel struct {
    name string {
        @doc value "names the channel."
    }

    @go marshal
}
`)
		})

		It("merges the live projection for a versioned resource", func(
			ctx SpecContext,
		) {
			write("schemas/synnax/channel.oracle", `Channel struct {
    name string {
        @doc value "names the channel."
        @validate required
    }

    @go output "core/pkg/service/channel"
}
`)
			result := MustSucceed(pipeline.Run(ctx, pipeline.Options{
				RepoRoot: repoRoot,
				Schemas:  MustSucceed(pipeline.DiscoverSchemas(repoRoot)),
			}))
			Expect(result.Chains).To(HaveKey("schemas/synnax/channel"))
			Expect(result.Versions).NotTo(BeNil())

			live := "schemas/synnax/channel.oracle"
			merged := string(result.MergedSources[live])
			Expect(merged).To(SatisfyAll(
				ContainSubstring("@go marshal"),
				ContainSubstring("@validate required"),
			))
			Expect(string(result.EffectiveSource(live))).To(Equal(merged))
		})

		It("returns the formatted source for unversioned schemas", func(
			ctx SpecContext,
		) {
			write("schemas/synnax/channel.oracle", "Channel struct {\n"+
				"    name string {\n"+
				"        @doc value \"names the channel.\"\n"+
				"    }\n}\n")
			plain := writeSchema("widget", `
@go output "x/go/widget"
Thing struct {
    name string
}
`)
			result := MustSucceed(pipeline.Run(ctx, pipeline.Options{
				RepoRoot: repoRoot,
				Schemas:  MustSucceed(pipeline.DiscoverSchemas(repoRoot)),
			}))
			Expect(result.MergedSources).NotTo(HaveKey(plain))
			Expect(result.EffectiveSource(plain)).
				To(Equal(result.FormattedSources[plain]))
		})

		It("adds a merged live file missing from the input set", func(
			ctx SpecContext,
		) {
			// Only the chain exists on disk; the live projection is synthesized
			// entirely from the version file and joins the schema set.
			writeSchema("widget", `
@go output "x/go/widget"
Thing struct {
    name string
}
`)
			result := MustSucceed(pipeline.Run(ctx, pipeline.Options{
				RepoRoot: repoRoot,
				Schemas:  MustSucceed(pipeline.DiscoverSchemas(repoRoot)),
			}))
			live := "schemas/synnax/channel.oracle"
			Expect(result.Schemas).To(ContainElement(live))
			Expect(string(result.MergedSources[live])).
				To(ContainSubstring("@go marshal"))
		})
	})

	It("fails when an input schema does not exist on disk", func(ctx SpecContext) {
		// Run returns a partial, non-nil Result alongside the error, so the
		// error and the result are asserted separately.
		result, err := pipeline.Run(ctx, pipeline.Options{
			RepoRoot: repoRoot,
			Schemas:  []string{"schemas/ghost.oracle"},
		})
		Expect(err).To(MatchError(
			ContainSubstring("read schema schemas/ghost.oracle"),
		))
		Expect(result.Sources).To(BeEmpty())
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

	It("excludes version chain directories", func() {
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
		write("schemas/synnax/versions/channel/v0.oracle")
		write("schemas/x/versions/telem/v0.oracle")
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
