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
	"os"
	"sort"
	"time"

	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/versions"
)

// FormatGate fails when any schema's on-disk source bytes differ from its
// canonical formatter output. Live schemas compare against the pipeline's
// FormattedSources; version files, which the pipeline never loads as
// schemas, are read from disk and formatted here.
type FormatGate struct{}

// NewFormatGate returns a Checker that asserts every schema is already
// in canonical form.
func NewFormatGate() *FormatGate { return &FormatGate{} }

func (FormatGate) Name() string { return "format" }

func (g FormatGate) Run(_ context.Context, p *pipeline.Result, env Env) GateReport {
	start := time.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}
	for _, rel := range p.Schemas {
		g.compare(&r, env, rel, string(p.Sources[rel]), string(p.FormattedSources[rel]))
	}
	for _, rel := range versionFilePaths(env.RepoRoot, &r) {
		raw, err := os.ReadFile(paths.Resolve(rel, env.RepoRoot))
		if err != nil {
			r.Findings = append(r.Findings, Finding{
				Path:     rel,
				Severity: SeverityError,
				Message:  "failed to read version file: " + err.Error(),
			})
			continue
		}
		canonical, err := formatter.Format(string(raw))
		if err != nil {
			r.Findings = append(r.Findings, Finding{
				Path:     rel,
				Severity: SeverityError,
				Message:  "failed to format version file: " + err.Error(),
			})
			continue
		}
		g.compare(&r, env, rel, string(raw), canonical)
	}
	if len(r.Findings) > 0 {
		r.Status = StatusFail
	}
	r.Elapsed = time.Since(start)
	return r
}

func (FormatGate) compare(r *GateReport, env Env, rel, raw, canonical string) {
	if raw == canonical {
		return
	}
	f := Finding{
		Path:     rel,
		Severity: SeverityError,
		Message:  "schema is not canonically formatted",
		FixHint:  "run `oracle fmt`",
	}
	if env.IncludeDiffs {
		f.Diff = unifiedDiff(rel, raw, canonical, 40)
	}
	r.Findings = append(r.Findings, f)
}

// versionFilePaths lists every version file in the repository, repo-relative
// and sorted. A discovery failure lands as a finding on the report.
func versionFilePaths(repoRoot string, r *GateReport) []string {
	chains, err := versions.Discover(repoRoot)
	if err != nil {
		r.Findings = append(r.Findings, Finding{
			Severity: SeverityError,
			Message:  "failed to discover version chains: " + err.Error(),
		})
		return nil
	}
	var rels []string
	for _, chain := range chains {
		for _, n := range chain.Numbers {
			rels = append(rels, paths.EnsureOracleExtension(chain.FilePath(n)))
		}
	}
	sort.Strings(rels)
	return rels
}
