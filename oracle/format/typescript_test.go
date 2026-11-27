// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package format_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/paths"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("TypeScript Formatter", func() {
	var (
		repoRoot string
		r        *format.Registry
	)

	BeforeEach(func() {
		repoRoot = MustSucceed(paths.RepoRoot())
		r = MustSucceed(format.Default(repoRoot, 2026, nil))
	})

	Context("Outside the repository packages", func() {
		var (
			root string
			tmp  *format.Registry
		)

		// The temporary root holds no node_modules, so Node finds no ESLint or
		// Prettier.
		BeforeEach(func() {
			root = GinkgoT().TempDir()
			template := MustSucceed(paths.ReadFile(
				filepath.Join("licenses", "headers", "template.txt"),
				repoRoot,
			))
			headers := filepath.Join(root, "licenses", "headers")
			Expect(os.MkdirAll(headers, 0o755)).To(Succeed())
			Expect(os.WriteFile(
				filepath.Join(headers, "template.txt"),
				template,
				0o644,
			)).To(Succeed())
			tmp = MustSucceed(format.Default(root, 2026, nil))
		})

		It(
			"Should fail every file of a package when one is not UTF-8",
			func(ctx SpecContext) {
				out := tmp.FormatBatch(ctx, []format.File{
					{Path: filepath.Join(root, "a.gen.ts"), Content: []byte{0xff}},
					{Path: filepath.Join(root, "b.gen.ts"), Content: []byte("1\n")},
				}, 1)
				for _, res := range out {
					Expect(res.Err).To(MatchError(ContainSubstring("invalid UTF-8")))
				}
			},
		)

		It(
			"Should fail every file of each package without ESLint",
			func(ctx SpecContext) {
				if _, err := exec.LookPath("node"); err != nil {
					Skip("node not on PATH")
				}
				pkg := filepath.Join(root, "pkg")
				Expect(os.MkdirAll(filepath.Join(pkg, "src"), 0o755)).To(Succeed())
				Expect(os.WriteFile(
					filepath.Join(pkg, "package.json"),
					[]byte("{}\n"),
					0o644,
				)).To(Succeed())
				loose := filepath.Join(root, "loose")
				Expect(os.Mkdir(loose, 0o755)).To(Succeed())
				out := tmp.FormatBatch(ctx, []format.File{
					{
						Path:    filepath.Join(pkg, "src", "a.gen.ts"),
						Content: []byte("1\n"),
					},
					{Path: filepath.Join(pkg, "b.gen.ts"), Content: []byte("2\n")},
					{Path: filepath.Join(loose, "c.gen.ts"), Content: []byte("3\n")},
				}, 2)
				for i, dir := range []string{pkg, pkg, loose} {
					Expect(out[i].Err).To(MatchError(And(
						ContainSubstring("node failed"),
						ContainSubstring(strconv.Quote(dir)),
						ContainSubstring("eslint"),
					)))
				}
			},
		)
	})

	Context("With ESLint and Prettier installed", func() {
		BeforeEach(func() {
			if _, err := exec.LookPath("node"); err != nil {
				Skip("node not on PATH")
			}
			for _, pkg := range []string{"x/ts", "client/ts"} {
				if _, err := os.Stat(
					filepath.Join(repoRoot, pkg, "node_modules"),
				); err != nil {
					Skip("node_modules not installed")
				}
			}
		})

		// stage writes each file to disk until the spec ends, as sync does for new
		// files, because typed ESLint lints only files on disk.
		stage := func(files ...format.File) []format.File {
			for _, f := range files {
				Expect(os.WriteFile(f.Path, f.Content, 0o644)).To(Succeed())
				DeferCleanup(os.Remove, f.Path)
			}
			return files
		}

		It(
			"Should sort imports and format files across packages",
			func(ctx SpecContext) {
				raw := "import { z } from \"zod\";\nimport { array } from \"@/array\";\n" +
					"export const  a = z.object({ b:array.nullishToEmpty(z.string()) })\n"
				out := r.FormatBatch(ctx, stage(
					format.File{
						Path: filepath.Join(
							repoRoot,
							"x/ts/src/oracle_fixture.gen.ts",
						),
						Content: []byte(raw),
					},
					format.File{
						Path: filepath.Join(
							repoRoot,
							"client/ts/src/oracle_fixture.gen.ts",
						),
						Content: []byte(raw),
					},
				), 2)
				for _, res := range out {
					Expect(res.Err).ToNot(HaveOccurred())
					Expect(
						string(res.Content),
					).To(HavePrefix("// Copyright 2026 Synnax Labs"))
					Expect(string(res.Content)).To(ContainSubstring(
						"import { z } from \"zod\";\n\nimport { array } from \"@/array\";\n",
					))
					Expect(string(res.Content)).To(ContainSubstring(
						"export const a = z.object({ b: array.nullishToEmpty(z.string()) });\n",
					))
				}
			},
		)

		It("Should fail only the file that does not parse", func(ctx SpecContext) {
			out := r.FormatBatch(ctx, stage(
				format.File{
					Path:    filepath.Join(repoRoot, "x/ts/src/oracle_ok.gen.ts"),
					Content: []byte("export const a = 1\n"),
				},
				format.File{
					Path:    filepath.Join(repoRoot, "x/ts/src/oracle_bad.gen.ts"),
					Content: []byte("export const = ;\n"),
				},
			), 1)
			Expect(out[0].Err).ToNot(HaveOccurred())
			Expect(string(out[0].Content)).To(HaveSuffix("export const a = 1;\n"))
			Expect(out[1].Err).To(MatchError(And(
				ContainSubstring("oracle_bad.gen.ts"),
				ContainSubstring("eslint fatal error"),
			)))
		})
	})
})
