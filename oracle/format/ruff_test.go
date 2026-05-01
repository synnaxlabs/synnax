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
	. "github.com/synnaxlabs/x/testutil"
)

// installRuffFixture writes a minimal pyproject.toml so the formatter's
// nearest-pyproject.toml lookup picks our temp dir. We invoke the ruff
// binary directly (Bin: "ruff") to keep the test independent of a `uv`
// install.
func installRuffFixture() (absFile, projectDir string) {
	GinkgoHelper()
	root := MustSucceed(os.MkdirTemp("", "ruff-fixture"))
	DeferCleanup(func() { Expect(os.RemoveAll(root)).To(Succeed()) })
	Expect(os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(`[tool.ruff]
line-length = 88
`), 0644)).To(Succeed())
	src := filepath.Join(root, "src")
	Expect(os.MkdirAll(src, 0755)).To(Succeed())
	return filepath.Join(src, "foo.py"), root
}

var _ = Describe("Ruff Formatter", func() {
	BeforeEach(func() {
		if _, err := exec.LookPath("ruff"); err != nil {
			Skip("ruff not on PATH")
		}
	})

	It("Should normalise quoting and spacing", func(ctx SpecContext) {
		absFile, _ := installRuffFixture()
		raw := []byte("def foo( ):\n  return  'hi'\n")
		r := &format.Ruff{Bin: "ruff"}
		out := MustSucceed(r.Format(ctx, raw, absFile))
		Expect(string(out)).To(ContainSubstring(`def foo():`))
		Expect(string(out)).To(ContainSubstring(`return "hi"`))
	})

	It("Should be idempotent", func(ctx SpecContext) {
		absFile, _ := installRuffFixture()
		raw := []byte(`def foo():
    return "hi"
`)
		r := &format.Ruff{Bin: "ruff"}
		first := MustSucceed(r.Format(ctx, raw, absFile))
		second := MustSucceed(r.Format(ctx, first, absFile))
		Expect(string(first)).To(Equal(string(second)))
	})

	It("Should surface a parse error for invalid Python", func(ctx SpecContext) {
		absFile, _ := installRuffFixture()
		raw := []byte("def foo(:\n")
		r := &format.Ruff{Bin: "ruff"}
		Expect(r.Format(ctx, raw, absFile)).Error().To(MatchError(ContainSubstring("ruff")))
	})
})
