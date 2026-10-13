// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versioning_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Versioning", func() {
	Describe("Dir", func() {
		It("Should format the version sub-directory name", func() {
			Expect(versioning.Dir(0)).To(Equal("v0"))
			Expect(versioning.Dir(12)).To(Equal("v12"))
		})
	})

	Describe("VersionedPath", func() {
		It("Should append the types/vN sub-path", func() {
			Expect(versioning.VersionedPath("core/out", 3)).To(
				Equal("core/out/versions/v3"),
			)
		})
	})

	Describe("VersionDirs", func() {
		It("Should return the version directories on disk, ascending", func() {
			root := GinkgoT().TempDir()
			for _, dir := range []string{"v5", "v0", "v2", "legacy", "testdata"} {
				Expect(os.MkdirAll(
					filepath.Join(root, "core/out/versions", dir), 0o755,
				)).To(Succeed())
			}
			Expect(versioning.VersionDirs(root, "core/out")).To(Equal([]int{0, 2, 5}))
		})

		It("Should return nothing when the versions tree is absent", func() {
			Expect(versioning.VersionDirs(GinkgoT().TempDir(), "core/out")).To(BeNil())
		})
	})

	Describe("Survey", func() {
		var (
			root string
			res  *versions.Resolver
		)

		analyzeLive := func(ctx context.Context, source string) *resolution.Table {
			GinkgoHelper()
			table := resolution.NewTable()
			diag := analyzer.AnalyzeSeeded(
				ctx, source, "schemas/synnax/channel.oracle", "channel",
				analyzer.NewStandardFileLoader(root), table,
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			return table
		}

		BeforeEach(func(ctx SpecContext) {
			root = GinkgoT().TempDir()
			full := filepath.Join(root, "schemas/synnax/versions/channel")
			Expect(os.MkdirAll(full, 0o755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(full, "v0.oracle"), []byte(`
Channel struct {
	key uuid @key

	@go marshal
}
`), 0o644)).To(Succeed())
			chains := MustSucceed(versions.Discover(root))
			res = versions.NewResolver(
				chains, analyzer.NewStandardFileLoader(root),
			)
		})

		const live = `
@go output "out"

Channel struct {
	key uuid @key

	@go marshal
}

Wire struct {
	name string
}
`

		It("Should map member output paths to the chain's current version",
			func(ctx SpecContext) {
				table := analyzeLive(ctx, live)
				entries, members := MustSucceed2(
					versioning.Survey(ctx, table, res),
				)
				Expect(entries).To(Equal(map[string]int{"out": 0}))
				Expect(members.Contains("channel.Channel")).To(BeTrue())
				Expect(members.Contains("channel.Wire")).To(BeFalse())
			})

		It("Should return nothing without a resolver", func(ctx SpecContext) {
			table := analyzeLive(ctx, live)
			entries, members := MustSucceed2(versioning.Survey(ctx, table, nil))
			Expect(entries).To(BeEmpty())
			Expect(members).To(BeEmpty())
		})

		It("Should rewrite member outputs and leave transient types",
			func(ctx SpecContext) {
				table := analyzeLive(ctx, live)
				rewritten, pathMap, members, err := versioning.RewriteCurrent(
					ctx, table, res,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(pathMap).To(Equal(map[string]string{"out": "out/versions/v0"}))
				Expect(members.Contains("channel.Channel")).To(BeTrue())
				entry := rewritten.MustGet("channel.Channel")
				Expect(output.GetPath(entry, "go")).To(Equal("out/versions/v0"))
				wire := rewritten.MustGet("channel.Wire")
				Expect(output.GetPath(wire, "go")).To(Equal("out"))
			})

		It("Should return the table unchanged with no chains", func(ctx SpecContext) {
			table := analyzeLive(ctx, live)
			rewritten, pathMap, _, err := versioning.RewriteCurrent(ctx, table, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(rewritten).To(BeIdenticalTo(table))
			Expect(pathMap).To(BeEmpty())
		})
	})

	Describe("Pins", func() {
		var (
			root string
			res  *versions.Resolver
		)

		write := func(rel, content string) {
			GinkgoHelper()
			full := filepath.Join(root, rel)
			Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
			Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
		}

		analyzeLive := func(ctx context.Context) *resolution.Table {
			GinkgoHelper()
			table := resolution.NewTable()
			diag := analyzer.AnalyzeSeeded(ctx, `
import "schemas/x/color"

@go output "out"

Note struct {
	tint color.Color

	@go marshal
}

Channel struct {
	key uuid @key
	note Note
	tint color.Color

	@go marshal
}
`, "schemas/synnax/channel.oracle", "channel",
				analyzer.NewStandardFileLoader(root), table,
			)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			return table
		}

		color := func(table *resolution.Table) resolution.Type {
			GinkgoHelper()
			return MustBeOk(table.Get("color.Color"))
		}

		BeforeEach(func() {
			root = GinkgoT().TempDir()
			write("schemas/x/color.oracle", `
@go output "colorout"

Color struct {
	r uint8
}
`)
			write("schemas/x/versions/color/v0.oracle", `
Color struct {
	r uint8
}
`)
			write("schemas/x/versions/color/v1.oracle", `
Color = v0.Color
`)
			write("schemas/synnax/versions/channel/v0.oracle", `
import "schemas/x/versions/color/v0"

Note struct {
	tint color.Color

	@go marshal
}

Channel struct {
	key uuid @key

	@go marshal
}
`)
			write("schemas/synnax/versions/channel/v1.oracle", `
import "schemas/x/versions/color/v1"

Note = v0.Note

Channel struct {
	key uuid @key
	note Note
	tint color.Color

	@go marshal
}
`)
			chains := MustSucceed(versions.Discover(root))
			res = versions.NewResolver(chains, analyzer.NewStandardFileLoader(root))
		})

		It("Should resolve a reference through the version its declaring file imports",
			func(ctx SpecContext) {
				table := analyzeLive(ctx)
				pins := MustSucceed(versioning.NewPins(ctx, table, res))
				Expect(MustBeOk(pins.Path("channel.Note", color(table)))).
					To(Equal("colorout/versions/v0"))
				Expect(MustBeOk(pins.Path("channel.Channel", color(table)))).
					To(Equal("colorout/versions/v1"))
			})

		It("Should resolve a current declaration through its schema's current pins",
			func(ctx SpecContext) {
				table := analyzeLive(ctx)
				pins := MustSucceed(versioning.NewPins(ctx, table, res))
				Expect(MustBeOk(pins.CurrentPath(
					"schemas/synnax/channel", color(table),
				))).To(Equal("colorout/versions/v1"))
			})

		It("Should not resolve a reference its declaring file does not pin",
			func(ctx SpecContext) {
				table := analyzeLive(ctx)
				pins := MustSucceed(versioning.NewPins(ctx, table, res))
				note := MustBeOk(table.Get("channel.Note"))
				Expect(pins.Path("channel.Channel", note)).Error().To(BeFalse())
				Expect(pins.Path("channel.Unknown", color(table))).Error().To(BeFalse())
				Expect(pins.CurrentPath("schemas/x/color", color(table))).Error().
					To(BeFalse())
			})

		It("Should resolve nothing without a resolver", func(ctx SpecContext) {
			table := analyzeLive(ctx)
			pins := MustSucceed(versioning.NewPins(ctx, table, nil))
			Expect(pins.Path("channel.Note", color(table))).Error().To(BeFalse())
		})

		It("Should return the error from reading a surface", func(ctx SpecContext) {
			table := analyzeLive(ctx)
			write("schemas/synnax/versions/channel/v1.oracle", "Channel struct {")
			Expect(versioning.NewPins(ctx, table, res)).Error().
				To(MatchError(ContainSubstring("mismatched input '<EOF>'")))
		})
	})
})
