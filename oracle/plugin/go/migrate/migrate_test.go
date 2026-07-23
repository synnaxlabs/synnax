// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate_test

import (
	"context"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/plugin/go/migrate"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func analyze(
	ctx context.Context,
	source, namespace string,
	loader *testutil.MockFileLoader,
) (*resolution.Table, error) {
	table, diag := analyzer.AnalyzeSource(ctx, source, namespace, loader)
	if diag != nil && !diag.Ok() {
		return nil, diag
	}
	if table == nil {
		return nil, diag
	}
	return table, nil
}

func generate(
	ctx context.Context,
	oldSource, newSource, namespace string,
	loader *testutil.MockFileLoader,
	p plugin.Plugin,
) (*plugin.Response, error) {
	newTable, err := analyze(ctx, newSource, namespace, loader)
	if err != nil {
		return nil, err
	}
	req := &plugin.Request{Resolutions: newTable, RepoRoot: loader.RepoRoot()}
	if oldSource != "" {
		oldTable, err := analyze(ctx, oldSource, namespace, loader)
		if err != nil {
			return nil, err
		}
		req.OldResolutions = oldTable
	}
	return p.Generate(req)
}

func fileContent(resp *plugin.Response, suffix string) string {
	for _, f := range resp.Files {
		if strings.HasSuffix(f.Path, suffix) {
			return string(f.Content)
		}
	}
	return ""
}

func filePaths(resp *plugin.Response) []string {
	paths := make([]string, len(resp.Files))
	for i, f := range resp.Files {
		paths[i] = f.Path
	}
	return paths
}

var _ = Describe("Go Migrate Plugin", func() {
	var (
		ctx    context.Context
		loader *testutil.MockFileLoader
		p      *migrate.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = testutil.NewMockFileLoader()
		p = migrate.New()
	})

	Describe("Plugin Interface", func() {
		It("Should have the correct name", func() {
			Expect(p.Name()).To(Equal("go/migrate"))
		})
		It("Should filter on go domain", func() {
			Expect(p.Domains()).To(Equal([]string{"go"}))
		})
		It("Should require go/types and go/marshal", func() {
			Expect(p.Requires()).To(Equal([]string{"go/types", "go/marshal"}))
		})
	})

	Describe("Generate", func() {
		Context("no schema change", func() {
			It("Should generate no files when schemas are identical", func() {
				schema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, schema, schema, "test", loader, p))
				Expect(resp.Files).To(BeEmpty())
			})
		})

		Context("pre-versioning snapshot", func() {
			It("Should generate no files against a snapshot with no @go version", func() {
				oldSchema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						email string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				Expect(resp.Files).To(BeEmpty())
			})
		})

		Context("no old resolutions", func() {
			It("Should generate no files with no old resolutions", func() {
				schema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, "", schema, "test", loader, p))
				Expect(resp.Files).To(BeEmpty())
			})
		})

		Context("version discipline", func() {
			It("Should error when the shape changed without a version bump", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						age int32
						@go migrate
					}
				`
				Expect(generate(ctx, oldSchema, newSchema, "test", loader, p)).
					Error().To(MatchError(ContainSubstring(
					"changed but @go version is still 1",
				)))
			})

			It("Should not require a bump when a new type is added", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
					Format enum {
						iso  = "ISO"
						date = "date"
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				Expect(resp.Files).To(BeEmpty())
			})

			It("Should error when a type is removed without a version bump", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
					Format enum {
						iso  = "ISO"
						date = "date"
					}
				`
				newSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				Expect(generate(ctx, oldSchema, newSchema, "test", loader, p)).
					Error().To(MatchError(ContainSubstring(
					"changed but @go version is still 1",
				)))
			})

			It("Should error when the version jumps by more than one", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 3
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						age int32
						@go migrate
					}
				`
				Expect(generate(ctx, oldSchema, newSchema, "test", loader, p)).
					Error().To(MatchError(ContainSubstring("jumped from 1 to 3")))
			})

			It("Should error when the version decreases", func() {
				oldSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				Expect(generate(ctx, oldSchema, newSchema, "test", loader, p)).
					Error().To(MatchError(ContainSubstring("decreased from 2 to 1")))
			})

			It("Should error when @go migrate lacks a @go version", func() {
				schema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				Expect(generate(ctx, schema, schema, "test", loader, p)).
					Error().To(MatchError(ContainSubstring(
					"@go migrate requires a @go version declaration",
				)))
			})

			It("Should scaffold against an unversioned dependency's live path", func() {
				loader.Add("schemas/dep", `
					@go output "dep"
					Inner struct { value int32 }
				`)
				oldSchema := `
					import "schemas/dep"
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key        {@key}
						inner dep.Inner
						@go migrate
					}
				`
				newSchema := `
					import "schemas/dep"
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key        {@key}
						inner dep.Inner
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				auto := fileContent(resp, "out/types/v2/migrate_auto.gen.go")
				Expect(auto).To(ContainSubstring("autoMigrateEntry"))
				Expect(auto).NotTo(ContainSubstring("dep/types/"))
			})
		})

		Context("codec-only version bump", func() {
			var resp *plugin.Response

			BeforeEach(func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				resp = MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
			})

			It("Should not re-emit the outgoing version", func() {
				for _, path := range filePaths(resp) {
					Expect(path).To(HavePrefix("out/types/v2/"))
				}
			})

			It("Should synthesize a full-copy passthrough auto-migrate", func() {
				autoCopy := fileContent(resp, "out/types/v2/migrate_auto.gen.go")
				Expect(autoCopy).To(ContainSubstring("func autoMigrateEntry"))
				Expect(autoCopy).To(ContainSubstring("Name: old.Name"))
			})

			It("Should scaffold the developer transform template", func() {
				tmpl := fileContent(resp, "out/types/v2/migrate.go")
				Expect(tmpl).To(ContainSubstring("package v2"))
				Expect(tmpl).To(ContainSubstring("func migrateEntry"))
				Expect(tmpl).To(ContainSubstring("autoMigrateEntry(ctx, old)"))
			})
		})

		Context("union type changes", func() {
			It("Should skip auto-copying fields whose union-ness changed", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						configs map<string, record>
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					TankConfig struct { width float64 }
					ElementConfig union on variant {
						tank TankConfig
					}
					Entry struct {
						key Key {@key}
						configs map<string, ElementConfig>
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).NotTo(ContainSubstring("Configs: old.Configs"))
			})
		})

		Context("enum value-set changes", func() {
			It("Should generate no files when a string enum value is removed", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Mode enum { a = "a"  b = "b"  c = "c" }
					Entry struct {
						key Key {@key}
						mode Mode
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Mode enum { a = "a"  b = "b" }
					Entry struct {
						key Key {@key}
						mode Mode
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				Expect(resp.Files).To(BeEmpty())
			})

			It("Should generate no files when a string enum value is added", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Mode enum { a = "a"  b = "b" }
					Entry struct {
						key Key {@key}
						mode Mode
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Mode enum { a = "a"  b = "b"  c = "c" }
					Entry struct {
						key Key {@key}
						mode Mode
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				Expect(resp.Files).To(BeEmpty())
			})

			It("Should generate no files when an int enum value is added", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Mode enum { a = 1  b = 2 }
					Entry struct {
						key Key {@key}
						mode Mode
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Mode enum { a = 1  b = 2  c = 3 }
					Entry struct {
						key Key {@key}
						mode Mode
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				Expect(resp.Files).To(BeEmpty())
			})

			It("Should freeze when an int enum renumbers an existing name", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Mode enum { a = 1  b = 2 }
					Entry struct {
						key Key {@key}
						mode Mode
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Mode enum { a = 1  b = 3 }
					Entry struct {
						key Key {@key}
						mode Mode
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				Expect(resp.Files).ToNot(BeEmpty())
			})
		})

		Context("field addition", func() {
			var resp *plugin.Response

			BeforeEach(func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						age int32
						@go migrate
					}
				`
				resp = MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
			})

			It("Should not re-emit the outgoing version", func() {
				for _, path := range filePaths(resp) {
					Expect(path).To(HavePrefix("out/types/v2/"))
				}
			})

			It("Should generate auto-copy with error propagation", func() {
				content := fileContent(resp, "out/types/v2/migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("func autoMigrateEntry"))
				Expect(content).NotTo(ContainSubstring("var _ = context.Background"))
				Expect(content).NotTo(ContainSubstring(", _ :="))
				Expect(content).NotTo(ContainSubstring(", _ ="))
			})

			It("Should not generate migrate.gen.go", func() {
				Expect(fileContent(resp, "migrate.gen.go")).To(BeEmpty())
			})

			It("Should generate developer transform template", func() {
				content := fileContent(resp, "out/types/v2/migrate.go")
				Expect(content).To(ContainSubstring("package v2"))
				Expect(content).To(ContainSubstring("func migrateEntry"))
				Expect(content).To(ContainSubstring("autoMigrateEntry"))
				Expect(content).To(ContainSubstring("Edit this file"))
			})
		})

		Context("field removal", func() {
			It("Should generate migration for removed field", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						age int32
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				for _, path := range filePaths(resp) {
					Expect(path).To(HavePrefix("out/types/v2/"))
				}
				autoCopy := fileContent(resp, "migrate_auto.gen.go")
				Expect(autoCopy).To(ContainSubstring("autoMigrateEntry"))
				Expect(autoCopy).NotTo(ContainSubstring("Age"))
			})
		})

		Context("marshal directive change", func() {
			It("Should generate a migration when @go marshal omit is added to a field", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						transient string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						transient string {
							@go marshal omit
						}
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				for _, path := range filePaths(resp) {
					Expect(path).To(HavePrefix("out/types/v2/"))
				}
				Expect(fileContent(resp, "migrate_auto.gen.go")).
					To(ContainSubstring("autoMigrateEntry"))
			})

			It("Should generate a migration when @go marshal omit is removed", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						transient string {
							@go marshal omit
						}
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						transient string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				for _, path := range filePaths(resp) {
					Expect(path).To(HavePrefix("out/types/v2/"))
				}
				Expect(fileContent(resp, "migrate_auto.gen.go")).
					To(ContainSubstring("autoMigrateEntry"))
			})

			It("Should migrate when json_only is added to a type-param field", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct<Details? = record> {
						key Key {@key}
						details Details?
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct<Details? = record> {
						key Key {@key}
						details Details? {
							@go marshal json_only
						}
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				for _, path := range filePaths(resp) {
					Expect(path).To(HavePrefix("out/types/v2/"))
				}
				Expect(fileContent(resp, "migrate_auto.gen.go")).
					To(ContainSubstring("autoMigrateEntry"))
			})
		})

		Context("generic types", func() {
			It("Should emit type-param decoration on a generic autoMigrate", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = string
					Entry struct<Details?> {
						key Key {@key}
						name string
						details Details?
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = string
					Entry struct<Details?> {
						key Key {@key}
						name string
						description string
						details Details?
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				autoCopy := fileContent(resp, "migrate_auto.gen.go")
				Expect(autoCopy).To(ContainSubstring("func autoMigrateEntry[Details any]"))
				Expect(autoCopy).To(ContainSubstring("old v1.Entry[Details]"))
				Expect(autoCopy).To(ContainSubstring(") (Entry[Details], error)"))
				Expect(autoCopy).To(ContainSubstring("return Entry[Details]{"))
			})

			It("Should emit type-param decoration on the Migrate developer template", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = string
					Entry struct<Details?> {
						key Key {@key}
						name string
						details Details?
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = string
					Entry struct<Details?> {
						key Key {@key}
						name string
						description string
						details Details?
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				transform := fileContent(resp, "out/types/v2/migrate.go")
				Expect(transform).To(ContainSubstring("func migrateEntry[Details any]"))
				Expect(transform).To(ContainSubstring("old v1.Entry[Details]"))
				Expect(transform).To(ContainSubstring(") (Entry[Details], error)"))
				Expect(transform).To(ContainSubstring("autoMigrateEntry[Details](ctx, old)"))
			})

			It("Should omit defaulted type params from the function signature", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = string
					Mode enum {a = "a"  b = "b"}
					Entry struct<Details?, M extends Mode = Mode> {
						key Key {@key}
						name string
						mode M
						details Details?
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = string
					Mode enum {a = "a"  b = "b"}
					Entry struct<Details?, M extends Mode = Mode> {
						key Key {@key}
						name string
						description string
						mode M
						details Details?
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				autoCopy := fileContent(resp, "migrate_auto.gen.go")
				Expect(autoCopy).To(ContainSubstring("func autoMigrateEntry[Details any]"))
				Expect(autoCopy).NotTo(ContainSubstring("[Details any, M"))
				Expect(autoCopy).NotTo(ContainSubstring("[Details, M]"))
			})

			It("Should substitute defaulted type params in field conversions", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = string
					Mode enum {a = "a"  b = "b"}
					Entry struct<Details?, M extends Mode = Mode> {
						key Key {@key}
						mode M
						details Details?
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = string
					Mode enum {a = "a"  b = "b"}
					Entry struct<Details?, M extends Mode = Mode> {
						key Key {@key}
						name string
						mode M
						details Details?
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				autoCopy := fileContent(resp, "migrate_auto.gen.go")
				Expect(autoCopy).To(ContainSubstring("Mode: old.Mode"))
				Expect(autoCopy).NotTo(ContainSubstring("Mode(old.Mode)"))
			})
		})

		Context("optional fields", func() {
			It("Should generate nil-check preamble for optional", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Inner struct { value int32 }
					Entry struct {
						key Key {@key}
						inner Inner?
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Inner struct { value int32  count int32 }
					Entry struct {
						key Key {@key}
						inner Inner?
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("if old.Inner != nil"))
				Expect(content).To(ContainSubstring("inner = &v"))
			})
		})

		Context("reserved word field names", func() {
			It("Should escape Go reserved words in preamble variable names", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					TypeInfo struct { kind int32 }
					Entry struct {
						key Key  {@key}
						type TypeInfo
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					TypeInfo struct { kind int32  extra string }
					Entry struct {
						key Key  {@key}
						type TypeInfo
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("typeVal, err :="))
				Expect(content).To(ContainSubstring("Type: typeVal"))
				Expect(content).NotTo(MatchRegexp(`(?m)^\ttype\b`))
			})
		})

		Context("slice fields", func() {
			It("Should generate loop preamble for arrays of Oracle element types", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Item struct { name string }
					Entry struct {
						key Key    {@key}
						items Item[]
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Item struct { name string  count int32 }
					Entry struct {
						key Key    {@key}
						items Item[]
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("lo.MapErr"))
				Expect(content).To(ContainSubstring("autoMigrateItem"))
				Expect(content).To(ContainSubstring("err"))
			})
		})

		Context("named slice types", func() {
			It("Should generate slice function for alias array types", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Item struct { name string }
					Items Item[]
					Entry struct {
						key Key    {@key}
						items Items
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Item struct { name string  priority int32 }
					Items Item[]
					Entry struct {
						key Key    {@key}
						items Items
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("func autoMigrateItems"))
				Expect(content).To(ContainSubstring("MigrateItem(ctx, v)"))
			})
		})

		Context("cast-only structs", func() {
			It("Should use type casts for unchanged structs with only builtins", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Pos struct { x float64  y float64 }
					Entry struct {
						key Key {@key}
						pos Pos
						name string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Pos struct { x float64  y float64 }
					Entry struct {
						key Key {@key}
						pos Pos
						name string
						age int32
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				Expect(fileContent(resp, "migrate_auto.gen.go")).
					NotTo(ContainSubstring("autoMigratePos"))
			})
		})

		Context("enum fields", func() {
			It("Should assign unchanged enum fields directly", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Mode enum { text = "text"  graph = "graph" }
					Entry struct {
						key Key  {@key}
						mode Mode
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Mode enum { text = "text"  graph = "graph" }
					Entry struct {
						key Key  {@key}
						mode Mode
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("Mode: old.Mode"))
				Expect(content).NotTo(ContainSubstring("autoMigrateMode"))
			})
		})

		Context("unchanged dependency", func() {
			It("Should copy the unchanged dep field without importing it", func() {
				loader.Add("schemas/dep", `
					@go output "dep"
					@go version 1
					Inner struct { value int32 }
				`)
				oldSchema := `
					import "schemas/dep"
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key        {@key}
						inner dep.Inner
						@go migrate
					}
				`
				newSchema := `
					import "schemas/dep"
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key        {@key}
						inner dep.Inner
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				for _, path := range filePaths(resp) {
					Expect(path).To(HavePrefix("out/types/v2/"))
				}
				content := fileContent(resp, "out/types/v2/migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("Inner: old.Inner"))
				Expect(content).NotTo(ContainSubstring("dep/types/v1"))
			})
		})

		Context("version-laid-out dependency", func() {
			It("Should copy the unchanged dep field without importing it", func() {
				loader.Add("schemas/dep", `
					@go output "dep"
					@go version 1
					DepKey = uuid
					Inner struct {
						key DepKey {@key}
						value int32
						@go marshal
					}
				`)
				oldSchema := `
					import "schemas/dep"
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key        {@key}
						inner dep.Inner
						@go migrate
					}
				`
				newSchema := `
					import "schemas/dep"
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key        {@key}
						inner dep.Inner
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				for _, path := range filePaths(resp) {
					Expect(path).To(HavePrefix("out/types/v2/"))
				}
				content := fileContent(resp, "out/types/v2/migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("Inner: old.Inner"))
				Expect(content).NotTo(ContainSubstring("dep/types/v1"))
			})
		})

		Context("helpers stay with the definer", func() {
			It("Should leave helpers.go in the outgoing version package", func() {
				tmpDir := GinkgoT().TempDir()
				v1Dir := tmpDir + "/out/types/v1"
				Expect(os.MkdirAll(v1Dir, 0755)).To(Succeed())
				helpers := "package v1\n\nfunc Helper() string { return \"h\" }\n"
				Expect(os.WriteFile(v1Dir+"/helpers.go", []byte(helpers), 0644)).
					To(Succeed())

				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						age int32
						@go migrate
					}
				`
				customLoader := testutil.NewMockFileLoaderWithRoot(tmpDir)
				resp := MustSucceed(generate(
					ctx, oldSchema, newSchema, "test", customLoader, p,
				))
				Expect(filePaths(resp)).NotTo(ContainElement("out/types/v2/helpers.go"))
				Expect(resp.Deletions).To(BeEmpty())
			})
		})

		Context("struct extension", func() {
			It("Should handle embedded struct migration", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Base struct { name string }
					Entry struct extends Base {
						key Key {@key}
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Base struct { name string  label string }
					Entry struct extends Base {
						key Key {@key}
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("autoMigrateEntry"))
				Expect(content).To(ContainSubstring("autoMigrateBase"))
			})
		})

		Context("unused context parameter", func() {
			It("Should use _ for context when not needed", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key   {@key}
						name string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key   {@key}
						name string
						age int32
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("_ context.Context"))
				Expect(content).To(ContainSubstring(`"context"`))
			})
		})

		Context("recursive types", func() {
			It("Should handle self-referential types without infinite loop", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Node struct {
						key Key       {@key}
						value int32
						child Node?
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Node struct {
						key Key       {@key}
						value int32
						child Node?
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("autoMigrateNode"))
				Expect(content).To(ContainSubstring("if old.Child != nil"))
			})
		})

		Context("field type changed", func() {
			It("Should detect when a field type changes", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key    {@key}
						value int32
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key    {@key}
						value float64
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				for _, path := range filePaths(resp) {
					Expect(path).To(HavePrefix("out/types/v2/"))
				}
				Expect(fileContent(resp, "migrate_auto.gen.go")).
					To(ContainSubstring("autoMigrateEntry"))
			})
		})

		Context("optionality changed", func() {
			It("Should detect when field becomes optional", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key    {@key}
						name string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key    {@key}
						name string?
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				Expect(fileContent(resp, "migrate_auto.gen.go")).NotTo(BeEmpty())
			})
		})

		Context("map fields", func() {
			It("Should handle map fields as direct copies", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key              {@key}
						tags map<string, string>
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key              {@key}
						tags map<string, string>
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				Expect(fileContent(resp, "migrate_auto.gen.go")).
					To(ContainSubstring("old.Tags"))
			})
		})

		Context("map with Oracle-defined values", func() {
			It("Should handle map fields with Oracle value types", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key                    {@key}
						channels map<uint32, string>
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key                    {@key}
						channels map<uint32, string>
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				Expect(fileContent(resp, "migrate_auto.gen.go")).
					To(ContainSubstring("old.Channels"))
			})
		})

		Context("distinct type fields", func() {
			It("Should use type cast for distinct types wrapping primitives", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Color string
					Entry struct {
						key Key    {@key}
						color Color
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Color string
					Entry struct {
						key Key    {@key}
						color Color
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("Color: old.Color"))
				Expect(content).NotTo(ContainSubstring("autoMigrateColor"))
			})
		})

		Context("multiple entry types", func() {
			It("Should generate a single migrate.go for multiple @go migrate types", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					KeyA = uuid
					EntryA struct {
						key KeyA {@key}
						name string
						@go migrate
					}
					KeyB = uuid
					EntryB struct {
						key KeyB {@key}
						value int32
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					KeyA = uuid
					EntryA struct {
						key KeyA {@key}
						name string
						age int32
						@go migrate
					}
					KeyB = uuid
					EntryB struct {
						key KeyB {@key}
						value int32
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				Expect(fileContent(resp, "migrate.gen.go")).To(BeEmpty())
				Expect(fileContent(resp, "out/types/v2/migrate_auto.gen.go")).
					NotTo(BeEmpty())
				tmpl := fileContent(resp, "out/types/v2/migrate.go")
				Expect(tmpl).To(ContainSubstring("func migrateEntryA"))
				Expect(tmpl).To(ContainSubstring("func migrateEntryB"))
				Expect(strings.Count(tmpl, "package v2")).To(Equal(1))
				Expect(strings.Count(tmpl, "Edit this file")).To(Equal(1))
			})
		})

		Context("cross-package unchanged types", func() {
			It("Should copy unchanged external Oracle types directly", func() {
				loader.Add("schemas/dep", `
					@go output "dep"
					@go version 1
					Color string
					Label struct {
						name string
						color Color
					}
				`)
				oldSchema := `
					import "schemas/dep"
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key           {@key}
						label dep.Label
						@go migrate
					}
				`
				newSchema := `
					import "schemas/dep"
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key           {@key}
						label dep.Label
						extra string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				content := fileContent(resp, "out/types/v2/migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("autoMigrateEntry"))
				Expect(content).To(ContainSubstring("Label: old.Label"))
				Expect(content).NotTo(ContainSubstring("autoMigrateLabel"))
				Expect(content).NotTo(ContainSubstring("dep/types"))
			})
		})

		Context("deeply nested changes", func() {
			It("Should propagate changes through multiple nesting levels", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Leaf struct { value int32 }
					Branch struct { leaf Leaf }
					Entry struct {
						key Key       {@key}
						branch Branch
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Leaf struct { value int32  extra string }
					Branch struct { leaf Leaf }
					Entry struct {
						key Key       {@key}
						branch Branch
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("autoMigrateEntry"))
				Expect(content).To(ContainSubstring("autoMigrateBranch"))
				Expect(content).To(ContainSubstring("autoMigrateLeaf"))
			})
		})

		Context("wrapper visibility", func() {
			It("Should unexport entry and locally consumed wrappers, keeping externally referenced ones exported", func() {
				loader.Add("schemas/dep", `
					@go output "dep"
					@go version 1
					Item struct { name string }
				`)
				oldSchema := `
					import "schemas/dep"
					@go output "out"
					@go version 1
					Key = uuid
					Inner struct { value int32 }
					Entry struct {
						key Key {@key}
						inner Inner
						item dep.Item
						@go migrate
					}
				`
				newSchema := `
					import "schemas/dep"
					@go output "out"
					@go version 2
					Key = uuid
					Inner struct { value int32  extra string }
					Entry struct {
						key Key {@key}
						inner Inner
						item dep.Item
						name string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				tmpl := fileContent(resp, "out/types/v2/migrate.go")
				Expect(tmpl).To(ContainSubstring("func migrateEntry"))
				Expect(tmpl).To(ContainSubstring("func migrateInner"))
				Expect(tmpl).NotTo(ContainSubstring("func MigrateEntry"))
				auto := fileContent(resp, "out/types/v2/migrate_auto.gen.go")
				Expect(auto).To(ContainSubstring("migrateInner(ctx,"))
			})
		})

		Context("struct with no @go migrate", func() {
			It("Should scaffold dep-style without a frozen re-emission", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go marshal
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						age int32
						@go marshal
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				for _, path := range filePaths(resp) {
					Expect(path).To(HavePrefix("out/types/v2/"))
				}
				auto := fileContent(resp, "out/types/v2/migrate_auto.gen.go")
				Expect(auto).To(ContainSubstring("package v2"))
				Expect(auto).To(ContainSubstring("autoMigrateEntry"))
				tmpl := fileContent(resp, "out/types/v2/migrate.go")
				Expect(tmpl).To(ContainSubstring("func MigrateEntry"))
			})
		})

		Context("struct without key field", func() {
			It("Should scaffold value-type paths standalone", func() {
				oldSchema := `
					@go output "out"
					@go version 1
					Entry struct {
						name string
					}
				`
				newSchema := `
					@go output "out"
					@go version 2
					Entry struct {
						name string
						age int32
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
				for _, path := range filePaths(resp) {
					Expect(path).To(HavePrefix("out/types/v2/"))
				}
				Expect(fileContent(resp, "out/types/v2/migrate_auto.gen.go")).
					To(ContainSubstring("autoMigrateEntry"))
			})
		})

		Context("regression: non-optional preamble should not add unused import", func() {
			It("Should not import the live package for a changed external field", func() {
				oldIR := `
					@go output "ir"
					@go version 1
					Leaf struct { value int32 }
					IR struct { leaf Leaf }
				`
				newIR := `
					@go output "ir"
					@go version 2
					Leaf struct { value int32  extra string }
					IR struct { leaf Leaf }
				`
				loader.Add("schemas/ir", oldIR)
				oldSchema := `
					import "schemas/ir"
					@go output "out"
					@go version 1
					Key = uuid
					Entry struct {
						key Key  {@key}
						ir ir.IR
						@go migrate
					}
				`
				newSchema := `
					import "schemas/ir"
					@go output "out"
					@go version 2
					Key = uuid
					Entry struct {
						key Key  {@key}
						ir ir.IR
						label string
						@go migrate
					}
				`
				oldTable := MustSucceed(analyze(ctx, oldSchema, "test", loader))
				loader.Add("schemas/ir", newIR)
				newTable := MustSucceed(analyze(ctx, newSchema, "test", loader))
				req := &plugin.Request{
					Resolutions:    newTable,
					OldResolutions: oldTable,
					RepoRoot:       loader.RepoRoot(),
				}
				resp := MustSucceed(p.Generate(req))
				content := fileContent(resp, "out/types/v2/migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("autoMigrateEntry"))
				Expect(content).NotTo(ContainSubstring(`"github.com/synnaxlabs/synnax/ir"`))
			})
		})
	})

	Describe("Auto-copy signature", func() {
		It("Should generate auto-copy functions with context and error propagation", func() {
			oldSchema := `
				@go output "out"
				@go version 1
				Key = uuid
				Inner struct { value int32 }
				Entry struct {
					key Key     {@key}
					inner Inner
					@go migrate
				}
			`
			newSchema := `
				@go output "out"
				@go version 2
				Key = uuid
				Inner struct { value int32  extra string }
				Entry struct {
					key Key     {@key}
					inner Inner
					@go migrate
				}
			`
			resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
			content := fileContent(resp, "migrate_auto.gen.go")
			Expect(content).To(ContainSubstring("context.Context"))
			Expect(content).To(ContainSubstring("error"))
			Expect(content).To(ContainSubstring("if err != nil"))
		})
	})

	Describe("Changed value-type dependency", func() {
		var resp *plugin.Response

		BeforeEach(func() {
			loader.Add("schemas/dep", `
				@go output "dep"
				@go version 1
				Item struct { name string }
			`)
			oldSchema := `
				import "schemas/dep"
				@go output "out"
				@go version 1
				Key = uuid
				Entry struct {
					key Key       {@key}
					item dep.Item
					@go migrate
				}
			`
			newSchema := `
				import "schemas/dep"
				@go output "out"
				@go version 2
				Key = uuid
				Entry struct {
					key Key       {@key}
					item dep.Item
					label string
					@go migrate
				}
			`
			oldTable := MustSucceed(analyze(ctx, oldSchema, "test", loader))
			loader.Add("schemas/dep", `
				@go output "dep"
				@go version 2
				Item struct { name string  priority int32 }
			`)
			newTable := MustSucceed(analyze(ctx, newSchema, "test", loader))
			req := &plugin.Request{
				Resolutions:    newTable,
				OldResolutions: oldTable,
				RepoRoot:       loader.RepoRoot(),
			}
			resp = MustSucceed(p.Generate(req))
		})

		It("Should scaffold the dependency's own incoming version", func() {
			tmpl := fileContent(resp, "dep/types/v2/migrate.go")
			Expect(tmpl).To(ContainSubstring("package v2"))
			Expect(tmpl).To(ContainSubstring("func MigrateItem"))
			Expect(tmpl).To(ContainSubstring("autoMigrateItem"))
			Expect(fileContent(resp, "dep/types/v2/migrate_auto.gen.go")).
				NotTo(BeEmpty())
		})

		It("Should call the dependency's MigrateX at its incoming version", func() {
			content := fileContent(resp, "out/types/v2/migrate_auto.gen.go")
			Expect(content).To(ContainSubstring(".MigrateItem"))
			Expect(content).To(ContainSubstring("dep/types/v2"))
			Expect(content).NotTo(ContainSubstring(".autoMigrateItem"))
		})
	})

	Describe("alias-to-slice field migration", func() {
		It("Should migrate alias-to-slice elements instead of casting", func() {
			oldSchema := `
				@go output "out"
				@go version 1
				Key = uuid
				Member struct {
					node string
				}
				Members = Member[]
				Scope struct {
					key Key {@key}
					strata Members[]
					@go migrate
				}
			`
			newSchema := `
				@go output "out"
				@go version 2
				Key = uuid
				Member struct {
					node string
					weight int32
				}
				Members = Member[]
				Scope struct {
					key Key {@key}
					strata Members[]
					@go migrate
				}
			`
			resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
			autoCopy := fileContent(resp, "migrate_auto.gen.go")
			Expect(autoCopy).To(ContainSubstring("autoMigrateMembers"))
			Expect(autoCopy).To(ContainSubstring("MigrateMember(ctx, v)"))
		})
	})

	Describe("context.Context in generated code", func() {
		It("Should use context.Context in developer transform template", func() {
			oldSchema := `
				@go output "out"
				@go version 1
				Key = uuid
				Entry struct {
					key Key {@key}
					name string
					@go migrate
				}
			`
			newSchema := `
				@go output "out"
				@go version 2
				Key = uuid
				Entry struct {
					key Key {@key}
					name string
					age int32
					@go migrate
				}
			`
			resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
			content := fileContent(resp, "out/types/v2/migrate.go")
			Expect(content).To(ContainSubstring("context.Context"))
			Expect(content).NotTo(ContainSubstring("MigrationContext"))
			Expect(content).To(ContainSubstring(`"context"`))
		})

		It("Should pass ctx to nested autoMigrate calls", func() {
			oldSchema := `
				@go output "out"
				@go version 1
				Key = uuid
				Inner struct { value int32 }
				Entry struct {
					key Key     {@key}
					inner Inner
					@go migrate
				}
			`
			newSchema := `
				@go output "out"
				@go version 2
				Key = uuid
				Inner struct { value int32  extra string }
				Entry struct {
					key Key     {@key}
					inner Inner
					@go migrate
				}
			`
			resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
			content := fileContent(resp, "migrate_auto.gen.go")
			Expect(content).To(ContainSubstring("= MigrateInner(ctx,"))
			Expect(content).To(ContainSubstring("ctx context.Context"))
		})

		It("Should not reference gorp in auto-copy imports", func() {
			oldSchema := `
				@go output "out"
				@go version 1
				Key = uuid
				Entry struct {
					key Key {@key}
					name string
					@go migrate
				}
			`
			newSchema := `
				@go output "out"
				@go version 2
				Key = uuid
				Entry struct {
					key Key {@key}
					name string
					age int32
					@go migrate
				}
			`
			resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p))
			content := fileContent(resp, "migrate_auto.gen.go")
			Expect(content).NotTo(ContainSubstring(`"github.com/synnaxlabs/x/gorp"`))
			Expect(content).To(ContainSubstring(`"context"`))
		})
	})

	Describe("schemadiff.SchemaDiff", func() {
		It("Should detect schemadiff.TypeChanged for added field", func() {
			oldTable := MustSucceed(analyze(ctx, `@go output "out"
				Entry struct { name string }`, "test", loader))
			newTable := MustSucceed(analyze(ctx, `@go output "out"
				Entry struct { name string  age int32 }`, "test", loader))
			oldEntry := MustBeOk(oldTable.Get("test.Entry"))
			newEntry := MustBeOk(newTable.Get("test.Entry"))
			diff := schemadiff.SchemaDiff(oldEntry, newEntry, oldTable, newTable)
			Expect(diff).To(HaveKey("test.Entry"))
			Expect(diff["test.Entry"].Kind).To(Equal(schemadiff.TypeChanged))
		})

		It("Should detect schemadiff.TypeDescendantChanged for nested type change", func() {
			oldTable := MustSucceed(analyze(ctx, `@go output "out"
				Inner struct { value int32 }
				Outer struct { inner Inner }`, "test", loader))
			newTable := MustSucceed(analyze(ctx, `@go output "out"
				Inner struct { value int32  extra string }
				Outer struct { inner Inner }`, "test", loader))
			oldEntry := MustBeOk(oldTable.Get("test.Outer"))
			newEntry := MustBeOk(newTable.Get("test.Outer"))
			diff := schemadiff.SchemaDiff(oldEntry, newEntry, oldTable, newTable)
			Expect(diff["test.Inner"].Kind).To(Equal(schemadiff.TypeChanged))
			Expect(diff["test.Outer"].Kind).To(Equal(schemadiff.TypeDescendantChanged))
		})

		It("Should return empty diff for identical schemas", func() {
			schema := `@go output "out"
				Entry struct { name string  age int32 }`
			oldTable := MustSucceed(analyze(ctx, schema, "test", loader))
			newTable := MustSucceed(analyze(ctx, schema, "test", loader))
			oldEntry := MustBeOk(oldTable.Get("test.Entry"))
			newEntry := MustBeOk(newTable.Get("test.Entry"))
			Expect(schemadiff.SchemaDiff(oldEntry, newEntry, oldTable, newTable)).To(BeEmpty())
		})

		It("Should detect field removal", func() {
			oldTable := MustSucceed(analyze(ctx, `@go output "out"
				Entry struct { name string  age int32 }`, "test", loader))
			newTable := MustSucceed(analyze(ctx, `@go output "out"
				Entry struct { name string }`, "test", loader))
			oldEntry := MustBeOk(oldTable.Get("test.Entry"))
			newEntry := MustBeOk(newTable.Get("test.Entry"))
			diff := schemadiff.SchemaDiff(oldEntry, newEntry, oldTable, newTable)
			Expect(diff["test.Entry"].Kind).To(Equal(schemadiff.TypeChanged))
			hasRemoved := false
			for _, fd := range diff["test.Entry"].ChangedFields {
				if fd.Kind == schemadiff.FieldKindRemoved {
					hasRemoved = true
				}
			}
			Expect(hasRemoved).To(BeTrue())
		})

		It("Should detect field type change", func() {
			oldTable := MustSucceed(analyze(ctx, `@go output "out"
				Entry struct { value int32 }`, "test", loader))
			newTable := MustSucceed(analyze(ctx, `@go output "out"
				Entry struct { value float64 }`, "test", loader))
			oldEntry := MustBeOk(oldTable.Get("test.Entry"))
			newEntry := MustBeOk(newTable.Get("test.Entry"))
			diff := schemadiff.SchemaDiff(oldEntry, newEntry, oldTable, newTable)
			Expect(diff["test.Entry"].Kind).To(Equal(schemadiff.TypeChanged))
			hasTypeChanged := false
			for _, fd := range diff["test.Entry"].ChangedFields {
				if fd.Kind == schemadiff.FieldKindTypeChanged {
					hasTypeChanged = true
				}
			}
			Expect(hasTypeChanged).To(BeTrue())
		})

		It("Should handle recursive types without infinite loop", func() {
			oldTable := MustSucceed(analyze(ctx, `@go output "out"
				Node struct { value int32  child Node? }`, "test", loader))
			newTable := MustSucceed(analyze(ctx, `@go output "out"
				Node struct { value int32  child Node?  label string }`, "test", loader))
			oldEntry := MustBeOk(oldTable.Get("test.Node"))
			newEntry := MustBeOk(newTable.Get("test.Node"))
			diff := schemadiff.SchemaDiff(oldEntry, newEntry, oldTable, newTable)
			Expect(diff["test.Node"].Kind).To(Equal(schemadiff.TypeChanged))
		})

		It("Should propagate changes through alias types", func() {
			oldTable := MustSucceed(analyze(ctx, `@go output "out"
				Item struct { name string }
				Items Item[]
				Container struct { items Items }`, "test", loader))
			newTable := MustSucceed(analyze(ctx, `@go output "out"
				Item struct { name string  priority int32 }
				Items Item[]
				Container struct { items Items }`, "test", loader))
			oldEntry := MustBeOk(oldTable.Get("test.Container"))
			newEntry := MustBeOk(newTable.Get("test.Container"))
			diff := schemadiff.SchemaDiff(oldEntry, newEntry, oldTable, newTable)
			Expect(diff["test.Item"].Kind).To(Equal(schemadiff.TypeChanged))
			Expect(diff["test.Items"].Kind).To(Equal(schemadiff.TypeDescendantChanged))
			Expect(diff["test.Container"].Kind).To(Equal(schemadiff.TypeDescendantChanged))
		})
	})
})
