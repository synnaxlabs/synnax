// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package module provides the compiled Arc module representation.
//
// A Module combines an intermediate representation (IR) with compiled WebAssembly
// bytecode, representing a complete executable Arc program. Modules are the final
// output of the Arc compilation pipeline and can be serialized for storage or
// executed by a WebAssembly runtime.
//
// # Compilation Pipeline
//
// Modules fit into the Arc compilation pipeline as follows:
//
//	Parser → AST → Analyzer → IR → Compiler → Module (IR + WASM)
//
// # Usage Example
//
// Creating a module from Arc source code:
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
//	// Compile to module with WASM
//	module, err := text.Compile(context.Background(), ir)
//	if err != nil {
//	    panic(err)
//	}
//
//	// Module now contains both IR and compiled WASM bytecode
//	wasm := module.WASM              // WebAssembly bytecode
//	functions := module.Functions    // Function definitions
//	symbols := module.Symbols        // Symbol table
package module

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/ir"
)

// Module represents a fully compiled Arc program combining intermediate representation
// with executable WebAssembly bytecode.
//
// A Module embeds both the IR (containing the dataflow graph, function definitions,
// and symbol table) and the compiler Output (containing WASM bytecode and memory
// layout information). This makes a Module self-contained and ready for execution
// by a WebAssembly runtime.
//
// The embedded compiler.Output provides:
//   - WASM: Compiled WebAssembly bytecode ready for execution
//   - OutputMemoryBases: Memory addresses for multi-output functions
//
// Modules can be serialized to disk for caching or distributed execution, though
// the Symbols and TypeMap fields are not serialized (they are only needed during
// compilation and tooling).
type Module struct {
	ir.IR
	compiler.Output
}

// IsZero reports whether the Module is empty (uninitialized or contains no content).
//
// A Module is considered zero if it has no compiled WASM bytecode and the embedded
// IR is also zero. This is useful for validating that compilation succeeded and
// produced a valid module.
//
// Example:
//
//	module, err := text.Compile(ctx, ir)
//	if err != nil {
//	    return err
//	}
//	if module.IsZero() {
//	    return errors.New("compilation produced empty module")
//	}
func (m Module) IsZero() bool { return len(m.WASM) == 0 && m.IR.IsZero() }

// String returns a human-readable string representation of the module.
// The output includes a summary of the WASM bytecode (size and SHA256 hash)
// and the full IR tree structure with functions, nodes, edges, strata, and sequences.
func (m Module) String() string {
	var b strings.Builder
	b.WriteString("Arc Module\n")

	hasContent := len(m.Functions) > 0 || len(m.Nodes) > 0 ||
		len(m.Edges) > 0 || len(m.Strata) > 0 || len(m.Sequences) > 0

	// WASM summary
	b.WriteString(ir.TreePrefix(!hasContent))
	b.WriteString(m.wasmSummary())
	b.WriteString("\n")

	// Delegate to IR for remaining content
	if hasContent {
		b.WriteString(m.IR.String())
	}

	return b.String()
}

// wasmSummary returns a summary of the WASM bytecode.
func (m Module) wasmSummary() string {
	if len(m.WASM) == 0 {
		return "WASM: (none)"
	}
	hash := sha256.Sum256(m.WASM)
	shortHash := hex.EncodeToString(hash[:])[:8]
	return fmt.Sprintf("WASM: %d bytes (sha256: %s...)", len(m.WASM), shortHash)
}
