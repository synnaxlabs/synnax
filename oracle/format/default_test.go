// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package format_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/format"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Default Registry", func() {
	var repoRoot string

	BeforeEach(func() {
		repoRoot = MustSucceed(os.MkdirTemp("", "default-registry"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(repoRoot)).To(Succeed())
		})
		Expect(
			os.MkdirAll(filepath.Join(repoRoot, "licenses", "headers"), 0o755),
		).To(Succeed())
		Expect(os.WriteFile(
			filepath.Join(repoRoot, "licenses/headers/template.txt"),
			[]byte("Copyright {{YEAR}} Synnax Labs, Inc.\n"), 0o644,
		)).To(Succeed())
	})

	It("Should propagate license-template read errors", func() {
		Expect(
			os.Remove(filepath.Join(repoRoot, "licenses/headers/template.txt")),
		).To(Succeed())
		Expect(
			format.Default(repoRoot, 2026, nil),
		).Error().
			To(MatchError(ContainSubstring("license template")))
	})

	// A routed file fails on the canceled context. A file no chain routes passes
	// through unchanged.
	canceled := func() context.Context {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}
	formatOne := func(r *format.Registry, ext string) format.Result {
		files := []format.File{
			{Path: filepath.Join(repoRoot, "file"+ext), Content: []byte("x")},
		}
		return r.FormatBatch(canceled(), files, 1)[0]
	}

	DescribeTable("Should register every supported extension",
		func(ext string) {
			r := MustSucceed(format.Default(repoRoot, 2026, nil))
			Expect(formatOne(r, ext).Err).To(MatchError(context.Canceled))
		},
		Entry("Go", ".go"),
		Entry("C++ source", ".cpp"),
		Entry("C++ source (.cc)", ".cc"),
		Entry("C++ source (.cxx)", ".cxx"),
		Entry("C++ header", ".h"),
		Entry("C++ header (.hpp)", ".hpp"),
		Entry("TypeScript", ".ts"),
		Entry("TypeScript JSX", ".tsx"),
		Entry("Python", ".py"),
		Entry("Protobuf", ".proto"),
	)

	DescribeTable(
		"Should not route unknown extensions through any formatter",
		func(ext string) {
			r := MustSucceed(format.Default(repoRoot, 2026, nil))
			Expect(formatOne(r, ext)).To(Equal(format.Result{Content: []byte("x")}))
		},
		Entry("Rust", ".rs"),
		Entry("Markdown", ".md"),
	)
})
