// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package output provides styled terminal output for oracle using lipgloss.
package output

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	purple  = lipgloss.Color("#9D4EDD")
	pink    = lipgloss.Color("#FF6B9D")
	cyan    = lipgloss.Color("#00D9FF")
	green   = lipgloss.Color("#39FF14")
	dimGray = lipgloss.Color("#6B7280")

	// Styles
	pluginStyle  = lipgloss.NewStyle().Foreground(pink)
	actionStyle  = lipgloss.NewStyle().Foreground(cyan)
	countStyle   = lipgloss.NewStyle().Bold(true).Foreground(purple)
	successStyle = lipgloss.NewStyle().Foreground(green)
	dimStyle     = lipgloss.NewStyle().Foreground(dimGray)
)

const (
	symbolArrow   = "→"
	symbolDot     = "·"
	symbolSpinner = "◌"
	symbolCheck   = "✓"
)

// PluginStart prints a message when a plugin starts generating.
func PluginStart(name string) {
	p := pluginStyle.Render(name)
	a := dimStyle.Render("generating...")
	fmt.Printf("  %s %s\n", p, a)
}

// PluginDone prints a message when a plugin finishes with file count.
func PluginDone(name string, fileCount int) {
	p := pluginStyle.Render(name)
	c := countStyle.Render(fmt.Sprintf("%d", fileCount))
	word := "file"
	if fileCount != 1 {
		word = "files"
	}
	fmt.Printf("  %s %s %s %s\n", p, symbolArrow, c, word)
}

// PostWriteStep prints a post-write step in progress.
func PostWriteStep(tool string, fileCount int, action string) {
	t := actionStyle.Render(tool)
	c := countStyle.Render(fmt.Sprintf("%d", fileCount))
	word := "file"
	if fileCount != 1 {
		word = "files"
	}
	a := dimStyle.Render(action)
	fmt.Printf("    %s %s %s %s %s\n", t, symbolArrow, a, c, word)
}
