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
	"strings"
	"time"

	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/freeze"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/set"
)

// VersionsGate verifies the explicitly managed version chains against the
// live schemas: every versioned path has a chain, the current version file
// matches canonical emission byte-for-byte, and every redeclaration differs
// structurally from its resolved predecessor.
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

	// Once any chain exists, every versioned path must have one.
	covered := make(set.Set[string])
	for _, chain := range chains {
		covered.Add(chain.LivePath())
	}
	for _, t := range p.Resolutions.Types {
		if !domain.HasExprFromType(t, "go", "version") {
			continue
		}
		livePath := versionFileLivePath(t.FilePath)
		if !covered.Contains(livePath) {
			covered.Add(livePath)
			r.fail(Finding{
				Path:     t.FilePath,
				Severity: SeverityError,
				Message: fmt.Sprintf(
					"%s declares versioned types but has no version chain",
					livePath,
				),
				FixHint: "create schemas/<domain>/versions/<resource>/v0.oracle",
			})
		}
	}

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
	f, err := resolver.File(ctx, livePath, current)
	if err != nil {
		r.fail(Finding{
			Path:     chain.FilePath(current) + ".oracle",
			Severity: SeverityError,
			Message:  err.Error(),
		})
		return
	}

	// Drift: the current file must equal canonical emission byte-for-byte.
	// A chain whose live schema is gone is history-only; nothing to compare.
	if len(p.Resolutions.TypesInNamespace(chain.Resource)) > 0 {
		canonical, err := freeze.Canonical(ctx, freeze.Input{
			Live:     p.Resolutions,
			Resolver: resolver,
			Chain:    chain,
			N:        current,
			Pins:     f.Pins,
			Docs:     docOverrides(f),
			Pinned:   pinnedNames(f),
		})
		if err != nil {
			r.fail(Finding{
				Path:     chain.FilePath(current) + ".oracle",
				Severity: SeverityError,
				Message:  err.Error(),
			})
		} else {
			filePath := chain.FilePath(current) + ".oracle"
			onDisk, err := os.ReadFile(filepath.Join(env.RepoRoot, filePath))
			if err != nil {
				r.fail(Finding{
					Path: filePath, Severity: SeverityError,
					Message: err.Error(),
				})
			} else if string(onDisk) != canonical {
				finding := Finding{
					Path:     filePath,
					Severity: SeverityError,
					Message: fmt.Sprintf(
						"%s drifts from its current version file", livePath,
					),
					FixHint: fmt.Sprintf(
						"run `oracle migrate %s` to bump, or "+
							"`oracle migrate --amend %s` if v%d has not shipped",
						chain.Resource, chain.Resource, current,
					),
				}
				if env.IncludeDiffs {
					finding.Diff = unifiedDiff(
						filePath, canonical, string(onDisk), 200,
					)
				}
				r.fail(finding)
			}
		}
	}

	// Minimality: every redeclaration must differ structurally from its
	// resolved predecessor.
	for k := 1; k <= current; k++ {
		fk, err := resolver.File(ctx, livePath, k)
		if err != nil {
			r.fail(Finding{
				Path:     chain.FilePath(k) + ".oracle",
				Severity: SeverityError,
				Message:  err.Error(),
			})
			return
		}
		surf, err := resolver.Surface(ctx, livePath, k-1)
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

// docOverrides extracts a file's explicit @doc declarations by type name.
func docOverrides(f *versions.File) map[string]string {
	docs := make(map[string]string)
	for _, t := range f.Defined {
		if d := doc.Get(t.Domains); d != "" {
			docs[t.Name] = d
		}
	}
	return docs
}

// pinnedNames extracts a file's @go pinned type names.
func pinnedNames(f *versions.File) set.Set[string] {
	pinned := make(set.Set[string])
	for _, t := range f.Defined {
		dom, ok := t.Domains["go"]
		if !ok {
			continue
		}
		if _, has := dom.Expressions.Find("pinned"); has {
			pinned.Add(t.Name)
		}
	}
	return pinned
}

// versionFileLivePath trims a schema file path to its import-path form.
func versionFileLivePath(filePath string) string {
	if p, err := versions.LiveFromFilePath(filePath); err == nil {
		return p
	}
	return strings.TrimSuffix(filePath, ".oracle")
}
