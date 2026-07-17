// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// IsZero reports whether the Program is empty (uninitialized or contains no content).
//
// A Program is considered zero if it has no compiled WASM bytecode and the embedded
// IR is also zero. This is useful for validating that compilation succeeded and
// produced a valid program.
//
// Example:
//
//	prog, err := text.Compile(ctx, ir)
//	if err != nil {
//	    return err
//	}
//	if prog.IsZero() {
//	    return errors.New("compilation produced empty program")
//	}
func (m Program) IsZero() bool { return len(m.WASM) == 0 && m.IR.IsZero() }

// String returns a human-readable string representation of the program.
// The output includes a summary of the WASM bytecode (size and SHA256 hash)
// and the full IR tree structure with functions, nodes, edges, and the
// Layer-2 execution shell rooted at IR.Root.
// treePrefix returns the branch prefix for a tree line: "└── " when last is true,
// "├── " otherwise.
func treePrefix(last bool) string {
	if last {
		return "└── "
	}
	return "├── "
}

func (m Program) String() string {
	var b strings.Builder
	b.WriteString("Arc Program\n")

	hasContent := len(m.Functions) > 0 || len(m.Nodes) > 0 ||
		len(m.Edges) > 0 || !m.Root.IsZero()

	// WASM summary
	b.WriteString(treePrefix(!hasContent))
	b.WriteString(m.wasmSummary())
	b.WriteString("\n")

	// Delegate to IR for remaining content
	if hasContent {
		b.WriteString(m.IR.String())
	}

	return b.String()
}

// wasmSummary returns a summary of the WASM bytecode.
func (m Program) wasmSummary() string {
	if len(m.WASM) == 0 {
		return "WASM: (none)"
	}
	hash := sha256.Sum256(m.WASM)
	shortHash := hex.EncodeToString(hash[:])[:8]
	return fmt.Sprintf("WASM: %d bytes (sha256: %s...)", len(m.WASM), shortHash)
}
