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
	"fmt"
	"os"
	"path/filepath"

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

// SimpleStructTemplate is a basic struct with common field types.
// Format: domain directive (e.g., @go output "path")
const simpleStructTemplate = `
%s

User struct {
	key uuid
	name string
	age int32
	active bool
}
`

// AllPrimitivesTemplate contains all primitive types supported by oracle.
// Format: domain directive
const allPrimitivesTemplate = `
%s

AllTypes struct {
	a uuid
	b string
	c bool
	d int8
	e int16
	f int32
	g int64
	h uint8
	i uint16
	j uint32
	k uint64
	l float32
	m float64
	n timestamp
	o timespan
	p time_range
	q record
	r bytes
}
`

// ArrayTypesTemplate contains various array field types.
// Format: domain directive
const arrayTypesTemplate = `
%s

Container struct {
	tags string[]
	labels uuid[]
	scores int32[]
	flags bool[]
}
`

// DistinctTypeTemplate demonstrates distinct type definition.
// Format: domain directive
const distinctTypeTemplate = `
%s

UserKey uuid

User struct {
	key UserKey
	name string
}
`

// FieldOmissionTemplate demonstrates field omission in struct extension.
// Format: domain directive
const fieldOmissionTemplate = `
%s

Parent struct {
	key uuid
	name string
	age int32
	secret string
}

Child struct extends Parent {
	-secret
	role string
}
`

// GenericStructTemplate defines a generic (type-parameterized) struct.
// Format: domain directive
const genericStructTemplate = `
%s

Container struct<T> {
	value T
	count int32
}
`

// IntEnumTemplate defines an integer-based enumeration.
// Format: domain directive
const intEnumTemplate = `
%s

Status enum {
	unknown = 0
	pending = 1
	active = 2
	completed = 3
}
`

// MultipleStructsTemplate contains multiple struct definitions.
// Format: domain directive
const multipleStructsTemplate = `
%s

User struct {
	key uuid
	name string
}

Group struct {
	key uuid
	name string
	owner uuid
}

Membership struct {
	user uuid
	group uuid
	role string
}
`

// OptionalFieldsTemplate contains fields with the optional modifier (?), which
// uses pointer semantics (can distinguish null from the zero value).
// Format: domain directive
const optionalFieldsTemplate = `
%s

OptionalFields struct {
	key uuid
	name string?
	age int32?
	parent uuid?
}
`

// StructExtensionTemplate demonstrates struct extension (inheritance).
// Format: domain directive
const structExtensionTemplate = `
%s

Base struct {
	key uuid
	name string
	created_at timestamp
}

Extended struct extends Base {
	description string
	updated_at timestamp
}
`

// StructReferenceTemplate contains structs that reference each other.
// Format: domain directive
const structReferenceTemplate = `
%s

Parent struct {
	key uuid
	name string
}

Child struct {
	key uuid
	parent Parent
	name string
}
`

// TypeAliasTemplate demonstrates type alias definition.
// Format: domain directive
const typeAliasTemplate = `
%s

UserKey = uuid

User struct {
	key UserKey
	name string
}
`

// DomainDirectives provides common domain directive strings for each plugin.
var domainDirectives = map[string]string{
	"go":  `@go output "out"`,
	"ts":  `@ts output "out"`,
	"py":  `@py output "out"`,
	"cpp": `@cpp output "out"`,
	"pb":  `@go output "out"`, // pb derives from go output
}

var _ = Describe("MustGenerateRequest", func() {
	var loader *MockFileLoader

	BeforeEach(func() {
		loader = NewMockFileLoader()
	})

	It("should return a request with resolved types", func(ctx SpecContext) {
		source := fmt.Sprintf(simpleStructTemplate, domainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "user", loader)
		Expect(req).NotTo(BeNil())
		Expect(req.Resolutions).NotTo(BeNil())
		Expect(req.RepoRoot).To(Equal("/mock/repo"))
	})

	It("should resolve all primitive types", func(ctx SpecContext) {
		source := fmt.Sprintf(allPrimitivesTemplate, domainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "all", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve enum definitions", func(ctx SpecContext) {
		source := fmt.Sprintf(intEnumTemplate, domainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "status", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve struct extension", func(ctx SpecContext) {
		source := fmt.Sprintf(structExtensionTemplate, domainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "ext", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve optional fields", func(ctx SpecContext) {
		source := fmt.Sprintf(optionalFieldsTemplate, domainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "opt", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve array types", func(ctx SpecContext) {
		source := fmt.Sprintf(arrayTypesTemplate, domainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "arr", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve generic structs", func(ctx SpecContext) {
		source := fmt.Sprintf(genericStructTemplate, domainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "gen", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve field omission in extension", func(ctx SpecContext) {
		source := fmt.Sprintf(fieldOmissionTemplate, domainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "omit", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve type aliases", func(ctx SpecContext) {
		source := fmt.Sprintf(typeAliasTemplate, domainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "alias", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve distinct types", func(ctx SpecContext) {
		source := fmt.Sprintf(distinctTypeTemplate, domainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "distinct", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve multiple structs in one schema", func(ctx SpecContext) {
		source := fmt.Sprintf(multipleStructsTemplate, domainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "multi", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})

	It("should resolve struct references", func(ctx SpecContext) {
		source := fmt.Sprintf(structReferenceTemplate, domainDirectives["go"])
		req := MustGenerateRequest(ctx, source, "ref", loader)
		Expect(req.Resolutions).NotTo(BeNil())
	})
})

var _ = Describe("MustGenerate", func() {
	var loader *MockFileLoader

	BeforeEach(func() {
		loader = NewMockFileLoader()
	})

	It("should return a response with generated files", func(ctx SpecContext) {
		p := &mockPlugin{
			files: []plugin.File{
				{Path: "out/user.go", Content: []byte("package user")},
			},
		}
		source := fmt.Sprintf(simpleStructTemplate, domainDirectives["go"])
		resp := MustGenerate(ctx, source, "user", loader, p)
		Expect(resp).NotTo(BeNil())
		Expect(resp.Files).To(HaveLen(1))
		Expect(resp.Files[0].Path).To(Equal("out/user.go"))
	})

	It("should pass the repo root through to the request", func(ctx SpecContext) {
		p := &reqCapturingPlugin{files: []plugin.File{
			{Path: "out/user.go", Content: []byte("package user")},
		}}
		source := fmt.Sprintf(simpleStructTemplate, domainDirectives["go"])
		MustGenerate(ctx, source, "user", loader, p)
		Expect(p.lastReq).NotTo(BeNil())
		Expect(p.lastReq.RepoRoot).To(Equal("/mock/repo"))
	})

	It("should pass resolutions to the plugin", func(ctx SpecContext) {
		p := &reqCapturingPlugin{files: []plugin.File{
			{Path: "out/user.go", Content: []byte("package user")},
		}}
		source := fmt.Sprintf(simpleStructTemplate, domainDirectives["go"])
		MustGenerate(ctx, source, "user", loader, p)
		Expect(p.lastReq.Resolutions).NotTo(BeNil())
	})

	It(
		"should return an empty response when plugin generates no files",
		func(ctx SpecContext) {
			p := &mockPlugin{files: []plugin.File{}}
			source := fmt.Sprintf(simpleStructTemplate, domainDirectives["go"])
			resp := MustGenerate(ctx, source, "user", loader, p)
			Expect(resp.Files).To(BeEmpty())
		},
	)

	It("should return multiple generated files", func(ctx SpecContext) {
		p := &mockPlugin{
			files: []plugin.File{
				{Path: "out/user.go", Content: []byte("package user")},
				{Path: "out/group.go", Content: []byte("package group")},
			},
		}
		source := fmt.Sprintf(multipleStructsTemplate, domainDirectives["go"])
		resp := MustGenerate(ctx, source, "multi", loader, p)
		Expect(resp.Files).To(HaveLen(2))
	})
})

var _ = Describe("DenyDirRead", func() {
	It("should make directory listings fail", func() {
		locked := filepath.Join(GinkgoT().TempDir(), "locked")
		Expect(os.Mkdir(locked, 0o755)).To(Succeed())
		DenyDirRead(locked)
		Expect(os.ReadDir(locked)).Error().To(MatchError(os.ErrPermission))
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
				Path: "out/user.go",
				Content: []byte(
					"package user\n\ntype User struct {\n\tKey uuid.UUID\n}",
				),
			})
			ExpectContent(resp, "user.go").
				ToContain("package user", "type User struct", "Key uuid.UUID")
		})

		It("should support chaining multiple ToContain calls", func() {
			resp := buildResponse(plugin.File{
				Path: "out/user.go",
				Content: []byte(
					"package user\n\ntype User struct {\n\tKey uuid.UUID\n\tName string\n}",
				),
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
				Path: "out/user.go",
				Content: []byte(
					"package user\n\nimport \"fmt\"\n\ntype User struct {}",
				),
			})
			ExpectContent(resp, "user.go").
				ToPreserveOrder("package user", "import", "type User struct")
		})

		It("should support chaining with other assertions", func() {
			resp := buildResponse(plugin.File{
				Path: "out/user.go",
				Content: []byte(
					"package user\n\ntype User struct {\n\tKey string\n\tName string\n}",
				),
			})
			ExpectContent(resp, "user.go").
				ToContain("package user").
				ToPreserveOrder("Key string", "Name string").
				ToNotContain("secret")
		})

		It("should verify ordering across many substrings", func() {
			resp := buildResponse(plugin.File{
				Path: "out/user.go",
				Content: []byte(
					"package user\n\nimport (\n\t\"fmt\"\n)\n\ntype User struct {\n\tA int\n\tB int\n\tC int\n}",
				),
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
