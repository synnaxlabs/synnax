// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gomod_test

import (
	"os"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ParseModuleName", func() {
	var tmpDir string

	BeforeEach(func() {
		tmpDir = MustSucceed(os.MkdirTemp("", "gomod-test"))
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("should extract the module name from a go.mod file", func() {
		modPath := filepath.Join(tmpDir, "go.mod")
		Expect(os.WriteFile(modPath, []byte("module github.com/example/pkg\n\ngo 1.21\n"), 0o644)).To(Succeed())
		Expect(MustSucceed(gomod.ParseModuleName(modPath))).To(Equal("github.com/example/pkg"))
	})

	It("should handle go.mod with require blocks", func() {
		modPath := filepath.Join(tmpDir, "go.mod")
		content := "module github.com/myorg/mymod\n\ngo 1.21\n\nrequire (\n\tgithub.com/foo/bar v1.0.0\n)\n"
		Expect(os.WriteFile(modPath, []byte(content), 0o644)).To(Succeed())
		Expect(MustSucceed(gomod.ParseModuleName(modPath))).To(Equal("github.com/myorg/mymod"))
	})

	It("should return an error for a nonexistent file", func() {
		_, err := gomod.ParseModuleName(filepath.Join(tmpDir, "nonexistent"))
		Expect(err).To(HaveOccurred())
	})

	It("should return an error for a go.mod without module directive", func() {
		modPath := filepath.Join(tmpDir, "go.mod")
		Expect(os.WriteFile(modPath, []byte("go 1.21\n"), 0o644)).To(Succeed())
		_, err := gomod.ParseModuleName(modPath)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no module directive"))
	})
})

var _ = Describe("ResolveImportPath", func() {
	var tmpDir string

	BeforeEach(func() {
		tmpDir = MustSucceed(os.MkdirTemp("", "gomod-test"))
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("should use fallback prefix when repo root is empty", func() {
		result := gomod.ResolveImportPath("core/pkg/user", "", "github.com/fallback/")
		Expect(result).To(Equal("github.com/fallback/core/pkg/user"))
	})

	It("should resolve import path using go.mod", func() {
		modDir := filepath.Join(tmpDir, "core")
		pkgDir := filepath.Join(modDir, "pkg", "user")
		Expect(os.MkdirAll(pkgDir, 0o755)).To(Succeed())
		Expect(os.WriteFile(
			filepath.Join(modDir, "go.mod"),
			[]byte("module github.com/synnaxlabs/synnax\n\ngo 1.21\n"),
			0o644,
		)).To(Succeed())

		result := gomod.ResolveImportPath("core/pkg/user", tmpDir, "github.com/fallback/")
		Expect(result).To(Equal("github.com/synnaxlabs/synnax/pkg/user"))
	})

	It("should return module name when output path matches module root", func() {
		Expect(os.WriteFile(
			filepath.Join(tmpDir, "go.mod"),
			[]byte("module github.com/synnaxlabs/x\n\ngo 1.21\n"),
			0o644,
		)).To(Succeed())

		result := gomod.ResolveImportPath("", tmpDir, "github.com/fallback/")
		Expect(result).To(Equal("github.com/synnaxlabs/x"))
	})

	It("should fall back to prefix when no go.mod found", func() {
		result := gomod.ResolveImportPath("some/path", tmpDir, "github.com/fallback/")
		Expect(result).To(Equal("github.com/fallback/some/path"))
	})
})

var _ = Describe("FindRepoRoot", func() {
	It("should find the repo root from the current file", func() {
		_, thisFile, _, _ := runtime.Caller(0)
		root := gomod.FindRepoRoot(thisFile)
		Expect(root).To(Equal(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(thisFile))))))
	})

	It("should return empty string when no .git directory exists", func() {
		root := gomod.FindRepoRoot("/tmp/nonexistent/path/file.go")
		Expect(root).To(BeEmpty())
	})
})
