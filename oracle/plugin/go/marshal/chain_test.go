// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package marshal_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/marshal"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Chain frozen codecs", func() {
	It("Should emit codecs for a frozen version's defined types only", func() {
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
		// The live source carries the version-owned marshal tag textually,
		// exactly as the merged live projection emits it.
		diag := analyzer.AnalyzeSeeded(
			GinkgoT().Context(), `
@go output "core/pkg/service/channel"

Key = uuid {
}

Channel struct {
	key Key @key
	name string
	virtual bool

	@go marshal
}
`,
			"schemas/synnax/channel.oracle", "channel",
			analyzer.NewStandardFileLoader(root), table,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		req := &plugin.Request{
			Resolutions: table, RepoRoot: root, Versions: resolver,
		}
		resp := MustSucceed(marshal.New(marshal.Options{
			FileNamePattern:     "codec.gen.go",
			TestFileNamePattern: "codec_gen_test.go",
			RequireVersioned:    true,
		}).Generate(req))
		var frozen string
		for _, f := range resp.Files {
			if f.Path == "core/pkg/service/channel/versions/v0/codec.gen.go" {
				frozen = string(f.Content)
			}
		}
		Expect(frozen).ToNot(BeEmpty())
		Expect(frozen).To(ContainSubstring("package v0"))
		Expect(frozen).To(ContainSubstring("func (c Channel) EncodeOrc"))
		// v1 aliases Key; only its defined Channel gets a codec there.
		var current string
		for _, f := range resp.Files {
			if f.Path == "core/pkg/service/channel/versions/v1/codec.gen.go" {
				current = string(f.Content)
			}
		}
		Expect(current).To(ContainSubstring("func (c Channel) EncodeOrc"))
	})

	It("Should keep frozen codecs generating for an ended chain", func() {
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
		req := &plugin.Request{
			Resolutions: table, RepoRoot: root, Versions: resolver,
		}
		resp := MustSucceed(marshal.New(marshal.Options{
			FileNamePattern:     "codec.gen.go",
			TestFileNamePattern: "codec_gen_test.go",
			RequireVersioned:    true,
		}).Generate(req))
		var frozen string
		for _, f := range resp.Files {
			Expect(f.Path).ToNot(ContainSubstring("versions/v1"))
			if f.Path == "core/thing/versions/v0/codec.gen.go" {
				frozen = string(f.Content)
			}
		}
		Expect(frozen).To(ContainSubstring("func (t Thing) EncodeOrc"))
	})
})

var _ = Describe("Stored reference through a lagging pin", func() {
	// log v0 still pins color v0, which color v1 only aliases, so the current log
	// package's codec and its specs import v0, the version its version file imports.
	It("Should import the pinned version from a current codec and its specs", func() {
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

	@go marshal
}

Band struct {
	color Color

	@go marshal
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

	@go marshal
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
		req := &plugin.Request{
			Resolutions: table,
			RepoRoot:    root,
			Versions:    versions.NewResolver(chains, loader),
		}
		resp := MustSucceed(marshal.New(marshal.Options{
			FileNamePattern:     "codec.gen.go",
			TestFileNamePattern: "codec_gen_test.go",
			GenerateTests:       true,
			RequireVersioned:    true,
		}).Generate(req))
		files := make(map[string]string)
		for _, f := range resp.Files {
			files[f.Path] = string(f.Content)
		}
		specs := files["core/pkg/service/log/versions/v0/codec_gen_test.go"]
		Expect(specs).To(ContainSubstring(`x/go/color/versions/v0"`))
		Expect(specs).ToNot(ContainSubstring(`x/go/color/versions/v1"`))
		Expect(files["core/pkg/service/log/versions/v0/codec.gen.go"]).
			ToNot(ContainSubstring(`x/go/color/versions/v1"`))
	})
})

var _ = Describe("Stored reference in a generic declaration", func() {
	// log v0's generic Wrapper pins color v0, which color v1 only aliases, so the
	// generic fixture in the current package's specs imports v0 too.
	It("Should import the pinned version from a generic declaration's specs", func() {
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

	@go marshal
}

Band struct {
	color Color

	@go marshal
}
`)
		generic := `
Wrapper struct<D?> {
	color color.Color
	details D {
		@go marshal json_only
	}

	@go marshal
}

Log struct {
	key uuid @key
	color color.Color

	@go marshal
}
`
		write("schemas/synnax/versions/log/v0.oracle",
			"\nimport \"schemas/x/versions/color/v0\"\n"+generic)
		write("schemas/synnax/log.oracle",
			"\nimport \"schemas/x/color\"\n\n@go output \"core/pkg/service/log\"\n"+
				generic)
		chains := MustSucceed(versions.Discover(root))
		loader := analyzer.NewStandardFileLoader(root)
		table, diag := analyzer.Analyze(
			GinkgoT().Context(),
			[]string{"schemas/x/color", "schemas/synnax/log"},
			loader,
		)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		resp := MustSucceed(marshal.New(marshal.Options{
			FileNamePattern:     "codec.gen.go",
			TestFileNamePattern: "codec_gen_test.go",
			GenerateTests:       true,
			RequireVersioned:    true,
		}).Generate(&plugin.Request{
			Resolutions: table,
			RepoRoot:    root,
			Versions:    versions.NewResolver(chains, loader),
		}))
		specs := generatedFile(
			resp,
			"core/pkg/service/log/versions/v0/codec_gen_test.go",
		)
		Expect(specs).To(ContainSubstring("Wrapper["))
		Expect(specs).To(ContainSubstring(`x/go/color/versions/v0"`))
		Expect(specs).ToNot(ContainSubstring(`x/go/color/versions/v1"`))
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
	generate := func(frozen bool) *plugin.Response {
		GinkgoHelper()
		return MustSucceed(marshal.New(marshal.Options{
			FileNamePattern:     "codec.gen.go",
			TestFileNamePattern: "codec_gen_test.go",
			GenerateTests:       true,
			RequireVersioned:    true,
		}).Generate(pinRequest(frozen)))
	}

	// log v1's Log is declared against color v1, so its codec names color v1 only.
	DescribeTable("Should name the version that log v1 imports in its codec",
		func(frozen bool) {
			codec := generatedFile(
				generate(frozen), "core/pkg/service/log/versions/v1/codec.gen.go",
			)
			Expect(codec).To(MatchRegexp(`\scolor "[^"]*/color/versions/v1"`))
			Expect(codec).ToNot(ContainSubstring(`color/versions/v0"`))
		},
		Entry("current", false),
		Entry("frozen", true),
	)

	// The specs fill Log's colors against color v1 and Note's tint against color v0,
	// since log v0 declares Note. The newest version takes the bare name.
	DescribeTable("Should name each field's version in the codec specs",
		func(frozen bool) {
			specs := generatedFile(
				generate(frozen), "core/pkg/service/log/versions/v1/codec_gen_test.go",
			)
			Expect(specs).To(MatchRegexp(`\scolor "[^"]*/color/versions/v1"`))
			Expect(specs).To(MatchRegexp(`\scolorv0 "[^"]*/color/versions/v0"`))
			Expect(specs).To(ContainSubstring("[]color.Color{"))
			Expect(specs).To(ContainSubstring("colorv0.Color{"))
		},
		Entry("current", false),
		Entry("frozen", true),
	)
})
