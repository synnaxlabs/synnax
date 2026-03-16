// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"fmt"
	"log"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	// Colors
	purple  = lipgloss.Color("#9D4EDD")
	pink    = lipgloss.Color("#FF6B9D")
	cyan    = lipgloss.Color("#00D9FF")
	green   = lipgloss.Color("#39FF14")
	yellow  = lipgloss.Color("#FFE66D")
	orange  = lipgloss.Color("#FF9F1C")
	red     = lipgloss.Color("#FF4757")
	dimGray = lipgloss.Color("#6B7280")

	// Styles
	successStyle = lipgloss.NewStyle().Bold(true).Foreground(green)
	errorStyle   = lipgloss.NewStyle().Bold(true).Foreground(red)
	infoStyle    = lipgloss.NewStyle().Foreground(cyan)
	dimStyle     = lipgloss.NewStyle().Foreground(dimGray)
	pluginStyle  = lipgloss.NewStyle().Foreground(pink)
	fileStyle    = lipgloss.NewStyle().Foreground(orange)
	countStyle   = lipgloss.NewStyle().Bold(true).Foreground(purple)
)

// Symbols for output
const (
	symbolSuccess = "✓"
	symbolError   = "✗"
	symbolArrow   = "→"
	symbolDot     = "·"
	symbolSpark   = "⚡"
	symbolFile    = "◈"
	symbolCheck   = "◆"
)

func printBanner() {
	banner := lipgloss.NewStyle().Bold(true).Foreground(purple).Render("oracle")
	spark := lipgloss.NewStyle().Foreground(yellow).Render(symbolSpark)
	if _, err := lipgloss.Printf("%s %s\n\n", spark, banner); err != nil {
		log.Println(err)
	}
}

func printSuccess(msg string) {
	sym := successStyle.Render(symbolSuccess)
	if _, err := lipgloss.Printf("%s %s\n", sym, successStyle.Render(msg)); err != nil {
		log.Println(err)
	}
}

func printError(msg string) {
	sym := errorStyle.Render(symbolError)
	if _, err := lipgloss.Printf("%s %s\n", sym, errorStyle.Render(msg)); err != nil {
		log.Println(err)
	}
}

func printInfo(msg string) {
	sym := infoStyle.Render(symbolArrow)
	if _, err := lipgloss.Printf("%s %s\n", sym, msg); err != nil {
		log.Println(err)
	}
}

func printDim(msg string) {
	if _, err := lipgloss.Println(dimStyle.Render(msg)); err != nil {
		log.Println(err)
	}
}

func printFileWritten(plugin, path string) {
	p := pluginStyle.Render(plugin)
	f := fileStyle.Render(path)
	if _, err := lipgloss.Printf(
		"  %s %s %s %s\n",
		dimStyle.Render(symbolFile),
		p,
		dimStyle.Render(symbolArrow),
		f,
	); err != nil {
		log.Println(err)
	}
}

func printSchemaCount(count int) {
	c := countStyle.Render(fmt.Sprintf("%d", count))
	word := "schema"
	if count != 1 {
		word = "schemas"
	}
	if _, err := lipgloss.Printf(
		"%s %s %s found\n", infoStyle.Render(symbolCheck), c, word,
	); err != nil {
		log.Println(err)
	}
}

func printSyncedCount(written, unchanged int) {
	if written == 0 {
		if _, err := lipgloss.Printf(
			"%s %s\n",
			dimStyle.Render(symbolDot),
			dimStyle.Render("already up to date"),
		); err != nil {
			log.Println(err)
		}
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
		parts = append(
			parts,
			fmt.Sprintf("%s structs", countStyle.Render(fmt.Sprintf("%d", structs))),
		)
	}
	if enums > 0 {
		parts = append(
			parts,
			fmt.Sprintf("%s enums", countStyle.Render(fmt.Sprintf("%d", enums))),
		)
	}
	msg := "valid " + dimStyle.Render("(") +
		strings.Join(parts, dimStyle.Render(", ")) + dimStyle.Render(")")
	printSuccess(msg)
}

func printDiagnostics(diagnosticStr string) {
	if diagnosticStr == "" {
		return
	}
	for line := range strings.SplitSeq(diagnosticStr, "\n") {
		if line == "" {
			continue
		}
		if _, err := lipgloss.Printf(
			"  %s %s\n", errorStyle.Render(symbolDot), line,
		); err != nil {
			log.Println(err)
		}
	}
}

func printFormattingStart(count int) {
	c := countStyle.Render(fmt.Sprintf("%d", count))
	word := "schema"
	if count != 1 {
		word = "schemas"
	}
	if _, err := lipgloss.Printf(
		"  %s %s %s\n", dimStyle.Render("formatting"), c, word,
	); err != nil {
		log.Println(err)
	}
}

func printFormattingDone(formatted int) {
	if formatted == 0 {
		if _, err := lipgloss.Printf(
			"    %s %s\n",
			dimStyle.Render(symbolArrow),
			dimStyle.Render("all schemas formatted"),
		); err != nil {
			log.Println(err)
		}
	} else {
		c := countStyle.Render(fmt.Sprintf("%d", formatted))
		word := "file"
		if formatted != 1 {
			word = "files"
		}
		if _, err := lipgloss.Printf(
			"    %s %s %s %s\n",
			infoStyle.Render(symbolArrow),
			dimStyle.Render("formatted"),
			c,
			word,
		); err != nil {
			log.Println(err)
		}
	}
}
