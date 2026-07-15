// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package program provides the compiled Arc program representation.
//
// A Program combines an intermediate representation (IR) with compiled WebAssembly
// bytecode, representing a complete executable Arc program. Programs are the final
// output of the Arc compilation pipeline and can be serialized for storage or
// executed by a WebAssembly runtime.
//
// # Compilation Pipeline
//
// Programs fit into the Arc compilation pipeline as follows:
//
//	Parser → AST → Analyzer → IR → Compiler → Program (IR + WASM)
//
// # Usage Example
//
// Creating a program from Arc source code:
//
//	import (
//	    "context"
//	    "github.com/synnaxlabs/arc/text"
//	)
//
//	// Parse Arc source code
//	src := text.Text{Raw: "func add(a i64, b i64) i64 { return a + b }"}
//	parsed, err := text.Parse(src)
//	if err != nil {
//	    panic(err)
//	}
//
//	// Analyze to produce IR
//	ir, diag := text.Analyze(context.Background(), parsed, nil)
//	if !diag.Ok() {
//	    panic(diag.Error())
//	}
//
//	// Compile to program with WASM
//	prog, err := text.Compile(context.Background(), ir)
//	if err != nil {
//	    panic(err)
//	}
//
//	// Program now contains both IR and compiled WASM bytecode
//	wasm := prog.WASM              // WebAssembly bytecode
//	functions := prog.Functions    // Function definitions
//	symbols := prog.Symbols        // Symbol table
package program
