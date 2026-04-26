// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package snapshot_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/snapshot"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Create", func() {
	var schemasDir, snapshotsDir string

	BeforeEach(func() {
		schemasDir = MustSucceed(os.MkdirTemp("", "schemas"))
		snapshotsDir = filepath.Join(schemasDir, ".snapshots")
		Expect(os.MkdirAll(snapshotsDir, 0755)).To(Succeed())
		DeferCleanup(func() {
			Expect(os.RemoveAll(schemasDir)).To(Succeed())
		})
	})

	It("should copy oracle files into the versioned snapshot directory", func() {
		Expect(os.WriteFile(
			filepath.Join(schemasDir, "user.oracle"),
			[]byte("User struct { key uuid }"), 0644,
		)).To(Succeed())

		Expect(snapshot.Create(schemasDir, snapshotsDir, 53)).To(Succeed())

		content := string(MustSucceed(os.ReadFile(
			filepath.Join(snapshotsDir, "v53", "user.oracle"),
		)))
		Expect(content).To(Equal("User struct { key uuid }"))
	})

	It("should preserve subdirectory structure", func() {
		subDir := filepath.Join(schemasDir, "core")
		Expect(os.MkdirAll(subDir, 0755)).To(Succeed())
		Expect(os.WriteFile(
			filepath.Join(subDir, "channel.oracle"),
			[]byte("Channel struct {}"), 0644,
		)).To(Succeed())

		Expect(snapshot.Create(schemasDir, snapshotsDir, 1)).To(Succeed())

		content := string(MustSucceed(os.ReadFile(
			filepath.Join(snapshotsDir, "v1", "core", "channel.oracle"),
		)))
		Expect(content).To(Equal("Channel struct {}"))
	})

	It("should skip the .snapshots directory itself", func() {
		Expect(os.WriteFile(
			filepath.Join(schemasDir, "root.oracle"),
			[]byte("Root struct {}"), 0644,
		)).To(Succeed())
		existingSnapshot := filepath.Join(snapshotsDir, "v1")
		Expect(os.MkdirAll(existingSnapshot, 0755)).To(Succeed())
		Expect(os.WriteFile(
			filepath.Join(existingSnapshot, "old.oracle"),
			[]byte("Old struct {}"), 0644,
		)).To(Succeed())

		Expect(snapshot.Create(schemasDir, snapshotsDir, 2)).To(Succeed())

		_, err := os.Stat(filepath.Join(snapshotsDir, "v2", "old.oracle"))
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("should skip non-oracle files", func() {
		Expect(os.WriteFile(
			filepath.Join(schemasDir, "readme.md"),
			[]byte("# Readme"), 0644,
		)).To(Succeed())
		Expect(os.WriteFile(
			filepath.Join(schemasDir, "schema.oracle"),
			[]byte("S struct {}"), 0644,
		)).To(Succeed())

		Expect(snapshot.Create(schemasDir, snapshotsDir, 3)).To(Succeed())

		_, err := os.Stat(filepath.Join(snapshotsDir, "v3", "readme.md"))
		Expect(os.IsNotExist(err)).To(BeTrue())
	})
})

var _ = Describe("LatestVersion", func() {
	var snapshotsDir string

	BeforeEach(func() {
		snapshotsDir = MustSucceed(os.MkdirTemp("", "snapshots"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(snapshotsDir)).To(Succeed())
		})
	})

	It("should return 0 when no snapshots exist", func() {
		v := MustSucceed(snapshot.LatestVersion(snapshotsDir))
		Expect(v).To(Equal(0))
	})

	It("should return 0 when the directory does not exist", func() {
		v := MustSucceed(snapshot.LatestVersion(filepath.Join(snapshotsDir, "nonexistent")))
		Expect(v).To(Equal(0))
	})

	It("should return the highest version number", func() {
		Expect(os.MkdirAll(filepath.Join(snapshotsDir, "v1"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(snapshotsDir, "v53"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(snapshotsDir, "v10"), 0755)).To(Succeed())

		v := MustSucceed(snapshot.LatestVersion(snapshotsDir))
		Expect(v).To(Equal(53))
	})

	It("should ignore non-version directories", func() {
		Expect(os.MkdirAll(filepath.Join(snapshotsDir, "v5"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(snapshotsDir, "other"), 0755)).To(Succeed())
		Expect(os.WriteFile(
			filepath.Join(snapshotsDir, "file.txt"),
			[]byte("not a dir"), 0644,
		)).To(Succeed())

		v := MustSucceed(snapshot.LatestVersion(snapshotsDir))
		Expect(v).To(Equal(5))
	})
})

var _ = Describe("Files", func() {
	var snapshotDir string

	BeforeEach(func() {
		snapshotDir = MustSucceed(os.MkdirTemp("", "snapshot"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(snapshotDir)).To(Succeed())
		})
	})

	It("should return sorted oracle files", func() {
		Expect(os.WriteFile(filepath.Join(snapshotDir, "b.oracle"), []byte("B"), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(snapshotDir, "a.oracle"), []byte("A"), 0644)).To(Succeed())

		files := MustSucceed(snapshot.Files(snapshotDir))
		Expect(files).To(HaveLen(2))
		Expect(files[0]).To(HaveSuffix("a.oracle"))
		Expect(files[1]).To(HaveSuffix("b.oracle"))
	})

	It("should include files in subdirectories", func() {
		subDir := filepath.Join(snapshotDir, "sub")
		Expect(os.MkdirAll(subDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(subDir, "nested.oracle"), []byte("N"), 0644)).To(Succeed())

		files := MustSucceed(snapshot.Files(snapshotDir))
		Expect(files).To(HaveLen(1))
		Expect(files[0]).To(HaveSuffix("nested.oracle"))
	})

	It("should ignore non-oracle files", func() {
		Expect(os.WriteFile(filepath.Join(snapshotDir, "schema.oracle"), []byte("S"), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(snapshotDir, "readme.md"), []byte("R"), 0644)).To(Succeed())

		files := MustSucceed(snapshot.Files(snapshotDir))
		Expect(files).To(HaveLen(1))
	})
})

var _ = Describe("FileLoader", func() {
	var (
		snapshotDir string
		repoRoot    string
		loader      *snapshot.FileLoader
	)

	BeforeEach(func() {
		snapshotDir = MustSucceed(os.MkdirTemp("", "snapshot"))
		repoRoot = "/fake/repo"
		loader = snapshot.NewFileLoader(snapshotDir, repoRoot)
		DeferCleanup(func() {
			Expect(os.RemoveAll(snapshotDir)).To(Succeed())
		})
	})

	It("should load a file from the snapshot directory", func() {
		Expect(os.WriteFile(
			filepath.Join(snapshotDir, "user.oracle"),
			[]byte("User struct {}"), 0644,
		)).To(Succeed())

		content, filePath := MustSucceed2(loader.Load("schemas/user"))
		Expect(content).To(Equal("User struct {}"))
		Expect(filePath).To(Equal("schemas/user.oracle"))
	})

	It("should handle nested import paths", func() {
		subDir := filepath.Join(snapshotDir, "core")
		Expect(os.MkdirAll(subDir, 0755)).To(Succeed())
		Expect(os.WriteFile(
			filepath.Join(subDir, "channel.oracle"),
			[]byte("Channel struct {}"), 0644,
		)).To(Succeed())

		content, filePath := MustSucceed2(loader.Load("schemas/core/channel"))
		Expect(content).To(Equal("Channel struct {}"))
		Expect(filePath).To(Equal("schemas/core/channel.oracle"))
	})

	It("should return an error for nonexistent files", func() {
		_, _, err := loader.Load("schemas/nonexistent")
		Expect(err).To(HaveOccurred())
	})

	It("should return the repo root", func() {
		Expect(loader.RepoRoot()).To(Equal(repoRoot))
	})
})
