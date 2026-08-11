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
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin/go/freeze"
	"github.com/synnaxlabs/oracle/versions"
)

// VersionsGate verifies the explicitly managed version chains: the live
// file's version-owned content matches chain resolution (the merged
// projection), and every redeclaration differs structurally from its
// resolved predecessor.
type VersionsGate struct{}

// Name implements Checker.
func (VersionsGate) Name() string { return "versions" }

// Run implements Checker.
func (g VersionsGate) Run(
	ctx context.Context, p *pipeline.Result, env Env,
) GateReport {
	start := time.Now()
	r := GateReport{Gate: g.Name(), Status: StatusPass}
	defer func() { r.Elapsed = time.Since(start) }()
	if p.Resolutions == nil {
		return r
	}
	chains, err := versions.Discover(env.RepoRoot)
	if err != nil {
		r.fail(Finding{Severity: SeverityError, Message: err.Error()})
		return r
	}
	if len(chains) == 0 {
		return r
	}
	resolver := versions.NewResolver(
		chains, analyzer.NewStandardFileLoader(env.RepoRoot),
	)

	for _, livePath := range slices.Sorted(maps.Keys(chains)) {
		g.checkChain(ctx, &r, p, env, resolver, chains[livePath])
	}
	return r
}

// checkChain runs the drift and minimality checks for one chain.
func (g VersionsGate) checkChain(
	ctx context.Context,
	r *GateReport,
	p *pipeline.Result,
	env Env,
	resolver *versions.Resolver,
	chain versions.Chain,
) {
	livePath := chain.LivePath()
	current := chain.Current()

	// Consistency: the live file's version-owned content must match chain
	// resolution — the merged projection is the canonical live file. The
	// version files are the authority; a live edit to version-owned content
	// is overwritten by sync, and a chain edit lands in the live file the
	// same way.
	if merged, ok := p.MergedSources[livePath+".oracle"]; ok {
		filePath := livePath + ".oracle"
		onDisk, err := os.ReadFile(filepath.Join(env.RepoRoot, filePath))
		if err != nil && !os.IsNotExist(err) {
			r.fail(Finding{
				Path: filePath, Severity: SeverityError, Message: err.Error(),
			})
		} else if string(onDisk) != string(merged) {
			finding := Finding{
				Path:     filePath,
				Severity: SeverityError,
				Message: fmt.Sprintf(
					"%s drifts from its version files", livePath,
				),
				FixHint: fmt.Sprintf(
					"the version files under %s are the authority for "+
						"version-owned content; run `oracle sync` to project them",
					chain.Dir(),
				),
			}
			if env.IncludeDiffs {
				finding.Diff = unifiedDiff(
					filePath, string(merged), string(onDisk), 200,
				)
			}
			r.fail(finding)
		}
	}

	// Minimality: every redeclaration must differ structurally from its
	// resolved predecessor.
	for i, k := range chain.Numbers {
		if i == 0 || k > current {
			continue
		}
		fk, err := resolver.File(ctx, livePath, k)
		if err != nil {
			r.fail(Finding{
				Path:     chain.FilePath(k) + ".oracle",
				Severity: SeverityError,
				Message:  err.Error(),
			})
			return
		}
		surf, err := resolver.Surface(ctx, livePath, chain.Numbers[i-1])
		if err != nil {
			r.fail(Finding{
				Path:     chain.FilePath(k) + ".oracle",
				Severity: SeverityError,
				Message:  err.Error(),
			})
			return
		}
		for _, t := range fk.Defined {
			if t.Synthetic {
				continue
			}
			// Pinned declarations track the live schema and may legitimately
			// match their predecessor.
			if dom, ok := t.Domains["go"]; ok {
				if _, pinned := dom.Expressions.Find("pinned"); pinned {
					continue
				}
			}
			def, ok := surf[t.Name]
			if !ok {
				continue
			}
			definer, err := resolver.File(ctx, livePath, def.Version)
			if err != nil {
				r.fail(Finding{
					Path:     chain.FilePath(k) + ".oracle",
					Severity: SeverityError,
					Message:  err.Error(),
				})
				return
			}
			if freeze.StructurallyEqual(def.Type, t, definer.Table, fk.Table) {
				r.fail(Finding{
					Path:     chain.FilePath(k) + ".oracle",
					Severity: SeverityError,
					Message: fmt.Sprintf(
						"%s v%d redeclares %s identically to v%d",
						livePath, k, t.Name, def.Version,
					),
					FixHint: fmt.Sprintf(
						"use an alias: %s = v%d.%s",
						t.Name, def.Version, t.Name,
					),
				})
			}
		}
	}
}

// fail records an error finding and flips the gate status.
func (r *GateReport) fail(f Finding) {
	r.Findings = append(r.Findings, f)
	r.Status = StatusFail
}

