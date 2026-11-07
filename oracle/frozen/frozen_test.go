// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package frozen_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/frozen"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	sumFile = "schemas/synnax/frozen/doc.sum"
	v0File  = "core/pkg/service/doc/versions/v0/types.gen.go"
	v1File  = "core/pkg/service/doc/versions/v1/types.gen.go"
)

var _ = Describe("Frozen", func() {
	var root string
	write := func(rel, content string) {
		GinkgoHelper()
		full := filepath.Join(root, rel)
		Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
		Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
	}
	compute := func(ctx context.Context, outputs ...string) frozen.Chain {
		GinkgoHelper()
		chains := MustSucceed(versions.Discover(root))
		loader := analyzer.NewStandardFileLoader(root)
		table, diag := analyzer.Analyze(ctx, []string{"schemas/synnax/doc"}, loader)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		files := make([]plugin.File, 0, len(outputs))
		for _, path := range outputs {
			files = append(files, plugin.File{Path: path})
		}
		computed := MustSucceed(frozen.Compute(ctx, &pipeline.Result{
			Resolutions: table,
			Versions:    versions.NewResolver(chains, loader),
			Outputs:     map[string][]plugin.File{"stub": files},
		}, root))
		Expect(computed).To(HaveLen(1))
		return computed[0]
	}

	BeforeEach(func() {
		root = GinkgoT().TempDir()
		write("schemas/synnax/versions/doc/v0.oracle", `
Doc struct {
	key uuid @key

	@go marshal
}
`)
		write("schemas/synnax/versions/doc/v1.oracle", `
Doc = v0.Doc
`)
		write("schemas/synnax/doc.oracle", `
@go output "core/pkg/service/doc"

Doc struct {
	key uuid @key

	@go marshal
}
`)
		write(v0File, "v0")
		write(v1File, "v1")
	})

	Describe("Chain", func() {
		It("Should name a version's package", func() {
			c := frozen.Chain{Root: "core/pkg/service/doc"}
			Expect(c.Package(3)).To(Equal("core/pkg/service/doc/versions/v3"))
		})

		DescribeTable("Moved",
			func(n int, recorded frozen.Sum, moved bool) {
				c := frozen.Chain{
					Current: 1,
					Sums:    map[int]frozen.Sum{0: {Schema: "s", Output: "o"}},
				}
				Expect(c.Moved(n, recorded)).To(Equal(moved))
			},
			Entry("output under the same schema", 0, frozen.Sum{Schema: "s"}, true),
			Entry("matching sum", 0, frozen.Sum{Schema: "s", Output: "o"}, false),
			Entry("changed schema", 0, frozen.Sum{Schema: "t"}, false),
			Entry("current version", 1, frozen.Sum{Schema: "s"}, false),
		)

		It("Should treat a frozen package that stopped generating as moved", func() {
			c := frozen.Chain{Current: 2, Sums: map[int]frozen.Sum{}}
			Expect(c.Moved(0, frozen.Sum{Schema: "s"})).To(BeTrue())
		})

		It("Should list computed and recorded versions in order", func() {
			c := frozen.Chain{Sums: map[int]frozen.Sum{2: {}}}
			Expect(c.Versions(map[int]frozen.Sum{1: {}, 2: {}})).To(Equal([]int{1, 2}))
		})
	})

	Describe("Compute", func() {
		It("Should locate the chain and its current version", func(ctx SpecContext) {
			c := compute(ctx, v0File, v1File)
			Expect(c.Path).To(Equal(sumFile))
			Expect(c.Root).To(Equal("core/pkg/service/doc"))
			Expect(c.Current).To(Equal(1))
			Expect(c.Sums).To(HaveLen(2))
		})

		It("Should skip a version with no generated files", func(ctx SpecContext) {
			Expect(compute(ctx, v1File).Sums).To(HaveLen(1))
		})

		It("Should change the output sum on a disk edit", func(ctx SpecContext) {
			before := compute(ctx, v0File, v1File).Sums[0]
			write(v0File, "edited")
			after := compute(ctx, v0File, v1File).Sums[0]
			Expect(after.Schema).To(Equal(before.Schema))
			Expect(after.Output).ToNot(Equal(before.Output))
		})

		It("Should change the output sum on a missing file", func(ctx SpecContext) {
			before := compute(ctx, v0File, v1File).Sums[0]
			Expect(os.Remove(filepath.Join(root, v0File))).To(Succeed())
			Expect(compute(ctx, v0File, v1File).Sums[0].Output).
				ToNot(Equal(before.Output))
		})

		It("Should carry a field change in an earlier version forward",
			func(ctx SpecContext) {
				before := compute(ctx, v0File, v1File).Sums[1]
				write("schemas/synnax/versions/doc/v0.oracle", `
Doc struct {
	key uuid @key
	name string

	@go marshal
}
`)
				Expect(compute(ctx, v0File, v1File).Sums[1].Schema).
					ToNot(Equal(before.Schema))
			},
		)

		It("Should keep every schema sum on a comment and format edit",
			func(ctx SpecContext) {
				before := compute(ctx, v0File, v1File).Sums
				write("schemas/synnax/versions/doc/v0.oracle", `
// Edited.
Doc struct {
		key    uuid    @key

	@go marshal
}
`)
				after := compute(ctx, v0File, v1File).Sums
				Expect(after[0].Schema).To(Equal(before[0].Schema))
				Expect(after[1].Schema).To(Equal(before[1].Schema))
			},
		)

		It("Should fail on a missing version file", func(ctx SpecContext) {
			chains := MustSucceed(versions.Discover(root))
			loader := analyzer.NewStandardFileLoader(root)
			table, diag := analyzer.Analyze(ctx, []string{"schemas/synnax/doc"}, loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			res := &pipeline.Result{
				Resolutions: table,
				Versions:    versions.NewResolver(chains, loader),
			}
			Expect(os.Remove(
				filepath.Join(root, "schemas/synnax/versions/doc/v1.oracle"),
			)).To(Succeed())
			Expect(frozen.Compute(ctx, res, root)).Error().To(SatisfyAll(
				MatchError(fs.ErrNotExist),
				MatchError(ContainSubstring("v1.oracle")),
			))
		})
	})

	Describe("Read", func() {
		It("Should hold no sums for a missing file", func() {
			Expect(MustSucceed(frozen.Read(sumFile, root))).To(BeEmpty())
		})

		It("Should parse versions and skip comments", func() {
			write(sumFile, "# header\nv4 s o\n")
			Expect(MustSucceed(frozen.Read(sumFile, root))).To(Equal(
				map[int]frozen.Sum{4: {Schema: "s", Output: "o"}},
			))
		})

		DescribeTable("Should fail on a malformed file",
			func(content string) {
				write(sumFile, content)
				Expect(frozen.Read(sumFile, root)).Error().
					To(MatchError(frozen.ErrMalformed))
			},
			Entry("a short line", "v0 s\n"),
			Entry("a conflict marker", "<<<<<<< HEAD\n"),
			Entry("a bad version", "vx s o\n"),
		)
	})

	Describe("Update", func() {
		chain := func(sums map[int]frozen.Sum) frozen.Chain {
			return frozen.Chain{Path: sumFile, Current: 1, Sums: sums}
		}
		recorded := map[int]frozen.Sum{
			0: {Schema: "s0", Output: "o0"},
			1: {Schema: "s1", Output: "o1"},
		}
		BeforeEach(func() {
			Expect(frozen.Update(
				[]frozen.Chain{chain(recorded)}, root, false,
			)).To(Succeed())
		})
		read := func() map[int]frozen.Sum {
			GinkgoHelper()
			return MustSucceed(frozen.Read(sumFile, root))
		}

		It("Should write what it computed", func() {
			Expect(read()).To(Equal(recorded))
		})

		It("Should keep a frozen sum whose schema did not change", func() {
			moved := map[int]frozen.Sum{
				0: {Schema: "s0", Output: "x"},
				1: {Schema: "s1", Output: "x"},
			}
			Expect(
				frozen.Update([]frozen.Chain{chain(moved)}, root, false),
			).To(Succeed())
			Expect(read()).To(Equal(map[int]frozen.Sum{
				0: {Schema: "s0", Output: "o0"},
				1: {Schema: "s1", Output: "x"},
			}))
		})

		It("Should take a frozen sum whose schema changed", func() {
			moved := map[int]frozen.Sum{
				0: {Schema: "t0", Output: "x"},
				1: recorded[1],
			}
			Expect(
				frozen.Update([]frozen.Chain{chain(moved)}, root, false),
			).To(Succeed())
			Expect(read()[0]).To(Equal(frozen.Sum{Schema: "t0", Output: "x"}))
		})

		It("Should keep the sum of a frozen package that stopped generating", func() {
			Expect(frozen.Update(
				[]frozen.Chain{chain(map[int]frozen.Sum{1: recorded[1]})}, root, false,
			)).To(Succeed())
			Expect(read()).To(Equal(recorded))
		})

		It("Should accept every change with refreeze", func() {
			moved := map[int]frozen.Sum{1: {Schema: "s1", Output: "x"}}
			Expect(
				frozen.Update([]frozen.Chain{chain(moved)}, root, true),
			).To(Succeed())
			Expect(read()).To(Equal(moved))
		})

		It("Should fail on a malformed sum file", func() {
			write(sumFile, "broken\n")
			Expect(frozen.Update(
				[]frozen.Chain{chain(recorded)}, root, false,
			)).To(MatchError(frozen.ErrMalformed))
		})

		It("Should remove a sum file that no chain owns", func() {
			const orphan = "schemas/synnax/frozen/gone.sum"
			write(orphan, "v0 s o\n")
			Expect(frozen.Update(
				[]frozen.Chain{chain(recorded)}, root, false,
			)).To(Succeed())
			Expect(filepath.Join(root, orphan)).ToNot(BeAnExistingFile())
			Expect(read()).To(Equal(recorded))
		})
	})

	Describe("Orphans", func() {
		owned := []frozen.Chain{{Path: sumFile}}

		It("Should list the sum files that no chain owns", func() {
			write(sumFile, "")
			write("schemas/synnax/frozen/b.sum", "")
			write("schemas/synnax/frozen/a.sum", "")
			write("schemas/synnax/frozen/notes.txt", "")
			Expect(MustSucceed(frozen.Orphans(owned, root))).To(Equal([]string{
				"schemas/synnax/frozen/a.sum",
				"schemas/synnax/frozen/b.sum",
			}))
		})

		It("Should list a sum file in a domain with no chains", func() {
			write(sumFile, "")
			write("schemas/x/frozen/gone.sum", "")
			Expect(MustSucceed(frozen.Orphans(owned, root))).To(Equal([]string{
				"schemas/x/frozen/gone.sum",
			}))
		})

		It("Should list nothing when no frozen directory exists", func() {
			Expect(MustSucceed(frozen.Orphans(owned, root))).To(BeEmpty())
		})
	})
})
