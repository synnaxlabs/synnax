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

