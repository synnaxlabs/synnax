// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("MockFileLoader", func() {
	Describe("NewMockFileLoader", func() {
		It("should create loader with empty files map", func() {
			loader := testutil.NewMockFileLoader()
			Expect(loader.Files).To(BeEmpty())
		})

		It("should use default repo root", func() {
			loader := testutil.NewMockFileLoader()
			Expect(loader.RepoRoot()).To(Equal("/mock/repo"))
		})
	})

	Describe("WithFiles", func() {
		It("should create loader with pre-populated files", func() {
			files := map[string]string{
				"schema/user":  "struct User {}",
				"schema/range": "struct Range {}",
			}
			loader := testutil.WithFiles(files)
			Expect(loader.Files).To(HaveLen(2))
			Expect(loader.Files["schema/user"]).To(Equal("struct User {}"))
		})
	})

	Describe("Add", func() {
		It("should add file and return loader for chaining", func() {
			loader := testutil.NewMockFileLoader().
				Add("schema/user", "struct User {}").
				Add("schema/range", "struct Range {}")
			Expect(loader.Files).To(HaveLen(2))
		})
	})

	Describe("WithRepoRoot", func() {
		It("should set custom repo root", func() {
			loader := testutil.NewMockFileLoader().WithRepoRoot("/custom/path")
			Expect(loader.RepoRoot()).To(Equal("/custom/path"))
		})
	})

	Describe("Load", func() {
		var loader *testutil.MockFileLoader

		BeforeEach(func() {
			loader = testutil.NewMockFileLoader().
				Add("schema/user", "struct User { field key uuid }").
				Add("schema/range.oracle", "struct Range { field key uuid }")
		})

		It("should load file by exact path", func() {
			content, path, err := loader.Load("schema/user")
			Expect(err).To(BeNil())
			Expect(content).To(Equal("struct User { field key uuid }"))
			Expect(path).To(Equal("schema/user.oracle"))
		})

		It("should load file with .oracle extension", func() {
			content, path, err := loader.Load("schema/range")
			Expect(err).To(BeNil())
			Expect(content).To(Equal("struct Range { field key uuid }"))
			Expect(path).To(Equal("schema/range.oracle"))
		})

		It("should return FileNotFoundError for missing file", func() {
			_, _, err := loader.Load("schema/nonexistent")
			Expect(err).To(HaveOccurred())
			var fnf *testutil.FileNotFoundError
			Expect(err).To(BeAssignableToTypeOf(fnf))
		})
	})

	Describe("FileNotFoundError", func() {
		It("should include path in error message", func() {
			err := &testutil.FileNotFoundError{Path: "schema/missing"}
			Expect(err.Error()).To(Equal("file not found: schema/missing"))
		})
	})
})
