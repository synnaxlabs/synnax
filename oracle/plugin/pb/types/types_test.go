// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/pb/types"
	"github.com/synnaxlabs/oracle/resolution"
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

var _ = Describe("Plugin", func() {
	var p *types.Plugin

	BeforeEach(func() {
		p = types.New(types.DefaultOptions())
	})

	Describe("Name", func() {
		It("Should return pb/types", func() {
			Expect(p.Name()).To(Equal("pb/types"))
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
			Expect(content).To(ContainSubstring("package api.user;"))
			Expect(content).To(ContainSubstring("message User"))
			Expect(content).To(ContainSubstring("string key = 1;"))
			Expect(content).To(ContainSubstring("string username = 2;"))
			// Soft optional (?) is a regular field in proto
			Expect(content).To(ContainSubstring("string email = 3;"))
		})

		It("Should generate enum with UNSPECIFIED value", func() {
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
			Expect(content).To(ContainSubstring("UNSPECIFIED = 0;"))
			Expect(content).To(ContainSubstring("ADMIN = 1;"))
			Expect(content).To(ContainSubstring("USER = 2;"))
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
	})
})
