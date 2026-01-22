// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build ignore

package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	content, err := os.ReadFile("ArcLexer.g4")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read grammar: %v\n", err)
		os.Exit(1)
	}

	// Match: TOKEN_NAME : 'literal';
	re := regexp.MustCompile(`(\w+)\s*:\s*'([^']+)'\s*;`)
	matches := re.FindAllStringSubmatch(string(content), -1)

	var b strings.Builder
	b.WriteString("// Copyright 2026 Synnax Labs, Inc.\n")
	b.WriteString("//\n")
	b.WriteString("// Use of this software is governed by the Business Source License included in the file\n")
	b.WriteString("// licenses/BSL.txt.\n")
	b.WriteString("//\n")
	b.WriteString("// As of the Change Date specified in that file, in accordance with the Business Source\n")
	b.WriteString("// License, use of this software will be governed by the Apache License, Version 2.0,\n")
	b.WriteString("// included in the file licenses/APL.txt.\n\n")
	b.WriteString("// Code generated from ArcLexer.g4 by gen/tokens/main.go. DO NOT EDIT.\n\n")
	b.WriteString("package parser\n\n")
	b.WriteString("const (\n")

	for _, m := range matches {
		name, literal := m[1], m[2]
		b.WriteString(fmt.Sprintf("\tLiteral%s = %q\n", name, literal))
	}

	b.WriteString(")\n")

	if err := os.WriteFile("token_literals.go", []byte(b.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated token_literals.go")
}
