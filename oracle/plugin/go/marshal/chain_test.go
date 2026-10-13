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

var _ = Describe("Stored reference to an aliased dependency member", func() {
	// color v1 only aliases v0's Color, so the current log package's codec and its
	// specs import v0, as they do once a later log version freezes it.
	It("Should import the member's definer from a current codec and its specs", func() {
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

// writeTwoVersionFixture writes a log whose current version, v1, reaches color's v0
// Color and v1 Band, and aliases its own v0 Note.
func writeTwoVersionFixture(root string) {
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
Note struct {
	text string

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
	write("schemas/synnax/log.oracle", `
import "schemas/x/color"

@go output "core/pkg/service/log"

Note struct {
	text string

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

// twoVersionRequest analyzes the fixture writeTwoVersionFixture writes.
func twoVersionRequest() *plugin.Request {
	GinkgoHelper()
	root := GinkgoT().TempDir()
	writeTwoVersionFixture(root)
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

var _ = Describe("Import aliases for two versions of a dependency", func() {
	DescribeTable("Should give the newest version the bare name",
		func(path string) {
			resp := MustSucceed(marshal.New(marshal.Options{
				FileNamePattern:     "codec.gen.go",
				TestFileNamePattern: "codec_gen_test.go",
				GenerateTests:       true,
				RequireVersioned:    true,
			}).Generate(twoVersionRequest()))
			var content string
			for _, f := range resp.Files {
				if f.Path == path {
					content = string(f.Content)
				}
			}
			Expect(content).To(MatchRegexp(`\scolorv0 "[^"]*/color/versions/v0"`))
			Expect(content).To(MatchRegexp(`\scolor "[^"]*/color/versions/v1"`))
		},
		Entry("codec", "core/pkg/service/log/versions/v1/codec.gen.go"),
		Entry("codec specs", "core/pkg/service/log/versions/v1/codec_gen_test.go"),
	)
})
