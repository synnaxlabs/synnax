// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package check is the read-only validation layer for oracle. It runs a set
// of named gates - format drift, analyzer diagnostics, generated-output
// drift, orphaned generated files, cache coherence - against a pipeline
// Result and produces a structured Report. It does not modify any file.
//
// Each gate is an independently testable Checker implementation. The
// Checker contract is intentionally narrow: take a pipeline Result, look
// at the on-disk projection of that result if needed, return a GateReport
// describing what passed, what failed, and what the user should do about
// it.
//
// Consumers (today only the `oracle check` command, tomorrow also CI hooks
// and editor integrations) drive the Checkers via Run, get back a Report,
// and choose how to render it. Pretty and JSON renderers live in this
// package so every consumer prints the same way.
package check

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/x/set"
)

// Checker is one validation gate. Name is a stable identifier used for
// flag selection (`--gates=format,analyze`), exit codes, and JSON output.
type Checker interface {
	Name() string
	Run(ctx context.Context, p *pipeline.Result, env Env) GateReport
}

// Env carries everything a Checker needs that is not part of the pipeline
// Result. Concrete fields are populated by the CLI driver and passed
// uniformly to every gate.
type Env struct {
	// RepoRoot is the absolute path to the repository root.
	RepoRoot string
	// IncludeDiffs requests that gates capture per-finding unified diffs.
	// Off by default because diffs blow up output size.
	IncludeDiffs bool
}

// Severity describes how a finding affects the gate's pass/fail decision.
type Severity int

const (
	// SeverityInfo is informational and does not affect status.
	SeverityInfo Severity = iota
	// SeverityWarning is a non-blocking concern. Promoted to Fail by the
	// driver only when warnings-as-errors is set.
	SeverityWarning
	// SeverityError causes the owning gate to Fail.
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

// MarshalJSON serializes Severity as its lower-case string form so the
// JSON report is consumable without an out-of-band enum table.
func (s Severity) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// Status is the outcome of a single gate.
type Status int

const (
	// StatusPass indicates the gate ran and produced no error-severity
	// findings.
	StatusPass Status = iota
	// StatusFail indicates the gate produced one or more error findings.
	StatusFail
	// StatusSkipped indicates the gate did not run (e.g. excluded via
	// --gates).
	StatusSkipped
)

func (s Status) String() string {
	switch s {
	case StatusPass:
		return "pass"
	case StatusFail:
		return "fail"
	case StatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// MarshalJSON serializes Status as its lower-case string form.
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// Finding is one observation produced by a Checker. The driver renders
// findings; gates do not print directly.
type Finding struct {
	Path     string   `json:"path,omitempty"`
	Line     int      `json:"line,omitempty"`
	Col      int      `json:"col,omitempty"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	// FixHint is a one-line suggestion shown to the user (e.g.
	// "run `oracle sync`").
	FixHint string `json:"fix_hint,omitempty"`
	// Diff is an optional unified-diff body for drift findings. Only
	// populated when Env.IncludeDiffs is true.
	Diff string `json:"diff,omitempty"`
}

// GateReport is the per-gate result, carried back to the driver.
type GateReport struct {
	Gate     string        `json:"gate"`
	Status   Status        `json:"status"`
	Findings []Finding     `json:"findings,omitempty"`
	Elapsed  time.Duration `json:"elapsed_ns"`
}

// Report aggregates every GateReport produced in one check run.
type Report struct {
	Gates       []GateReport  `json:"gates"`
	TotalRun    int           `json:"total_run"`
	TotalPassed int           `json:"total_passed"`
	TotalFailed int           `json:"total_failed"`
	Elapsed     time.Duration `json:"elapsed_ns"`
}

// FailureCodes is the per-gate exit-code contract. CI consumers can branch
// on these to attribute failures. Codes are stable; never re-number.
//
// The values intentionally start at 10 to leave 1 for "internal error" and
// 2 for cobra's usage-error default. 13 is reserved for a future orphan
// gate; do not reuse.
var FailureCodes = map[string]int{
	"format":    10,
	"analyze":   11,
	"generated": 12,
	"cache":     14,
}

// FirstExitCode returns the exit code for the first failed gate, in the
// order Run executed them, or 0 when every gate passed. The driver uses
// this when no other failure (e.g. internal error) takes precedence.
func (r *Report) FirstExitCode() int {
	for _, g := range r.Gates {
		if g.Status != StatusFail {
			continue
		}
		if code, ok := FailureCodes[g.Gate]; ok {
			return code
		}
		return 1
	}
	return 0
}

// Run executes every Checker in checkers against the pipeline result and
// aggregates their reports. Skipped gates (excluded by `gates` filter, if
// non-empty) record StatusSkipped so JSON consumers see the full set
// regardless of which subset ran.
//
// The order in checkers is preserved in the output so callers can rely on
// "format runs before analyze runs before generated" without the gate
// definitions having to know about each other.
func Run(
	ctx context.Context,
	p *pipeline.Result,
	env Env,
	checkers []Checker,
	gates []string,
) *Report {
	want := set.New(gates...)
	report := &Report{}
	start := time.Now()
	for _, c := range checkers {
		gr := GateReport{Gate: c.Name(), Status: StatusSkipped}
		if len(want) == 0 || want.Contains(c.Name()) {
			gr = c.Run(ctx, p, env)
		}
		sortFindings(gr.Findings)
		report.Gates = append(report.Gates, gr)
		switch gr.Status {
		case StatusPass:
			report.TotalRun++
			report.TotalPassed++
		case StatusFail:
			report.TotalRun++
			report.TotalFailed++
		}
	}
	report.Elapsed = time.Since(start)
	return report
}

func sortFindings(f []Finding) {
	sort.Slice(f, func(i, j int) bool {
		if f[i].Path != f[j].Path {
			return f[i].Path < f[j].Path
		}
		if f[i].Line != f[j].Line {
			return f[i].Line < f[j].Line
		}
		return f[i].Message < f[j].Message
	})
}
