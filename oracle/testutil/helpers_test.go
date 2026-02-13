// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin"
	. "github.com/synnaxlabs/oracle/testutil"
)

type mockPlugin struct {
	files []plugin.File
	err   error
}

func (m *mockPlugin) Name() string                { return "mock" }
func (m *mockPlugin) Domains() []string           { return []string{"go"} }
func (m *mockPlugin) Requires() []string          { return nil }
func (m *mockPlugin) Check(*plugin.Request) error { return nil }

func (m *mockPlugin) Generate(*plugin.Request) (*plugin.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &plugin.Response{Files: m.files}, nil
}

type reqCapturingPlugin struct {
	files   []plugin.File
	lastReq *plugin.Request
}

func (m *reqCapturingPlugin) Name() string                { return "mock" }
func (m *reqCapturingPlugin) Domains() []string           { return []string{"go"} }
func (m *reqCapturingPlugin) Requires() []string          { return nil }
func (m *reqCapturingPlugin) Check(*plugin.Request) error { return nil }

func (m *reqCapturingPlugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	m.lastReq = req
	return &plugin.Response{Files: m.files}, nil
}

var _ = Describe("MustGenerateRequest", func() {
	var (
		ctx    context.Context
		loader *MockFileLoader
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = NewMockFileLoader()
	})

	It("should return a request with resolved types", func() {
		source := fmt.Sprintf(SimpleStructTemplate, DomainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "user", loader)
		Expect(req).NotTo(BeNil())
		Expect(req.Resolutions).NotTo(BeNil())
		Expect(req.RepoRoot).To(Equal("/mock/repo"))
	})

	It("should resolve all primitive types", func() {
		source := fmt.Sprintf(AllPrimitivesTemplate, DomainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "all", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve enum definitions", func() {
		source := fmt.Sprintf(IntEnumTemplate, DomainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "status", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve struct extension", func() {
		source := fmt.Sprintf(StructExtensionTemplate, DomainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "ext", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve soft optional fields", func() {
		source := fmt.Sprintf(SoftOptionalTemplate, DomainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "opt", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve hard optional fields", func() {
		source := fmt.Sprintf(HardOptionalTemplate, DomainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "nullable", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve array types", func() {
		source := fmt.Sprintf(ArrayTypesTemplate, DomainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "arr", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve generic structs", func() {
		source := fmt.Sprintf(GenericStructTemplate, DomainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "gen", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve field omission in extension", func() {
		source := fmt.Sprintf(FieldOmissionTemplate, DomainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "omit", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve type aliases", func() {
		source := fmt.Sprintf(TypeAliasTemplate, DomainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "alias", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve distinct types", func() {
		source := fmt.Sprintf(DistinctTypeTemplate, DomainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "distinct", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve multiple structs in one schema", func() {
		source := fmt.Sprintf(MultipleStructsTemplate, DomainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "multi", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve struct references", func() {
		source := fmt.Sprintf(StructReferenceTemplate, DomainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "ref", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})
})

var _ = Describe("MustGenerate", func() {
	var (
		ctx    context.Context
		loader *MockFileLoader
	)

	BeforeEach(func() {
		ctx = context.Background()
		loader = NewMockFileLoader()
	})

	It("should return a response with generated files", func() {
		p := &mockPlugin{
			files: []plugin.File{
				{Path: "out/user.go", Content: []byte("package user")},
			},
		}
		source := fmt.Sprintf(SimpleStructTemplate, DomainDirectives["go"])
		resp := MustGenerate(ctx, source, "user", loader, p)
		Expect(resp).NotTo(BeNil())
		Expect(resp.Files).To(HaveLen(1))
		Expect(resp.Files[0].Path).To(Equal("out/user.go"))
	})

	It("should pass the repo root through to the request", func() {
		p := &reqCapturingPlugin{files: []plugin.File{
			{Path: "out/user.go", Content: []byte("package user")},
		}}
		source := fmt.Sprintf(SimpleStructTemplate, DomainDirectives["go"])
		MustGenerate(ctx, source, "user", loader, p)
		Expect(p.lastReq).NotTo(BeNil())
		Expect(p.lastReq.RepoRoot).To(Equal("/mock/repo"))
	})

	It("should pass resolutions to the plugin", func() {
		p := &reqCapturingPlugin{files: []plugin.File{
			{Path: "out/user.go", Content: []byte("package user")},
		}}
		source := fmt.Sprintf(SimpleStructTemplate, DomainDirectives["go"])
		MustGenerate(ctx, source, "user", loader, p)
		Expect(p.lastReq.Resolutions).NotTo(BeNil())
	})

	It("should return an empty response when plugin generates no files", func() {
		p := &mockPlugin{files: []plugin.File{}}
		source := fmt.Sprintf(SimpleStructTemplate, DomainDirectives["go"])
		resp := MustGenerate(ctx, source, "user", loader, p)
		Expect(resp.Files).To(BeEmpty())
	})

	It("should return multiple generated files", func() {
		p := &mockPlugin{
			files: []plugin.File{
				{Path: "out/user.go", Content: []byte("package user")},
				{Path: "out/group.go", Content: []byte("package group")},
			},
		}
		source := fmt.Sprintf(MultipleStructsTemplate, DomainDirectives["go"])
		resp := MustGenerate(ctx, source, "multi", loader, p)
		Expect(resp.Files).To(HaveLen(2))
	})
})

var _ = Describe("MustContentOf", func() {
	It("should return content of a matching file", func() {
		resp := &plugin.Response{
			Files: []plugin.File{
				{Path: "out/user.go", Content: []byte("package user")},
			},
		}
		Expect(MustContentOf(resp, "user.go")).To(Equal("package user"))
	})

	It("should match by suffix across deep paths", func() {
		resp := &plugin.Response{
			Files: []plugin.File{
				{Path: "a/b/c/d/schema.go", Content: []byte("package schema")},
			},
		}
		Expect(MustContentOf(resp, "schema.go")).To(Equal("package schema"))
	})

	It("should return the first matching file", func() {
		resp := &plugin.Response{
			Files: []plugin.File{
				{Path: "out/user.go", Content: []byte("first")},
				{Path: "other/user.go", Content: []byte("second")},
			},
		}
		Expect(MustContentOf(resp, "user.go")).To(Equal("first"))
	})
})

var _ = Describe("ExpectContent", func() {
	buildResponse := func(files ...plugin.File) *plugin.Response {
		return &plugin.Response{Files: files}
	}

	Describe("ToContain", func() {
		It("should pass when content contains all substrings", func() {
			resp := buildResponse(plugin.File{
				Path:    "out/user.go",
				Content: []byte("package user\n\ntype User struct {\n\tKey uuid.UUID\n}"),
			})
			ExpectContent(resp, "user.go").
				ToContain("package user", "type User struct", "Key uuid.UUID")
		})

		It("should support chaining multiple ToContain calls", func() {
			resp := buildResponse(plugin.File{
				Path:    "out/user.go",
				Content: []byte("package user\n\ntype User struct {\n\tKey uuid.UUID\n\tName string\n}"),
			})
			ExpectContent(resp, "user.go").
				ToContain("package user").
				ToContain("type User struct").
				ToContain("Key uuid.UUID")
		})
	})

	Describe("ToNotContain", func() {
		It("should pass when content does not contain any substrings", func() {
			resp := buildResponse(plugin.File{
				Path:    "out/user.go",
				Content: []byte("package user\n\ntype User struct {}"),
			})
			ExpectContent(resp, "user.go").
				ToNotContain("secret", "password", "private")
		})

		It("should support chaining with ToContain", func() {
			resp := buildResponse(plugin.File{
				Path:    "out/user.go",
				Content: []byte("package user\n\ntype User struct {\n\tName string\n}"),
			})
			ExpectContent(resp, "user.go").
				ToContain("Name string").
				ToNotContain("secret")
		})
	})

	Describe("ToPreserveOrder", func() {
		It("should pass when substrings appear in order", func() {
			resp := buildResponse(plugin.File{
				Path:    "out/user.go",
				Content: []byte("package user\n\nimport \"fmt\"\n\ntype User struct {}"),
			})
			ExpectContent(resp, "user.go").
				ToPreserveOrder("package user", "import", "type User struct")
		})

		It("should support chaining with other assertions", func() {
			resp := buildResponse(plugin.File{
				Path:    "out/user.go",
				Content: []byte("package user\n\ntype User struct {\n\tKey string\n\tName string\n}"),
			})
			ExpectContent(resp, "user.go").
				ToContain("package user").
				ToPreserveOrder("Key string", "Name string").
				ToNotContain("secret")
		})

		It("should verify ordering across many substrings", func() {
			resp := buildResponse(plugin.File{
				Path:    "out/user.go",
				Content: []byte("package user\n\nimport (\n\t\"fmt\"\n)\n\ntype User struct {\n\tA int\n\tB int\n\tC int\n}"),
			})
			ExpectContent(resp, "user.go").
				ToPreserveOrder("package", "import", "type User", "A int", "B int", "C int")
		})
	})

	Describe("file matching", func() {
		It("should match by path suffix", func() {
			resp := buildResponse(plugin.File{
				Path:    "some/deep/path/user.go",
				Content: []byte("package user"),
			})
			ExpectContent(resp, "user.go").ToContain("package user")
		})

		It("should match the correct file among multiple", func() {
			resp := buildResponse(
				plugin.File{Path: "out/user.go", Content: []byte("package user")},
				plugin.File{Path: "out/group.go", Content: []byte("package group")},
				plugin.File{Path: "out/role.go", Content: []byte("package role")},
			)
			ExpectContent(resp, "group.go").
				ToContain("package group").
				ToNotContain("package user", "package role")
		})

		It("should match a partial directory suffix", func() {
			resp := buildResponse(plugin.File{
				Path:    "core/pkg/service/user/types.gen.go",
				Content: []byte("package user"),
			})
			ExpectContent(resp, "user/types.gen.go").ToContain("package user")
		})
	})
})
