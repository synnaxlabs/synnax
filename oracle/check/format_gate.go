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

	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/x/telem"
)

// FormatGate fails when any schema's on-disk source bytes differ from its canonical
// formatter output. Live schemas compare against the pipeline's FormattedSources;
// version files, which the pipeline never loads as schemas, are read from disk and
// formatted here.
type formatGate struct{}

// NewFormatGate returns a Checker that asserts every schema is already in canonical
// form.
func NewFormatGate() Checker { return formatGate{} }

func (formatGate) Name() string { return "format" }

func (g formatGate) Run(_ context.Context, res *pipeline.Result, env Env) GateReport {
	start := telem.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}
	for _, rel := range res.Schemas {
		compareFormat(
			&r, env, rel, string(res.Sources[rel]), string(res.FormattedSources[rel]),
		)
	}
	for _, rel := range versionFilePaths(res) {
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
		compareFormat(&r, env, rel, string(raw), canonical)
	}
	if len(r.Findings) > 0 {
		r.Status = StatusFail
	}
	r.Elapsed = telem.Since(start)
	return r
}

func compareFormat(r *GateReport, env Env, rel, raw, canonical string) {
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

// versionFilePaths lists every version file in the repository, repo-relative and
// sorted.
func versionFilePaths(res *pipeline.Result) []string {
	var rels []string
	for _, chain := range res.Chains {
		for _, n := range chain.Numbers {
			rels = append(rels, paths.EnsureOracleExtension(chain.FilePath(n)))
		}
	}
	sort.Strings(rels)
	return rels
}
