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
)

var _ = Describe("Paths", func() {
	var repoRoot string

	BeforeEach(func() {
		var err error
		repoRoot, err = paths.RepoRoot()
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("RepoRoot", func() {
		It("Should find repo root from current directory", func() {
			root, err := paths.RepoRoot()
			Expect(err).ToNot(HaveOccurred())
			Expect(root).To(HaveSuffix("synnax"))
			Expect(filepath.Join(root, ".git")).To(BeADirectory())
		})

		It("Should find repo root from a subdirectory", func() {
			// We're already in a subdirectory (oracle/paths)
			root, err := paths.RepoRoot()
			Expect(err).ToNot(HaveOccurred())
			Expect(root).ToNot(BeEmpty())
		})
	})

	Describe("Normalize", func() {
		It("Should normalize absolute paths to repo-relative", func() {
			absPath := filepath.Join(repoRoot, "oracle", "paths")
			rel, err := paths.Normalize(absPath, repoRoot)
			Expect(err).ToNot(HaveOccurred())
			Expect(rel).To(Equal(filepath.Join("oracle", "paths")))
		})

		It("Should handle already repo-relative paths", func() {
			rel, err := paths.Normalize("oracle/paths", repoRoot)
			Expect(err).ToNot(HaveOccurred())
			Expect(rel).To(Equal(filepath.Join("oracle", "paths")))
		})

		It("Should reject paths that escape the repository", func() {
			_, err := paths.Normalize("../../../etc/passwd", repoRoot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("escapes"))
		})

		It("Should reject empty paths", func() {
			_, err := paths.Normalize("", repoRoot)
			Expect(err).To(HaveOccurred())
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
			err := paths.ValidateOutput("client/ts/src/user", repoRoot)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should reject path traversal attempts", func() {
			err := paths.ValidateOutput("../../../etc/passwd", repoRoot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("path traversal"))
		})

		It("Should reject paths containing ..", func() {
			err := paths.ValidateOutput("client/../../../etc", repoRoot)
			Expect(err).To(HaveOccurred())
		})

		It("Should reject absolute paths", func() {
			err := paths.ValidateOutput("/etc/passwd", repoRoot)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("repo-relative"))
		})

		It("Should reject empty paths", func() {
			err := paths.ValidateOutput("", repoRoot)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("RelativeImport", func() {
		It("Should compute relative path between sibling directories", func() {
			rel, err := paths.RelativeImport("client/ts/src/user", "client/ts/src/group")
			Expect(err).ToNot(HaveOccurred())
			Expect(rel).To(Equal("../group"))
		})

		It("Should compute relative path to nested directory", func() {
			rel, err := paths.RelativeImport("client/ts/src", "client/ts/src/user")
			Expect(err).ToNot(HaveOccurred())
			Expect(rel).To(Equal("./user"))
		})

		It("Should compute relative path to parent directory", func() {
			rel, err := paths.RelativeImport("client/ts/src/user", "client/ts/src")
			Expect(err).ToNot(HaveOccurred())
			Expect(rel).To(Equal(".."))
		})

		It("Should return . for same directory", func() {
			rel, err := paths.RelativeImport("client/ts/src", "client/ts/src")
			Expect(err).ToNot(HaveOccurred())
			Expect(rel).To(Equal("."))
		})

		It("Should compute relative path across different branches", func() {
			rel, err := paths.RelativeImport("client/ts/src/user", "core/pkg/service/group")
			Expect(err).ToNot(HaveOccurred())
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
			originalWd, err := os.Getwd()
			Expect(err).ToNot(HaveOccurred())
			defer os.Chdir(originalWd)

			// Get repo root from current location
			root1, err := paths.RepoRoot()
			Expect(err).ToNot(HaveOccurred())

			// Change to a subdirectory
			err = os.Chdir(filepath.Join(repoRoot, "oracle"))
			Expect(err).ToNot(HaveOccurred())

			// Get repo root from subdirectory
			root2, err := paths.RepoRoot()
			Expect(err).ToNot(HaveOccurred())

			// Both should return the same root
			Expect(root1).To(Equal(root2))
		})
	})
})
