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
		if _, err := exec.LookPath("node"); err != nil {
			Skip("node not on PATH")
		}
		repoRoot = MustSucceed(paths.RepoRoot())
		for _, pkg := range []string{"x/ts", "client/ts"} {
			if _, err := os.Stat(
				filepath.Join(repoRoot, pkg, "node_modules"),
			); err != nil {
				Skip("node_modules not installed")
			}
		}
		r = MustSucceed(format.Default(repoRoot, 2026, nil))
	})

	It("Should sort imports and format files across packages", func(ctx SpecContext) {
		raw := "import { z } from \"zod\";\nimport { array } from \"@/array\";\n" +
			"export const  a = z.object({ b:array.nullishToEmpty(z.string()) })\n"
		out := r.FormatBatch(ctx, []format.File{
			{
				Path:    filepath.Join(repoRoot, "x/ts/src/oracle_fixture.gen.ts"),
				Content: []byte(raw),
			},
			{
				Path:    filepath.Join(repoRoot, "client/ts/src/oracle_fixture.gen.ts"),
				Content: []byte(raw),
			},
		}, 2)
		for _, res := range out {
			Expect(res.Err).ToNot(HaveOccurred())
			Expect(string(res.Content)).To(HavePrefix("// Copyright 2026 Synnax Labs"))
			Expect(string(res.Content)).To(ContainSubstring(
				"import { z } from \"zod\";\n\nimport { array } from \"@/array\";\n",
			))
			Expect(string(res.Content)).To(ContainSubstring(
				"export const a = z.object({ b: array.nullishToEmpty(z.string()) });\n",
			))
		}
	})

	It("Should fail only the file that does not parse", func(ctx SpecContext) {
		out := r.FormatBatch(ctx, []format.File{
			{
				Path:    filepath.Join(repoRoot, "x/ts/src/oracle_ok.gen.ts"),
				Content: []byte("export const a = 1\n"),
			},
			{
				Path:    filepath.Join(repoRoot, "x/ts/src/oracle_bad.gen.ts"),
				Content: []byte("export const = ;\n"),
			},
		}, 1)
		Expect(out[0].Err).ToNot(HaveOccurred())
		Expect(string(out[0].Content)).To(HaveSuffix("export const a = 1;\n"))
		Expect(out[1].Err).To(MatchError(And(
			ContainSubstring("oracle_bad.gen.ts"),
			ContainSubstring("eslint fatal error"),
		)))
	})
})
