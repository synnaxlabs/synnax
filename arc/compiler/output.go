// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler

// Output represents the result of compiling an Arc program to WebAssembly.
type Output struct {
	// WASM contains the compiled WebAssembly bytecode.
	WASM []byte
	// OutputMemoryBases maps function/stage names to their output memory base addresses.
	// Only includes functions/stages with multi-output (named outputs).
	// The runtime uses this to read outputs from linear memory after execution.
	OutputMemoryBases map[string]uint32
}