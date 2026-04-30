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
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/format"
	. "github.com/synnaxlabs/x/testutil"
)

const fakeTemplate = "Copyright {{YEAR}} Synnax Labs, Inc.\n\nGoverned by BSL.\n"

var _ = Describe("License", func() {
	var repoRoot string

	BeforeEach(func() {
		repoRoot = MustSucceed(os.MkdirTemp("", "license"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(repoRoot)).To(Succeed())
		})
		Expect(os.MkdirAll(filepath.Join(repoRoot, "licenses", "headers"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(repoRoot, format.LicenseTemplatePath), []byte(fakeTemplate), 0644)).To(Succeed())
	})

	It("Should prepend a slash header to a Go file with no header", func(ctx SpecContext) {
		l := MustSucceed(format.NewLicense(repoRoot))
		out := MustSucceed(l.Format(ctx, []byte("package x\n"), "/abs/foo.go"))
		Expect(string(out)).To(HavePrefix("// Copyright"))
		Expect(string(out)).To(ContainSubstring("Synnax Labs, Inc."))
		Expect(string(out)).To(HaveSuffix("package x\n"))
	})

	It("Should prepend a hash header to a Python file", func(ctx SpecContext) {
		l := MustSucceed(format.NewLicense(repoRoot))
		out := MustSucceed(l.Format(ctx, []byte("def foo(): pass\n"), "/abs/foo.py"))
		Expect(string(out)).To(HavePrefix("#  Copyright"))
		Expect(strings.Contains(string(out), "#  Governed by BSL.")).To(BeTrue())
	})

	It("Should prepend a block header to a CSS file", func(ctx SpecContext) {
		l := MustSucceed(format.NewLicense(repoRoot))
		out := MustSucceed(l.Format(ctx, []byte(".cls { color: red; }\n"), "/abs/foo.css"))
		Expect(string(out)).To(HavePrefix("/*\n * Copyright"))
		Expect(string(out)).To(ContainSubstring(" */\n\n.cls"))
	})

	It("Should be idempotent on a file with the correct header already", func(ctx SpecContext) {
		l := MustSucceed(format.NewLicense(repoRoot))
		first := MustSucceed(l.Format(ctx, []byte("package x\n"), "/abs/foo.go"))
		second := MustSucceed(l.Format(ctx, first, "/abs/foo.go"))
		Expect(string(first)).To(Equal(string(second)))
	})

	It("Should replace a stale-year header", func(ctx SpecContext) {
		l := MustSucceed(format.NewLicense(repoRoot))
		stale := []byte("// Copyright 1999 Synnax Labs, Inc.\n//\n// Governed by BSL.\n\npackage x\n")
		out := MustSucceed(l.Format(ctx, stale, "/abs/foo.go"))
		Expect(string(out)).ToNot(ContainSubstring("1999"))
		Expect(string(out)).To(HaveSuffix("package x\n"))
	})

	It("Should leave files with non-Synnax leading comments alone", func(ctx SpecContext) {
		l := MustSucceed(format.NewLicense(repoRoot))
		raw := []byte("// some unrelated comment\npackage x\n")
		out := MustSucceed(l.Format(ctx, raw, "/abs/foo.go"))
		Expect(string(out)).To(HavePrefix("// Copyright"))
		Expect(string(out)).To(ContainSubstring("// some unrelated comment"))
	})

	It("Should pass through unsupported extensions unchanged", func(ctx SpecContext) {
		l := MustSucceed(format.NewLicense(repoRoot))
		out := MustSucceed(l.Format(ctx, []byte("body"), "/abs/foo.unknown"))
		Expect(string(out)).To(Equal("body"))
	})
})
