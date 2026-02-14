// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package doc provides composable blocks for generating consistent markdown documentation.
// It is used for hover docs, diagnostics, and other LSP features.
package doc

import "strings"

// Block represents a renderable documentation block.
type Block interface {
	Render() string
}

// Doc is a composition of documentation blocks.
type Doc struct {
	blocks []Block
}

// New creates a new Doc with the given blocks.
func New(blocks ...Block) Doc {
	return Doc{blocks: blocks}
}

// Add appends additional blocks to the document and returns the modified Doc.
func (d *Doc) Add(blocks ...Block) *Doc {
	d.blocks = append(d.blocks, blocks...)
	return d
}

// Render joins all blocks with double newlines and returns the result.
func (d Doc) Render() string {
	if len(d.blocks) == 0 {
		return ""
	}
	parts := make([]string, 0, len(d.blocks))
	for _, b := range d.blocks {
		if rendered := b.Render(); rendered != "" {
			parts = append(parts, rendered)
		}
	}
	return strings.Join(parts, "\n\n")
}
