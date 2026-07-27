// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package tree provides box-drawing glyphs for rendering trees as text.
package tree

// Prefix returns the branch glyph for a tree item: "└── " when last is true, "├── "
// otherwise.
func Prefix(last bool) string {
	if last {
		return "└── "
	}
	return "├── "
}

// Indent returns the indent for lines under a tree item: "    " when last is true,
// "│   " otherwise.
func Indent(last bool) string {
	if last {
		return "    "
	}
	return "│   "
}
