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

	"github.com/samber/lo"
)

// Find searches for a node by key. Returns the node and true if found,
// or zero value and false otherwise.
func (n Nodes) Find(key string) (Node, bool) {
	return lo.Find(n, func(n Node) bool { return n.Key == key })
}

// Get returns the node with the given key. Panics if not found.
func (n Nodes) Get(key string) Node { return lo.Must(n.Find(key)) }

// String returns the string representation of the node.
func (n Node) String() string {
	return n.stringWithPrefix("")
}

// stringWithPrefix returns the string representation with tree formatting.
func (n Node) stringWithPrefix(prefix string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s (type: %s)\n", n.Key, n.Type))

	hasConfig := len(n.Config) > 0
	hasInputs := len(n.Inputs) > 0
	hasOutputs := len(n.Outputs) > 0

	isLast := !hasConfig && !hasInputs && !hasOutputs
	b.WriteString(prefix)
	b.WriteString(treePrefix(isLast))
	b.WriteString("channels: ")
	b.WriteString(formatChannels(n.Channels))
	b.WriteString("\n")

	if hasConfig {
		isLast = !hasInputs && !hasOutputs
		b.WriteString(prefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString("config: ")
		b.WriteString(formatParams(n.Config))
		b.WriteString("\n")
	}

	if hasInputs {
		isLast = !hasOutputs
		b.WriteString(prefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString("inputs: ")
		b.WriteString(formatParams(n.Inputs))
		b.WriteString("\n")
	}

	if hasOutputs {
		b.WriteString(prefix)
		b.WriteString(treePrefix(true))
		b.WriteString("outputs: ")
		b.WriteString(formatParams(n.Outputs))
		b.WriteString("\n")
	}

	return b.String()
}
