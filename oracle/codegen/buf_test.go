// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codegen_test

import (
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/codegen"
	"github.com/synnaxlabs/oracle/format"
	. "github.com/synnaxlabs/x/testutil"
)

// fixture writes a minimal repoRoot containing buf.yaml,
// buf.gen.yaml, and one .proto file. The gen template lists no
// plugins so `buf generate` runs successfully without producing any
// output, which is all we need to exercise the cache.
func fixture() string {
	GinkgoHelper()
	root := MustSucceed(os.MkdirTemp("", "buf-fixture"))
	DeferCleanup(func() {
		Expect(os.RemoveAll(root)).To(Succeed())
	})
	Expect(os.MkdirAll(filepath.Join(root, "x"), 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(root, "buf.yaml"), []byte("version: v2\n"), 0644)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(root, "buf.gen.yaml"), []byte("version: v2\n"), 0644)).To(Succeed())
	Expect(os.WriteFile(
		filepath.Join(root, "x", "a.proto"),
		[]byte(`syntax = "proto3";`+"\n"+`package x;`+"\n"),
		0644,
	)).To(Succeed())
	return root
}

var _ = Describe("RunBufGenerate", func() {
	BeforeEach(func() {
		if _, err := exec.LookPath("buf"); err != nil {
			Skip("buf not on PATH")
		}
	})

	It("Should be a cache hit on the second run when nothing changed", func(ctx SpecContext) {
		root := fixture()
		cache := format.LoadCache(root)

		first := MustSucceed(codegen.RunBufGenerate(ctx, root, nil, cache))
		Expect(first.Cached).To(BeFalse(), "first run cannot be cached: no prior stamp")

		second := MustSucceed(codegen.RunBufGenerate(ctx, root, nil, cache))
		Expect(second.Cached).To(BeTrue(), "second run with identical inputs must hit the stamp cache")
	})

	It("Should re-run when a .proto file changes", func(ctx SpecContext) {
		root := fixture()
		cache := format.LoadCache(root)
		MustSucceed(codegen.RunBufGenerate(ctx, root, nil, cache))

		Expect(os.WriteFile(
			filepath.Join(root, "x", "a.proto"),
			[]byte(`syntax = "proto3";`+"\n"+`package x;`+"\n"+`message M {}`+"\n"),
			0644,
		)).To(Succeed())

		result := MustSucceed(codegen.RunBufGenerate(ctx, root, nil, cache))
		Expect(result.Cached).To(BeFalse())
	})

	It("Should re-run when changedProtos is non-empty even if the stamp matches", func(ctx SpecContext) {
		root := fixture()
		cache := format.LoadCache(root)
		MustSucceed(codegen.RunBufGenerate(ctx, root, nil, cache))

		// Stamp is fresh; nil changedProtos would be a cache hit.
		// Passing a non-empty changedProtos forces a re-run so the
		// caller can scope generation to a slice of changed files.
		result := MustSucceed(codegen.RunBufGenerate(ctx, root, []string{"x/a.proto"}, cache))
		Expect(result.Cached).To(BeFalse())
	})
})
