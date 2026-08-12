// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package pipeline is the single in-memory execution path that takes a set of .oracle
// schema files and produces (a) the canonical formatted schema source for each input
// and (b) the canonical generated output for each registered plugin. Every other oracle
// entrypoint - sync, generate, check - is a consumer of this pipeline. There is no
// other code path that turns schemas into outputs; sync and generate cannot disagree
// with check about what is "valid", because they all run the same Run.
//
// The pipeline does not touch the filesystem outside of reading inputs. Writing
// canonical schema source, writing generated outputs, or invoking post-write hooks are
// all consumer concerns layered on top of the Result.
package pipeline

import (
	"context"
	"maps"
	"os"
	"runtime"
	"sort"
	"sync"

	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"golang.org/x/sync/errgroup"
)

// Options configures a single pipeline run.
type Options struct {
	// RepoRoot is the absolute path to the repository root.
	RepoRoot string
	// Schemas is the set of repo-relative .oracle file paths to analyze. Callers must
	// normalize these via paths.Normalize before passing them in; see DiscoverSchemas
	// for the canonical discovery helper.
	Schemas []string
	// Plugins is the registry of code generators to run. Pass nil to skip plugin
	// generation entirely (analyze-only mode).
	Plugins *plugin.Registry
}

// Result is the artifact set produced by a single pipeline run, held entirely in
// memory.
//
// Field rules:
//   - Sources is always populated with the as-read source bytes for every
//     input schema, even when later stages fail.
//   - FormattedSources holds the canonical formatter output for every input.
//     A schema's entry equals its Sources entry when no formatting drift
//     exists.
//   - MergedSources holds the canonical live-file projection for every
//     versioned resource: version-owned content from the chain, live-owned
//     annotations from the on-disk source. Analysis and sync use these bytes
//     in place of FormattedSources; check compares them against disk.
//   - Resolutions and Diagnostics are populated by the analyzer. Resolutions
//     is nil when the analyzer produced fatal errors. Diagnostics is always
//     non-nil and may carry warnings even on success.
//   - Outputs is the per-plugin set of generated files, byte-identical to
//     what plugins emitted (no formatter chain applied yet). Empty when
//     Options.Plugins is nil or the analyzer failed.
type Result struct {
	Schemas          []string
	Sources          map[string][]byte
	FormattedSources map[string][]byte
	MergedSources    map[string][]byte
	Resolutions      *resolution.Table
	Diagnostics      *diagnostics.Files
	Outputs          map[string][]plugin.File
	// Deletions holds repo-relative paths of files plugins requested be removed from
	// disk (e.g. when migrate retargets a transform). Keyed by plugin name to mirror
	// Outputs.
	Deletions map[string][]string
	// Chains holds the discovered version chains, keyed by live import path.
	Chains map[string]versions.Chain
	// Versions resolves the version chains through the same overlay loader
	// analysis used; nil when the repository declares no chains.
	Versions *versions.Resolver
}

// Run executes the pipeline end to end. The returned Result is always non-nil and
// reflects whatever was completed before the first fatal failure. Diagnostics surface
// non-fatal issues (warnings, info, hints). The error return is reserved for IO
// failures and unexpected errors that prevent the pipeline from running at all;
// analyzer or plugin diagnostics do not surface as a Go error.
func Run(ctx context.Context, opts Options) (*Result, error) {
	if opts.RepoRoot == "" {
		return nil, errors.New("pipeline: RepoRoot is required")
	}
	if len(opts.Schemas) == 0 {
		return nil, errors.New("pipeline: at least one schema is required")
	}
	loader := analyzer.NewStandardFileLoader(opts.RepoRoot)
	workers := runtime.GOMAXPROCS(0)

	r := &Result{
		Schemas:          append([]string(nil), opts.Schemas...),
		Sources:          make(map[string][]byte, len(opts.Schemas)),
		FormattedSources: make(map[string][]byte, len(opts.Schemas)),
		Outputs:          make(map[string][]plugin.File),
		Deletions:        make(map[string][]string),
		Diagnostics:      diagnostics.NewFiles(),
	}
	sort.Strings(r.Schemas)

	if err := readAndFormat(ctx, r, opts, workers); err != nil {
		return r, err
	}

	chains, err := versions.Discover(opts.RepoRoot)
	if err != nil {
		return r, err
	}
	r.Chains = chains
	if len(chains) > 0 {
		if err := mergeLiveSources(ctx, r, loader, chains); err != nil {
			return r, err
		}
	}
	effective := newOverlayLoader(loader, r.effectiveSources())
	if len(chains) > 0 {
		r.Versions = versions.NewResolver(chains, effective)
	}

	if err := analyze(ctx, r, effective); err != nil {
		return r, err
	}

	if opts.Plugins != nil && r.Resolutions != nil {
		if err := generate(ctx, r, opts, workers); err != nil {
			return r, err
		}
	}

	return r, nil
}

// EffectiveSource returns the canonical bytes a schema contributes to analysis and to
// disk after sync: the merged live projection when one exists, the formatted source
// otherwise.
func (r *Result) EffectiveSource(path string) []byte {
	if merged, ok := r.MergedSources[path]; ok {
		return merged
	}
	return r.FormattedSources[path]
}

// effectiveSources overlays MergedSources onto FormattedSources.
func (r *Result) effectiveSources() map[string][]byte {
	out := make(map[string][]byte, len(r.FormattedSources)+len(r.MergedSources))
	maps.Copy(out, r.FormattedSources)
	maps.Copy(out, r.MergedSources)
	return out
}

// mergeLiveSources computes the canonical live-file projection for every version chain,
// iterating to a fixpoint: a merged live file can change what another chain's version
// files resolve through a live import, so merges re-run over their own output until
// stable.
func mergeLiveSources(
	ctx context.Context,
	r *Result,
	loader analyzer.FileLoader,
	chains map[string]versions.Chain,
) error {
	livePaths := make([]string, 0, len(chains))
	for livePath := range chains {
		livePaths = append(livePaths, livePath)
	}
	sort.Strings(livePaths)
	prev := make(map[string][]byte)
	for range 3 {
		overlay := make(map[string][]byte, len(r.FormattedSources)+len(prev))
		maps.Copy(overlay, r.FormattedSources)
		maps.Copy(overlay, prev)
		resolver := versions.NewResolver(chains, newOverlayLoader(loader, overlay))
		next := make(map[string][]byte, len(chains))
		for _, livePath := range livePaths {
			file := livePath + ".oracle"
			merged, err := versions.MergeLive(
				ctx, resolver, chains[livePath], string(r.FormattedSources[file]),
			)
			if err != nil {
				return err
			}
			if merged == "" {
				continue
			}
			next[file] = []byte(merged)
		}
		if sourcesEqual(prev, next) {
			r.MergedSources = next
			for file := range next {
				if _, known := r.Sources[file]; !known {
					r.Schemas = append(r.Schemas, file)
					sort.Strings(r.Schemas)
				}
			}
			return nil
		}
		prev = next
	}
	return errors.New("live-file merge did not converge")
}

// sourcesEqual reports whether two source maps hold identical bytes.
func sourcesEqual(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for p, ab := range a {
		if bb, ok := b[p]; !ok || string(ab) != string(bb) {
			return false
		}
	}
	return true
}

// DiscoverSchemas finds every .oracle file under <repoRoot>/schemas and returns the
// repo-relative paths in sorted order. This is the discovery helper sync, generate, and
// check share; it is the only correct way to build Options.Schemas from a glob.
func DiscoverSchemas(repoRoot string) ([]string, error) {
	abs, err := globOracleSchemas(repoRoot)
	if err != nil {
		return nil, err
	}
	rel := make([]string, 0, len(abs))
	for _, p := range abs {
		n, err := paths.Normalize(p, repoRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "normalize schema path %q", p)
		}
		rel = append(rel, n)
	}
	sort.Strings(rel)
	return rel, nil
}

func readAndFormat(ctx context.Context, r *Result, opts Options, workers int) error {
	type entry struct {
		path string
		raw  []byte
	}
	entries := make([]entry, len(r.Schemas))
	eg, gctx := errgroup.WithContext(ctx)
	eg.SetLimit(workers)
	for i, rel := range r.Schemas {
		eg.Go(func() error {
			if err := gctx.Err(); err != nil {
				return err
			}
			abs := paths.Resolve(rel, opts.RepoRoot)
			raw, err := os.ReadFile(abs)
			if err != nil {
				return errors.Wrapf(err, "read schema %s", rel)
			}
			entries[i] = entry{path: rel, raw: raw}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	formattedRaw := make([][]byte, len(entries))
	eg2, gctx2 := errgroup.WithContext(ctx)
	eg2.SetLimit(workers)
	for i, e := range entries {
		eg2.Go(func() error {
			if err := gctx2.Err(); err != nil {
				return err
			}
			out, err := formatter.Format(string(e.raw))
			if err != nil {
				return errors.Wrapf(err, "format schema %s", e.path)
			}
			formattedRaw[i] = []byte(out)
			return nil
		})
	}
	if err := eg2.Wait(); err != nil {
		return err
	}

	for i, e := range entries {
		r.Sources[e.path] = e.raw
		r.FormattedSources[e.path] = formattedRaw[i]
	}
	return nil
}

func analyze(ctx context.Context, r *Result, loader analyzer.FileLoader) error {
	// The loader already overlays the canonical in-memory bytes (formatted sources plus
	// merged live projections), so analysis sees exactly what sync would write to disk.
	table, diag := analyzer.Analyze(ctx, r.Schemas, loader)
	if diag != nil {
		r.Diagnostics.Combine(diag)
	}
	if r.Diagnostics.Ok() {
		r.Resolutions = table
	}
	return nil
}

func generate(ctx context.Context, r *Result, opts Options, workers int) error {
	// Explicitly managed version chains are the versioning baseline. The resolver
	// loads through the same overlay analysis used, so frozen surfaces resolve live
	// imports against the merged projections.
	if r.Versions != nil {
		if err := r.Versions.Annotate(ctx, r.Resolutions); err != nil {
			return err
		}
	}

	levels := topoLevels(opts.Plugins)
	var mu sync.Mutex
	for _, level := range levels {
		eg, gctx := errgroup.WithContext(ctx)
		eg.SetLimit(workers)
		for _, p := range level {
			eg.Go(func() error {
				if err := gctx.Err(); err != nil {
					return err
				}
				req := &plugin.Request{
					Resolutions: r.Resolutions,
					RepoRoot:    opts.RepoRoot,
					Versions:    r.Versions,
				}
				for _, depName := range p.Requires() {
					if opts.Plugins.Get(depName) == nil {
						return errors.Newf(
							"plugin %q requires unknown plugin %q",
							p.Name(), depName,
						)
					}
				}
				resp, err := p.Generate(req)
				if err != nil {
					return errors.Wrapf(err, "plugin %s", p.Name())
				}
				if resp != nil {
					mu.Lock()
					r.Outputs[p.Name()] = resp.Files
					if len(resp.Deletions) > 0 {
						r.Deletions[p.Name()] = resp.Deletions
					}
					mu.Unlock()
				}
				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			r.Diagnostics.Add(diagnostics.Error(err, nil))
			return nil
		}
	}
	return nil
}
