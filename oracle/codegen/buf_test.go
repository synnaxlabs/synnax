// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codegen

import (
	"os"
	"path/filepath"
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/format"
	. "github.com/synnaxlabs/x/testutil"
)

// writeFile is a tiny helper that creates parent dirs and writes
// content. It panics through Gomega so callers don't need to thread
// errors through the fixture builders.
func writeFile(path, content string) {
	GinkgoHelper()
	Expect(os.MkdirAll(filepath.Dir(path), 0755)).To(Succeed())
	Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())
}

var _ = Describe("findProtoFiles", func() {
	var root string

	BeforeEach(func() {
		root = MustSucceed(os.MkdirTemp("", "proto-walk"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(root)).To(Succeed())
		})
	})

	It("Should return every .proto under repoRoot", func() {
		writeFile(filepath.Join(root, "x", "a.proto"), "syntax = \"proto3\";")
		writeFile(filepath.Join(root, "y", "z", "b.proto"), "syntax = \"proto3\";")
		out := MustSucceed(findProtoFiles(root))
		sort.Strings(out)
		Expect(out).To(Equal([]string{
			filepath.Join(root, "x", "a.proto"),
			filepath.Join(root, "y", "z", "b.proto"),
		}))
	})

	It("Should ignore non-.proto files", func() {
		writeFile(filepath.Join(root, "a.go"), "package x")
		writeFile(filepath.Join(root, "a.proto"), "syntax = \"proto3\";")
		out := MustSucceed(findProtoFiles(root))
		Expect(out).To(Equal([]string{filepath.Join(root, "a.proto")}))
	})

	It("Should skip vendored / build directories", func() {
		writeFile(filepath.Join(root, "node_modules", "x", "vendored.proto"), "")
		writeFile(filepath.Join(root, "dist", "build.proto"), "")
		writeFile(filepath.Join(root, ".git", "stash.proto"), "")
		writeFile(filepath.Join(root, ".oracle", "cache.proto"), "")
		writeFile(filepath.Join(root, "src", "real.proto"), "")
		out := MustSucceed(findProtoFiles(root))
		Expect(out).To(Equal([]string{filepath.Join(root, "src", "real.proto")}))
	})
})

var _ = Describe("readIfExists", func() {
	var root string

	BeforeEach(func() {
		root = MustSucceed(os.MkdirTemp("", "read-if-exists"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(root)).To(Succeed())
		})
	})

	It("Should return file contents in input order", func() {
		writeFile(filepath.Join(root, "a"), "AAA")
		writeFile(filepath.Join(root, "b"), "BBB")
		out := MustSucceed(readIfExists(
			filepath.Join(root, "a"),
			filepath.Join(root, "b"),
		))
		Expect(out).To(HaveLen(2))
		Expect(string(out[0])).To(Equal("AAA"))
		Expect(string(out[1])).To(Equal("BBB"))
	})

	It("Should leave a nil entry for missing files instead of erroring", func() {
		writeFile(filepath.Join(root, "present"), "X")
		out := MustSucceed(readIfExists(
			filepath.Join(root, "present"),
			filepath.Join(root, "missing"),
		))
		Expect(out).To(HaveLen(2))
		Expect(string(out[0])).To(Equal("X"))
		Expect(out[1]).To(BeNil())
	})

	It("Should propagate non-IsNotExist errors", func() {
		// Pass a path that resolves through a non-directory parent so
		// the read returns an error other than IsNotExist.
		writeFile(filepath.Join(root, "file"), "X")
		Expect(readIfExists(filepath.Join(root, "file", "no-can-do"))).Error().
			To(MatchError(ContainSubstring("read")))
	})
})

var _ = Describe("RunBufGenerate", func() {
	var root string

	BeforeEach(func() {
		root = MustSucceed(os.MkdirTemp("", "buf-run"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(root)).To(Succeed())
		})
		writeFile(filepath.Join(root, "buf.yaml"), "version: v2\n")
		writeFile(filepath.Join(root, "x", "a.proto"), "syntax = \"proto3\";\n")
	})

	It("Should report Cached=true when the stamp matches and no protos changed", func(ctx SpecContext) {
		stamp := MustSucceed(bufInputStamp(ctx, root))
		cache := format.LoadCache(root)
		cache.PutStamp(BufGenerateStampKey, stamp)

		result := MustSucceed(RunBufGenerate(ctx, root, nil, cache))
		Expect(result.Cached).To(BeTrue())
	})

	It("Should not invoke buf when changedProtos is empty and stamp hits", func(ctx SpecContext) {
		// If buf were invoked we'd need it on PATH; reaching this
		// case without an error proves the cache short-circuit ran
		// before any exec.Command happened.
		stamp := MustSucceed(bufInputStamp(ctx, root))
		cache := format.LoadCache(root)
		cache.PutStamp(BufGenerateStampKey, stamp)

		MustSucceed(RunBufGenerate(ctx, root, nil, cache))
	})
})

var _ = Describe("bufInputStamp", func() {
	var root string

	BeforeEach(func() {
		root = MustSucceed(os.MkdirTemp("", "buf-stamp"))
		DeferCleanup(func() {
			Expect(os.RemoveAll(root)).To(Succeed())
		})
	})

	It("Should be deterministic for identical input trees", func(ctx SpecContext) {
		writeFile(filepath.Join(root, "buf.yaml"), "version: v2\n")
		writeFile(filepath.Join(root, "buf.gen.yaml"), "version: v2\n")
		writeFile(filepath.Join(root, "x", "a.proto"), "syntax = \"proto3\";\n")
		first := MustSucceed(bufInputStamp(ctx, root))
		second := MustSucceed(bufInputStamp(ctx, root))
		Expect(first).To(Equal(second))
		Expect(first).ToNot(BeEmpty())
	})

	It("Should change when a proto file changes", func(ctx SpecContext) {
		writeFile(filepath.Join(root, "buf.yaml"), "version: v2\n")
		writeFile(filepath.Join(root, "x", "a.proto"), "syntax = \"proto3\";\n")
		before := MustSucceed(bufInputStamp(ctx, root))
		writeFile(filepath.Join(root, "x", "a.proto"), "syntax = \"proto3\";\n// changed\n")
		after := MustSucceed(bufInputStamp(ctx, root))
		Expect(after).ToNot(Equal(before))
	})

	It("Should change when buf.gen.yaml changes", func(ctx SpecContext) {
		writeFile(filepath.Join(root, "buf.yaml"), "version: v2\n")
		writeFile(filepath.Join(root, "buf.gen.yaml"), "version: v2\nplugins: []\n")
		writeFile(filepath.Join(root, "x", "a.proto"), "syntax = \"proto3\";\n")
		before := MustSucceed(bufInputStamp(ctx, root))
		writeFile(filepath.Join(root, "buf.gen.yaml"), "version: v2\nplugins: [{remote: x}]\n")
		after := MustSucceed(bufInputStamp(ctx, root))
		Expect(after).ToNot(Equal(before))
	})

	It("Should be independent of proto traversal order", func(ctx SpecContext) {
		// Walks are sorted internally, so even with files added in
		// different orders the stamp must match for identical bytes.
		writeFile(filepath.Join(root, "buf.yaml"), "version: v2\n")
		writeFile(filepath.Join(root, "z.proto"), "syntax = \"proto3\";\n")
		writeFile(filepath.Join(root, "a.proto"), "syntax = \"proto3\";\n")
		first := MustSucceed(bufInputStamp(ctx, root))

		// Re-create the same files to bump mtimes; bytes are identical.
		writeFile(filepath.Join(root, "a.proto"), "syntax = \"proto3\";\n")
		writeFile(filepath.Join(root, "z.proto"), "syntax = \"proto3\";\n")
		second := MustSucceed(bufInputStamp(ctx, root))
		Expect(second).To(Equal(first))
	})
})
