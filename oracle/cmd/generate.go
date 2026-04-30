// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/output"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"golang.org/x/sync/errgroup"
)

func generate(
	ctx context.Context,
	files []string,
	repoRoot string,
	registry *plugin.Registry,
) (*generateResult, *diagnostics.Diagnostics) {
	loader := analyzer.NewStandardFileLoader(repoRoot)
	table, diag := analyzer.Analyze(ctx, files, loader)
	if diag != nil && !diag.Ok() {
		return nil, diag
	}

	result := &generateResult{
		Resolutions: table,
		Files:       make(map[string][]plugin.File),
	}
	levels := topoLevels(registry)
	var mu sync.Mutex
	for _, level := range levels {
		eg, gctx := errgroup.WithContext(ctx)
		for _, p := range level {
			p := p
			eg.Go(func() error {
				if err := gctx.Err(); err != nil {
					return err
				}
				output.PluginStart(p.Name())
				req := &plugin.Request{Resolutions: table, RepoRoot: repoRoot}
				for _, depName := range p.Requires() {
					dep := registry.Get(depName)
					if dep == nil {
						return errors.Newf("plugin %q requires unknown plugin %q", p.Name(), depName)
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
					result.Files[p.Name()] = resp.Files
					mu.Unlock()
					output.PluginDone(p.Name(), len(resp.Files))
				}
				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			if staleErr, ok := err.(*plugin.DependencyStaleError); ok {
				errDiag := &diagnostics.Diagnostics{}
				errDiag.Add(diagnostics.Error(staleErr, nil))
				return nil, errDiag
			}
			diag.Add(diagnostics.Error(err, nil))
		}
	}
	return result, diag
}

// topoLevels returns the plugins grouped into levels where each level can
// run concurrently. Level N contains plugins whose Requires() are all
// satisfied by levels < N. Plugins with cyclic or unknown dependencies
// fall through to the final level so the regular Check() error path can
// surface the actual failure. Plugins within a level are sorted by name
// to keep generation order deterministic.
func topoLevels(registry *plugin.Registry) [][]plugin.Plugin {
	remaining := make(map[string]plugin.Plugin)
	for _, p := range registry.All() {
		remaining[p.Name()] = p
	}
	var levels [][]plugin.Plugin
	placed := make(map[string]struct{})
	for len(remaining) > 0 {
		var level []plugin.Plugin
		for _, p := range remaining {
			satisfied := true
			for _, dep := range p.Requires() {
				if _, ok := placed[dep]; ok {
					continue
				}
				if _, exists := remaining[dep]; exists {
					satisfied = false
					break
				}
			}
			if satisfied {
				level = append(level, p)
			}
		}
		if len(level) == 0 {
			for _, p := range remaining {
				level = append(level, p)
			}
		}
		sort.Slice(level, func(i, j int) bool { return level[i].Name() < level[j].Name() })
		for _, p := range level {
			placed[p.Name()] = struct{}{}
			delete(remaining, p.Name())
		}
		levels = append(levels, level)
	}
	return levels
}

type generateResult struct {
	Resolutions *resolution.Table
	Files       map[string][]plugin.File
}

type syncResult struct {
	ByPlugin   map[string][]string
	Written    []string
	Unchanged  []string
	Skipped    []string // skipped via cache hit
	WrittenAbs []string // absolute paths of written files, for downstream steps
}

// syncFiles formats every generated file and writes those whose canonical
// bytes differ from the on-disk file. The format-then-write step is short-
// circuited when the cache shows the same raw content was generated for
// the same path on the previous run AND the on-disk file still exists,
// since formatters and the file are presumed to already be in their
// canonical state.
func (r *generateResult) syncFiles(
	ctx context.Context,
	repoRoot string,
	formatters *format.Registry,
	cache *format.Cache,
	workers int,
) (*syncResult, error) {
	result := &syncResult{
		Written:    make([]string, 0),
		Unchanged:  make([]string, 0),
		Skipped:    make([]string, 0),
		WrittenAbs: make([]string, 0),
		ByPlugin:   make(map[string][]string),
	}

	type pending struct {
		Plugin   string
		RelPath  string
		AbsPath  string
		Raw      []byte
		RawHash  string
		Existing []byte
	}

	keep := make(map[string]struct{})
	var toFormat []pending
	for pluginName, files := range r.Files {
		for _, f := range files {
			absPath := filepath.Join(repoRoot, f.Path)
			rawHash := format.Hash(f.Content)
			keep[f.Path] = struct{}{}

			cachedHash, hit := cache.LookupRaw(f.Path)
			if hit && cachedHash == rawHash {
				if _, err := os.Stat(absPath); err == nil {
					result.Skipped = append(result.Skipped, f.Path)
					continue
				}
			}

			existing, err := os.ReadFile(absPath)
			if err != nil && !os.IsNotExist(err) {
				return nil, errors.Wrapf(err, "read existing %s", absPath)
			}
			toFormat = append(toFormat, pending{
				Plugin:   pluginName,
				RelPath:  f.Path,
				AbsPath:  absPath,
				Raw:      f.Content,
				RawHash:  rawHash,
				Existing: existing,
			})
		}
	}

	cache.PruneRawTo(keep)

	if len(toFormat) == 0 {
		printFormatPlan(0, len(result.Skipped))
		return result, nil
	}

	printFormatPlan(len(toFormat), len(result.Skipped))
	batch := make([]format.File, len(toFormat))
	for i, p := range toFormat {
		batch[i] = format.File{Path: p.AbsPath, Content: p.Raw}
	}
	formatStart := time.Now()
	formatted, err := formatters.FormatBatch(ctx, batch, workers)
	if err != nil {
		return nil, err
	}
	printFormatDone(time.Since(formatStart))

	var mu sync.Mutex
	eg, gctx := errgroup.WithContext(ctx)
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	eg.SetLimit(workers)
	for i, p := range toFormat {
		i, p := i, p
		eg.Go(func() error {
			if err := gctx.Err(); err != nil {
				return err
			}
			canonical := formatted[i].Content
			if p.Existing != nil && string(p.Existing) == string(canonical) {
				mu.Lock()
				result.Unchanged = append(result.Unchanged, p.RelPath)
				cache.PutRaw(p.RelPath, p.RawHash)
				mu.Unlock()
				return nil
			}
			if err := os.MkdirAll(filepath.Dir(p.AbsPath), 0755); err != nil {
				return errors.Wrapf(err, "mkdir %s", filepath.Dir(p.AbsPath))
			}
			if err := os.WriteFile(p.AbsPath, canonical, 0644); err != nil {
				return errors.Wrapf(err, "write %s", p.AbsPath)
			}
			mu.Lock()
			result.Written = append(result.Written, p.RelPath)
			result.WrittenAbs = append(result.WrittenAbs, p.AbsPath)
			result.ByPlugin[p.Plugin] = append(result.ByPlugin[p.Plugin], p.RelPath)
			cache.PutRaw(p.RelPath, p.RawHash)
			mu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	printWritePlan(len(result.Written), len(result.Unchanged))
	return result, nil
}
