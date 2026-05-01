// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package check

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// Format selects an output renderer.
type Format string

const (
	// FormatText renders a styled, human-readable report. Default.
	FormatText Format = "text"
	// FormatJSON renders a stable, machine-readable report. The exact
	// JSON shape is defined by the Report and GateReport struct tags.
	FormatJSON Format = "json"
)

// Render writes the report to w in the requested format.
func Render(w io.Writer, r *Report, f Format, verbose bool) error {
	switch f {
	case FormatJSON:
		return renderJSON(w, r)
	default:
		return renderText(w, r, verbose)
	}
}

func renderJSON(w io.Writer, r *Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

var (
	checkPass    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#39FF14"))
	checkFail    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF4757"))
	checkSkip    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	checkPath    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9F1C"))
	checkHint    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00D9FF"))
	checkDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	checkSummary = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9D4EDD"))
)

const (
	symGatePass = "✓"
	symGateFail = "✗"
	symGateSkip = "·"
	symFinding  = "›"
)

func renderText(w io.Writer, r *Report, verbose bool) error {
	for _, g := range r.Gates {
		var sym, label string
		switch g.Status {
		case StatusPass:
			sym = checkPass.Render(symGatePass)
			label = checkPass.Render(g.Gate)
		case StatusFail:
			sym = checkFail.Render(symGateFail)
			label = checkFail.Render(g.Gate)
		case StatusSkipped:
			sym = checkSkip.Render(symGateSkip)
			label = checkSkip.Render(g.Gate + " (skipped)")
		}
		elapsed := checkDim.Render(fmt.Sprintf(" %s", fmtDuration(g.Elapsed)))
		if _, err := fmt.Fprintf(w, "%s %s%s\n", sym, label, elapsed); err != nil {
			return err
		}
		// In a passing gate we suppress info/warning findings unless
		// verbose; failures always print every finding so the user sees
		// the full picture.
		for _, f := range g.Findings {
			if g.Status == StatusPass && f.Severity != SeverityError && !verbose {
				continue
			}
			if err := renderFinding(w, f); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	return renderSummary(w, r)
}

func renderFinding(w io.Writer, f Finding) error {
	loc := f.Path
	if f.Line > 0 {
		loc = fmt.Sprintf("%s:%d", f.Path, f.Line)
		if f.Col > 0 {
			loc = fmt.Sprintf("%s:%d", loc, f.Col)
		}
	}
	prefix := checkDim.Render(symFinding)
	severity := f.Severity.String()
	switch f.Severity {
	case SeverityError:
		severity = checkFail.Render(severity)
	case SeverityWarning:
		severity = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFE66D")).Render(severity)
	default:
		severity = checkDim.Render(severity)
	}
	if loc != "" {
		loc = checkPath.Render(loc) + " "
	}
	if _, err := fmt.Fprintf(w, "  %s %s%s: %s\n", prefix, loc, severity, f.Message); err != nil {
		return err
	}
	if f.FixHint != "" {
		if _, err := fmt.Fprintf(w, "      %s %s\n", checkDim.Render("fix:"), checkHint.Render(f.FixHint)); err != nil {
			return err
		}
	}
	if f.Diff != "" {
		for line := range strings.SplitSeq(strings.TrimRight(f.Diff, "\n"), "\n") {
			if _, err := fmt.Fprintf(w, "      %s\n", checkDim.Render(line)); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderSummary(w io.Writer, r *Report) error {
	if r.TotalFailed == 0 {
		_, err := fmt.Fprintf(w, "%s %s\n",
			checkPass.Render(symGatePass),
			checkSummary.Render(fmt.Sprintf("all gates passed (%d)", r.TotalPassed)),
		)
		return err
	}
	_, err := fmt.Fprintf(w, "%s %s\n",
		checkFail.Render(symGateFail),
		checkSummary.Render(fmt.Sprintf("%d gate(s) failed", r.TotalFailed)),
	)
	return err
}

func fmtDuration(d time.Duration) string {
	ms := d.Milliseconds()
	if ms < 1000 {
		return fmt.Sprintf("(%dms)", ms)
	}
	return fmt.Sprintf("(%.1fs)", d.Seconds())
}
