// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package text provides parsing, analysis, and compilation of Arc source code.
//
// The package implements a three-stage pipeline:
//   - Parse: Converts raw text into an Abstract Syntax Tree (AST)
//   - Analyze: Performs semantic analysis and builds Intermediate Representation (IR)
//   - Compile: Generates WebAssembly bytecode from IR
//
// The analyzer uses a multi-pass approach:
//  1. Analyze function declarations and build the symbol table
//  2. Process flow statements to construct nodes and edges
//  3. Calculate execution stratification for deterministic reactive execution
package text

import (
	"github.com/synnaxlabs/arc/parser"
)

// Text represents Arc source code with its parsed AST.
type Text struct {
	AST parser.IProgramContext `json:"-"`
	Raw string                 `json:"raw" msgpack:"raw"`
}
