// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"context"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/pb/types"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/testutil"
)

func TestPBTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PB Types Plugin Suite")
}

// pbDomains creates domains that enable pb generation with the given go output path.
// The pb plugin derives its output from @go output + "/pb".
func pbDomains(goOutputPath string) map[string]resolution.Domain {
	return map[string]resolution.Domain{
		"pb": {Name: "pb"}, // Just having @pb is enough to enable
		"go": {
			Name: "go",
			Expressions: []resolution.Expression{
				{Name: "output", Values: []resolution.ExpressionValue{{StringValue: goOutputPath}}},
			},
		},
	}
}

var _ = Describe("PbFormatter", func() {
	var f *types.PbFormatter

	BeforeEach(func() {
		f = &types.PbFormatter{}
	})

	Describe("FormatQualified", func() {
		It("Should format qualified names with dot separator", func() {
			Expect(f.FormatQualified("pkg", "Type")).To(Equal("pkg.Type"))
		})

		It("Should return type name when qualifier is empty", func() {
			Expect(f.FormatQualified("", "Type")).To(Equal("Type"))
		})
	})

	Describe("FormatGeneric", func() {
		It("Should return base name unchanged (protobuf has no generics)", func() {
			Expect(f.FormatGeneric("Container", []string{"string"})).To(Equal("Container"))
		})
	})

	Describe("FormatArray", func() {
		It("Should return element type unchanged (repeated handled at field level)", func() {
			Expect(f.FormatArray("string")).To(Equal("string"))
		})
	})

	Describe("FormatMap", func() {
		It("Should format map types", func() {
			Expect(f.FormatMap("string", "int32")).To(Equal("map<string, int32>"))
		})
	})

	Describe("FallbackType", func() {
		It("Should return bytes as fallback", func() {
			Expect(f.FallbackType()).To(Equal("bytes"))
		})
	})
})

var _ = Describe("PbImportResolver", func() {
	var r *types.PbImportResolver

	BeforeEach(func() {
		r = &types.PbImportResolver{}
	})

	Describe("ResolveImport", func() {
		It("Should return proto path with qualifier", func() {
			importPath, qualifier, shouldImport := r.ResolveImport("core/pkg/task", nil)
			Expect(importPath).To(Equal("core/pkg/task/types.gen.proto"))
			Expect(qualifier).To(Equal("task"))
			Expect(shouldImport).To(BeTrue())
		})
	})
})

var _ = Describe("Plugin", func() {
	var (
		ctx    context.Context
		loader *testutil.MockFileLoader
		p      *types.Plugin
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = testutil.NewMockFileLoader()
		p = types.New(types.DefaultOptions())
	})

	Describe("Name", func() {
		It("Should return pb/types", func() {
			Expect(p.Name()).To(Equal("pb/types"))
		})
	})

	Describe("Check", func() {
		It("Should return nil (no validation required)", func() {
			Expect(p.Check(&plugin.Request{})).To(BeNil())
		})
	})

	Describe("Domains", func() {
		It("Should return pb domain", func() {
			Expect(p.Domains()).To(Equal([]string{"pb"}))
		})
	})

	Describe("Requires", func() {
		It("Should return nil", func() {
			Expect(p.Requires()).To(BeNil())
		})
	})

	Describe("Generate", func() {
		Context("primitive type mappings", func() {
			DescribeTable("should generate correct proto type",
				func(oracleType, expectedProtoType string) {
					source := `
						@go output "core/pkg/api/grpc/v1"
						@pb

						Test struct {
							field ` + oracleType + `
						}
					`
					resp := testutil.MustGenerate(ctx, source, "test", loader, p)
					testutil.ExpectContent(resp, "test.proto").ToContain(expectedProtoType + " field = 1;")
				},
				Entry("string", "string", "string"),
				Entry("bool", "bool", "bool"),
				Entry("int32", "int32", "int32"),
				Entry("int64", "int64", "int64"),
				Entry("uint32", "uint32", "uint32"),
				Entry("uint64", "uint64", "uint64"),
				Entry("float32", "float32", "float"),
				Entry("float64", "float64", "double"),
				Entry("bytes", "bytes", "bytes"),
			)

			It("Should map uuid to string", func() {
				source := `
					@go output "core/pkg/api/grpc/v1"
					@pb

					Test struct {
						key uuid
					}
				`
				resp := testutil.MustGenerate(ctx, source, "test", loader, p)
				testutil.ExpectContent(resp, "test.proto").ToContain("string key = 1;")
			})

			It("Should map json to google.protobuf.Struct", func() {
				source := `
					@go output "core/pkg/api/grpc/v1"
					@pb

					Test struct {
						data json
					}
				`
				resp := testutil.MustGenerate(ctx, source, "test", loader, p)
				testutil.ExpectContent(resp, "test.proto").
					ToContain(
						`import "google/protobuf/struct.proto";`,
						`google.protobuf.Struct data = 1;`,
					)
			})
		})

		It("Should generate proto file for simple struct", func() {
			table := resolution.NewTable()
			table.Add(resolution.Type{
				Name:          "User",
				Namespace:     "user",
				QualifiedName: "user.User",
				Form: resolution.StructForm{
					Fields: []resolution.Field{
						{Name: "key", Type: resolution.TypeRef{Name: "uuid"}},
						{Name: "username", Type: resolution.TypeRef{Name: "string"}},
						{Name: "email", Type: resolution.TypeRef{Name: "string"}, IsOptional: true},
					},
				},
				Domains: pbDomains("core/pkg/api/grpc/v1"),
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Files).To(HaveLen(1))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring("syntax = \"proto3\";"))
			Expect(content).To(ContainSubstring("package api.user.pb;"))
			Expect(content).To(ContainSubstring("message User"))
			Expect(content).To(ContainSubstring("string key = 1;"))
			Expect(content).To(ContainSubstring("string username = 2;"))
			// Soft optional (?) is a regular field in proto
			Expect(content).To(ContainSubstring("string email = 3;"))
		})

		It("Should generate enum with prefixed values per protobuf style guide", func() {
			table := resolution.NewTable()
			table.Add(resolution.Type{
				Name:          "Role",
				Namespace:     "user",
				QualifiedName: "user.Role",
				Form: resolution.EnumForm{
					Values: []resolution.EnumValue{
						{Name: "admin"},
						{Name: "user"},
					},
				},
				Domains: map[string]resolution.Domain{},
			})
			table.Add(resolution.Type{
				Name:          "User",
				Namespace:     "user",
				QualifiedName: "user.User",
				Form: resolution.StructForm{
					Fields: []resolution.Field{
						{Name: "role", Type: resolution.TypeRef{Name: "user.Role"}},
					},
				},
				Domains: pbDomains("core/pkg/api/grpc/v1"),
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Files).To(HaveLen(1))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring("enum Role"))
			// Enum values should be prefixed with enum name per protobuf style guide
			Expect(content).To(ContainSubstring("ROLE_ADMIN = 0;"))
			Expect(content).To(ContainSubstring("ROLE_USER = 1;"))
		})

		It("Should handle array fields as repeated", func() {
			table := resolution.NewTable()
			table.Add(resolution.Type{
				Name:          "User",
				Namespace:     "user",
				QualifiedName: "user.User",
				Form: resolution.StructForm{
					Fields: []resolution.Field{
						{Name: "tags", Type: resolution.TypeRef{Name: "Array", TypeArgs: []resolution.TypeRef{{Name: "string"}}}},
					},
				},
				Domains: pbDomains("core/pkg/api/grpc/v1"),
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Files).To(HaveLen(1))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring("repeated string tags = 1;"))
		})

		It("Should handle map fields", func() {
			table := resolution.NewTable()
			table.Add(resolution.Type{
				Name:          "Config",
				Namespace:     "config",
				QualifiedName: "config.Config",
				Form: resolution.StructForm{
					Fields: []resolution.Field{
						{
							Name: "settings",
							Type: resolution.TypeRef{
								Name: "Map",
								TypeArgs: []resolution.TypeRef{
									{Name: "string"},
									{Name: "string"},
								},
							},
						},
					},
				},
				Domains: pbDomains("core/pkg/api/grpc/v1"),
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Files).To(HaveLen(1))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring("map<string, string> settings = 1;"))
		})

		It("Should import google.protobuf.Struct for json type", func() {
			table := resolution.NewTable()
			table.Add(resolution.Type{
				Name:          "Config",
				Namespace:     "config",
				QualifiedName: "config.Config",
				Form: resolution.StructForm{
					Fields: []resolution.Field{
						{Name: "data", Type: resolution.TypeRef{Name: "json"}},
					},
				},
				Domains: pbDomains("core/pkg/api/grpc/v1"),
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Files).To(HaveLen(1))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring("import \"google/protobuf/struct.proto\";"))
			Expect(content).To(ContainSubstring("google.protobuf.Struct data = 1;"))
		})

		It("Should skip type aliases", func() {
			table := resolution.NewTable()
			table.Add(resolution.Type{
				Name:          "Parent",
				Namespace:     "user",
				QualifiedName: "user.Parent",
				Form: resolution.StructForm{
					Fields: []resolution.Field{
						{Name: "id", Type: resolution.TypeRef{Name: "uuid"}},
					},
				},
				Domains: pbDomains("core/pkg/api/grpc/v1"),
			})
			table.Add(resolution.Type{
				Name:          "Alias",
				Namespace:     "user",
				QualifiedName: "user.Alias",
				Form: resolution.AliasForm{
					Target: resolution.TypeRef{Name: "user.Parent"},
				},
				Domains: pbDomains("core/pkg/api/grpc/v1"),
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Files).To(HaveLen(1))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring("message Parent"))
			Expect(strings.Count(content, "message")).To(Equal(1)) // Only Parent, not Alias
		})

		It("Should preserve struct declaration order", func() {
			table := resolution.NewTable()
			domains := pbDomains("core/pkg/api/grpc/v1")
			table.Add(resolution.Type{
				Name:          "Zebra",
				Namespace:     "animals",
				QualifiedName: "animals.Zebra",
				Form: resolution.StructForm{
					Fields: []resolution.Field{{Name: "name", Type: resolution.TypeRef{Name: "string"}}},
				},
				Domains: domains,
			})
			table.Add(resolution.Type{
				Name:          "Apple",
				Namespace:     "animals",
				QualifiedName: "animals.Apple",
				Form: resolution.StructForm{
					Fields: []resolution.Field{{Name: "color", Type: resolution.TypeRef{Name: "string"}}},
				},
				Domains: domains,
			})
			table.Add(resolution.Type{
				Name:          "Mango",
				Namespace:     "animals",
				QualifiedName: "animals.Mango",
				Form: resolution.StructForm{
					Fields: []resolution.Field{{Name: "ripe", Type: resolution.TypeRef{Name: "bool"}}},
				},
				Domains: domains,
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Files).To(HaveLen(1))

			content := string(resp.Files[0].Content)
			zebraIdx := strings.Index(content, "message Zebra")
			appleIdx := strings.Index(content, "message Apple")
			mangoIdx := strings.Index(content, "message Mango")
			Expect(zebraIdx).To(BeNumerically("<", appleIdx))
			Expect(appleIdx).To(BeNumerically("<", mangoIdx))
		})

		It("Should preserve field declaration order", func() {
			table := resolution.NewTable()
			table.Add(resolution.Type{
				Name:          "Record",
				Namespace:     "order",
				QualifiedName: "order.Record",
				Form: resolution.StructForm{
					Fields: []resolution.Field{
						{Name: "zebra", Type: resolution.TypeRef{Name: "string"}},
						{Name: "apple", Type: resolution.TypeRef{Name: "int32"}},
						{Name: "mango", Type: resolution.TypeRef{Name: "bool"}},
					},
				},
				Domains: pbDomains("core/pkg/api/grpc/v1"),
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Files).To(HaveLen(1))

			content := string(resp.Files[0].Content)
			zebraIdx := strings.Index(content, "zebra = 1")
			appleIdx := strings.Index(content, "apple = 2")
			mangoIdx := strings.Index(content, "mango = 3")
			Expect(zebraIdx).To(BeNumerically("<", appleIdx))
			Expect(appleIdx).To(BeNumerically("<", mangoIdx))
		})

		Context("@omit directive", func() {
			It("Should skip types with @pb omit directive", func() {
				source := `
					@go output "core/pkg/api/grpc/v1"
					@pb

					User struct {
						key uuid
						name string
					}

					InternalState struct {
						cache json
						@pb omit
					}
				`
				resp := testutil.MustGenerate(ctx, source, "user", loader, p)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`message User`))
				Expect(content).NotTo(ContainSubstring(`InternalState`))
			})

			It("Should skip enums with @pb omit directive", func() {
				source := `
					@go output "core/pkg/api/grpc/v1"
					@pb

					Status enum {
						active = 1
						inactive = 2
					}

					DebugLevel enum {
						verbose = 0
						trace = 1
						@pb omit
					}

					Task struct {
						status Status
					}
				`
				resp := testutil.MustGenerate(ctx, source, "status", loader, p)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`enum Status`))
				Expect(content).NotTo(ContainSubstring(`DebugLevel`))
			})
		})

		Context("type parameters", func() {
			It("Should resolve type param defaults to concrete types", func() {
				table := resolution.NewTable()
				Expect(table.Add(resolution.Type{
					Name:          "Generic",
					Namespace:     "test",
					QualifiedName: "test.Generic",
					Form: resolution.StructForm{
						TypeParams: []resolution.TypeParam{
							{Name: "T", Default: &resolution.TypeRef{Name: "string"}},
						},
						Fields: []resolution.Field{
							{
								Name: "value",
								Type: resolution.TypeRef{
									TypeParam: &resolution.TypeParam{
										Name:    "T",
										Default: &resolution.TypeRef{Name: "string"},
									},
								},
							},
						},
					},
					Domains: pbDomains("core/pkg/api/grpc/v1"),
				})).To(Succeed())

				req := &plugin.Request{Resolutions: table, RepoRoot: "/tmp/test"}
				resp, err := p.Generate(req)
				Expect(err).ToNot(HaveOccurred())
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring("string value = 1;"))
			})

			It("Should resolve type params without default to google.protobuf.Any", func() {
				table := resolution.NewTable()
				Expect(table.Add(resolution.Type{
					Name:          "Container",
					Namespace:     "test",
					QualifiedName: "test.Container",
					Form: resolution.StructForm{
						TypeParams: []resolution.TypeParam{{Name: "T"}},
						Fields: []resolution.Field{
							{
								Name: "value",
								Type: resolution.TypeRef{
									TypeParam: &resolution.TypeParam{Name: "T"},
								},
							},
						},
					},
					Domains: pbDomains("core/pkg/api/grpc/v1"),
				})).To(Succeed())

				req := &plugin.Request{Resolutions: table, RepoRoot: "/tmp/test"}
				resp, err := p.Generate(req)
				Expect(err).ToNot(HaveOccurred())
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring(`import "google/protobuf/any.proto";`))
				Expect(content).To(ContainSubstring("google.protobuf.Any value = 1;"))
			})
		})

		Context("nested arrays", func() {
			It("Should generate wrapper messages for nested arrays", func() {
				table := resolution.NewTable()
				Expect(table.Add(resolution.Type{
					Name:          "Matrix",
					Namespace:     "math",
					QualifiedName: "math.Matrix",
					Form: resolution.StructForm{
						Fields: []resolution.Field{
							{
								Name: "rows",
								Type: resolution.TypeRef{
									Name: "Array",
									TypeArgs: []resolution.TypeRef{
										{
											Name:     "Array",
											TypeArgs: []resolution.TypeRef{{Name: "float64"}},
										},
									},
								},
							},
						},
					},
					Domains: pbDomains("core/pkg/api/grpc/v1"),
				})).To(Succeed())

				req := &plugin.Request{Resolutions: table, RepoRoot: "/tmp/test"}
				resp, err := p.Generate(req)
				Expect(err).ToNot(HaveOccurred())
				content := string(resp.Files[0].Content)
				// Should have wrapper message and use it
				Expect(content).To(ContainSubstring("message ArrayWrapper"))
				Expect(content).To(ContainSubstring("repeated double values = 1;"))
				Expect(content).To(ContainSubstring("repeated ArrayWrapper rows = 1;"))
			})
		})

		Context("hard optional fields", func() {
			It("Should mark hard optional fields with optional keyword", func() {
				table := resolution.NewTable()
				Expect(table.Add(resolution.Type{
					Name:          "User",
					Namespace:     "user",
					QualifiedName: "user.User",
					Form: resolution.StructForm{
						Fields: []resolution.Field{
							{Name: "name", Type: resolution.TypeRef{Name: "string"}},
							{Name: "nickname", Type: resolution.TypeRef{Name: "string"}, IsHardOptional: true},
						},
					},
					Domains: pbDomains("core/pkg/api/grpc/v1"),
				})).To(Succeed())

				req := &plugin.Request{Resolutions: table, RepoRoot: "/tmp/test"}
				resp, err := p.Generate(req)
				Expect(err).ToNot(HaveOccurred())
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring("string name = 1;"))
				Expect(content).To(ContainSubstring("optional string nickname = 2;"))
			})
		})

		Context("distinct types", func() {
			It("Should resolve distinct types to their underlying type", func() {
				table := resolution.NewTable()
				Expect(table.Add(resolution.Type{
					Name:          "UserID",
					Namespace:     "user",
					QualifiedName: "user.UserID",
					Form:          resolution.DistinctForm{Base: resolution.TypeRef{Name: "uuid"}},
					Domains:       pbDomains("core/pkg/api/grpc/v1"),
				})).To(Succeed())
				Expect(table.Add(resolution.Type{
					Name:          "User",
					Namespace:     "user",
					QualifiedName: "user.User",
					Form: resolution.StructForm{
						Fields: []resolution.Field{
							{Name: "id", Type: resolution.TypeRef{Name: "user.UserID"}},
						},
					},
					Domains: pbDomains("core/pkg/api/grpc/v1"),
				})).To(Succeed())

				req := &plugin.Request{Resolutions: table, RepoRoot: "/tmp/test"}
				resp, err := p.Generate(req)
				Expect(err).ToNot(HaveOccurred())
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring("string id = 1;"))
			})
		})

		Context("@pb name override", func() {
			It("Should use @pb name for struct if specified", func() {
				source := `
					@go output "core/pkg/api/grpc/v1"
					@pb name "MyProtoMessage"
					@pb

					OriginalName struct {
						value string
					}
				`
				resp := testutil.MustGenerate(ctx, source, "test", loader, p)
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring("message MyProtoMessage"))
				Expect(content).NotTo(ContainSubstring("OriginalName"))
			})
		})

		Context("cross-namespace struct references", func() {
			It("Should import and qualify cross-namespace struct", func() {
				table := resolution.NewTable()
				Expect(table.Add(resolution.Type{
					Name:          "Address",
					Namespace:     "shared",
					QualifiedName: "shared.Address",
					Form: resolution.StructForm{
						Fields: []resolution.Field{
							{Name: "city", Type: resolution.TypeRef{Name: "string"}},
						},
					},
					Domains: pbDomains("core/pkg/shared"),
				})).To(Succeed())
				Expect(table.Add(resolution.Type{
					Name:          "User",
					Namespace:     "user",
					QualifiedName: "user.User",
					Form: resolution.StructForm{
						Fields: []resolution.Field{
							{Name: "address", Type: resolution.TypeRef{Name: "shared.Address"}},
						},
					},
					Domains: pbDomains("core/pkg/user"),
				})).To(Succeed())

				req := &plugin.Request{Resolutions: table, RepoRoot: "/tmp/test"}
				resp, err := p.Generate(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Files).To(HaveLen(2))

				// Find the user file
				var userContent string
				for _, f := range resp.Files {
					if strings.Contains(f.Path, "user.proto") {
						userContent = string(f.Content)
						break
					}
				}
				Expect(userContent).To(ContainSubstring(`import "core/pkg/shared/pb/shared.proto";`))
				Expect(userContent).To(ContainSubstring(".shared.shared.pb.Address address = 1;"))
			})
		})

		Context("cross-namespace enum references", func() {
			It("Should import and qualify cross-namespace enum", func() {
				table := resolution.NewTable()
				// Enum in shared namespace with struct for output path lookup
				Expect(table.Add(resolution.Type{
					Name:          "Helper",
					Namespace:     "shared",
					QualifiedName: "shared.Helper",
					Form: resolution.StructForm{
						Fields: []resolution.Field{
							{Name: "dummy", Type: resolution.TypeRef{Name: "string"}},
						},
					},
					Domains: pbDomains("core/pkg/shared"),
				})).To(Succeed())
				Expect(table.Add(resolution.Type{
					Name:          "Priority",
					Namespace:     "shared",
					QualifiedName: "shared.Priority",
					Form: resolution.EnumForm{
						Values: []resolution.EnumValue{{Name: "low"}, {Name: "high"}},
					},
				})).To(Succeed())
				Expect(table.Add(resolution.Type{
					Name:          "Task",
					Namespace:     "task",
					QualifiedName: "task.Task",
					Form: resolution.StructForm{
						Fields: []resolution.Field{
							{Name: "priority", Type: resolution.TypeRef{Name: "shared.Priority"}},
						},
					},
					Domains: pbDomains("core/pkg/task"),
				})).To(Succeed())

				req := &plugin.Request{Resolutions: table, RepoRoot: "/tmp/test"}
				resp, err := p.Generate(req)
				Expect(err).ToNot(HaveOccurred())

				// Find the task file
				var taskContent string
				for _, f := range resp.Files {
					if strings.Contains(f.Path, "task.proto") {
						taskContent = string(f.Content)
						break
					}
				}
				Expect(taskContent).To(ContainSubstring(`import "core/pkg/shared/pb/shared.proto";`))
				Expect(taskContent).To(ContainSubstring(".shared.shared.pb.Priority priority = 1;"))
			})
		})

		Context("map with struct values", func() {
			It("Should handle maps with struct value types", func() {
				table := resolution.NewTable()
				Expect(table.Add(resolution.Type{
					Name:          "Info",
					Namespace:     "config",
					QualifiedName: "config.Info",
					Form: resolution.StructForm{
						Fields: []resolution.Field{
							{Name: "value", Type: resolution.TypeRef{Name: "string"}},
						},
					},
					Domains: pbDomains("core/pkg/api/grpc/v1"),
				})).To(Succeed())
				Expect(table.Add(resolution.Type{
					Name:          "Config",
					Namespace:     "config",
					QualifiedName: "config.Config",
					Form: resolution.StructForm{
						Fields: []resolution.Field{
							{
								Name: "items",
								Type: resolution.TypeRef{
									Name: "Map",
									TypeArgs: []resolution.TypeRef{
										{Name: "string"},
										{Name: "config.Info"},
									},
								},
							},
						},
					},
					Domains: pbDomains("core/pkg/api/grpc/v1"),
				})).To(Succeed())

				req := &plugin.Request{Resolutions: table, RepoRoot: "/tmp/test"}
				resp, err := p.Generate(req)
				Expect(err).ToNot(HaveOccurred())
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring("map<string, Info> items = 1;"))
			})
		})

		Context("array of structs", func() {
			It("Should handle arrays of struct types", func() {
				table := resolution.NewTable()
				Expect(table.Add(resolution.Type{
					Name:          "Item",
					Namespace:     "list",
					QualifiedName: "list.Item",
					Form: resolution.StructForm{
						Fields: []resolution.Field{
							{Name: "name", Type: resolution.TypeRef{Name: "string"}},
						},
					},
					Domains: pbDomains("core/pkg/api/grpc/v1"),
				})).To(Succeed())
				Expect(table.Add(resolution.Type{
					Name:          "Container",
					Namespace:     "list",
					QualifiedName: "list.Container",
					Form: resolution.StructForm{
						Fields: []resolution.Field{
							{
								Name: "items",
								Type: resolution.TypeRef{
									Name:     "Array",
									TypeArgs: []resolution.TypeRef{{Name: "list.Item"}},
								},
							},
						},
					},
					Domains: pbDomains("core/pkg/api/grpc/v1"),
				})).To(Succeed())

				req := &plugin.Request{Resolutions: table, RepoRoot: "/tmp/test"}
				resp, err := p.Generate(req)
				Expect(err).ToNot(HaveOccurred())
				content := string(resp.Files[0].Content)
				Expect(content).To(ContainSubstring("repeated Item items = 1;"))
			})
		})
	})
})
