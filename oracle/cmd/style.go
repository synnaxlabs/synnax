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
	"strings"
	"time"

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
	fmt.Printf(
		"  %s %s %s %s\n",
		dimStyle.Render(symbolFile),
		p,
		dimStyle.Render(symbolArrow),
		f,
	)
}

func printSchemaCount(count int) {
	c := countStyle.Render(fmt.Sprintf("%d", count))
	word := "schema"
	if count != 1 {
		word = "schemas"
	}
	fmt.Printf("%s %s %s found\n", infoStyle.Render(symbolCheck), c, word)
}

func printSyncedCount(written, unchanged int) {
	if written == 0 {
		fmt.Printf(
			"%s %s\n",
			dimStyle.Render(symbolDot),
			dimStyle.Render("already up to date"),
		)
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
		fmt.Printf(
			"    %s %s\n",
			dimStyle.Render(symbolArrow),
			dimStyle.Render("all schemas formatted"),
		)
		return
	}
	c := countStyle.Render(fmt.Sprintf("%d", formatted))
	word := "file"
	if formatted != 1 {
		word = "files"
	}
	fmt.Printf(
		"    %s %s %s %s\n",
		infoStyle.Render(symbolArrow),
		dimStyle.Render("formatted"),
		c,
		word,
	)
}

// printFormatPlan announces the format pass: how many generated files
// need to run through the formatter chain and how many are short-
// circuited by the on-disk cache. Emitted once, immediately before the
// format batch runs, so the user sees activity even though the batch
// itself takes a few seconds and is silent internally.
func printFormatPlan(toFormat, cached int) {
	printPlan("formatting", toFormat, "file", cached, "cached")
}

// printFormatDone reports duration of the format batch.
func printFormatDone(d time.Duration) {
	printArrowDone(d)
}

// printWritePlan announces the parallel write pass after formatting.
// Files whose canonical bytes already match the on-disk file are
// reported as "unchanged" and not rewritten.
func printWritePlan(toWrite, unchanged int) {
	if toWrite == 0 && unchanged == 0 {
		return
	}
	printPlan("writing", toWrite, "file", unchanged, "unchanged")
}

// printBufGenerateStart announces the start of the buf-generate step.
// When changedProtos is 0 the cache decides whether to run; the
// banner is suppressed in that case to avoid noise.
func printBufGenerateStart(changedProtos int) {
	if changedProtos == 0 {
		fmt.Printf("  %s\n", dimStyle.Render("buf generate"))
		return
	}
	fmt.Printf(
		"  %s %s %s\n",
		dimStyle.Render("buf generate"),
		dimStyle.Render("over"),
		countWord(changedProtos, "proto"),
	)
}

// printBufGenerateDone reports the outcome: cache hit or actual run
// with elapsed time. cached=true means the input-content stamp was
// unchanged and no protoc plugins were invoked.
func printBufGenerateDone(cached bool, d time.Duration) {
	if cached {
		fmt.Printf("    %s %s\n", infoStyle.Render(symbolArrow), dimStyle.Render("cached"))
		return
	}
	printArrowDone(d)
}

// printPlan renders a "<verb> <n> <noun>(s) (<aux> <m>)" line. The
// auxiliary clause is omitted when m == 0; when n == 0 the auxiliary
// drives the message instead. Used for "formatting / writing"
// announcements that share a shape but differ in nouns.
func printPlan(verb string, n int, noun string, aux int, auxLabel string) {
	if n == 0 {
		fmt.Printf(
			"  %s %s\n",
			dimStyle.Render(verb),
			dimStyle.Render(fmt.Sprintf("%d %s, nothing to do", aux, auxLabel)),
		)
		return
	}
	suffix := ""
	if aux > 0 {
		suffix = dimStyle.Render(fmt.Sprintf(" (%d %s)", aux, auxLabel))
	}
	fmt.Printf("  %s %s%s\n", dimStyle.Render(verb), countWord(n, noun), suffix)
}

// printArrowDone renders the post-step "→ done in <duration>" line
// shared by every timed phase.
func printArrowDone(d time.Duration) {
	fmt.Printf(
		"    %s %s %s\n",
		infoStyle.Render(symbolArrow),
		dimStyle.Render("done in"),
		dimStyle.Render(fmtDuration(d)),
	)
}

// countWord renders "<n> <singular>" or "<n> <singular>s" with n in
// the count style. The singular form is the bare noun ("file",
// "proto", "schema"); the plural is formed by appending "s".
func countWord(n int, singular string) string {
	noun := singular
	if n != 1 {
		noun = singular + "s"
	}
	return countStyle.Render(fmt.Sprintf("%d", n)) + " " + noun
}

// fmtDuration renders a duration in a tight, human-friendly form: ms
// under a second, decimal seconds otherwise.
func fmtDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
