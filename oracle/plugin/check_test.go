// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package plugin_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Suite")
}

var _ = Describe("Check Utilities", func() {
	var tempDir string

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "oracle-check-test")
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("FileModTime", func() {
		It("Should return modification time for existing file", func() {
			filePath := filepath.Join(tempDir, "test.txt")
			err := os.WriteFile(filePath, []byte("test"), 0644)
			Expect(err).To(BeNil())

			modTime := plugin.FileModTime(filePath)
			Expect(modTime.IsZero()).To(BeFalse())
			Expect(modTime.Before(time.Now().Add(time.Second))).To(BeTrue())
		})

		It("Should return zero time for non-existent file", func() {
			modTime := plugin.FileModTime(filepath.Join(tempDir, "nonexistent.txt"))
			Expect(modTime.IsZero()).To(BeTrue())
		})
	})

	Describe("CheckFreshness", func() {
		It("Should return nil when all files are fresh", func() {
			// Create schema file
			schemaPath := filepath.Join(tempDir, "schema.oracle")
			err := os.WriteFile(schemaPath, []byte("struct Test {}"), 0644)
			Expect(err).To(BeNil())

			// Wait a bit, then create generated file (newer)
			time.Sleep(10 * time.Millisecond)
			genPath := filepath.Join(tempDir, "test.gen.go")
			err = os.WriteFile(genPath, []byte("package test"), 0644)
			Expect(err).To(BeNil())

			genFiles := map[string][]string{
				genPath: {schemaPath},
			}
			result := plugin.CheckFreshness("test", genFiles)
			Expect(result).To(BeNil())
		})

		It("Should return StaleError when generated file is older than schema", func() {
			// Create generated file first
			genPath := filepath.Join(tempDir, "test.gen.go")
			err := os.WriteFile(genPath, []byte("package test"), 0644)
			Expect(err).To(BeNil())

			// Wait a bit, then create schema file (newer)
			time.Sleep(10 * time.Millisecond)
			schemaPath := filepath.Join(tempDir, "schema.oracle")
			err = os.WriteFile(schemaPath, []byte("struct Test {}"), 0644)
			Expect(err).To(BeNil())

			genFiles := map[string][]string{
				genPath: {schemaPath},
			}
			result := plugin.CheckFreshness("test", genFiles)
			Expect(result).NotTo(BeNil())

			staleErr, ok := result.(*plugin.StaleError)
			Expect(ok).To(BeTrue())
			Expect(staleErr.Plugin).To(Equal("test"))
			Expect(staleErr.Files).To(HaveLen(1))
			Expect(staleErr.Files[0].Generated).To(Equal(genPath))
			Expect(staleErr.Files[0].Schema).To(Equal(schemaPath))
		})

		It("Should return StaleError when generated file is missing", func() {
			schemaPath := filepath.Join(tempDir, "schema.oracle")
			err := os.WriteFile(schemaPath, []byte("struct Test {}"), 0644)
			Expect(err).To(BeNil())

			genPath := filepath.Join(tempDir, "test.gen.go") // Does not exist

			genFiles := map[string][]string{
				genPath: {schemaPath},
			}
			result := plugin.CheckFreshness("test", genFiles)
			Expect(result).NotTo(BeNil())

			staleErr, ok := result.(*plugin.StaleError)
			Expect(ok).To(BeTrue())
			Expect(staleErr.Files).To(HaveLen(1))
			Expect(staleErr.Files[0].GenTime.IsZero()).To(BeTrue()) // Missing file has zero time
		})

		It("Should handle multiple generated files", func() {
			schemaPath := filepath.Join(tempDir, "schema.oracle")
			err := os.WriteFile(schemaPath, []byte("struct Test {}"), 0644)
			Expect(err).To(BeNil())

			time.Sleep(10 * time.Millisecond)

			// One fresh, one stale
			freshPath := filepath.Join(tempDir, "fresh.gen.go")
			err = os.WriteFile(freshPath, []byte("package test"), 0644)
			Expect(err).To(BeNil())

			stalePath := filepath.Join(tempDir, "stale.gen.go") // Missing

			genFiles := map[string][]string{
				freshPath: {schemaPath},
				stalePath: {schemaPath},
			}
			result := plugin.CheckFreshness("test", genFiles)
			Expect(result).NotTo(BeNil())

			staleErr, ok := result.(*plugin.StaleError)
			Expect(ok).To(BeTrue())
			Expect(staleErr.Files).To(HaveLen(1)) // Only the stale one
			Expect(staleErr.Files[0].Generated).To(Equal(stalePath))
		})
	})

	Describe("StaleError", func() {
		It("Should format error message with stale files", func() {
			err := &plugin.StaleError{
				Plugin: "go-types",
				Files: []plugin.StaleFile{
					{
						Generated:  "core/ranger/types.gen.go",
						Schema:     "schemas/ranger.oracle",
						GenTime:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
						SchemaTime: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
					},
				},
			}

			msg := err.Error()
			Expect(msg).To(ContainSubstring("go-types"))
			Expect(msg).To(ContainSubstring("1 stale file(s)"))
			Expect(msg).To(ContainSubstring("core/ranger/types.gen.go"))
			Expect(msg).To(ContainSubstring("schemas/ranger.oracle"))
		})

		It("Should indicate missing files", func() {
			err := &plugin.StaleError{
				Plugin: "go-types",
				Files: []plugin.StaleFile{
					{
						Generated:  "core/ranger/types.gen.go",
						Schema:     "schemas/ranger.oracle",
						GenTime:    time.Time{}, // Zero = missing
						SchemaTime: time.Now(),
					},
				},
			}

			msg := err.Error()
			Expect(msg).To(ContainSubstring("missing"))
		})
	})

	Describe("DependencyStaleError", func() {
		It("Should format error message with dependency info", func() {
			innerErr := &plugin.StaleError{
				Plugin: "go-types",
				Files: []plugin.StaleFile{
					{
						Generated: "core/ranger/types.gen.go",
						Schema:    "schemas/ranger.oracle",
					},
				},
			}

			err := &plugin.DependencyStaleError{
				Plugin:     "go-query",
				Dependency: "go-types",
				Reason:     innerErr,
			}

			msg := err.Error()
			Expect(msg).To(ContainSubstring("go-query"))
			Expect(msg).To(ContainSubstring("go-types"))
			Expect(msg).To(ContainSubstring("oracle generate -p go-types"))
		})
	})
})
