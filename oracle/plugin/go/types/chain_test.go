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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

const chainLiveSource = `
@go output "core/pkg/service/channel"

Key = uuid

Channel struct {
	key Key @key
	name string
	virtual bool
}
`

var _ = Describe("Chain-driven alias split", func() {
	var req *plugin.Request

	BeforeEach(func() {
		root := GinkgoT().TempDir()
		write := func(rel, content string) {
			GinkgoHelper()
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("schemas/synnax/versions/channel/v0.oracle", `
Key = uuid

Channel struct {
	key Key @key
	name string

	@go marshal
}
`)
		write("schemas/synnax/versions/channel/v1.oracle", `
Key = v0.Key

Channel struct {
	key Key @key
	name string
	virtual bool

	@go marshal
}
`)
		chains := MustSucceed(versions.Discover(root))
		resolver := versions.NewResolver(
			chains, analyzer.NewStandardFileLoader(root),
		)
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), chainLiveSource,
			"schemas/synnax/channel.oracle", "channel",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		req = &plugin.Request{Resolutions: table, RepoRoot: root, Versions: resolver}
	})

	It("Should alias per the current file's enumeration", func() {
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(req))
		var current string
		for _, f := range resp.Files {
			if f.Path == "core/pkg/service/channel/versions/v1/types.gen.go" {
				current = string(f.Content)
			}
		}
		Expect(current).ToNot(BeEmpty())
		Expect(current).To(ContainSubstring("type Key = v0.Key"))
		Expect(current).To(ContainSubstring("type Channel struct"))
	})

	It("Should regenerate frozen packages from the version files", func() {
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(req))
		var frozen string
		for _, f := range resp.Files {
			if f.Path == "core/pkg/service/channel/versions/v0/types.gen.go" {
				frozen = string(f.Content)
			}
		}
		Expect(frozen).ToNot(BeEmpty())
		Expect(frozen).To(ContainSubstring("package v0"))
		Expect(frozen).To(ContainSubstring("type Key = uuid.UUID"))
		Expect(frozen).To(ContainSubstring("type Channel struct"))
		// The frozen shape is v0's: no Virtual field, no alias lines.
		Expect(frozen).ToNot(ContainSubstring("Virtual"))
		Expect(frozen).ToNot(ContainSubstring("= v"))
	})
})

var _ = Describe("Frozen union with a borrowed enum", func() {
	It("Should type the frozen discriminator with the frozen enum", func() {
		root := GinkgoT().TempDir()
		write := func(rel, content string) {
			GinkgoHelper()
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		const decls = `
Method enum {
	password = "password"
}

Credentials struct {
	username string
}

Proof union on method Method {
	password Credentials

	@go marshal
}
`
		write("schemas/synnax/versions/auth/v0.oracle", decls)
		write("schemas/synnax/versions/auth/v1.oracle", `
Method enum {
	password = "password"
	api_key  = "api_key"
}

Credentials = v0.Credentials

Proof union on method Method {
	password v0.Credentials
	api_key  v0.Credentials

	@go marshal
}
`)
		chains := MustSucceed(versions.Discover(root))
		resolver := versions.NewResolver(
			chains, analyzer.NewStandardFileLoader(root),
		)
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), `@go output "core/pkg/service/auth"`+"\n"+`
Method enum {
	password = "password"
	api_key  = "api_key"
}

Credentials struct {
	username string
}

Proof union on method Method {
	password Credentials
	api_key  Credentials

	@go marshal
}
`,
			"schemas/synnax/auth.oracle", "auth",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		req := &plugin.Request{Resolutions: table, RepoRoot: root, Versions: resolver}
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(req))
		var frozen string
		for _, f := range resp.Files {
			if f.Path == "core/pkg/service/auth/versions/v0/types.gen.go" {
				frozen = string(f.Content)
			}
		}
		Expect(frozen).ToNot(BeEmpty())
		Expect(frozen).To(ContainSubstring("Type Method `json:\"method\"`"))
		Expect(frozen).To(ContainSubstring("MethodPassword Method"))
		Expect(frozen).ToNot(ContainSubstring("ProofType"))
	})
})

var _ = Describe("Ended chain", func() {
	var req *plugin.Request

	BeforeEach(func() {
		root := GinkgoT().TempDir()
		write := func(rel, content string) {
			GinkgoHelper()
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("schemas/synnax/versions/thing/v0.oracle", `
Thing struct {
	key uuid @key
	name string

	@go marshal
}
`)
		write("schemas/synnax/versions/thing/v1.oracle", "// gone\n")
		chains := MustSucceed(versions.Discover(root))
		resolver := versions.NewResolver(
			chains, analyzer.NewStandardFileLoader(root),
		)
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), `
@go output "core/thing"

Thing struct {
	key uuid @key
	name string
}
`,
			"schemas/synnax/thing.oracle", "thing",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		req = &plugin.Request{Resolutions: table, RepoRoot: root, Versions: resolver}
	})

	It("Should keep frozen packages generating without a current package", func() {
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(req))
		byPath := make(map[string]string, len(resp.Files))
		for _, f := range resp.Files {
			byPath[f.Path] = string(f.Content)
		}
		frozen := byPath["core/thing/versions/v0/types.gen.go"]
		Expect(frozen).To(ContainSubstring("package v0"))
		Expect(frozen).To(ContainSubstring("type Thing struct"))
		// The live type generates concretely at the package root; no package
		// exists for the tombstone and no root alias files point into versions.
		Expect(byPath["core/thing/types.gen.go"]).
			To(ContainSubstring("type Thing struct"))
		for path := range byPath {
			Expect(path).ToNot(ContainSubstring("thing/versions/v1"))
			Expect(path).ToNot(Equal("core/thing/versions/types.gen.go"))
		}
	})
})

var _ = Describe("Frozen hand-written dependency", func() {
	var req *plugin.Request

	BeforeEach(func() {
		root := GinkgoT().TempDir()
		write := func(rel, content string) {
			GinkgoHelper()
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("schemas/synnax/versions/channel/v0.oracle", `
Key uint32 {
	@go hand
}
`)
		write("schemas/synnax/channel.oracle", `
@go output "core/pkg/service/channel"

Key uint32 {
	@go hand
}
`)
		write("schemas/synnax/versions/log/v0.oracle", `
import "schemas/synnax/versions/channel/v0"

Log struct {
	key uuid @key
	channel channel.Key

	@go marshal
}
`)
		write("schemas/synnax/versions/log/v1.oracle", `
import "schemas/synnax/versions/channel/v0"

Log struct {
	key uuid @key
	channel channel.Key
	name string

	@go marshal
}
`)
		write("schemas/synnax/log.oracle", `
import "schemas/synnax/channel"

@go output "core/pkg/service/log"

Log struct {
	key uuid @key
	channel channel.Key
	name string
}
`)
		chains := MustSucceed(versions.Discover(root))
		loader := analyzer.NewStandardFileLoader(root)
		table, diag := analyzer.Analyze(
			GinkgoT().Context(),
			[]string{"schemas/synnax/channel", "schemas/synnax/log"},
			loader,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		req = &plugin.Request{
			Resolutions: table,
			RepoRoot:    root,
			Versions:    versions.NewResolver(chains, loader),
		}
	})

	// channel v0 holds its own hand-written Key, so a frozen package pinning v0 must
	// import it there and not follow the live type.
	It("Should import the pinned version directory of a hand-written type", func() {
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(req))
		var frozen string
		for _, f := range resp.Files {
			if f.Path == "core/pkg/service/log/versions/v0/types.gen.go" {
				frozen = string(f.Content)
			}
		}
		Expect(frozen).To(ContainSubstring(`service/channel/versions/v0"`))
		Expect(frozen).ToNot(ContainSubstring(`service/channel"`))
	})
})

var _ = Describe("Live validation on a frozen definer", func() {
	const liveSource = `
@go output "core/doc"

Entry struct {
	precision int32 {
		@validate min -1
		@validate max 17
	}
}

Doc struct {
	key uuid @key
	entries Entry[]
	name string
}
`
	generate := func(v1 string) map[string]string {
		GinkgoHelper()
		root := GinkgoT().TempDir()
		write := func(rel, content string) {
			GinkgoHelper()
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("schemas/synnax/versions/doc/v0.oracle", `
Entry struct {
	precision int32

	@go marshal
}

Doc struct {
	key uuid @key
	entries Entry[]

	@go marshal
}
`)
		write("schemas/synnax/versions/doc/v1.oracle", v1)
		chains := MustSucceed(versions.Discover(root))
		resolver := versions.NewResolver(chains, analyzer.NewStandardFileLoader(root))
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), liveSource, "schemas/synnax/doc.oracle", "doc",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		req := &plugin.Request{Resolutions: table, RepoRoot: root, Versions: resolver}
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(req))
		byPath := make(map[string]string, len(resp.Files))
		for _, f := range resp.Files {
			byPath[f.Path] = string(f.Content)
		}
		return byPath
	}

	It(
		"Should validate a type the current version aliases from a frozen definer",
		func() {
			files := generate(`
Entry = v0.Entry

Doc struct {
	key uuid @key
	entries Entry[]
	name string

	@go marshal
}
`)
			frozen := files["core/doc/versions/v0/types.gen.go"]
			Expect(
				frozen,
			).To(ContainSubstring(`v.GreaterThanEq("precision", e.Precision, -1)`))
			Expect(
				frozen,
			).To(ContainSubstring(`v.LessThanEq("precision", e.Precision, 17)`))
			Expect(files["core/doc/versions/v1/types.gen.go"]).To(ContainSubstring(
				"type Entry = v0.Entry",
			))
		},
	)

	It(
		"Should leave a frozen definer unvalidated once a later version redeclares it",
		func() {
			files := generate(`
Entry struct {
	precision int32

	@go marshal
}

Doc struct {
	key uuid @key
	entries Entry[]
	name string

	@go marshal
}
`)
			Expect(files["core/doc/versions/v0/types.gen.go"]).ToNot(ContainSubstring(
				`Eq("precision"`,
			))
			Expect(files["core/doc/versions/v1/types.gen.go"]).To(ContainSubstring(
				`v.LessThanEq("precision", e.Precision, 17)`,
			))
		},
	)
})

var _ = Describe("Live to_any on a frozen definer", func() {
	const livePath = "schemas/synnax/box.oracle"
	generate := func(ctx SpecContext, v1 string) map[string]string {
		GinkgoHelper()
		resp := chainGenerate(
			ctx,
			gotypes.New(gotypes.DefaultOptions()),
			map[string]string{
				livePath: `
@go output "core/box"

Box struct<T> {
	key uuid @key
	value T
	name string

	@go to_any
}
`,
				"schemas/synnax/versions/box/v0.oracle": `
Box struct<T> {
	key uuid @key
	value T
	name string

	@go marshal
}
`,
				"schemas/synnax/versions/box/v1.oracle": v1,
			},
			livePath,
			"box",
		)
		byPath := make(map[string]string, len(resp.Files))
		for _, f := range resp.Files {
			byPath[f.Path] = string(f.Content)
		}
		return byPath
	}

	It(
		"Should convert a type the current version aliases from a frozen definer",
		func(ctx SpecContext) {
			files := generate(ctx, `
Box = v0.Box
`)
			Expect(files["core/box/versions/v0/types.gen.go"]).To(ContainSubstring(
				"func (b Box[T]) ToAny() Box[any] {",
			))
			current := files["core/box/versions/v1/types.gen.go"]
			Expect(current).To(ContainSubstring("type Box[T any] = v0.Box[T]"))
			Expect(current).ToNot(ContainSubstring("ToAny"))
		},
	)

	It(
		"Should leave a frozen definer unconverted once a later version redeclares it",
		func(ctx SpecContext) {
			files := generate(ctx, `
Box struct<T> {
	key uuid @key
	value T
	name string

	@go marshal
}
`)
			Expect(files["core/box/versions/v0/types.gen.go"]).ToNot(ContainSubstring(
				"ToAny",
			))
			Expect(files["core/box/versions/v1/types.gen.go"]).To(ContainSubstring(
				"func (b Box[T]) ToAny() Box[any] {",
			))
		},
	)
})

var _ = Describe("Live validation on a frozen union", func() {
	const v0Source = `
Payload struct {
	size int32

	@go marshal
}

Window union on variant {
	rolling {
		span int32
	}
	named Payload

	@go marshal
}

Doc struct {
	key uuid @key
	window Window

	@go marshal
}
`
	const liveSource = `
@go output "core/doc"

Payload struct {
	size int32 {
		@validate min 1
	}
}

Window union on variant {
	rolling {
		span int32 {
			@validate min 1
		}
	}
	named Payload
}

Doc struct {
	key uuid @key
	window Window
	name string
}
`
	generate := func(v1 string) map[string]string {
		GinkgoHelper()
		root := GinkgoT().TempDir()
		write := func(rel, content string) {
			GinkgoHelper()
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("schemas/synnax/versions/doc/v0.oracle", v0Source)
		write("schemas/synnax/versions/doc/v1.oracle", v1)
		chains := MustSucceed(versions.Discover(root))
		resolver := versions.NewResolver(chains, analyzer.NewStandardFileLoader(root))
		table := resolution.NewTable()
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), liveSource, "schemas/synnax/doc.oracle", "doc",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		req := &plugin.Request{Resolutions: table, RepoRoot: root, Versions: resolver}
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(req))
		byPath := make(map[string]string, len(resp.Files))
		for _, f := range resp.Files {
			byPath[f.Path] = string(f.Content)
		}
		return byPath
	}
	const docV1 = `
Doc struct {
	key uuid @key
	window Window
	name string

	@go marshal
}
`

	It(
		"Should validate the variant fields of a union the current version aliases",
		func() {
			files := generate("Payload = v0.Payload\n\nWindow = v0.Window\n" + docV1)
			frozen := files["core/doc/versions/v0/types.gen.go"]
			Expect(frozen).To(ContainSubstring(`GreaterThanEq("span"`))
			Expect(frozen).To(ContainSubstring(`GreaterThanEq("size"`))
		},
	)

	It(
		"Should leave a frozen union unvalidated once a later version redeclares it",
		func() {
			files := generate(`
Payload struct {
	size int32

	@go marshal
}

Window union on variant {
	rolling {
		span int32
	}
	named Payload

	@go marshal
}
` + docV1)
			Expect(files["core/doc/versions/v0/types.gen.go"]).ToNot(ContainSubstring(
				`GreaterThanEq("span"`,
			))
			Expect(files["core/doc/versions/v1/types.gen.go"]).To(ContainSubstring(
				`GreaterThanEq("span"`,
			))
		},
	)
})

var _ = Describe("Stored reference through a lagging pin", func() {
	var req *plugin.Request

	BeforeEach(func() {
		root := GinkgoT().TempDir()
		write := func(rel, content string) {
			GinkgoHelper()
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		write("schemas/x/versions/color/v0.oracle", `
Color struct {
	r uint8

	@go marshal
}
`)
		write("schemas/x/versions/color/v1.oracle", `
Color = v0.Color

Band struct {
	color Color

	@go marshal
}
`)
		write("schemas/x/color.oracle", `
@go output "x/go/color"

Color struct {
	r uint8
}

Band struct {
	color Color
}
`)
		write("schemas/synnax/versions/log/v0.oracle", `
import "schemas/x/versions/color/v0"

Log struct {
	key uuid @key
	color color.Color

	@go marshal
}
`)
		write("schemas/synnax/log.oracle", `
import "schemas/x/color"

@go output "core/pkg/service/log"

Log struct {
	key uuid @key
	color color.Color
}
`)
		chains := MustSucceed(versions.Discover(root))
		loader := analyzer.NewStandardFileLoader(root)
		table, diag := analyzer.Analyze(
			GinkgoT().Context(),
			[]string{"schemas/x/color", "schemas/synnax/log"},
			loader,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		req = &plugin.Request{
			Resolutions: table,
			RepoRoot:    root,
			Versions:    versions.NewResolver(chains, loader),
		}
	})

	file := func(resp *plugin.Response, path string) string {
		GinkgoHelper()
		for _, f := range resp.Files {
			if f.Path == path {
				return string(f.Content)
			}
		}
		Fail("no generated file " + path)
		return ""
	}

	// log v0 still pins color v0, which color v1 only aliases, so the current log
	// package imports v0, the version its version file imports.
	It("Should import the pinned version from a current package", func() {
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(req))
		current := file(resp, "core/pkg/service/log/versions/v0/types.gen.go")
		Expect(current).To(ContainSubstring(`x/go/color/versions/v0"`))
		Expect(current).ToNot(ContainSubstring(`x/go/color/versions/v1"`))
	})

	It("Should keep the alias in the dependency's current package", func() {
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(req))
		dep := file(resp, "x/go/color/versions/v1/types.gen.go")
		Expect(dep).To(ContainSubstring("type Color = v0.Color"))
		Expect(dep).To(ContainSubstring("Color Color"))
	})
})

// writePinFixture writes a log whose v0 declares Note against color v0 and whose v1
// declares Log against color v1, where color v1 only aliases v0's Color. With frozen
// set, a log v2 aliases both, freezing v1.
func writePinFixture(root string, frozen bool) {
	GinkgoHelper()
	write := func(rel, content string) {
		GinkgoHelper()
		full := filepath.Join(root, rel)
		Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
		Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
	}
	write("schemas/x/versions/color/v0.oracle", `
Color struct {
	r uint8

	@go marshal
}
`)
	write("schemas/x/versions/color/v1.oracle", `
Color = v0.Color

Band struct {
	color Color

	@go marshal
}
`)
	write("schemas/x/color.oracle", `
@go output "x/go/color"

Color struct {
	r uint8

	@go marshal
}

Band struct {
	color Color

	@go marshal
}
`)
	write("schemas/synnax/versions/log/v0.oracle", `
import "schemas/x/versions/color/v0"

Note struct {
	tint color.Color

	@go marshal
}

Log struct {
	key uuid @key
	note Note

	@go marshal
}
`)
	write("schemas/synnax/versions/log/v1.oracle", `
import "schemas/x/versions/color/v1"

Note = v0.Note

Log struct {
	key uuid @key
	note Note
	colors color.Color[]
	bands color.Band[]

	@go marshal
}
`)
	if frozen {
		write("schemas/synnax/versions/log/v2.oracle", `
Note = v0.Note

Log = v1.Log
`)
	}
	write("schemas/synnax/log.oracle", `
import "schemas/x/color"

@go output "core/pkg/service/log"

Note struct {
	tint color.Color

	@go marshal
}

Log struct {
	key uuid @key
	note Note
	colors color.Color[]
	bands color.Band[]

	@go marshal
}
`)
}

// pinRequest analyzes the fixture writePinFixture writes.
func pinRequest(frozen bool) *plugin.Request {
	GinkgoHelper()
	root := GinkgoT().TempDir()
	writePinFixture(root, frozen)
	chains := MustSucceed(versions.Discover(root))
	loader := analyzer.NewStandardFileLoader(root)
	table, diag := analyzer.Analyze(
		GinkgoT().Context(),
		[]string{"schemas/x/color", "schemas/synnax/log"},
		loader,
	)
	Expect(diag.Ok()).To(BeTrue(), diag.String())
	return &plugin.Request{
		Resolutions: table,
		RepoRoot:    root,
		Versions:    versions.NewResolver(chains, loader),
	}
}

// generatedFile returns the content resp holds at path.
func generatedFile(resp *plugin.Response, path string) string {
	GinkgoHelper()
	for _, f := range resp.Files {
		if f.Path == path {
			return string(f.Content)
		}
	}
	Fail("no generated file " + path)
	return ""
}

var _ = Describe("Stored references through the importing version file", func() {
	// log v1 imports color v1, which aliases v0's Color, so the package references
	// color v1 only, and freezing v1 keeps that import.
	DescribeTable("Should import the version that log v1 imports",
		func(frozen bool) {
			resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).
				Generate(pinRequest(frozen)))
			v1 := generatedFile(resp, "core/pkg/service/log/versions/v1/types.gen.go")
			Expect(v1).To(MatchRegexp(`\scolor "[^"]*/color/versions/v1"`))
			Expect(v1).ToNot(ContainSubstring(`color/versions/v0"`))
			Expect(v1).To(MatchRegexp(`\sv0 "[^"]*/log/versions/v0"`))
		},
		Entry("current", false),
		Entry("frozen", true),
	)

	It("Should import the version that log v0 imports", func() {
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).
			Generate(pinRequest(false)))
		v0 := generatedFile(resp, "core/pkg/service/log/versions/v0/types.gen.go")
		Expect(v0).To(ContainSubstring(`x/go/color/versions/v0"`))
		Expect(v0).ToNot(ContainSubstring(`color/versions/v1"`))
	})
})
