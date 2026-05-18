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
	})

	Describe("Normalize", func() {
		It("Should normalize absolute paths to repo-relative", func() {
			absPath := filepath.Join(repoRoot, "oracle", "paths")
			rel := MustSucceed(paths.Normalize(absPath, repoRoot))
			Expect(rel).To(Equal(filepath.Join("oracle", "paths")))
		})

		It("Should handle already repo-relative paths", func() {
			rel := MustSucceed(paths.Normalize("oracle/paths", repoRoot))
			Expect(rel).To(Equal(filepath.Join("oracle", "paths")))
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
	})

	Describe("RelativeImport", func() {
		It("Should compute relative path between sibling directories", func() {
			rel := MustSucceed(paths.RelativeImport("client/ts/src/user", "client/ts/src/group"))
			Expect(rel).To(Equal("../group"))
		})

		It("Should compute relative path to nested directory", func() {
			rel := MustSucceed(paths.RelativeImport("client/ts/src", "client/ts/src/user"))
			Expect(rel).To(Equal("./user"))
		})

		It("Should compute relative path to parent directory", func() {
			rel := MustSucceed(paths.RelativeImport("client/ts/src/user", "client/ts/src"))
			Expect(rel).To(Equal(".."))
		})

		It("Should return . for same directory", func() {
			rel := MustSucceed(paths.RelativeImport("client/ts/src", "client/ts/src"))
			Expect(rel).To(Equal("."))
		})

		It("Should compute relative path across different branches", func() {
			rel := MustSucceed(paths.RelativeImport("client/ts/src/user", "core/pkg/service/group"))
			Expect(rel).To(Equal("../../../../core/pkg/service/group"))
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
