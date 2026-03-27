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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
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
	version int,
) (*plugin.Response, error) {
	newTable, err := analyze(ctx, newSource, namespace, loader)
	if err != nil {
		return nil, err
	}
	req := &plugin.Request{
		Resolutions:     newTable,
		SnapshotVersion: version,
		RepoRoot:        loader.RepoRoot(),
	}
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
			It("Should generate only migrate.gen.go when schemas are identical", func() {
				schema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, schema, schema, "test", loader, p, 1))
				Expect(fileContent(resp, "migrate.gen.go")).NotTo(BeEmpty())
				Expect(fileContent(resp, "migrate_auto.gen.go")).To(BeEmpty())
				Expect(fileContent(resp, "migrations/v1/types.gen.go")).To(BeEmpty())
			})
		})

		Context("no old resolutions", func() {
			It("Should generate only migrate.gen.go with no migration chain", func() {
				schema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, "", schema, "test", loader, p, 1))
				content := fileContent(resp, "migrate.gen.go")
				Expect(content).To(ContainSubstring("package out"))
				Expect(content).To(ContainSubstring("EntryMigrations"))
				Expect(content).NotTo(ContainSubstring("v1"))
			})
		})

		Context("field addition", func() {
			var resp *plugin.Response

			BeforeEach(func() {
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
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						age int32
						@go migrate
					}
				`
				resp = MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
			})

			It("Should generate frozen types", func() {
				content := fileContent(resp, "migrations/v1/types.gen.go")
				Expect(content).To(ContainSubstring("package v1"))
				Expect(content).To(ContainSubstring("type Entry struct"))
				Expect(content).To(ContainSubstring("Name string"))
				Expect(content).NotTo(ContainSubstring("Age"))
			})

			It("Should generate frozen codec", func() {
				content := fileContent(resp, "migrations/v1/codec.gen.go")
				Expect(content).To(ContainSubstring("package v1"))
				Expect(content).To(ContainSubstring("EntryCodec"))
			})

			It("Should generate auto-copy with error propagation", func() {
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("func AutoMigrateEntry"))
				Expect(content).NotTo(ContainSubstring("var _ = context.Background"))
				Expect(content).NotTo(ContainSubstring(", _ :="))
				Expect(content).NotTo(ContainSubstring(", _ ="))
			})

			It("Should generate migration registration", func() {
				content := fileContent(resp, "migrate.gen.go")
				Expect(content).To(ContainSubstring("EntryMigrations"))
				Expect(content).To(ContainSubstring("v1_schema_migration"))
				Expect(content).To(ContainSubstring("MigrateEntry"))
			})

			It("Should generate developer transform template", func() {
				content := fileContent(resp, "out/migrate.go")
				Expect(content).To(ContainSubstring("func MigrateEntry"))
				Expect(content).To(ContainSubstring("AutoMigrateEntry"))
				Expect(content).To(ContainSubstring("Edit this file"))
			})
		})

		Context("field removal", func() {
			It("Should generate migration for removed field", func() {
				oldSchema := `
					@go output "out"
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
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				Expect(fileContent(resp, "migrations/v1/types.gen.go")).To(ContainSubstring("Age int32"))
				autoCopy := fileContent(resp, "migrate_auto.gen.go")
				Expect(autoCopy).To(ContainSubstring("AutoMigrateEntry"))
				Expect(autoCopy).NotTo(ContainSubstring("Age"))
			})
		})

		Context("optional fields", func() {
			It("Should generate nil-check preamble for hard optional", func() {
				oldSchema := `
					@go output "out"
					Key = uuid
					Inner struct { value int32 }
					Entry struct {
						key Key {@key}
						inner Inner??
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					Key = uuid
					Inner struct { value int32  count int32 }
					Entry struct {
						key Key {@key}
						inner Inner??
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("if old.Inner != nil"))
				Expect(content).To(ContainSubstring("inner = &v"))
			})
		})

		Context("reserved word field names", func() {
			It("Should escape Go reserved words in preamble variable names", func() {
				oldSchema := `
					@go output "out"
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
					Key = uuid
					TypeInfo struct { kind int32  extra string }
					Entry struct {
						key Key  {@key}
						type TypeInfo
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("typeVal"))
				Expect(content).NotTo(MatchRegexp(`[^a-zA-Z]type[^VN]`))
			})
		})

		Context("slice fields", func() {
			It("Should generate loop preamble for array fields with Oracle element types", func() {
				oldSchema := `
					@go output "out"
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
					Key = uuid
					Item struct { name string  count int32 }
					Entry struct {
						key Key    {@key}
						items Item[]
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("for i, v := range"))
				Expect(content).To(ContainSubstring("AutoMigrateItem"))
				Expect(content).To(ContainSubstring("err"))
			})
		})

		Context("named slice types", func() {
			It("Should generate slice function for alias array types", func() {
				oldSchema := `
					@go output "out"
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
					Key = uuid
					Item struct { name string  priority int32 }
					Items Item[]
					Entry struct {
						key Key    {@key}
						items Items
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("func AutoMigrateItems"))
				Expect(content).To(ContainSubstring("AutoMigrateItem(ctx, v)"))
			})
		})

		Context("cast-only structs", func() {
			It("Should use type casts for unchanged structs with only builtins", func() {
				oldSchema := `
					@go output "out"
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
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				Expect(fileContent(resp, "migrate_auto.gen.go")).NotTo(ContainSubstring("AutoMigratePos"))
			})
		})

		Context("enum fields", func() {
			It("Should use type cast for enum fields", func() {
				oldSchema := `
					@go output "out"
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
					Key = uuid
					Mode enum { text = "text"  graph = "graph" }
					Entry struct {
						key Key  {@key}
						mode Mode
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("Mode(old.Mode)"))
				Expect(content).NotTo(ContainSubstring("AutoMigrateMode"))
			})
		})

		Context("cross-package references", func() {
			It("Should generate frozen types in each dependency package", func() {
				loader.Add("schemas/dep", `
					@go output "dep"
					Inner struct { value int32 }
				`)
				oldSchema := `
					import "schemas/dep"
					@go output "out"
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
					Key = uuid
					Entry struct {
						key Key        {@key}
						inner dep.Inner
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				paths := filePaths(resp)
				Expect(paths).To(ContainElement(ContainSubstring("dep/migrations/v1/types.gen.go")))
				Expect(paths).To(ContainElement(ContainSubstring("out/migrations/v1/types.gen.go")))
			})
		})

		Context("struct extension", func() {
			It("Should handle embedded struct migration", func() {
				oldSchema := `
					@go output "out"
					Key = uuid
					Base struct { name string }
					Entry struct extends Base {
						key Key {@key}
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					Key = uuid
					Base struct { name string  label string }
					Entry struct extends Base {
						key Key {@key}
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("AutoMigrateEntry"))
				Expect(content).To(ContainSubstring("AutoMigrateBase"))
			})
		})

		Context("unused context parameter", func() {
			It("Should use _ for context when not needed", func() {
				oldSchema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key   {@key}
						name string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key   {@key}
						name string
						age int32
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				Expect(fileContent(resp, "migrate_auto.gen.go")).To(ContainSubstring("_ context.Context"))
			})
		})

		Context("recursive types", func() {
			It("Should handle self-referential types without infinite loop", func() {
				oldSchema := `
					@go output "out"
					Key = uuid
					Node struct {
						key Key       {@key}
						value int32
						child Node??
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					Key = uuid
					Node struct {
						key Key       {@key}
						value int32
						child Node??
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("AutoMigrateNode"))
				Expect(content).To(ContainSubstring("if old.Child != nil"))
			})
		})

		Context("field type changed", func() {
			It("Should detect when a field type changes", func() {
				oldSchema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key    {@key}
						value int32
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key    {@key}
						value float64
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				Expect(fileContent(resp, "migrations/v1/types.gen.go")).To(ContainSubstring("int32"))
				Expect(fileContent(resp, "migrate_auto.gen.go")).To(ContainSubstring("AutoMigrateEntry"))
			})
		})

		Context("optionality changed", func() {
			It("Should detect when field becomes optional", func() {
				oldSchema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key    {@key}
						name string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key    {@key}
						name string?
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				Expect(fileContent(resp, "migrate_auto.gen.go")).NotTo(BeEmpty())
			})
		})

		Context("map fields", func() {
			It("Should handle map fields as direct copies", func() {
				oldSchema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key              {@key}
						tags map<string, string>
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key              {@key}
						tags map<string, string>
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				Expect(fileContent(resp, "migrate_auto.gen.go")).To(ContainSubstring("old.Tags"))
			})
		})

		Context("map with Oracle-defined values", func() {
			It("Should handle map fields with Oracle value types", func() {
				oldSchema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key                    {@key}
						channels map<uint32, string>
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key                    {@key}
						channels map<uint32, string>
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				Expect(fileContent(resp, "migrate_auto.gen.go")).To(ContainSubstring("old.Channels"))
			})
		})

		Context("distinct type fields", func() {
			It("Should use type cast for distinct types wrapping primitives", func() {
				oldSchema := `
					@go output "out"
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
					Key = uuid
					Color string
					Entry struct {
						key Key    {@key}
						color Color
						label string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("Color(old.Color)"))
				Expect(content).NotTo(ContainSubstring("AutoMigrateColor"))
			})
		})

		Context("multiple entry types", func() {
			It("Should generate migrations for multiple @go migrate types in same package", func() {
				oldSchema := `
					@go output "out"
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
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				regContent := fileContent(resp, "migrate.gen.go")
				Expect(regContent).To(ContainSubstring("EntryAMigrations"))
				Expect(regContent).To(ContainSubstring("EntryBMigrations"))
			})
		})

		Context("cross-package unchanged types", func() {
			It("Should generate local helpers for unchanged external Oracle types", func() {
				loader.Add("schemas/dep", `
					@go output "dep"
					Color string
					Label struct {
						name string
						color Color
					}
				`)
				oldSchema := `
					import "schemas/dep"
					@go output "out"
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
					Key = uuid
					Entry struct {
						key Key           {@key}
						label dep.Label
						extra string
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("AutoMigrateLabel"))
				Expect(content).To(ContainSubstring("AutoMigrateEntry"))
			})
		})

		Context("deeply nested changes", func() {
			It("Should propagate changes through multiple nesting levels", func() {
				oldSchema := `
					@go output "out"
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
					Key = uuid
					Leaf struct { value int32  extra string }
					Branch struct { leaf Leaf }
					Entry struct {
						key Key       {@key}
						branch Branch
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				content := fileContent(resp, "migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("AutoMigrateEntry"))
				Expect(content).To(ContainSubstring("AutoMigrateBranch"))
				Expect(content).To(ContainSubstring("AutoMigrateLeaf"))
			})
		})

		Context("struct with no @go migrate", func() {
			It("Should skip structs without migrate annotation", func() {
				oldSchema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
					}
				`
				newSchema := `
					@go output "out"
					Key = uuid
					Entry struct {
						key Key {@key}
						name string
						age int32
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				Expect(resp.Files).To(BeEmpty())
			})
		})

		Context("struct without key field", func() {
			It("Should skip structs without @key annotation", func() {
				oldSchema := `
					@go output "out"
					Entry struct {
						name string
						@go migrate
					}
				`
				newSchema := `
					@go output "out"
					Entry struct {
						name string
						age int32
						@go migrate
					}
				`
				resp := MustSucceed(generate(ctx, oldSchema, newSchema, "test", loader, p, 1))
				Expect(resp.Files).To(BeEmpty())
			})
		})

		Context("regression: non-optional preamble should not add unused import", func() {
			It("Should not import the live package for a non-optional changed external field", func() {
				loader.Add("schemas/ir", `
					@go output "ir"
					Leaf struct { value int32 }
					IR struct { leaf Leaf }
				`)
				oldIR := `
					@go output "ir"
					Leaf struct { value int32 }
					IR struct { leaf Leaf }
				`
				newIR := `
					@go output "ir"
					Leaf struct { value int32  extra string }
					IR struct { leaf Leaf }
				`
				loader.Add("schemas/ir", oldIR)
				oldSchema := `
					import "schemas/ir"
					@go output "out"
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
					Resolutions:     newTable,
					OldResolutions:  oldTable,
					SnapshotVersion: 1,
					RepoRoot:        loader.RepoRoot(),
				}
				resp := MustSucceed(p.Generate(req))
				content := fileContent(resp, "out/migrate_auto.gen.go")
				Expect(content).To(ContainSubstring("AutoMigrateEntry"))
				Expect(content).NotTo(ContainSubstring(`ir "github.com`))
			})
		})
	})

	Describe("SchemaDiff", func() {
		It("Should detect TypeChanged for added field", func() {
			oldTable := MustSucceed(analyze(ctx, `@go output "out"
				Entry struct { name string }`, "test", loader))
			newTable := MustSucceed(analyze(ctx, `@go output "out"
				Entry struct { name string  age int32 }`, "test", loader))
			oldEntry := MustBeOk(oldTable.Get("test.Entry"))
			newEntry := MustBeOk(newTable.Get("test.Entry"))
			diff := migrate.SchemaDiff(oldEntry, newEntry, oldTable, newTable)
			Expect(diff).To(HaveKey("test.Entry"))
			Expect(diff["test.Entry"].Kind).To(Equal(migrate.TypeChanged))
		})

		It("Should detect TypeDescendantChanged for nested type change", func() {
			oldTable := MustSucceed(analyze(ctx, `@go output "out"
				Inner struct { value int32 }
				Outer struct { inner Inner }`, "test", loader))
			newTable := MustSucceed(analyze(ctx, `@go output "out"
				Inner struct { value int32  extra string }
				Outer struct { inner Inner }`, "test", loader))
			oldEntry := MustBeOk(oldTable.Get("test.Outer"))
			newEntry := MustBeOk(newTable.Get("test.Outer"))
			diff := migrate.SchemaDiff(oldEntry, newEntry, oldTable, newTable)
			Expect(diff["test.Inner"].Kind).To(Equal(migrate.TypeChanged))
			Expect(diff["test.Outer"].Kind).To(Equal(migrate.TypeDescendantChanged))
		})

		It("Should return empty diff for identical schemas", func() {
			schema := `@go output "out"
				Entry struct { name string  age int32 }`
			oldTable := MustSucceed(analyze(ctx, schema, "test", loader))
			newTable := MustSucceed(analyze(ctx, schema, "test", loader))
			oldEntry := MustBeOk(oldTable.Get("test.Entry"))
			newEntry := MustBeOk(newTable.Get("test.Entry"))
			Expect(migrate.SchemaDiff(oldEntry, newEntry, oldTable, newTable)).To(BeEmpty())
		})

		It("Should detect field removal", func() {
			oldTable := MustSucceed(analyze(ctx, `@go output "out"
				Entry struct { name string  age int32 }`, "test", loader))
			newTable := MustSucceed(analyze(ctx, `@go output "out"
				Entry struct { name string }`, "test", loader))
			oldEntry := MustBeOk(oldTable.Get("test.Entry"))
			newEntry := MustBeOk(newTable.Get("test.Entry"))
			diff := migrate.SchemaDiff(oldEntry, newEntry, oldTable, newTable)
			Expect(diff["test.Entry"].Kind).To(Equal(migrate.TypeChanged))
			hasRemoved := false
			for _, fd := range diff["test.Entry"].ChangedFields {
				if fd.Kind == migrate.FieldKindRemoved {
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
			diff := migrate.SchemaDiff(oldEntry, newEntry, oldTable, newTable)
			Expect(diff["test.Entry"].Kind).To(Equal(migrate.TypeChanged))
			hasTypeChanged := false
			for _, fd := range diff["test.Entry"].ChangedFields {
				if fd.Kind == migrate.FieldKindTypeChanged {
					hasTypeChanged = true
				}
			}
			Expect(hasTypeChanged).To(BeTrue())
		})

		It("Should handle recursive types without infinite loop", func() {
			oldTable := MustSucceed(analyze(ctx, `@go output "out"
				Node struct { value int32  child Node?? }`, "test", loader))
			newTable := MustSucceed(analyze(ctx, `@go output "out"
				Node struct { value int32  child Node??  label string }`, "test", loader))
			oldEntry := MustBeOk(oldTable.Get("test.Node"))
			newEntry := MustBeOk(newTable.Get("test.Node"))
			Expect(migrate.SchemaDiff(oldEntry, newEntry, oldTable, newTable)["test.Node"].Kind).To(Equal(migrate.TypeChanged))
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
			diff := migrate.SchemaDiff(oldEntry, newEntry, oldTable, newTable)
			Expect(diff["test.Item"].Kind).To(Equal(migrate.TypeChanged))
			Expect(diff["test.Items"].Kind).To(Equal(migrate.TypeDescendantChanged))
			Expect(diff["test.Container"].Kind).To(Equal(migrate.TypeDescendantChanged))
		})
	})
})
