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
	"fmt"
	"strings"

	"github.com/samber/lo"
)

func (i *IR) IsZero() bool {
	return len(i.Functions) == 0 &&
		len(i.Nodes) == 0 &&
		len(i.Edges) == 0 &&
		i.Root.IsZero() &&
		i.Symbols == nil &&
		i.TypeMap == nil
}

// String returns the string representation of the IR.
func (i *IR) String() string { return i.stringWithPrefix("") }

// stringWithPrefix returns the string representation with tree formatting.
func (i *IR) stringWithPrefix(prefix string) string {
	var b strings.Builder

	hasFunctions := len(i.Functions) > 0
	hasNodes := len(i.Nodes) > 0
	hasEdges := len(i.Edges) > 0
	hasRoot := !i.Root.IsZero()

	if hasFunctions {
		isLast := !hasNodes && !hasEdges && !hasRoot
		i.writeFunctions(&b, prefix, isLast)
	}

	if hasNodes {
		isLast := !hasEdges && !hasRoot
		i.writeNodes(&b, prefix, isLast)
	}

	if hasEdges {
		isLast := !hasRoot
		i.writeEdges(&b, prefix, isLast)
	}

	if hasRoot {
		i.writeRoot(&b, prefix, true)
	}

	return b.String()
}

func (i *IR) writeFunctions(b *strings.Builder, prefix string, last bool) {
	b.WriteString(prefix)
	b.WriteString(treePrefix(last))
	lo.Must(fmt.Fprintf(b, "Functions (%d)\n", len(i.Functions)))
	childPrefix := prefix + treeIndent(last)
	for j, f := range i.Functions {
		isLast := j == len(i.Functions)-1
		b.WriteString(childPrefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString(f.stringWithPrefix(childPrefix + treeIndent(isLast)))
	}
}

func (i *IR) writeNodes(b *strings.Builder, prefix string, last bool) {
	b.WriteString(prefix)
	b.WriteString(treePrefix(last))
	lo.Must(fmt.Fprintf(b, "Nodes (%d)\n", len(i.Nodes)))
	childPrefix := prefix + treeIndent(last)
	for j, n := range i.Nodes {
		isLast := j == len(i.Nodes)-1
		b.WriteString(childPrefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString(n.stringWithPrefix(childPrefix + treeIndent(isLast)))
	}
}

func (i *IR) writeEdges(b *strings.Builder, prefix string, last bool) {
	b.WriteString(prefix)
	b.WriteString(treePrefix(last))
	lo.Must(fmt.Fprintf(b, "Edges (%d)\n", len(i.Edges)))
	childPrefix := prefix + treeIndent(last)
	for j, e := range i.Edges {
		isLast := j == len(i.Edges)-1
		b.WriteString(childPrefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString(e.String())
		b.WriteString("\n")
	}
}

func (i *IR) writeRoot(b *strings.Builder, prefix string, last bool) {
	b.WriteString(prefix)
	b.WriteString(treePrefix(last))
	b.WriteString("Root\n")
	childPrefix := prefix + treeIndent(last)
	b.WriteString(i.Root.StringWithPrefix(childPrefix))
}
