// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package embed_test

import (
	"embed"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	localembed "github.com/synnaxlabs/x/embed"
)

//go:embed testdata/*
var testFiles embed.FS

var emptyFS embed.FS

/*
	Expected file structure located in the directory that contains this test file:
	└──  emptydata
	└── testdata
		├── nested
		│	 └── inner.txt
		└── simple.txt - "This is a simple file."
*/

var _ = Describe("Extract", func() {
	var (
		tempDir  string
		dirPerm  os.FileMode
		filePerm os.FileMode
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "extract_test_*")
		Expect(err).ToNot(HaveOccurred())
		dirPerm = 0o755
		filePerm = 0o644
	})

	AfterEach(func() {
		_ = os.RemoveAll(tempDir)
	})

	It("should extract a simple set of files", func() {
		err := localembed.Extract(testFiles, tempDir, dirPerm, filePerm)
		Expect(err).ToNot(HaveOccurred())

		contents, err := os.ReadFile(filepath.Join(tempDir, "testdata", "simple.txt"))
		Expect(err).ToNot(HaveOccurred())
		Expect(string(contents)).To(Equal("This is a simple file.\n"))
	})

	It("should extract nested directories", func() {
		err := localembed.Extract(testFiles, tempDir, dirPerm, filePerm)
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Stat(filepath.Join(tempDir, "testdata", "nested", "inner.txt"))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should set directory permissions correctly", func() {
		// Use a custom dirPerm and verify it
		customDirPerm := os.FileMode(0o700)
		err := localembed.Extract(testFiles, tempDir, customDirPerm, filePerm)
		Expect(err).ToNot(HaveOccurred())

		info, err := os.Stat(filepath.Join(tempDir, "testdata"))
		Expect(err).ToNot(HaveOccurred())
		Expect(info.Mode().Perm()).To(Equal(customDirPerm))
	})

	It("should set file permissions correctly", func() {
		// Use a custom filePerm and verify it
		customFilePerm := os.FileMode(0o600)
		err := localembed.Extract(testFiles, tempDir, dirPerm, customFilePerm)
		Expect(err).ToNot(HaveOccurred())

		info, err := os.Stat(filepath.Join(tempDir, "testdata", "simple.txt"))
		Expect(err).ToNot(HaveOccurred())
		Expect(info.Mode().Perm()).To(Equal(customFilePerm))
	})

	Context("when directory creation fails", func() {
		It("should return an error", func() {
			// Try extracting to a directory where we don't have permission
			noPermDir := "/root/no_access_dir"
			err := localembed.Extract(testFiles, noPermDir, dirPerm, filePerm)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when writing a file fails", func() {
		It("should return an error", func() {
			// Make the directory read-only and attempt extraction
			err := os.Chmod(tempDir, 0o500) // read/execute only
			Expect(err).ToNot(HaveOccurred())

			err = localembed.Extract(testFiles, tempDir, dirPerm, filePerm)
			Expect(err).To(HaveOccurred())
		})
	})

	It("should handle empty embedded file systems gracefully", func() {
		err := localembed.Extract(emptyFS, tempDir, dirPerm, filePerm)
		Expect(err).ToNot(HaveOccurred())

		entries, err := os.ReadDir(tempDir)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(entries)).To(Equal(0))
	})

	It("should be idempotent on repeated extractions", func() {
		err := localembed.Extract(testFiles, tempDir, dirPerm, filePerm)
		Expect(err).ToNot(HaveOccurred())

		err = localembed.Extract(testFiles, tempDir, dirPerm, filePerm)
		Expect(err).ToNot(HaveOccurred())

		contents, err := os.ReadFile(filepath.Join(tempDir, "testdata", "simple.txt"))
		Expect(err).ToNot(HaveOccurred())
		Expect(string(contents)).To(Equal("This is a simple file.\n"))
	})

	It("should recreate directory structure only as needed", func() {
		// Remove write permissions from the tempDir after creating a file.
		// This ensures no extraneous directories are created unless needed.
		err := localembed.Extract(testFiles, tempDir, dirPerm, filePerm)
		Expect(err).ToNot(HaveOccurred())

		err = os.Chmod(tempDir, 0o500)
		Expect(err).ToNot(HaveOccurred())

		// Try extracting again. Since directories already exist, it should still work,
		// but if new directories were needed, it might fail.
		err = localembed.Extract(testFiles, tempDir, dirPerm, filePerm)
		// Depending on your scenario, this might or might not fail, but typically since
		// all directories exist, it won't need to create new directories.
		// If new directories are needed, this could fail. Adjust test data as needed.
		Expect(err).ToNot(HaveOccurred())
	})
})
