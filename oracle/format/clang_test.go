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
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/format"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Clang Formatter", func() {
	BeforeEach(func() {
		if _, err := exec.LookPath("clang-format"); err != nil {
			Skip("clang-format not on PATH")
		}
	})

	It("Should normalise spacing in C++ source", func(ctx SpecContext) {
		raw := []byte("int   main(  ) {return  0 ;}\n")
		out := MustSucceed(format.NewClang().Format(ctx, raw, "/abs/main.cpp"))
		Expect(string(out)).To(ContainSubstring("int main()"))
		Expect(string(out)).To(ContainSubstring("return 0;"))
	})

	It("Should be idempotent", func(ctx SpecContext) {
		raw := []byte("int main() {\n    return 0;\n}\n")
		first := MustSucceed(format.NewClang().Format(ctx, raw, "/abs/main.cpp"))
		second := MustSucceed(format.NewClang().Format(ctx, first, "/abs/main.cpp"))
		Expect(string(first)).To(Equal(string(second)))
	})

	It("Should pick a header-style indent for .h files via --assume-filename", func(ctx SpecContext) {
		// Header content: clang-format treats the same content differently
		// depending on the assumed filename's extension (e.g. include
		// guard handling). Verify the assume-filename round-trip works
		// rather than asserting on a specific style, which is config-
		// dependent.
		raw := []byte("#pragma once\nint  foo(  );\n")
		out := MustSucceed(format.NewClang().Format(ctx, raw, "/abs/foo.h"))
		Expect(string(out)).To(ContainSubstring("#pragma once"))
		Expect(string(out)).To(ContainSubstring("int foo();"))
	})
})
