// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("readCoreVersion", func() {
	var repoRoot string

	BeforeEach(func() {
		repoRoot = MustSucceed(os.MkdirTemp("", "repo"))
		versionDir := filepath.Join(repoRoot, "core", "pkg", "version")
		Expect(os.MkdirAll(versionDir, 0755)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(repoRoot)).To(Succeed())
	})

	It("should parse a standard version", func() {
		Expect(os.WriteFile(
			filepath.Join(repoRoot, "core", "pkg", "version", "VERSION"),
			[]byte("0.53.4"), 0644,
		)).To(Succeed())
		v := MustSucceed(readCoreVersion(repoRoot))
		Expect(v).To(Equal(53))
	})

	It("should handle major version > 0", func() {
		Expect(os.WriteFile(
			filepath.Join(repoRoot, "core", "pkg", "version", "VERSION"),
			[]byte("2.10.0"), 0644,
		)).To(Succeed())
		v := MustSucceed(readCoreVersion(repoRoot))
		Expect(v).To(Equal(2010))
	})

	It("should handle version with trailing newline", func() {
		Expect(os.WriteFile(
			filepath.Join(repoRoot, "core", "pkg", "version", "VERSION"),
			[]byte("0.53.4\n"), 0644,
		)).To(Succeed())
		v := MustSucceed(readCoreVersion(repoRoot))
		Expect(v).To(Equal(53))
	})

	It("should error on missing VERSION file", func() {
		_, err := readCoreVersion(filepath.Join(repoRoot, "nonexistent"))
		Expect(err).To(HaveOccurred())
	})

	It("should error on invalid version format", func() {
		Expect(os.WriteFile(
			filepath.Join(repoRoot, "core", "pkg", "version", "VERSION"),
			[]byte("invalid"), 0644,
		)).To(Succeed())
		_, err := readCoreVersion(repoRoot)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("invalid version format")))
	})

	It("should error on non-numeric major version", func() {
		Expect(os.WriteFile(
			filepath.Join(repoRoot, "core", "pkg", "version", "VERSION"),
			[]byte("abc.53.4"), 0644,
		)).To(Succeed())
		_, err := readCoreVersion(repoRoot)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("invalid major version")))
	})

	It("should error on non-numeric minor version", func() {
		Expect(os.WriteFile(
			filepath.Join(repoRoot, "core", "pkg", "version", "VERSION"),
			[]byte("0.abc.4"), 0644,
		)).To(Succeed())
		_, err := readCoreVersion(repoRoot)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("invalid minor version")))
	})
})

var _ = Describe("writeFileIfChanged", func() {
	var tmpDir string

	BeforeEach(func() {
		tmpDir = MustSucceed(os.MkdirTemp("", "write"))
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("should create a new file", func() {
		path := filepath.Join(tmpDir, "new.go")
		Expect(writeFileIfChanged(path, []byte("package main"))).To(Succeed())
		content := string(MustSucceed(os.ReadFile(path)))
		Expect(content).To(Equal("package main"))
	})

	It("should overwrite a file with different content", func() {
		path := filepath.Join(tmpDir, "existing.go")
		Expect(os.WriteFile(path, []byte("old content"), 0644)).To(Succeed())
		Expect(writeFileIfChanged(path, []byte("new content"))).To(Succeed())
		content := string(MustSucceed(os.ReadFile(path)))
		Expect(content).To(Equal("new content"))
	})

	It("should not write when content is identical", func() {
		path := filepath.Join(tmpDir, "same.go")
		Expect(os.WriteFile(path, []byte("unchanged"), 0644)).To(Succeed())
		info := MustSucceed(os.Stat(path))
		origModTime := info.ModTime()

		Expect(writeFileIfChanged(path, []byte("unchanged"))).To(Succeed())
		info2 := MustSucceed(os.Stat(path))
		Expect(info2.ModTime()).To(Equal(origModTime))
	})

	It("should create intermediate directories", func() {
		path := filepath.Join(tmpDir, "a", "b", "c", "file.go")
		Expect(writeFileIfChanged(path, []byte("deep"))).To(Succeed())
		content := string(MustSucceed(os.ReadFile(path)))
		Expect(content).To(Equal("deep"))
	})
})

var _ = Describe("findMigrationVersions", func() {
	var migrationsDir string

	BeforeEach(func() {
		migrationsDir = MustSucceed(os.MkdirTemp("", "migrations"))
	})

	AfterEach(func() {
		Expect(os.RemoveAll(migrationsDir)).To(Succeed())
	})

	It("should return nil for nonexistent directory", func() {
		versions := MustSucceed(findMigrationVersions(filepath.Join(migrationsDir, "nonexistent")))
		Expect(versions).To(BeNil())
	})

	It("should find version directories", func() {
		Expect(os.MkdirAll(filepath.Join(migrationsDir, "v53"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(migrationsDir, "v54"), 0755)).To(Succeed())

		versions := MustSucceed(findMigrationVersions(migrationsDir))
		Expect(versions).To(ConsistOf(53, 54))
	})

	It("should ignore non-version entries", func() {
		Expect(os.MkdirAll(filepath.Join(migrationsDir, "v10"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(migrationsDir, "other"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(migrationsDir, "file.txt"), []byte("x"), 0644)).To(Succeed())

		versions := MustSucceed(findMigrationVersions(migrationsDir))
		Expect(versions).To(Equal([]int{10}))
	})

	It("should return empty for directory with no versions", func() {
		versions := MustSucceed(findMigrationVersions(migrationsDir))
		Expect(versions).To(BeNil())
	})
})
