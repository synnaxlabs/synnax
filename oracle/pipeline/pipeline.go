// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package pipeline is the single in-memory execution path that takes a set of
// .oracle schema files and produces (a) the canonical formatted schema source
// for each input and (b) the canonical generated output for each registered
// plugin. Every other oracle entrypoint - sync, generate, check - is a
// consumer of this pipeline. There is no other code path that turns schemas
// into outputs; sync and generate cannot disagree with check about what is
// "valid", because they all run the same Run.
//
// The pipeline does not touch the filesystem outside of reading inputs.
// Writing canonical schema source, writing generated outputs, or invoking
// post-write hooks are all consumer concerns layered on top of the Result.
package pipeline

import (
	"context"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"golang.org/x/sync/errgroup"
)

// Options configures a single pipeline run.
type Options struct {
	// RepoRoot is the absolute path to the repository root.
	RepoRoot string
	// Schemas is the set of repo-relative .oracle file paths to analyze.
	// Callers must normalize these via paths.Normalize before passing them
	// in; see DiscoverSchemas for the canonical discovery helper.
	Schemas []string
	// Plugins is the registry of code generators to run. Pass nil to skip
	// plugin generation entirely (analyze-only mode).
	Plugins *plugin.Registry
	// Loader is the analyzer file loader for resolving schema imports. Pass
	// nil to use a StandardFileLoader rooted at RepoRoot.
	Loader analyzer.FileLoader
	// Workers caps fan-out across schema formatting and plugin generation.
	// Zero (or negative) defaults to GOMAXPROCS.
	Workers int
}

// Result is the artifact set produced by a single pipeline run, held entirely
// in memory.
//
// Field rules:
//   - Sources is always populated with the as-read source bytes for every
//     input schema, even when later stages fail.
//   - FormattedSources holds the canonical formatter output for every input.
//     A schema's entry equals its Sources entry when no formatting drift
//     exists.
//   - Resolutions and Diagnostics are populated by the analyzer. Resolutions
//     is nil when the analyzer produced fatal errors. Diagnostics is always
//     non-nil and may carry warnings even on success.
//   - Outputs is the per-plugin set of generated files, byte-identical to
//     what plugins emitted (no formatter chain applied yet). Empty when
//     Options.Plugins is nil or the analyzer failed.
//   - Timings records elapsed wall time for each pipeline phase. Useful for
//     verbose / profile output.
type Result struct {
	Schemas          []string
	Sources          map[string][]byte
	FormattedSources map[string][]byte
	Resolutions      *resolution.Table
	Diagnostics      *diagnostics.Diagnostics
	Outputs          map[string][]plugin.File
	// Deletions holds repo-relative paths of files plugins requested be
	// removed from disk (e.g. when migrate retargets a transform). Keyed
	// by plugin name to mirror Outputs.
	Deletions map[string][]string
	Timings   Timings
}

// Timings records the wall-clock duration of each pipeline phase.
type Timings struct {
	Read     time.Duration
	Format   time.Duration
	Analyze  time.Duration
	Generate time.Duration
}

// Run executes the pipeline end to end. The returned Result is always non-nil
// and reflects whatever was completed before the first fatal failure.
// Diagnostics surface non-fatal issues (warnings, info, hints). The error
// return is reserved for IO failures and unexpected errors that prevent the
// pipeline from running at all; analyzer or plugin diagnostics do not
// surface as a Go error.
func Run(ctx context.Context, opts Options) (*Result, error) {
	if opts.RepoRoot == "" {
		return nil, errors.New("pipeline: RepoRoot is required")
	}
	if len(opts.Schemas) == 0 {
		return nil, errors.New("pipeline: at least one schema is required")
	}
	loader := opts.Loader
	if loader == nil {
		loader = analyzer.NewStandardFileLoader(opts.RepoRoot)
	}
	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}

	r := &Result{
		Schemas:          append([]string(nil), opts.Schemas...),
		Sources:          make(map[string][]byte, len(opts.Schemas)),
		FormattedSources: make(map[string][]byte, len(opts.Schemas)),
		Outputs:          make(map[string][]plugin.File),
		Deletions:        make(map[string][]string),
		Diagnostics:      &diagnostics.Diagnostics{},
	}
	sort.Strings(r.Schemas)

	if err := readAndFormat(ctx, r, opts, workers); err != nil {
		return r, err
	}

	if err := analyze(ctx, r, loader); err != nil {
		return r, err
	}

	if opts.Plugins != nil && r.Resolutions != nil {
		if err := generate(ctx, r, opts, workers); err != nil {
			return r, err
		}
	}

	return r, nil
}

// DiscoverSchemas finds every .oracle file under <repoRoot>/schemas and
// returns the repo-relative paths in sorted order. This is the discovery
// helper sync, generate, and check share; it is the only correct way to
// build Options.Schemas from a glob.
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
	readStart := time.Now()
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
	r.Timings.Read = time.Since(readStart)

	formatStart := time.Now()
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
	r.Timings.Format = time.Since(formatStart)

	for i, e := range entries {
		r.Sources[e.path] = e.raw
		r.FormattedSources[e.path] = formattedRaw[i]
	}
	return nil
}

func analyze(ctx context.Context, r *Result, loader analyzer.FileLoader) error {
	start := time.Now()
	// Use the freshly formatted bytes for analysis. This gives format
	// drift the same semantic check the canonical source would receive,
	// without requiring the caller to write them to disk first.
	overlay := newOverlayLoader(loader, r.FormattedSources)
	table, diag := analyzer.Analyze(ctx, r.Schemas, overlay)
	r.Timings.Analyze = time.Since(start)
	if diag != nil {
		r.Diagnostics.Merge(*diag)
	}
	if r.Diagnostics.Ok() {
		r.Resolutions = table
	}
	return nil
}

func generate(ctx context.Context, r *Result, opts Options, workers int) error {
	start := time.Now()
	defer func() { r.Timings.Generate = time.Since(start) }()

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
				}
				for _, depName := range p.Requires() {
					dep := opts.Plugins.Get(depName)
					if dep == nil {
						return errors.Newf(
							"plugin %q requires unknown plugin %q",
							p.Name(), depName,
						)
					}
					if err := dep.Check(req); err != nil {
						return &plugin.DependencyStaleError{
							Plugin:     p.Name(),
							Dependency: depName,
							Reason:     err,
						}
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
			if staleErr, ok := err.(*plugin.DependencyStaleError); ok {
				r.Diagnostics.Add(diagnostics.Error(staleErr, nil))
				return nil
			}
			r.Diagnostics.Add(diagnostics.Error(err, nil))
			return nil
		}
	}
	return nil
}

// FormatGenerated runs the on-disk formatter chain over a single plugin file
// and returns its canonical bytes. This is the post-pipeline normalisation
// step that turns plugin output (raw template render) into what would
// actually land on disk after sync. Sharing this between sync (writes the
// result) and check (compares the result against disk) is what makes the
// generated-drift gate impossible to mismatch with what sync would write.
func FormatGenerated(
	ctx context.Context,
	formatters *format.Registry,
	repoRoot string,
	files []plugin.File,
	workers int,
) ([]plugin.File, error) {
	if len(files) == 0 {
		return nil, nil
	}
	batch := make([]format.File, len(files))
	for i, f := range files {
		batch[i] = format.File{Path: paths.Resolve(f.Path, repoRoot), Content: f.Content}
	}
	formatted, err := formatters.FormatBatch(ctx, batch, workers)
	if err != nil {
		return nil, err
	}
	out := make([]plugin.File, len(files))
	for i, f := range files {
		out[i] = plugin.File{Path: f.Path, Content: formatted[i].Content}
	}
	return out, nil
}
