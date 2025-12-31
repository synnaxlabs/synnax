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
			table.AddStruct(resolution.Struct{
				Name:          "User",
				Namespace:     "user",
				QualifiedName: "user.User",
				Fields: resolution.Fields{
					{Name: "key", TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "uuid"}},
					{Name: "username", TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "string"}},
					{Name: "email", TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "string", IsOptional: true}},
				},
				Domains: map[string]resolution.Domain{
					"pb": {
						Name: "pb",
						Expressions: resolution.Expressions{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "core/pkg/api/grpc/v1"}}},
							{Name: "package", Values: []resolution.ExpressionValue{{StringValue: "api.v1"}}},
						},
					},
				},
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
			Expect(content).To(ContainSubstring("package api.v1;"))
			Expect(content).To(ContainSubstring("message PBUser"))
			Expect(content).To(ContainSubstring("string key = 1;"))
			Expect(content).To(ContainSubstring("string username = 2;"))
			Expect(content).To(ContainSubstring("optional string email = 3;"))
		})

		It("Should generate enum with UNSPECIFIED value", func() {
			table := resolution.NewTable()
			roleEnum := resolution.Enum{
				Name:          "Role",
				Namespace:     "user",
				QualifiedName: "user.Role",
				Values: []resolution.EnumEntry{
					{Name: "admin", ExpressionValue: resolution.ExpressionValue{StringValue: "admin"}},
					{Name: "user", ExpressionValue: resolution.ExpressionValue{StringValue: "user"}},
				},
				Domains: map[string]resolution.Domain{},
			}
			table.AddEnum(roleEnum)
			table.AddStruct(resolution.Struct{
				Name:          "User",
				Namespace:     "user",
				QualifiedName: "user.User",
				Fields: resolution.Fields{
					{Name: "role", TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindEnum, EnumRef: &roleEnum}},
				},
				Domains: map[string]resolution.Domain{
					"pb": {
						Name: "pb",
						Expressions: resolution.Expressions{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "core/pkg/api/grpc/v1"}}},
						},
					},
				},
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Files).To(HaveLen(1))

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring("enum PBRole"))
			Expect(content).To(ContainSubstring("PB_ROLE_UNSPECIFIED = 0;"))
			Expect(content).To(ContainSubstring("PB_ROLE_ADMIN = 1;"))
			Expect(content).To(ContainSubstring("PB_ROLE_USER = 2;"))
		})

		It("Should handle array fields as repeated", func() {
			table := resolution.NewTable()
			table.AddStruct(resolution.Struct{
				Name:          "User",
				Namespace:     "user",
				QualifiedName: "user.User",
				Fields: resolution.Fields{
					{Name: "tags", TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "string", IsArray: true}},
				},
				Domains: map[string]resolution.Domain{
					"pb": {
						Name: "pb",
						Expressions: resolution.Expressions{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "core/pkg/api/grpc/v1"}}},
						},
					},
				},
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring("repeated string tags = 1;"))
		})

		It("Should handle map fields", func() {
			table := resolution.NewTable()
			table.AddStruct(resolution.Struct{
				Name:          "Config",
				Namespace:     "config",
				QualifiedName: "config.Config",
				Fields: resolution.Fields{
					{
						Name: "settings",
						TypeRef: &resolution.TypeRef{
							Kind:         resolution.TypeKindMap,
							MapKeyType:   &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "string"},
							MapValueType: &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "string"},
						},
					},
				},
				Domains: map[string]resolution.Domain{
					"pb": {
						Name: "pb",
						Expressions: resolution.Expressions{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "core/pkg/api/grpc/v1"}}},
						},
					},
				},
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring("map<string, string> settings = 1;"))
		})

		It("Should import google.protobuf.Struct for json type", func() {
			table := resolution.NewTable()
			table.AddStruct(resolution.Struct{
				Name:          "Config",
				Namespace:     "config",
				QualifiedName: "config.Config",
				Fields: resolution.Fields{
					{Name: "data", TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "json"}},
				},
				Domains: map[string]resolution.Domain{
					"pb": {
						Name: "pb",
						Expressions: resolution.Expressions{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "core/pkg/api/grpc/v1"}}},
						},
					},
				},
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring("import \"google/protobuf/struct.proto\";"))
			Expect(content).To(ContainSubstring("google.protobuf.Struct data = 1;"))
		})

		It("Should skip type aliases", func() {
			table := resolution.NewTable()
			parentStruct := resolution.Struct{
				Name:          "Parent",
				Namespace:     "user",
				QualifiedName: "user.Parent",
				Fields: resolution.Fields{
					{Name: "id", TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "uuid"}},
				},
				Domains: map[string]resolution.Domain{
					"pb": {
						Name: "pb",
						Expressions: resolution.Expressions{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "core/pkg/api/grpc/v1"}}},
						},
					},
				},
			}
			table.AddStruct(parentStruct)
			table.AddStruct(resolution.Struct{
				Name:          "Alias",
				Namespace:     "user",
				QualifiedName: "user.Alias",
				AliasOf:       &resolution.TypeRef{Kind: resolution.TypeKindStruct, StructRef: &parentStruct},
				Domains: map[string]resolution.Domain{
					"pb": {
						Name: "pb",
						Expressions: resolution.Expressions{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "core/pkg/api/grpc/v1"}}},
						},
					},
				},
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())

			content := string(resp.Files[0].Content)
			Expect(content).To(ContainSubstring("message PBParent"))
			Expect(strings.Count(content, "message")).To(Equal(1)) // Only Parent, not Alias
		})

		It("Should preserve struct declaration order", func() {
			table := resolution.NewTable()
			pbDomain := map[string]resolution.Domain{
				"pb": {
					Name: "pb",
					Expressions: resolution.Expressions{
						{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "core/pkg/api/grpc/v1"}}},
					},
				},
			}
			table.AddStruct(resolution.Struct{
				Name:          "Zebra",
				Namespace:     "animals",
				QualifiedName: "animals.Zebra",
				Fields:        resolution.Fields{{Name: "name", TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "string"}}},
				Domains:       pbDomain,
			})
			table.AddStruct(resolution.Struct{
				Name:          "Apple",
				Namespace:     "animals",
				QualifiedName: "animals.Apple",
				Fields:        resolution.Fields{{Name: "color", TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "string"}}},
				Domains:       pbDomain,
			})
			table.AddStruct(resolution.Struct{
				Name:          "Mango",
				Namespace:     "animals",
				QualifiedName: "animals.Mango",
				Fields:        resolution.Fields{{Name: "ripe", TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "bool"}}},
				Domains:       pbDomain,
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())

			content := string(resp.Files[0].Content)
			zebraIdx := strings.Index(content, "message PBZebra")
			appleIdx := strings.Index(content, "message PBApple")
			mangoIdx := strings.Index(content, "message PBMango")
			Expect(zebraIdx).To(BeNumerically("<", appleIdx))
			Expect(appleIdx).To(BeNumerically("<", mangoIdx))
		})

		It("Should preserve field declaration order", func() {
			table := resolution.NewTable()
			table.AddStruct(resolution.Struct{
				Name:          "Record",
				Namespace:     "order",
				QualifiedName: "order.Record",
				Fields: resolution.Fields{
					{Name: "zebra", TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "string"}},
					{Name: "apple", TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "int32"}},
					{Name: "mango", TypeRef: &resolution.TypeRef{Kind: resolution.TypeKindPrimitive, Primitive: "bool"}},
				},
				Domains: map[string]resolution.Domain{
					"pb": {
						Name: "pb",
						Expressions: resolution.Expressions{
							{Name: "output", Values: []resolution.ExpressionValue{{StringValue: "core/pkg/api/grpc/v1"}}},
						},
					},
				},
			})

			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    "/tmp/test",
			}

			resp, err := p.Generate(req)
			Expect(err).ToNot(HaveOccurred())

			content := string(resp.Files[0].Content)
			zebraIdx := strings.Index(content, "zebra = 1")
			appleIdx := strings.Index(content, "apple = 2")
			mangoIdx := strings.Index(content, "mango = 3")
			Expect(zebraIdx).To(BeNumerically("<", appleIdx))
			Expect(appleIdx).To(BeNumerically("<", mangoIdx))
		})
	})
})
