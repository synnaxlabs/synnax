// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package paths_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/paths"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Paths", func() {
	var repoRoot string

	BeforeEach(func() {
		repoRoot = MustSucceed(paths.RepoRoot())
	})

	Describe("RepoRoot", func() {
		It("Should find repo root from current directory", func() {
			root := MustSucceed(paths.RepoRoot())
			Expect(filepath.Join(root, "pnpm-workspace.yaml")).To(BeAnExistingFile())
			Expect(filepath.Join(root, "MODULE.bazel")).To(BeAnExistingFile())
		})

		It("Should find repo root from a subdirectory", func() {
			// We're already in a subdirectory (oracle/paths)
			root := MustSucceed(paths.RepoRoot())
			Expect(root).ToNot(BeEmpty())
		})

		It("Should fall back to walking up for a .git entry", func() {
			// An empty .git directory is not a valid repository, so `git rev-parse`
			// fails and RepoRoot falls back to the directory walk.
			dir := GinkgoT().TempDir()
			origWd := MustSucceed(os.Getwd())
			DeferCleanup(func() { Expect(os.Chdir(origWd)).To(Succeed()) })
			Expect(os.MkdirAll(filepath.Join(dir, ".git"), 0o755)).To(Succeed())
			nested := filepath.Join(dir, "a", "b")
			Expect(os.MkdirAll(nested, 0o755)).To(Succeed())
			Expect(os.Chdir(nested)).To(Succeed())
			root := MustSucceed(paths.RepoRoot())
			Expect(filepath.Join(root, ".git")).To(BeADirectory())
		})

		It("Should error outside any git repository", func() {
			dir := GinkgoT().TempDir()
			origWd := MustSucceed(os.Getwd())
			DeferCleanup(func() { Expect(os.Chdir(origWd)).To(Succeed()) })
			Expect(os.Chdir(dir)).To(Succeed())
			Expect(paths.RepoRoot()).Error().
				To(MatchError(ContainSubstring("no .git entry found")))
		})
	})

	Describe("Normalize", func() {
		It("Should normalize absolute paths to repo-relative", func() {
			absPath := filepath.Join(repoRoot, "oracle", "paths")
			rel := MustSucceed(paths.Normalize(absPath, repoRoot))
			Expect(rel).To(Equal("oracle/paths"))
		})

		It("Should handle already repo-relative paths", func() {
			rel := MustSucceed(paths.Normalize("oracle/paths", repoRoot))
			Expect(rel).To(Equal("oracle/paths"))
		})

		It("Should resolve cwd-relative paths that are not repo-relative", func() {
			// paths_test.go exists relative to the cwd (oracle/paths) but not
			// relative to the repo root, forcing the cwd-resolution branch.
			rel := MustSucceed(paths.Normalize("paths_test.go", repoRoot))
			Expect(rel).To(Equal("oracle/paths/paths_test.go"))
		})

		It("Should reject paths that escape the repository", func() {
			Expect(paths.Normalize("../../../etc/passwd", repoRoot)).
				Error().To(MatchError(ContainSubstring("escapes")))
		})

		It("Should reject empty paths", func() {
			Expect(paths.Normalize("", repoRoot)).
				Error().To(MatchError(ContainSubstring("path cannot be empty")))
		})
	})

	Describe("Resolve", func() {
		It("Should convert repo-relative to absolute", func() {
			abs := paths.Resolve("oracle/paths", repoRoot)
			Expect(abs).To(Equal(filepath.Join(repoRoot, "oracle", "paths")))
		})

		It("Should return absolute paths unchanged", func() {
			absPath := filepath.Join(repoRoot, "oracle", "paths")
			result := paths.Resolve(absPath, repoRoot)
			Expect(result).To(Equal(absPath))
		})
	})

	Describe("ValidateOutput", func() {
		It("Should accept valid repo-relative paths", func() {
			Expect(paths.ValidateOutput("client/ts/src/user", repoRoot)).To(Succeed())
		})

		It("Should reject path traversal attempts", func() {
			Expect(paths.ValidateOutput("../../../etc/passwd", repoRoot)).Error().
				To(MatchError(ContainSubstring("path traversal")))
		})

		It("Should reject paths containing ..", func() {
			Expect(paths.ValidateOutput("client/../../../etc", repoRoot)).Error().
				To(MatchError(ContainSubstring("output path \"client/../../../etc\" contains path traversal")))
		})

		It("Should reject absolute paths", func() {
			Expect(paths.ValidateOutput("/etc/passwd", repoRoot)).Error().
				To(MatchError(ContainSubstring("repo-relative")))
		})

		It("Should reject empty paths", func() {
			Expect(paths.ValidateOutput("", repoRoot)).
				Error().To(MatchError(ContainSubstring("output path cannot be empty")))
		})

		It("Should reject backslash-prefixed paths", func() {
			Expect(paths.ValidateOutput(`\client\ts`, repoRoot)).Error().
				To(MatchError(ContainSubstring("must be repo-relative")))
		})
	})

	Describe("EnsureOracleExtension", func() {
		It("Should add .oracle extension if missing", func() {
			result := paths.EnsureOracleExtension("schema/core/label")
			Expect(result).To(Equal("schema/core/label.oracle"))
		})

		It("Should not duplicate .oracle extension", func() {
			result := paths.EnsureOracleExtension("schema/core/label.oracle")
			Expect(result).To(Equal("schema/core/label.oracle"))
		})
	})

	Describe("DeriveNamespace", func() {
		It("Should extract namespace from file path", func() {
			ns := paths.DeriveNamespace("schema/core/label.oracle")
			Expect(ns).To(Equal("label"))
		})

		It("Should handle nested paths", func() {
			ns := paths.DeriveNamespace("schema/core/nested/user.oracle")
			Expect(ns).To(Equal("user"))
		})

		It("Should handle paths without .oracle extension", func() {
			ns := paths.DeriveNamespace("schema/core/label")
			Expect(ns).To(Equal("label"))
		})

		It("Should derive a version file's resource as its namespace", func() {
			ns := paths.DeriveNamespace("schemas/x/versions/telem/v0.oracle")
			Expect(ns).To(Equal("telem"))
			Expect(paths.DeriveNamespace("schemas/x/versions/telem/v12")).
				To(Equal("telem"))
		})
	})

	Describe("VersionFile", func() {
		It("Should recognize version file paths", func() {
			resource, version, ok := paths.VersionFile(
				"schemas/synnax/versions/channel/v3.oracle",
			)
			Expect(ok).To(BeTrue())
			Expect(resource).To(Equal("channel"))
			Expect(version).To(Equal(3))
		})

		It("Should reject non-version paths", func() {
			_, _, ok := paths.VersionFile("schemas/synnax/channel.oracle")
			Expect(ok).To(BeFalse())
			_, _, ok = paths.VersionFile("schemas/synnax/channel/v3.oracle")
			Expect(ok).To(BeFalse())
			_, _, ok = paths.VersionFile("schemas/synnax/versions/channel/v03.oracle")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("VersionsDir", func() {
		It("Should return the chain directory for a live resource schema", func() {
			dir := MustBeOk(paths.VersionsDir("schemas/synnax/channel.oracle"))
			Expect(dir).To(Equal("schemas/synnax/versions/channel"))
		})

		DescribeTable("Should report false for non-live paths",
			func(path string) {
				_, ok := paths.VersionsDir(path)
				Expect(ok).To(BeFalse())
			},
			Entry("version file", "schemas/synnax/versions/channel/v0.oracle"),
			Entry("nested path", "schemas/synnax/nested/channel.oracle"),
			Entry("outside schemas", "client/ts/channel.oracle"),
		)
	})

	Describe("LiveSchema", func() {
		It("Should return the domain and resource for a live schema", func() {
			domain, resource, ok := paths.LiveSchema("schemas/x/telem.oracle")
			Expect(ok).To(BeTrue())
			Expect(domain).To(Equal("x"))
			Expect(resource).To(Equal("telem"))
		})

		DescribeTable("Should report false for non-live paths",
			func(path string) {
				_, _, ok := paths.LiveSchema(path)
				Expect(ok).To(BeFalse())
			},
			Entry("version file", "schemas/x/versions/telem/v0.oracle"),
			Entry("too shallow", "schemas/telem.oracle"),
			Entry("too deep", "schemas/x/nested/telem.oracle"),
			Entry("outside schemas", "client/x/telem.oracle"),
		)
	})

	Describe("Integration", func() {
		It("Should work correctly from different working directories", func() {
			// Save original working directory
			originalWd := MustSucceed(os.Getwd())
			defer func() { Expect(os.Chdir(originalWd)).To(Succeed()) }()

			// Get repo root from current location
			root1 := MustSucceed(paths.RepoRoot())

			// Change to a subdirectory
			Expect(os.Chdir(filepath.Join(repoRoot, "oracle"))).To(Succeed())

			// Get repo root from subdirectory
			root2 := MustSucceed(paths.RepoRoot())

			// Both should return the same root
			Expect(root1).To(Equal(root2))
		})
	})
})
