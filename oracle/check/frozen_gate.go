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

	"github.com/synnaxlabs/oracle/frozen"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/x/telem"
)

// frozenGate fails when a frozen version package no longer matches its recorded sum.
// The generated gate cannot see this case: it compares the disk to the generator, so a
// generator change passes it once synced.
type frozenGate struct{}

// NewFrozenGate returns a Checker that asserts frozen version packages stand still.
func NewFrozenGate() Checker { return frozenGate{} }

func (frozenGate) Name() string { return "frozen" }

func (g frozenGate) Run(
	ctx context.Context, res *pipeline.Result, env Env,
) (r GateReport) {
	start := telem.Now()
	r = GateReport{Gate: g.Name(), Status: StatusPass}
	defer func() { r.Elapsed = telem.Since(start) }()
	if res.Resolutions == nil || res.Versions == nil {
		return r
	}
	chains, err := frozen.Compute(ctx, res, env.RepoRoot)
	if err != nil {
		r.fail(Finding{Severity: SeverityError, Message: err.Error()})
		return r
	}
	for _, c := range chains {
		recorded, err := frozen.Read(c.Path, env.RepoRoot)
		if err != nil {
			r.fail(Finding{Path: c.Path, Severity: SeverityError, Message: err.Error()})
			continue
		}
		for _, n := range c.Versions(recorded) {
			checkSum(&r, c, n, recorded)
		}
	}
	return r
}

func checkSum(r *GateReport, c frozen.Chain, n int, recorded map[int]frozen.Sum) {
	sum, current := c.Sums[n]
	old, ok := recorded[n]
	if ok && current && old == sum {
		return
	}
	if ok && c.Moved(n, old) {
		r.fail(Finding{
			Path:     c.Package(n),
			Severity: SeverityError,
			Message:  "frozen package changed, but its version files did not",
			FixHint: "find what moved it. A new version must leave it byte-identical: " +
				"restate a live rule in the version file, or fix the generator. If " +
				"every version must change, run `oracle sync --refreeze`",
		})
		return
	}
	r.fail(Finding{
		Path:     c.Path,
		Severity: SeverityError,
		Message:  "sum file does not match " + c.Package(n),
		FixHint:  "run `oracle sync`",
	})
}
