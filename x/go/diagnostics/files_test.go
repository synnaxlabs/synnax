// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package diagnostics_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/diagnostics"
)

var _ = Describe("Files", func() {
	var fd *diagnostics.Files

	errAt := func(msg string) diagnostics.Diagnostic {
		return diagnostics.Errorf(nil, "%s", msg)
	}
	warnAt := func(msg string) diagnostics.Diagnostic {
		return diagnostics.Warningf(nil, "%s", msg)
	}

	BeforeEach(func() { fd = diagnostics.NewFiles() })

	Describe("Empty and Ok", func() {
		It("Should be empty and ok with no diagnostics", func() {
			Expect(fd.Empty()).To(BeTrue())
			Expect(fd.Ok()).To(BeTrue())
		})

		It("Should stay ok when only warnings are reported", func() {
			fd.Report("a.oracle", warnAt("unresolved type"))
			Expect(fd.Empty()).To(BeFalse())
			Expect(fd.Ok()).To(BeTrue())
		})

		It("Should not be ok once any file holds an error", func() {
			fd.Report("a.oracle", warnAt("unresolved type"))
			fd.Report("b.oracle", errAt("duplicate type"))
			Expect(fd.Ok()).To(BeFalse())
		})
	})

	Describe("Report and Add", func() {
		It("Should deduplicate within a file but not across files", func() {
			fd.Report("a.oracle", errAt("boom"))
			fd.Report("a.oracle", errAt("boom"))
			fd.Report("b.oracle", errAt("boom"))
			Expect(fd.Flat()).To(HaveLen(2))
		})

		It("Should record file-less diagnostics under the empty file", func() {
			fd.Add(errAt("global failure"))
			var files []string
			fd.Each(func(file string, _ diagnostics.Diagnostic) {
				files = append(files, file)
			})
			Expect(files).To(Equal([]string{""}))
		})
	})

	Describe("MergeFile", func() {
		It("Should fold a flat set of diagnostics under one file", func() {
			fd.MergeFile("a.oracle", diagnostics.Diagnostics{
				errAt("first"),
				errAt("second"),
			})
			Expect(fd.Flat()).To(HaveLen(2))
			Expect(fd.Ok()).To(BeFalse())
		})
	})

	Describe("Combine", func() {
		It("Should fold another collection preserving file grouping", func() {
			fd.Report("a.oracle", errAt("from a"))
			other := diagnostics.NewFiles()
			other.Report("a.oracle", errAt("from a"))
			other.Report("b.oracle", warnAt("from b"))
			fd.Combine(other)
			var files []string
			fd.Each(func(file string, _ diagnostics.Diagnostic) {
				files = append(files, file)
			})
			Expect(files).To(Equal([]string{"a.oracle", "b.oracle"}))
		})
	})

	Describe("Errors, Each, and Flat ordering", func() {
		BeforeEach(func() {
			fd.Report("b.oracle", errAt("b error"))
			fd.Report("a.oracle", warnAt("a warning"))
			fd.Report("a.oracle", errAt("a error"))
		})

		It("Should return only error-level diagnostics in first-seen file order", func() {
			errs := fd.Errors()
			Expect(errs).To(HaveLen(2))
			Expect(errs[0].Message).To(Equal("b error"))
			Expect(errs[1].Message).To(Equal("a error"))
		})

		It("Should visit every diagnostic with its origin file", func() {
			var visited []string
			fd.Each(func(file string, d diagnostics.Diagnostic) {
				visited = append(visited, file+": "+d.Message)
			})
			Expect(visited).To(Equal([]string{
				"b.oracle: b error",
				"a.oracle: a warning",
				"a.oracle: a error",
			}))
		})

		It("Should flatten all diagnostics in file order", func() {
			flat := fd.Flat()
			Expect(flat).To(HaveLen(3))
			Expect(flat[0].Message).To(Equal("b error"))
		})
	})

	Describe("String and Error", func() {
		It("Should report success with no diagnostics", func() {
			Expect(fd.String()).To(Equal("analysis successful"))
		})

		It("Should format every file's diagnostics under its file name", func() {
			fd.Report("a.oracle", errAt("a error"))
			fd.Report("b.oracle", warnAt("b warning"))
			Expect(fd.String()).To(ContainSubstring("a.oracle:\n"))
			Expect(fd.String()).To(ContainSubstring("a error"))
			Expect(fd.String()).To(ContainSubstring("b.oracle:\n"))
			Expect(fd.String()).To(ContainSubstring("b warning"))
		})

		It("Should omit the file header for diagnostics with no file", func() {
			fd.Add(errAt("free-floating error"))
			Expect(fd.String()).ToNot(ContainSubstring(":\n"))
			Expect(fd.String()).To(ContainSubstring("free-floating error"))
		})

		It("Should report the formatted diagnostics as the error message", func() {
			fd.Report("a.oracle", errAt("a error"))
			var err error = fd
			Expect(err.Error()).To(Equal(fd.String()))
		})
	})
})
