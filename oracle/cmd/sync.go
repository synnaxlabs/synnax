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
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"golang.org/x/sync/errgroup"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync generated code, only writing changed files",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runSync(cmd) },
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

	schemas, err := discoverSchemas(repoRoot)
	if err != nil {
		return err
	}
	printSchemaCount(len(schemas))
	result, err := runPipeline(ctx, repoRoot, schemas)
	if err != nil {
		return err
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
		printPluginDone(name, len(files))
	}

	formatters, err := buildSyncFormatters(repoRoot)
	if err != nil {
		return err
	}
	cache := format.LoadCache(repoRoot)

	syncResult, err := syncOutputs(ctx, result, repoRoot, formatters, cache, 0)
	if err != nil {
		return errors.Wrap(err, "failed to sync files")
	}
	if err := cache.Save(); err != nil {
		printDim(fmt.Sprintf("save cache: %v", err))
	}
	if verbose {
		printWrittenFiles(syncResult)
	}
	removeDeletions(result, repoRoot)
	changedProtos := syncResult.ByPlugin["pb/types"]
	if err := generateBuf(ctx, repoRoot, changedProtos, formatters, cache); err != nil {
		return err
	}

	printSyncedCount(
		len(syncResult.Written),
		len(syncResult.Unchanged)+len(syncResult.Skipped),
	)
	return nil
}

// buildSyncFormatters builds the formatter registry for generated files and buf
// outputs.
func buildSyncFormatters(repoRoot string) (*format.Registry, error) {
	// buf generate rewrites these without a header before they are formatted.
	pbFiles, err := codegen.FindPbGoFiles(repoRoot)
	if err != nil {
		return nil, errors.Wrap(err, "find buf outputs")
	}
	formatters, err := format.Default(repoRoot, time.Now().Year(), pbFiles)
	if err != nil {
		return nil, errors.Wrap(err, "build formatter registry")
	}
	return formatters, nil
}

// printWrittenFiles prints each written file with its plugin, sorted by path.
func printWrittenFiles(r *syncResult) {
	if len(r.Written) == 0 {
		return
	}
	pluginByPath := make(map[string]string, len(r.Written))
	for name, files := range r.ByPlugin {
		for _, f := range files {
			pluginByPath[f] = name
		}
	}
	writtenSorted := append([]string(nil), r.Written...)
	sort.Strings(writtenSorted)
	for _, f := range writtenSorted {
		printFileWritten(pluginByPath[f], f)
	}
}

// removeDeletions removes the files the plugins asked to delete. It prints a failed
// removal and continues.
func removeDeletions(result *pipeline.Result, repoRoot string) {
	for _, files := range result.Deletions {
		for _, deletePath := range files {
			abs := filepath.Join(repoRoot, deletePath)
			if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
				printDim(fmt.Sprintf("remove %s: %v", deletePath, err))
			}
		}
	}
}

// generateBuf runs buf generate for changedProtos and formats its outputs.
func generateBuf(
	ctx context.Context,
	repoRoot string,
	changedProtos []string,
	formatters *format.Registry,
	cache *format.Cache,
) error {
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

	// buf generate emits .pb.go and _grpc.pb.go files alongside their .proto sources
	// without a license header. Run them through the same formatter chain the oracle
	// plugin outputs use so the header is added and gofmt is applied. Idempotent on
	// cached / unchanged files because the license formatter no-ops when a header is
	// already present and gofmt is stable.
	if bufResult.Cached {
		return nil
	}
	if _, err := codegen.FormatBufOutputs(ctx, repoRoot, formatters, 0); err != nil {
		return errors.Wrap(err, "format buf outputs")
	}
	return nil
}

// writeSchemaSources rewrites each schema file whose canonical bytes differ from the
// on-disk source: the formatter's output, or the merged live projection for a versioned
// resource. Returns the count of files actually rewritten.
func writeSchemaSources(result *pipeline.Result, repoRoot string) (int, error) {
	written := 0
	for _, rel := range result.Schemas {
		canonical := result.EffectiveSource(rel)
		raw := result.Sources[rel]
		if string(canonical) == string(raw) {
			continue
		}
		abs := paths.Resolve(rel, repoRoot)
		if err := os.WriteFile(abs, canonical, 0o644); err != nil {
			return written, errors.Wrapf(err, "failed to write %s", abs)
		}
		written++
	}
	return written, nil
}

// syncOutputs is the on-disk projection of the pipeline's plugin outputs. For each
// generated file it consults the cache, formats only on cache miss, byte-compares
// against the existing on-disk file, and writes only when the canonical bytes differ.
// The cache stores the SHA-256 of the raw (pre-format) plugin bytes for each path so
// repeat runs can skip the formatter chain entirely when nothing has changed.
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
	toFormat, err := planOutputs(result, repoRoot, cache, r)
	if err != nil {
		return nil, err
	}
	if len(toFormat) == 0 {
		printFormatPlan(0, len(r.Skipped))
		return r, nil
	}

	printFormatPlan(len(toFormat), len(r.Skipped))
	batch, err := stageOutputs(toFormat)
	if err != nil {
		return nil, err
	}
	formatStart := time.Now()
	formatted, err := formatters.FormatBatch(ctx, batch, workers)
	if err != nil {
		return nil, err
	}
	printFormatDone(time.Since(formatStart))

	if err := writeOutputs(ctx, toFormat, formatted, cache, r, workers); err != nil {
		return nil, err
	}
	printWritePlan(len(r.Written), len(r.Unchanged))
	return r, nil
}

// pendingOutput is a generated file that the cache cannot skip.
type pendingOutput struct {
	Plugin   string
	RelPath  string
	AbsPath  string
	Raw      []byte
	RawHash  string
	Existing []byte
}

// planOutputs returns the generated files to format, adds cache hits to r.Skipped, and
// prunes the cache to the current outputs.
func planOutputs(
	result *pipeline.Result,
	repoRoot string,
	cache *format.Cache,
	r *syncResult,
) ([]pendingOutput, error) {
	keep := set.New[string]()
	var toFormat []pendingOutput
	for pluginName, files := range result.Outputs {
		for _, f := range files {
			absPath := filepath.Join(repoRoot, f.Path)
			rawHash := format.Hash(f.Content)
			keep.Add(f.Path)

			existing, err := paths.ReadFile(f.Path, repoRoot)
			if err != nil && !os.IsNotExist(err) {
				return nil, errors.Wrapf(err, "read existing %s", absPath)
			}
			if cacheHit(cache, f.Path, rawHash, existing) {
				r.Skipped = append(r.Skipped, f.Path)
				continue
			}
			toFormat = append(toFormat, pendingOutput{
				Plugin:   pluginName,
				RelPath:  f.Path,
				AbsPath:  absPath,
				Raw:      f.Content,
				RawHash:  rawHash,
				Existing: existing,
			})
		}
	}
	cache.PruneTo(keep)
	return toFormat, nil
}

// cacheHit reports whether the cache says BOTH the raw plugin output is unchanged AND
// the on-disk file still matches the canonical bytes the previous run wrote. The second
// check is what makes the cache safe across formatter upgrades and hand edits.
func cacheHit(cache *format.Cache, relPath, rawHash string, existing []byte) bool {
	if existing == nil {
		return false
	}
	entry, hit := cache.Lookup(relPath)
	return hit && entry.Raw == rawHash && entry.Canonical == format.Hash(existing)
}

// stageOutputs returns the format batch for toFormat. It writes the raw bytes of each
// new file to disk first, because ESLint's TypeScript project service lints only files
// on disk.
func stageOutputs(toFormat []pendingOutput) ([]format.File, error) {
	batch := make([]format.File, len(toFormat))
	for i, p := range toFormat {
		batch[i] = format.File{Path: p.AbsPath, Content: p.Raw}
		if p.Existing == nil {
			if err := writeFile(p.AbsPath, p.Raw); err != nil {
				return nil, err
			}
		}
	}
	return batch, nil
}

// writeOutputs writes each formatted file whose bytes differ from the file on disk,
// records the outcome in r, and updates the cache. It runs on workers goroutines, or
// GOMAXPROCS when workers is not positive.
func writeOutputs(
	ctx context.Context,
	toFormat []pendingOutput,
	formatted []format.File,
	cache *format.Cache,
	r *syncResult,
	workers int,
) error {
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
			entry := format.Entry{Raw: p.RawHash, Canonical: format.Hash(canonical)}
			if p.Existing != nil && string(p.Existing) == string(canonical) {
				mu.Lock()
				defer mu.Unlock()
				r.Unchanged = append(r.Unchanged, p.RelPath)
				cache.Put(p.RelPath, entry)
				return nil
			}
			if err := writeFile(p.AbsPath, canonical); err != nil {
				return err
			}
			mu.Lock()
			defer mu.Unlock()
			r.Written = append(r.Written, p.RelPath)
			r.ByPlugin[p.Plugin] = append(r.ByPlugin[p.Plugin], p.RelPath)
			cache.Put(p.RelPath, entry)
			return nil
		})
	}
	return eg.Wait()
}

func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.Wrapf(err, "mkdir %s", filepath.Dir(path))
	}
	return errors.Wrapf(os.WriteFile(path, content, 0o644), "write %s", path)
}

type syncResult struct {
	ByPlugin  map[string][]string
	Written   []string
	Unchanged []string
	Skipped   []string
}

// expandGlobs is preserved for `oracle fmt` which still accepts arbitrary user-provided
// patterns (e.g. `oracle fmt schemas/rack.oracle`). The canonical schema-discovery path
// is pipeline.DiscoverSchemas.
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
