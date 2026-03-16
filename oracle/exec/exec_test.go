// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package exec_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/exec"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("GroupByConfigDir", func() {
	var tmpDir string

	BeforeEach(func() {
		tmpDir = MustSucceed(os.MkdirTemp("", "exec-test"))
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("should group files by the nearest directory containing the config file", func() {
		projA := filepath.Join(tmpDir, "projA")
		projB := filepath.Join(tmpDir, "projB")
		Expect(os.MkdirAll(filepath.Join(projA, "src"), 0o755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(projB, "src"), 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(projA, "package.json"), []byte("{}"), 0o644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(projB, "package.json"), []byte("{}"), 0o644)).To(Succeed())

		files := []string{
			filepath.Join(projA, "src", "a.ts"),
			filepath.Join(projA, "src", "b.ts"),
			filepath.Join(projB, "src", "c.ts"),
		}

		result := exec.GroupByConfigDir(files, "package.json")
		Expect(result).To(HaveLen(2))
		Expect(result[projA]).To(HaveLen(2))
		Expect(result[projB]).To(HaveLen(1))
	})

	It("should skip files with no ancestor containing the config file", func() {
		files := []string{filepath.Join(tmpDir, "orphan", "file.ts")}
		result := exec.GroupByConfigDir(files, "nonexistent.json")
		Expect(result).To(BeEmpty())
	})

	It("should return empty map for empty input", func() {
		result := exec.GroupByConfigDir([]string{}, "package.json")
		Expect(result).To(BeEmpty())
	})
})

var _ = Describe("FindConfigDir", func() {
	var tmpDir string

	BeforeEach(func() {
		tmpDir = MustSucceed(os.MkdirTemp("", "exec-test"))
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("should find the nearest ancestor directory with the config file", func() {
		projDir := filepath.Join(tmpDir, "proj")
		subDir := filepath.Join(projDir, "src", "sub")
		Expect(os.MkdirAll(subDir, 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(projDir, "go.mod"), []byte("module test"), 0o644)).To(Succeed())

		result := exec.FindConfigDir(filepath.Join(subDir, "file.go"), "go.mod")
		Expect(result).To(Equal(projDir))
	})

	It("should return empty string when no config file exists", func() {
		result := exec.FindConfigDir(filepath.Join(tmpDir, "no", "config", "file.go"), "go.mod")
		Expect(result).To(BeEmpty())
	})
})

var _ = Describe("OnFiles", func() {
	It("should run a command successfully", func() {
		err := exec.OnFiles([]string{"echo", "hello"}, []string{"a.txt"}, "")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return an error for a failing command", func() {
		err := exec.OnFiles([]string{"false"}, []string{"a.txt"}, "")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("PostWriter", func() {
	It("should return nil for empty file list", func() {
		w := &exec.PostWriter{Commands: [][]string{{"echo"}}}
		Expect(w.PostWrite([]string{})).To(Succeed())
	})

	It("should filter files by extension", func() {
		w := &exec.PostWriter{
			Extensions: []string{".go"},
			Commands:   [][]string{{"echo"}},
		}
		Expect(w.PostWrite([]string{"file.ts", "other.js"})).To(Succeed())
	})

	It("should run commands on files matching extensions", func() {
		w := &exec.PostWriter{
			Extensions: []string{".txt"},
			Commands:   [][]string{{"echo"}},
		}
		Expect(w.PostWrite([]string{"a.txt", "b.txt"})).To(Succeed())
	})

	It("should run commands without config file from current directory", func() {
		w := &exec.PostWriter{
			Commands: [][]string{{"echo"}},
		}
		Expect(w.PostWrite([]string{"a.txt"})).To(Succeed())
	})
})
