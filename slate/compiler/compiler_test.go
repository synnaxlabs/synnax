// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/parser"
	. "github.com/synnaxlabs/x/testutil"
)

func compile(source string) ([]byte, error) {
	prog := MustSucceed(parser.Parse(source))
	analysis := analyzer.Analyze(analyzer.Config{Program: prog})
	return compiler.Compile(prog, &analysis)
}

var _ = Describe("Compiler", func() {
	Describe("Empty Program", func() {
		It("should compile an empty program with all imports", func() {
			wasm := MustSucceed(compile(`
			func dog(b i64) i64 {
				a i64 := 2
				if b == a {
					return 2
				} else if a > b {
					c i64 := 5
					return c
				} else {
					return 3
				}
			}
			`))
			os.WriteFile("main.wasm", wasm, 0644)
		})
	})
})
