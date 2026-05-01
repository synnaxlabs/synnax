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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/oracle/codegen"
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/output"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"golang.org/x/sync/errgroup"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync generated code, only writing changed files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runSync(cmd); err != nil {
				printError(err.Error())
				return err
			}
			return nil
		},
	}
}

func runSync(cmd *cobra.Command) error {
	ctx := cmd.Context()
	verbose := viper.GetBool(verboseFlag)
	printBanner()
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "sync must be run within a git repository")
	}

	schemas, err := pipeline.DiscoverSchemas(repoRoot)
	if err != nil {
		return err
	}
	if len(schemas) == 0 {
		return errors.New("no schema files found")
	}
	printSchemaCount(len(schemas))

	registry := buildPluginRegistry()
	result, err := pipeline.Run(ctx, pipeline.Options{
		RepoRoot: repoRoot,
		Schemas:  schemas,
		Plugins:  registry,
	})
	if err != nil {
		return errors.Wrap(err, "pipeline")
	}
	if !result.Diagnostics.Ok() {
		printDiagnostics(result.Diagnostics.String())
		return errors.New("generation failed")
	}

	formattedSchemas, err := writeSchemaSources(result, repoRoot)
	if err != nil {
		return err
	}
	printFormattingStart(len(schemas))
	printFormattingDone(formattedSchemas)

	for name, files := range result.Outputs {
		output.PluginDone(name, len(files))
	}

	formatters, err := format.Default(repoRoot)
	if err != nil {
		return errors.Wrap(err, "build formatter registry")
	}
	cache := format.LoadCache(repoRoot)

	syncResult, err := syncOutputs(ctx, result, repoRoot, formatters, cache, 0)
	if err != nil {
		return errors.Wrap(err, "failed to sync files")
	}
	if err := cache.Save(); err != nil {
		printDim(fmt.Sprintf("save cache: %v", err))
	}

	if verbose && len(syncResult.Written) > 0 {
		pluginByPath := make(map[string]string, len(syncResult.Written))
		for name, files := range syncResult.ByPlugin {
			for _, f := range files {
				pluginByPath[f] = name
			}
		}
		writtenSorted := append([]string(nil), syncResult.Written...)
		sort.Strings(writtenSorted)
		for _, f := range writtenSorted {
			printFileWritten(pluginByPath[f], f)
		}
	}

	for pluginName, files := range syncResult.ByPlugin {
		p := registry.Get(pluginName)
		if pw, ok := p.(plugin.PostWriter); ok {
			absPaths := make([]string, len(files))
			for i, f := range files {
				absPaths[i] = filepath.Join(repoRoot, f)
			}
			if err := pw.PostWrite(absPaths); err != nil {
				printDim(fmt.Sprintf("post-write hook for %s failed: %v", pluginName, err))
			}
		}
	}

	for _, files := range result.Deletions {
		for _, deletePath := range files {
			abs := filepath.Join(repoRoot, deletePath)
			if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
				printDim(fmt.Sprintf("remove %s: %v", deletePath, err))
			}
		}
	}

	changedProtos := syncResult.ByPlugin["pb/types"]
	printBufGenerateStart(len(changedProtos))
	bufStart := time.Now()
	bufResult, err := codegen.RunBufGenerate(ctx, repoRoot, changedProtos, cache)
	if err != nil {
		return errors.Wrap(err, "buf generate")
	}
	printBufGenerateDone(bufResult.Cached, time.Since(bufStart))
	if err := cache.Save(); err != nil {
		printDim(fmt.Sprintf("save cache: %v", err))
	}

	printSyncedCount(len(syncResult.Written), len(syncResult.Unchanged)+len(syncResult.Skipped))
	return nil
}

// writeSchemaSources rewrites each schema file whose canonical formatted
// bytes differ from the on-disk source. Returns the count of files actually
// rewritten. The pipeline already produced FormattedSources in memory; this
// is the on-disk projection of that step.
func writeSchemaSources(result *pipeline.Result, repoRoot string) (int, error) {
	formatted := 0
	for _, rel := range result.Schemas {
		canonical := result.FormattedSources[rel]
		raw := result.Sources[rel]
		if string(canonical) == string(raw) {
			continue
		}
		abs := paths.Resolve(rel, repoRoot)
		if err := os.WriteFile(abs, canonical, 0644); err != nil {
			return formatted, errors.Wrapf(err, "failed to write %s", abs)
		}
		formatted++
	}
	return formatted, nil
}

// syncOutputs is the on-disk projection of the pipeline's plugin outputs.
// For each generated file it consults the cache, formats only on cache
// miss, byte-compares against the existing on-disk file, and writes only
// when the canonical bytes differ. The cache stores the SHA-256 of the
// raw (pre-format) plugin bytes for each path so repeat runs can skip the
// formatter chain entirely when nothing has changed.
func syncOutputs(
	ctx context.Context,
	result *pipeline.Result,
	repoRoot string,
	formatters *format.Registry,
	cache *format.Cache,
	workers int,
) (*syncResult, error) {
	r := &syncResult{
		Written:   make([]string, 0),
		Unchanged: make([]string, 0),
		Skipped:   make([]string, 0),
		ByPlugin:  make(map[string][]string),
	}

	type pending struct {
		Plugin   string
		RelPath  string
		AbsPath  string
		Raw      []byte
		RawHash  string
		Existing []byte
	}

	keep := set.New[string]()
	var toFormat []pending
	for pluginName, files := range result.Outputs {
		for _, f := range files {
			absPath := filepath.Join(repoRoot, f.Path)
			rawHash := format.Hash(f.Content)
			keep.Add(f.Path)

			cachedHash, hit := cache.LookupRaw(f.Path)
			if hit && cachedHash == rawHash {
				if _, err := os.Stat(absPath); err == nil {
					r.Skipped = append(r.Skipped, f.Path)
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
		printFormatPlan(0, len(r.Skipped))
		return r, nil
	}

	printFormatPlan(len(toFormat), len(r.Skipped))
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
		eg.Go(func() error {
			if err := gctx.Err(); err != nil {
				return err
			}
			canonical := formatted[i].Content
			if p.Existing != nil && string(p.Existing) == string(canonical) {
				mu.Lock()
				r.Unchanged = append(r.Unchanged, p.RelPath)
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
			r.Written = append(r.Written, p.RelPath)
			r.ByPlugin[p.Plugin] = append(r.ByPlugin[p.Plugin], p.RelPath)
			cache.PutRaw(p.RelPath, p.RawHash)
			mu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	printWritePlan(len(r.Written), len(r.Unchanged))
	return r, nil
}

type syncResult struct {
	ByPlugin  map[string][]string
	Written   []string
	Unchanged []string
	Skipped   []string
}

// expandGlobs is preserved for `oracle fmt` which still accepts arbitrary
// user-provided patterns (e.g. `oracle fmt schemas/rack.oracle`). The
// canonical schema-discovery path is pipeline.DiscoverSchemas.
func expandGlobs(patterns []string, baseDir string) ([]string, error) {
	var files []string
	for _, pattern := range patterns {
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(baseDir, pattern)
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid glob pattern %q", pattern)
		}
		files = append(files, matches...)
	}
	sort.Strings(files)
	return files, nil
}
