// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import (
	"fmt"
	"strings"

	"github.com/synnaxlabs/x/tree"
)

// IsEntryNode reports whether n is an entry node: it has no incoming edges and
// reads no channels. Entry nodes fire once per activation.
func (n Node) IsEntryNode(edges Edges) bool {
	return len(edges.GetInputs(n.Key)) == 0 && len(n.Channels.Read) == 0
}

// String returns the string representation of the node.
func (n Node) String() string { return n.stringWithPrefix("") }

// stringWithPrefix returns the string representation with tree formatting.
func (n Node) stringWithPrefix(prefix string) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "%s (type: %s)\n", n.Key, n.Type)

	hasInputs := len(n.Inputs) > 0
	hasOutputs := len(n.Outputs) > 0

	isLast := !hasInputs && !hasOutputs
	b.WriteString(prefix)
	b.WriteString(tree.Prefix(isLast))
	b.WriteString("channels: ")
	b.WriteString(n.Channels.String())
	b.WriteString("\n")

	if hasInputs {
		isLast = !hasOutputs
		b.WriteString(prefix)
		b.WriteString(tree.Prefix(isLast))
		b.WriteString("inputs: ")
		b.WriteString(n.Inputs.String())
		b.WriteString("\n")
	}

	if hasOutputs {
		b.WriteString(prefix)
		b.WriteString(tree.Prefix(true))
		b.WriteString("outputs: ")
		b.WriteString(n.Outputs.String())
		b.WriteString("\n")
	}

	return b.String()
}
