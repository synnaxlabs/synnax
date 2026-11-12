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
	"maps"
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
		"Should keep live rules out of a frozen definer the current version aliases",
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
			Expect(files["core/doc/versions/v0/types.gen.go"]).ToNot(ContainSubstring(
				`Eq("precision"`,
			))
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

var _ = Describe("Live default group on a frozen definer", func() {
	const liveSource = `
@go output "core/doc"

Range struct {
	min_val float64 = 0 {
		@default group "range"
	}
	max_val float64 = 1 {
		@default group "range"
	}
}

Doc struct {
	key uuid @key
	ranges Range[]
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
Range struct {
	min_val float64 = 0
	max_val float64 = 1

	@go marshal
}

Doc struct {
	key uuid @key
	ranges Range[]

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
		"Should keep a live default group out of an aliased frozen definer",
		func() {
			files := generate(`
Range = v0.Range

Doc struct {
	key uuid @key
	ranges Range[]
	name string

	@go marshal
}
`)
			Expect(files["core/doc/versions/v0/types.gen.go"]).ToNot(ContainSubstring(
				"r.MinVal == 0 && r.MaxVal == 0",
			))
			Expect(files["core/doc/versions/v1/types.gen.go"]).To(ContainSubstring(
				"type Range = v0.Range",
			))
		},
	)

	It(
		"Should leave a frozen definer ungrouped once a later version redeclares it",
		func() {
			files := generate(`
Range struct {
	min_val float64 = 0
	max_val float64 = 1

	@go marshal
}

Doc struct {
	key uuid @key
	ranges Range[]
	name string

	@go marshal
}
`)
			Expect(files["core/doc/versions/v0/types.gen.go"]).ToNot(ContainSubstring(
				"r.MinVal == 0 && r.MaxVal == 0",
			))
			Expect(files["core/doc/versions/v1/types.gen.go"]).To(ContainSubstring(
				"if r.MinVal == 0 && r.MaxVal == 0 {",
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
		"Should keep a live to_any out of a frozen definer the current version aliases",
		func(ctx SpecContext) {
			files := generate(ctx, `
Box = v0.Box
`)
			Expect(files["core/box/versions/v0/types.gen.go"]).ToNot(ContainSubstring(
				"ToAny",
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
		"Should keep live rules out of a frozen union the current version aliases",
		func() {
			files := generate("Payload = v0.Payload\n\nWindow = v0.Window\n" + docV1)
			frozen := files["core/doc/versions/v0/types.gen.go"]
			Expect(frozen).ToNot(ContainSubstring(`GreaterThanEq("span"`))
			Expect(frozen).ToNot(ContainSubstring(`GreaterThanEq("size"`))
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

var _ = Describe("Frozen enum that the live file omits", func() {
	generate := func(v0, v1, live string) *plugin.Response {
		GinkgoHelper()
		root := GinkgoT().TempDir()
		for rel, content := range map[string]string{
			"schemas/synnax/versions/doc/v0.oracle": v0,
			"schemas/synnax/versions/doc/v1.oracle": v1,
			"schemas/synnax/doc.oracle":             live,
		} {
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		chains := MustSucceed(versions.Discover(root))
		loader := analyzer.NewStandardFileLoader(root)
		table, diag := analyzer.Analyze(
			GinkgoT().Context(), []string{"schemas/synnax/doc"}, loader,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		return MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(
			&plugin.Request{
				Resolutions: table,
				RepoRoot:    root,
				Versions:    versions.NewResolver(chains, loader),
			},
		))
	}
	frozenV0 := func(v0, v1, live string) string {
		GinkgoHelper()
		return generatedFile(
			generate(v0, v1, live), "core/pkg/service/doc/versions/v0/types.gen.go",
		)
	}
	const (
		sharedSide = `
Side = v0.Side

Doc struct {
	key uuid @key

	@go marshal
}
`
		liveSide = `
@go output "core/pkg/service/doc"

Side enum {
	left = "left"

	@go omit
}

Doc struct {
	key uuid @key
}
`
	)
	DescribeTable("Should keep a shared enum that a frozen declaration uses",
		func(v0 string) {
			frozen := frozenV0(v0, sharedSide, liveSide)
			Expect(frozen).To(ContainSubstring("type Side string"))
		},
		Entry("through a field", `
Side enum {
	left = "left"
}

Doc struct {
	key uuid @key
	side Side

	@go marshal
}
`),
		Entry("as a union's discriminator", `
Side enum {
	left = "left"
}

Shape union on side Side {
	left {
		width int32
	}

	@go marshal
}

Doc struct {
	key uuid @key
	shape Shape

	@go marshal
}
`),
	)
	It("Should keep a shared enum that a later frozen version uses", func() {
		root := GinkgoT().TempDir()
		for rel, content := range map[string]string{
			"schemas/synnax/versions/doc/v0.oracle": `
Side enum {
	left = "left"
}

Doc struct {
	key uuid @key

	@go marshal
}
`,
			"schemas/synnax/versions/doc/v1.oracle": `
Side = v0.Side

Doc struct {
	key uuid @key
	side Side

	@go marshal
}
`,
			"schemas/synnax/versions/doc/v2.oracle": sharedSide,
			"schemas/synnax/doc.oracle":             liveSide,
		} {
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		chains := MustSucceed(versions.Discover(root))
		loader := analyzer.NewStandardFileLoader(root)
		table, diag := analyzer.Analyze(
			GinkgoT().Context(), []string{"schemas/synnax/doc"}, loader,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(
			&plugin.Request{
				Resolutions: table,
				RepoRoot:    root,
				Versions:    versions.NewResolver(chains, loader),
			},
		))
		Expect(generatedFile(resp, "core/pkg/service/doc/versions/v0/types.gen.go")).
			To(ContainSubstring("type Side string"))
		Expect(generatedFile(resp, "core/pkg/service/doc/versions/v1/types.gen.go")).
			To(ContainSubstring("Side = v0.Side"))
	})
	It("Should declare live-omitted members that only each other use", func() {
		frozen := frozenV0(`
Side enum {
	left = "left"
}

Note struct {
	side Side
}

Doc struct {
	key uuid @key

	@go marshal
}
`, `
Side = v0.Side

Note = v0.Note

Doc struct {
	key uuid @key
	name string

	@go marshal
}
`, `
@go output "core/pkg/service/doc"

Side enum {
	left = "left"

	@go omit
}

Note struct {
	side Side

	@go omit
}

Doc struct {
	key uuid @key
	name string
}
`)
		Expect(frozen).To(ContainSubstring("type Side string"))
		Expect(frozen).To(ContainSubstring("type Note struct"))
	})
	It("Should keep a shared enum that another resource's frozen version uses",
		func() {
			root := GinkgoT().TempDir()
			for rel, content := range map[string]string{
				"schemas/synnax/versions/doc/v0.oracle": `
Side enum {
	left = "left"
}

Doc struct {
	key uuid @key

	@go marshal
}
`,
				"schemas/synnax/versions/doc/v1.oracle": sharedSide,
				"schemas/synnax/doc.oracle":             liveSide,
				"schemas/synnax/versions/job/v0.oracle": `
import "schemas/synnax/versions/doc/v0"

Job struct {
	key uuid @key
	side doc.Side

	@go marshal
}
`,
				"schemas/synnax/versions/job/v1.oracle": `
Job struct {
	key uuid @key

	@go marshal
}
`,
				"schemas/synnax/job.oracle": `
@go output "core/pkg/service/job"

Job struct {
	key uuid @key
}
`,
			} {
				full := filepath.Join(root, rel)
				Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
				Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
			}
			chains := MustSucceed(versions.Discover(root))
			loader := analyzer.NewStandardFileLoader(root)
			table, diag := analyzer.Analyze(
				GinkgoT().Context(),
				[]string{"schemas/synnax/doc", "schemas/synnax/job"},
				loader,
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(
				&plugin.Request{
					Resolutions: table,
					RepoRoot:    root,
					Versions:    versions.NewResolver(chains, loader),
				},
			))
			Expect(generatedFile(
				resp, "core/pkg/service/job/versions/v0/types.gen.go",
			)).To(ContainSubstring(".Side"))
			Expect(generatedFile(
				resp, "core/pkg/service/doc/versions/v0/types.gen.go",
			)).To(ContainSubstring("type Side string"))
		},
	)
	It("Should declare a live-omitted member that no declaration uses", func() {
		resp := generate(`
Side enum {
	left = "left"
}

Doc struct {
	key uuid @key

	@go marshal
}
`, sharedSide, liveSide)
		Expect(generatedFile(resp, "core/pkg/service/doc/versions/v0/types.gen.go")).
			To(ContainSubstring("type Side string"))
		Expect(generatedFile(resp, "core/pkg/service/doc/versions/v1/types.gen.go")).
			To(ContainSubstring("Side = v0.Side"))
	})
	It("Should keep an enum that a frozen declaration uses or defines alone", func() {
		frozen := frozenV0(`
Side enum {
	left = "left"
}

Mode enum {
	slow = "slow"
}

Doc struct {
	key uuid @key
	side Side

	@go marshal
}
`, `
Mode enum {
	fast = "fast"
}

Doc struct {
	key uuid @key

	@go marshal
}
`, `
@go output "core/pkg/service/doc"

Side enum {
	left = "left"

	@go omit
}

Mode enum {
	fast = "fast"

	@go omit
}

Doc struct {
	key uuid @key
}
`)
		Expect(frozen).To(ContainSubstring("type Side string"))
		Expect(frozen).To(ContainSubstring("type Mode string"))
	})
})

var _ = Describe("Frozen omitted field to a dependency type with no root", func() {
	DescribeTable("Should keep the reference on the pinned version", func(
		v1Extra, liveExtra string,
	) {
		root := GinkgoT().TempDir()
		for rel, content := range map[string]string{
			"schemas/x/versions/color/v0.oracle": `
Old struct {
	r uint8
}

Color struct {
	r uint8

	@go marshal
}
`,
			"schemas/x/versions/color/v1.oracle": `
Color = v0.Color
` + v1Extra,
			"schemas/x/color.oracle": `
@go output "x/go/color"

Color struct {
	r uint8

	@go marshal
}
` + liveExtra,
			"schemas/synnax/versions/doc/v0.oracle": `
import "schemas/x/versions/color/v0"

Doc struct {
	key uuid @key
	tint color.Color
	old color.Old {
		@go marshal omit
	}

	@go marshal
}
`,
			"schemas/synnax/versions/doc/v1.oracle": `
import "schemas/x/versions/color/v1"

Doc struct {
	key uuid @key
	tint color.Color

	@go marshal
}
`,
			"schemas/synnax/doc.oracle": `
import "schemas/x/color"

@go output "core/pkg/service/doc"

Doc struct {
	key uuid @key
	tint color.Color
}
`,
		} {
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		chains := MustSucceed(versions.Discover(root))
		loader := analyzer.NewStandardFileLoader(root)
		table, diag := analyzer.Analyze(
			GinkgoT().Context(),
			[]string{"schemas/x/color", "schemas/synnax/doc"},
			loader,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(
			&plugin.Request{
				Resolutions: table,
				RepoRoot:    root,
				Versions:    versions.NewResolver(chains, loader),
			},
		))
		frozen := generatedFile(resp, "core/pkg/service/doc/versions/v0/types.gen.go")
		Expect(frozen).To(ContainSubstring("Old color.Old"))
		const colorPkg = `"github.com/synnaxlabs/synnax/x/go/color`
		Expect(frozen).To(ContainSubstring(`color ` + colorPkg + `/versions/v0"`))
		Expect(frozen).ToNot(ContainSubstring(colorPkg + `"`))
	},
		Entry("when the live file removes the type", "", ""),
		Entry("when the live file omits the type from Go",
			"\nOld = v0.Old\n", "\nOld struct {\n\tr uint8\n\n\t@go omit\n}\n"),
	)
})

var _ = Describe("Renamed predecessor aliases", func() {
	It(
		"Should alias the frozen names of a renamed enum and union",
		func(ctx SpecContext) {
			const livePath = "schemas/synnax/doc.oracle"
			resp := chainGenerate(
				ctx,
				gotypes.New(gotypes.DefaultOptions()),
				map[string]string{
					"schemas/synnax/versions/doc/v0.oracle": `
Mode enum {
	light = "light"
	dark  = "dark"
}

Shape union on kind {
	circle {
		radius int32
	}
	square {
		side int32
	}

	@go marshal
}

Doc struct {
	key uuid @key
	mode Mode
	shape Shape

	@go marshal
}
`,
					"schemas/synnax/versions/doc/v1.oracle": `
Mode = v0.Mode

Shape = v0.Shape

Doc struct {
	key uuid @key
	mode Mode
	shape Shape
	name string

	@go marshal
}
`,
					livePath: `
@go output "core/doc"

Mode enum {
	light = "light"
	dark  = "dark"

	@go name "Tone"
}

Shape union on kind {
	circle {
		radius int32
	}
	square {
		side int32
	}

	@go name "Form"
}

Doc struct {
	key uuid @key
	mode Mode
	shape Shape
	name string
}
`,
				},
				livePath,
				"doc",
			)
			frozen := generatedFile(resp, "core/doc/versions/v0/types.gen.go")
			Expect(frozen).To(ContainSubstring("type Mode string"))
			Expect(frozen).To(ContainSubstring("type Shape struct"))
			current := generatedFile(resp, "core/doc/versions/v1/types.gen.go")
			for _, line := range []string{
				"type Tone = v0.Mode",
				"ToneLight Tone = v0.ModeLight",
				"ToneDark Tone = v0.ModeDark",
				"type Form = v0.Shape",
				"type FormVariant = v0.ShapeVariant",
				"type FormType = v0.ShapeType",
				"CircleFormType FormType = v0.CircleShapeType",
				"SquareFormType FormType = v0.SquareShapeType",
				"type CircleForm = v0.CircleShape",
				"type SquareForm = v0.SquareShape",
			} {
				Expect(current).To(ContainSubstring(line))
			}
		},
	)
})

var _ = Describe("Frozen pins of a dependency's current and frozen versions", func() {
	It("Should take the live Go name and output only at the current version", func() {
		root := GinkgoT().TempDir()
		for rel, content := range map[string]string{
			"schemas/x/versions/color/v0.oracle": `
Color struct {
	r uint8

	@go marshal
}

Shade struct {
	r uint8
}
`,
			"schemas/x/versions/color/v1.oracle": `
Color struct {
	r uint8
	g uint8

	@go marshal
}

Shade = v0.Shade
`,
			"schemas/x/color.oracle": `
@go output "x/go/color"

Color struct {
	r uint8
	g uint8

	@go marshal
	@go name "Tint"
}

Shade struct {
	r uint8

	@go output "x/go/shade"
}
`,
			"schemas/synnax/versions/doc/v0.oracle": `
import "schemas/x/versions/color/v0"

Doc struct {
	key uuid @key
	tint color.Color
	shade color.Shade

	@go marshal
}
`,
			"schemas/synnax/versions/doc/v1.oracle": `
import "schemas/x/versions/color/v1"

Doc struct {
	key uuid @key
	tint color.Color
	shade color.Shade
	name string

	@go marshal
}
`,
			"schemas/synnax/versions/doc/v2.oracle": `
import "schemas/x/versions/color/v1"

Doc struct {
	key uuid @key
	tint color.Color
	shade color.Shade
	name string
	size int32

	@go marshal
}
`,
			"schemas/synnax/doc.oracle": `
import "schemas/x/color"

@go output "core/pkg/service/doc"

Doc struct {
	key uuid @key
	tint color.Color
	shade color.Shade
	name string
	size int32
}
`,
		} {
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		chains := MustSucceed(versions.Discover(root))
		loader := analyzer.NewStandardFileLoader(root)
		table, diag := analyzer.Analyze(
			GinkgoT().Context(),
			[]string{"schemas/x/color", "schemas/synnax/doc"},
			loader,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(
			&plugin.Request{
				Resolutions: table,
				RepoRoot:    root,
				Versions:    versions.NewResolver(chains, loader),
			},
		))
		const pkg = `"github.com/synnaxlabs/synnax/x/go/`
		currentPin := generatedFile(
			resp, "core/pkg/service/doc/versions/v1/types.gen.go",
		)
		Expect(currentPin).To(ContainSubstring("Tint color.Tint"))
		Expect(currentPin).To(ContainSubstring("Shade shade.Shade"))
		Expect(currentPin).To(ContainSubstring(pkg + `color/versions/v1"`))
		Expect(currentPin).To(ContainSubstring(pkg + `shade/versions/v1"`))
		frozenPin := generatedFile(
			resp, "core/pkg/service/doc/versions/v0/types.gen.go",
		)
		Expect(frozenPin).To(ContainSubstring("Tint color.Color"))
		Expect(frozenPin).To(ContainSubstring("Shade color.Shade"))
		Expect(frozenPin).To(ContainSubstring(pkg + `color/versions/v0"`))
		Expect(frozenPin).ToNot(ContainSubstring("x/go/shade"))
	})
})

var _ = Describe("Pinned dependency Go names", func() {
	const colorPkg = `"github.com/synnaxlabs/synnax/x/go/color/versions/`
	generate := func(liveColor string) *plugin.Response {
		GinkgoHelper()
		root := GinkgoT().TempDir()
		for rel, content := range map[string]string{
			"schemas/x/versions/color/v0.oracle": `
Old struct {
	r uint8
}

Shade struct {
	r uint8
}

Color struct {
	r uint8

	@go marshal
}
`,
			"schemas/x/versions/color/v1.oracle": `
Shade = v0.Shade

Color = v0.Color
`,
			"schemas/x/color.oracle": liveColor,
			"schemas/synnax/versions/doc/v0.oracle": `
import "schemas/x/versions/color/v0"

Doc struct {
	key uuid @key
	tint color.Color
	shade color.Shade
	old color.Old {
		@go marshal omit
	}

	@go marshal
}
`,
			"schemas/synnax/versions/doc/v1.oracle": `
import "schemas/x/versions/color/v1"

Doc struct {
	key uuid @key
	tint color.Color
	shade color.Shade

	@go marshal
}
`,
			"schemas/synnax/doc.oracle": `
import "schemas/x/color"

@go output "core/pkg/service/doc"

Doc struct {
	key uuid @key
	tint color.Color
	shade color.Shade
}
`,
		} {
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		chains := MustSucceed(versions.Discover(root))
		loader := analyzer.NewStandardFileLoader(root)
		table, diag := analyzer.Analyze(
			GinkgoT().Context(),
			[]string{"schemas/x/color", "schemas/synnax/doc"},
			loader,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		return MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(
			&plugin.Request{
				Resolutions: table,
				RepoRoot:    root,
				Versions:    versions.NewResolver(chains, loader),
			},
		))
	}

	It("Should keep a frozen pin on the name its version file declares", func() {
		resp := generate(`
@go output "x/go/color"

Shade struct {
	r uint8
}

Color struct {
	r uint8

	@go marshal
	@go name "Tint"
}
`)
		Expect(generatedFile(resp, "x/go/color/versions/v0/types.gen.go")).
			To(ContainSubstring("type Color struct"))
		Expect(generatedFile(resp, "x/go/color/versions/v1/types.gen.go")).
			To(ContainSubstring("type Tint = v0.Color"))
		frozen := generatedFile(resp, "core/pkg/service/doc/versions/v0/types.gen.go")
		Expect(frozen).To(ContainSubstring("Tint color.Color"))
		Expect(frozen).ToNot(ContainSubstring("color.Tint"))
		Expect(generatedFile(resp, "core/pkg/service/doc/versions/v1/types.gen.go")).
			To(ContainSubstring("Tint color.Tint"))
	})

	It(
		"Should keep a frozen pin at the chain root when the live file moves a type",
		func() {
			resp := generate(`
@go output "x/go/color"

Color struct {
	r uint8

	@go marshal
}

Shade struct {
	r uint8

	@go output "x/go/shade"
}
`)
			Expect(generatedFile(resp, "x/go/color/versions/v0/types.gen.go")).
				To(ContainSubstring("type Shade struct"))
			frozen := generatedFile(
				resp,
				"core/pkg/service/doc/versions/v0/types.gen.go",
			)
			Expect(frozen).To(ContainSubstring("Shade color.Shade"))
			Expect(frozen).To(ContainSubstring(`color ` + colorPkg + `v0"`))
			Expect(frozen).ToNot(ContainSubstring("x/go/shade"))
		},
	)

	It(
		"Should keep a frozen pin at the chain root when the live file drops a type",
		func() {
			resp := generate(`
@go output "x/go/color"

Color struct {
	r uint8

	@go marshal
}

Shade struct {
	r uint8
}
`)
			frozen := generatedFile(
				resp,
				"core/pkg/service/doc/versions/v0/types.gen.go",
			)
			Expect(frozen).To(ContainSubstring("Old color.Old"))
			Expect(frozen).To(ContainSubstring(`color ` + colorPkg + `v0"`))
		},
	)
})

var _ = Describe("Frozen omitted field with a default", func() {
	DescribeTable("Should resolve the default against the same table as the field",
		func(decls, field string) {
			root := GinkgoT().TempDir()
			for rel, content := range map[string]string{
				"schemas/x/versions/color/v0.oracle": `
Mode enum {
	light = "light"
}
`,
				"schemas/x/versions/color/v1.oracle": `
Mode enum {
	light = "light"
	dark  = "dark"
}
`,
				"schemas/x/color.oracle": `
@go output "x/go/color"

Mode enum {
	light = "light"
	dark  = "dark"
}
`,
				"schemas/synnax/versions/doc/v0.oracle": `
import "schemas/x/versions/color/v0"
` + decls + `
Doc struct {
	key uuid @key
	` + field + `

	@go marshal
}
`,
				"schemas/synnax/versions/doc/v1.oracle": `
import "schemas/x/versions/color/v1"
` + decls + `
Doc struct {
	key uuid @key
	` + field + `
	name string

	@go marshal
}
`,
				"schemas/synnax/doc.oracle": `
import "schemas/x/color"

@go output "core/pkg/service/doc"
` + decls + `
Doc struct {
	key uuid @key
	` + field + `
	name string
}
`,
			} {
				full := filepath.Join(root, rel)
				Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
				Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
			}
			chains := MustSucceed(versions.Discover(root))
			loader := analyzer.NewStandardFileLoader(root)
			table, diag := analyzer.Analyze(
				GinkgoT().Context(),
				[]string{"schemas/x/color", "schemas/synnax/doc"},
				loader,
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(
				&plugin.Request{
					Resolutions: table,
					RepoRoot:    root,
					Versions:    versions.NewResolver(chains, loader),
				},
			))
			frozen := generatedFile(
				resp, "core/pkg/service/doc/versions/v0/types.gen.go",
			)
			Expect(frozen).To(ContainSubstring("Mode color.Mode"))
			Expect(frozen).To(ContainSubstring("= color.ModeLight"))
			Expect(frozen).To(ContainSubstring(".Mode.IsValid()"))
		},
		Entry("on a struct", "", `mode color.Mode = light {
		@go marshal omit
	}`),
		Entry("on an inline union variant", `
Shape union on kind {
	circle {
		mode color.Mode = light {
			@go marshal omit
		}
	}

	@go marshal
}
`, "shape Shape"),
	)
})

var _ = Describe("Minting on top of a version", func() {
	generate := func(files map[string]string, live []string) *plugin.Response {
		GinkgoHelper()
		root := GinkgoT().TempDir()
		for rel, content := range files {
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}
		chains := MustSucceed(versions.Discover(root))
		loader := analyzer.NewStandardFileLoader(root)
		table, diag := analyzer.Analyze(GinkgoT().Context(), live, loader)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		return MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(
			&plugin.Request{
				Resolutions: table,
				RepoRoot:    root,
				Versions:    versions.NewResolver(chains, loader),
			},
		))
	}
	const (
		docV0 = "core/pkg/service/doc/versions/v0/types.gen.go"
		docV1 = "core/pkg/service/doc/versions/v1/types.gen.go"
		color = `
Color struct {
	r uint8

	@go marshal
}
`
		colorLive = `
@go output "x/go/color"

Color struct {
	r uint8

	@go marshal
}
`
	)
	DescribeTable("Should leave the earlier version's file byte-identical",
		func(demoted, mintPath, mint string, files map[string]string, live ...string) {
			before := generatedFile(generate(files, live), demoted)
			minted := maps.Clone(files)
			minted[mintPath] = mint
			Expect(generatedFile(generate(minted, live), demoted)).To(Equal(before))
		},
		Entry("when the live file declares in another order",
			docV0, "schemas/synnax/versions/doc/v1.oracle", `
Note = v0.Note

Doc = v0.Doc
`, map[string]string{
				"schemas/synnax/versions/doc/v0.oracle": `
Note struct {
	text string
}

Doc struct {
	key uuid @key
	note Note

	@go marshal
}
`,
				"schemas/synnax/doc.oracle": `
@go output "core/pkg/service/doc"

Doc struct {
	key uuid @key
	note Note
}

Note struct {
	text string
}
`,
			}, "schemas/synnax/doc"),
		Entry("when the live file omits a declaration",
			docV0, "schemas/synnax/versions/doc/v1.oracle", `
Side = v0.Side

Note = v0.Note

Doc = v0.Doc
`, map[string]string{
				"schemas/synnax/versions/doc/v0.oracle": `
Side enum {
	left = "left"
}

Note struct {
	text string
}

Doc struct {
	key uuid @key

	@go marshal
}
`,
				"schemas/synnax/doc.oracle": `
@go output "core/pkg/service/doc"

Side enum {
	left = "left"

	@go omit
}

Note struct {
	text string

	@go omit
}

Doc struct {
	key uuid @key
}
`,
			}, "schemas/synnax/doc"),
		Entry("when the new version drops a stored enum",
			docV0, "schemas/synnax/versions/doc/v1.oracle", `
Doc struct {
	key uuid @key

	@go marshal
}
`, map[string]string{
				"schemas/synnax/versions/doc/v0.oracle": `
Side enum {
	left = "left"
}

Doc struct {
	key uuid @key
	side Side

	@go marshal
}
`,
				"schemas/synnax/doc.oracle": `
@go output "core/pkg/service/doc"

Side enum {
	left = "left"
}

Doc struct {
	key uuid @key
	side Side
}
`,
			}, "schemas/synnax/doc"),
		Entry("when an omitted field resolves a dependency",
			docV0, "schemas/synnax/versions/doc/v1.oracle", `
import "schemas/x/versions/color/v0"

Doc struct {
	key uuid @key
	tint color.Color {
		@go marshal omit
	}
	name string

	@go marshal
}
`, map[string]string{
				"schemas/x/versions/color/v0.oracle": color,
				"schemas/x/color.oracle":             colorLive,
				"schemas/synnax/versions/doc/v0.oracle": `
import "schemas/x/versions/color/v0"

Doc struct {
	key uuid @key
	tint color.Color {
		@go marshal omit
	}

	@go marshal
}
`,
				"schemas/synnax/doc.oracle": `
import "schemas/x/color"

@go output "core/pkg/service/doc"

Doc struct {
	key uuid @key
	tint color.Color {
		@go marshal omit
	}

	@go marshal
}
`,
			}, "schemas/x/color", "schemas/synnax/doc"),
		Entry("when only another resource stores a declaration",
			docV0, "schemas/synnax/versions/doc/v1.oracle", `
Config = v0.Config

Doc = v0.Doc
`, map[string]string{
				"schemas/x/versions/color/v0.oracle": color,
				"schemas/x/color.oracle":             colorLive,
				"schemas/synnax/versions/doc/v0.oracle": `
import "schemas/x/versions/color/v0"

Config struct {
	tint color.Color
}

Doc struct {
	key uuid @key

	@go marshal
}
`,
				"schemas/synnax/doc.oracle": `
import "schemas/x/color"

@go output "core/pkg/service/doc"

Config struct {
	tint color.Color
}

Doc struct {
	key uuid @key

	@go marshal
}
`,
				"schemas/synnax/versions/job/v0.oracle": `
import "schemas/synnax/versions/doc/v0"

Job struct {
	key uuid @key
	config doc.Config

	@go marshal
}
`,
				"schemas/synnax/job.oracle": `
import "schemas/synnax/doc"

@go output "core/pkg/service/job"

Job struct {
	key uuid @key
	config doc.Config

	@go marshal
}
`,
			}, "schemas/x/color", "schemas/synnax/doc", "schemas/synnax/job"),
	)

	It("Should drop live rules and Go fields from the demoted version", func() {
		files := map[string]string{
			"schemas/synnax/versions/doc/v0.oracle": `
Doc struct {
	key uuid @key
	name string

	@go marshal
}
`,
			"schemas/synnax/doc.oracle": `
@go output "core/pkg/service/doc"

Doc struct {
	key uuid @key
	name string {
		@validate required
	}

	@go fields  "Size int ` + "`json:\\\"-\\\"`" + `"
}
`,
		}
		Expect(generatedFile(generate(files, []string{"schemas/synnax/doc"}), docV0)).
			To(And(ContainSubstring(`NotEmptyString("name"`), ContainSubstring("Size int")))
		files["schemas/synnax/versions/doc/v1.oracle"] = "\nDoc = v0.Doc\n"
		demoted := generatedFile(generate(files, []string{"schemas/synnax/doc"}), docV0)
		Expect(demoted).ToNot(ContainSubstring(`NotEmptyString("name"`))
		Expect(demoted).ToNot(ContainSubstring("Size int"))
	})
})

var _ = Describe("Frozen type that only a frozen version embeds", func() {
	It(
		"Should give the frozen type defaults but no decode hooks",
		func(ctx SpecContext) {
			root := GinkgoT().TempDir()
			write := func(rel, content string) {
				GinkgoHelper()
				full := filepath.Join(root, rel)
				Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
				Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
			}
			write("schemas/synnax/versions/cfg/v0.oracle", `
Base struct {
	rate float64 = 0.2
}

Cfg struct {
	key uuid @key
	name string

	@go marshal
}
`)
			write("schemas/synnax/versions/cfg/v1.oracle", `
Base = v0.Base

Cfg struct extends Base {
	key uuid @key
	name string

	@go marshal
}
`)
			write("schemas/synnax/versions/cfg/v2.oracle", `
Base = v0.Base

Cfg struct {
	key uuid @key
	name string
	rate float64 = 0.2

	@go marshal
}
`)
			chains := MustSucceed(versions.Discover(root))
			resolver := versions.NewResolver(
				chains,
				analyzer.NewStandardFileLoader(root),
			)
			table := resolution.NewTable()
			diag := analyzer.AnalyzeSeeded(ctx, `
@go output "core/pkg/service/cfg"

Base struct {
	rate float64 = 0.2
}

Cfg struct {
	key uuid @key
	name string
	rate float64 = 0.2
}
`, "schemas/synnax/cfg.oracle", "cfg", analyzer.NewStandardFileLoader(root), table)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    root,
				Versions:    resolver,
			}
			resp := MustSucceed(gotypes.New(gotypes.DefaultOptions()).Generate(req))
			files := make(map[string]string)
			for _, f := range resp.Files {
				files[f.Path] = string(f.Content)
			}
			frozen := files["core/pkg/service/cfg/versions/v0/types.gen.go"]
			Expect(frozen).To(ContainSubstring("func (b *Base) SetDefaults() {"))
			Expect(frozen).ToNot(ContainSubstring("func (b *Base) UnmarshalJSONFrom("))
			Expect(frozen).ToNot(ContainSubstring("func (b *Base) DecodeMsgpack("))
		},
	)
})
