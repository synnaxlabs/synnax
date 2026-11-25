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
	"context"
	"io/fs"

	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// GeneratedGate is the load-bearing gate of `oracle check`: it asserts that what the
// pipeline would write to disk on the next `oracle sync` already matches what is on
// disk today.
//
// The mechanics: for every plugin output, run the same formatter chain `oracle sync`
// would, then byte-compare the canonical bytes to the existing file. Any divergence -
// missing file, content mismatch - is reported as drift with `oracle sync` as the fix
// hint.
//
// Because the formatter chain is the *same* one sync uses (passed in via the
// constructor, not reconstructed here), it is structurally impossible for this gate to
// disagree with what sync would produce.
type generatedGate struct {
	formatters *format.Registry
	workers    int
}

// NewGeneratedGate constructs a generated-drift gate. The formatter registry must be
// the same one sync uses; passing a different one turns the gate into a lie.
func NewGeneratedGate(formatters *format.Registry, workers int) Checker {
	return generatedGate{formatters: formatters, workers: workers}
}

func (generatedGate) Name() string { return "generated" }

func (g generatedGate) Run(
	ctx context.Context,
	res *pipeline.Result,
	env Env,
) GateReport {
	start := telem.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}

	all := make([]plugin.File, 0)
	for _, files := range res.Outputs {
		all = append(all, files...)
	}

	batch := make([]format.File, len(all))
	for i, f := range all {
		batch[i] = format.File{
			Path:    paths.Resolve(f.Path, env.RepoRoot),
			Content: f.Content,
		}
	}
	formatted := g.formatters.FormatBatch(ctx, batch, g.workers)
	if err := ctx.Err(); err != nil {
		r.Status = StatusFail
		r.Findings = append(r.Findings, Finding{
			Severity: SeverityError,
			Message:  "generated gate aborted: " + err.Error(),
		})
		r.Elapsed = telem.Since(start)
		return r
	}
	for i, f := range all {
		if finding, ok := checkOne(env, f, formatted[i]); ok {
			r.Findings = append(r.Findings, finding)
		}
	}
	if len(r.Findings) > 0 {
		r.Status = StatusFail
	}
	r.Elapsed = telem.Since(start)
	return r
}

// checkOne compares the formatted bytes of one plugin output to the on-disk file.
// Returns (finding, true) when there is something to report, (zero, false) when the
// file is up to date.
//
// Errors from the formatter chain are reported as findings rather than surfaced as Go
// errors. Sync would experience the same error on the next run; for a CI gate the right
// thing is to attribute the failure to the file that triggered it and keep going so the
// user sees every failing file at once.
func checkOne(env Env, f plugin.File, formatted format.Result) (Finding, bool) {
	if formatted.Err != nil {
		return Finding{
			Path:     f.Path,
			Severity: SeverityError,
			Message:  "formatter failed for generated output: " + formatted.Err.Error(),
			FixHint:  "fix the formatter for this extension and re-run `oracle sync`",
		}, true
	}
	existing, err := paths.ReadFile(f.Path, env.RepoRoot)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Finding{
				Path:     f.Path,
				Severity: SeverityError,
				Message:  "generated file is missing on disk",
				FixHint:  "run `oracle sync`",
			}, true
		}
		return Finding{
			Path:     f.Path,
			Severity: SeverityError,
			Message:  "read existing file: " + err.Error(),
		}, true
	}
	if string(existing) == string(formatted.Content) {
		return Finding{}, false
	}
	finding := Finding{
		Path:     f.Path,
		Severity: SeverityError,
		Message:  "on-disk content does not match generated output",
		FixHint:  "run `oracle sync`",
	}
	if env.IncludeDiffs {
		finding.Diff = unifiedDiff(
			f.Path,
			string(existing),
			string(formatted.Content),
			60,
		)
	}
	return finding, true
}
