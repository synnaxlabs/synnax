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
	"time"

	"github.com/synnaxlabs/oracle/pipeline"
)

// FormatGate fails when any input schema's on-disk source bytes differ
// from its canonical formatter output. The pipeline already produced
// FormattedSources in memory; this gate just compares them against the
// raw Sources for the same path. No filesystem reads.
type FormatGate struct{}

// NewFormatGate returns a Checker that asserts every schema is already
// in canonical form.
func NewFormatGate() *FormatGate { return &FormatGate{} }

func (FormatGate) Name() string { return "format" }

func (g FormatGate) Run(_ context.Context, p *pipeline.Result, env Env) GateReport {
	start := time.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}
	for _, rel := range p.Schemas {
		raw := string(p.Sources[rel])
		canonical := string(p.FormattedSources[rel])
		if raw == canonical {
			continue
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
	if len(r.Findings) > 0 {
		r.Status = StatusFail
	}
	r.Elapsed = time.Since(start)
	return r
}
