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
	"errors"
	"io/fs"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin"
	"golang.org/x/sync/errgroup"
)

// GeneratedGate is the load-bearing gate of `oracle check`: it asserts
// that what the pipeline would write to disk on the next `oracle sync`
// already matches what is on disk today.
//
// The mechanics: for every plugin output, run the same formatter chain
// `oracle sync` would, then byte-compare the canonical bytes to the
// existing file. Any divergence - missing file, content mismatch - is
// reported as drift with `oracle sync` as the fix hint.
//
// Because the formatter chain is the *same* one sync uses (passed in via
// the constructor, not reconstructed here), it is structurally
// impossible for this gate to disagree with what sync would produce.
type GeneratedGate struct {
	formatters *format.Registry
	workers    int
}

// NewGeneratedGate constructs a generated-drift gate. The formatter
// registry must be the same one sync uses; passing a different one
// turns the gate into a lie.
func NewGeneratedGate(formatters *format.Registry, workers int) *GeneratedGate {
	return &GeneratedGate{formatters: formatters, workers: workers}
}

func (GeneratedGate) Name() string { return "generated" }

func (g GeneratedGate) Run(ctx context.Context, p *pipeline.Result, env Env) GateReport {
	start := time.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}

	all := make([]plugin.File, 0)
	for _, files := range p.Outputs {
		all = append(all, files...)
	}

	workers := g.workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}

	findings := make([]Finding, len(all))
	var fillFinding sync.Mutex
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(workers)
	for i, f := range all {
		eg.Go(func() error {
			if err := gctx.Err(); err != nil {
				return err
			}
			finding, ok := g.checkOne(gctx, env, f)
			if !ok {
				return nil
			}
			fillFinding.Lock()
			findings[i] = finding
			fillFinding.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		r.Status = StatusFail
		r.Findings = append(r.Findings, Finding{
			Severity: SeverityError,
			Message:  "generated gate aborted: " + err.Error(),
		})
		r.Elapsed = time.Since(start)
		return r
	}
	for _, f := range findings {
		if f.Severity == 0 && f.Message == "" {
			continue
		}
		r.Findings = append(r.Findings, f)
	}
	if len(r.Findings) > 0 {
		r.Status = StatusFail
	}
	r.Elapsed = time.Since(start)
	return r
}

// checkOne runs the formatter chain on a single plugin output and
// compares the canonical bytes to the on-disk file. Returns (finding,
// true) when there is something to report, (zero, false) when the file
// is up to date.
//
// Errors from the formatter chain are reported as findings rather than
// surfaced as Go errors. Sync would experience the same error on the
// next run; for a CI gate the right thing is to attribute the failure
// to the file that triggered it and keep going so the user sees every
// failing file at once.
func (g GeneratedGate) checkOne(ctx context.Context, env Env, f plugin.File) (Finding, bool) {
	abs := paths.Resolve(f.Path, env.RepoRoot)
	canonical, err := g.formatters.Format(ctx, f.Content, abs)
	if err != nil {
		return Finding{
			Path:     f.Path,
			Severity: SeverityError,
			Message:  "formatter failed for generated output: " + err.Error(),
			FixHint:  "fix the formatter for this extension and re-run `oracle sync`",
		}, true
	}
	existing, err := os.ReadFile(abs)
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
	if string(existing) == string(canonical) {
		return Finding{}, false
	}
	finding := Finding{
		Path:     f.Path,
		Severity: SeverityError,
		Message:  "on-disk content does not match generated output",
		FixHint:  "run `oracle sync`",
	}
	if env.IncludeDiffs {
		finding.Diff = unifiedDiff(f.Path, string(existing), string(canonical), 60)
	}
	return finding, true
}
