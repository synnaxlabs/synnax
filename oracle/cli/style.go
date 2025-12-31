// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	purple    = lipgloss.Color("#9D4EDD")
	pink      = lipgloss.Color("#FF6B9D")
	cyan      = lipgloss.Color("#00D9FF")
	green     = lipgloss.Color("#39FF14")
	yellow    = lipgloss.Color("#FFE66D")
	orange    = lipgloss.Color("#FF9F1C")
	red       = lipgloss.Color("#FF4757")
	dimGray   = lipgloss.Color("#6B7280")

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(purple)

	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(green)

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(red)

	warnStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(yellow)

	infoStyle = lipgloss.NewStyle().
			Foreground(cyan)

	dimStyle = lipgloss.NewStyle().
			Foreground(dimGray)

	pluginStyle = lipgloss.NewStyle().
			Foreground(pink)

	fileStyle = lipgloss.NewStyle().
			Foreground(orange)

	countStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(purple)
)

// Symbols for output
const (
	symbolOracle  = "✦"
	symbolSuccess = "✓"
	symbolError   = "✗"
	symbolArrow   = "→"
	symbolDot     = "·"
	symbolSpark   = "⚡"
	symbolFile    = "◈"
	symbolCheck   = "◆"
)

func printBanner() {
	banner := lipgloss.NewStyle().
		Bold(true).
		Foreground(purple).
		Render("oracle")
	spark := lipgloss.NewStyle().Foreground(yellow).Render(symbolSpark)
	fmt.Printf("%s %s\n\n", spark, banner)
}

func printSuccess(msg string) {
	sym := successStyle.Render(symbolSuccess)
	fmt.Printf("%s %s\n", sym, successStyle.Render(msg))
}

func printError(msg string) {
	sym := errorStyle.Render(symbolError)
	fmt.Printf("%s %s\n", sym, errorStyle.Render(msg))
}

func printInfo(msg string) {
	sym := infoStyle.Render(symbolArrow)
	fmt.Printf("%s %s\n", sym, msg)
}

func printDim(msg string) {
	fmt.Println(dimStyle.Render(msg))
}

func printFileWritten(plugin, path string) {
	p := pluginStyle.Render(plugin)
	f := fileStyle.Render(path)
	fmt.Printf("  %s %s %s %s\n", dimStyle.Render(symbolFile), p, dimStyle.Render(symbolArrow), f)
}

func printSchemaCount(count int) {
	c := countStyle.Render(fmt.Sprintf("%d", count))
	word := "schema"
	if count != 1 {
		word = "schemas"
	}
	fmt.Printf("%s %s %s found\n", infoStyle.Render(symbolCheck), c, word)
}

func printGeneratedCount(count int) {
	c := countStyle.Render(fmt.Sprintf("%d", count))
	word := "file"
	if count != 1 {
		word = "files"
	}
	msg := fmt.Sprintf("%s %s generated", c, word)
	printSuccess(msg)
}

func printSyncedCount(written, unchanged int) {
	if written == 0 {
		fmt.Printf("%s %s\n", dimStyle.Render(symbolDot), dimStyle.Render("already up to date"))
		return
	}
	w := countStyle.Render(fmt.Sprintf("%d", written))
	word := "file"
	if written != 1 {
		word = "files"
	}
	msg := fmt.Sprintf("%s %s synced", w, word)
	if unchanged > 0 {
		msg += dimStyle.Render(fmt.Sprintf(" (%d unchanged)", unchanged))
	}
	printSuccess(msg)
}

func printValidationPassed(structs, enums int) {
	parts := []string{}
	if structs > 0 {
		parts = append(parts, fmt.Sprintf("%s structs", countStyle.Render(fmt.Sprintf("%d", structs))))
	}
	if enums > 0 {
		parts = append(parts, fmt.Sprintf("%s enums", countStyle.Render(fmt.Sprintf("%d", enums))))
	}
	msg := "valid " + dimStyle.Render("(") + strings.Join(parts, dimStyle.Render(", ")) + dimStyle.Render(")")
	printSuccess(msg)
}

func printDiagnostics(diagnosticStr string) {
	if diagnosticStr == "" {
		return
	}
	lines := strings.Split(diagnosticStr, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		fmt.Printf("  %s %s\n", errorStyle.Render(symbolDot), line)
	}
}

func printFormattingStart(count int) {
	c := countStyle.Render(fmt.Sprintf("%d", count))
	word := "schema"
	if count != 1 {
		word = "schemas"
	}
	fmt.Printf("  %s %s %s\n", dimStyle.Render("formatting"), c, word)
}

func printFormattingDone(formatted int) {
	if formatted == 0 {
		fmt.Printf("    %s %s\n", dimStyle.Render(symbolArrow), dimStyle.Render("all schemas formatted"))
	} else {
		c := countStyle.Render(fmt.Sprintf("%d", formatted))
		word := "file"
		if formatted != 1 {
			word = "files"
		}
		fmt.Printf("    %s %s %s %s\n", infoStyle.Render(symbolArrow), dimStyle.Render("formatted"), c, word)
	}
}
