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
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/x/errors"
)

// Flag names for the check command. Constants so test code can set them
// the same way the cobra binding does.
const (
	checkGatesFlag            = "gates"
	checkFormatFlag           = "format"
	checkDiffFlag             = "diff"
	checkWarningsAsErrorsFlag = "warnings-as-errors"
)

func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Validate schemas, generated outputs, and cache",
		Long: `Run a set of read-only validation gates against the oracle workspace.

Gates:
  format       schemas are canonically formatted
  analyze      schemas pass semantic analysis (errors and warnings surfaced)
  generated    on-disk generated files match what 'oracle sync' would produce
  orphans      no abandoned generated files remain on disk
  cache        sync cache is internally consistent

Exit codes:
  0   all gates passed
  1   internal error
  10  format drift
  11  analyzer errors
  12  generated drift
  13  orphan files
  14  cache incoherence

Examples:
  oracle check
  oracle check --gates=format,analyze
  oracle check --diff
  oracle check --format=json > report.json`,
	}
	cmd.Flags().StringSlice(checkGatesFlag, nil,
		"Comma-separated subset of gates to run (default: all)")
	cmd.Flags().String(checkFormatFlag, "text",
		"Output format: text or json")
	cmd.Flags().Bool(checkDiffFlag, false,
		"Include unified diffs in drift findings")
	cmd.Flags().Bool(checkWarningsAsErrorsFlag, false,
		"Treat analyzer warnings as errors")
	cmd.RunE = runCheck
	return cmd
}

func runCheck(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	gates, _ := cmd.Flags().GetStringSlice(checkGatesFlag)
	outputFormat, _ := cmd.Flags().GetString(checkFormatFlag)
	includeDiffs, _ := cmd.Flags().GetBool(checkDiffFlag)
	warningsAsErrors, _ := cmd.Flags().GetBool(checkWarningsAsErrorsFlag)
	verbose := viper.GetBool(verboseFlag)

	if outputFormat != string(check.FormatText) && outputFormat != string(check.FormatJSON) {
		return errors.Newf("invalid --format %q: must be 'text' or 'json'", outputFormat)
	}

	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "check must be run within a git repository")
	}

	if outputFormat == string(check.FormatText) {
		printBanner()
	}

	schemas, err := pipeline.DiscoverSchemas(repoRoot)
	if err != nil {
		return err
	}
	if len(schemas) == 0 {
		return errors.New("no schema files found")
	}

	if outputFormat == string(check.FormatText) {
		printSchemaCount(len(schemas))
	}

	registry := buildPluginRegistry()
	result, err := pipeline.Run(ctx, pipeline.Options{
		RepoRoot: repoRoot,
		Schemas:  schemas,
		Plugins:  registry,
	})
	if err != nil {
		return errors.Wrap(err, "pipeline")
	}

	// Resource initialization is lazy because each gate has different
	// dependencies and a failure to build (e.g. missing license template
	// in a minimal test repo) should not poison gates that do not need
	// the resource. Whether a gate is in the enabled set determines
	// whether we even try.
	wantedGates := wantedSet(gates)
	var formatters *format.Registry
	if wantedGates.has("generated") {
		formatters, err = buildCheckFormatters(repoRoot)
		if err != nil {
			return errors.Wrap(err, "build formatter registry")
		}
	}
	var cache *format.Cache
	if wantedGates.has("orphans") || wantedGates.has("cache") {
		cache = loadCheckCache(repoRoot)
	}

	checkers := buildCheckers(formatters, cache, warningsAsErrors)
	report := check.Run(ctx, result, check.Env{
		RepoRoot:     repoRoot,
		IncludeDiffs: includeDiffs,
	}, checkers, gates)

	if err := check.Render(os.Stdout, report, check.Format(outputFormat), verbose); err != nil {
		return errors.Wrap(err, "render report")
	}

	if code := report.FirstExitCode(); code != 0 {
		return &exitCodeError{code: code, msg: fmt.Sprintf("%d gate(s) failed", report.TotalFailed)}
	}
	return nil
}

// migrateOwnedPatterns lists generated-file basenames owned by the
// migrate plugin (`oracle migrate`) rather than the regular sync
// pipeline. The orphan gate skips these so it does not flag migration
// artifacts as drift.
//
// This list will go away once the check pipeline runs the migrate
// plugin alongside the sync registry; until then, it is the single
// source of truth for migrate-owned file shapes.
var migrateOwnedPatterns = []string{
	"migrate_auto.gen.*",
}

// buildCheckers wires the canonical gate set. The order matters and is
// part of the documented contract: format runs before analyze runs
// before generated, etc.
func buildCheckers(
	formatters *format.Registry,
	cache *format.Cache,
	warningsAsErrors bool,
) []check.Checker {
	return []check.Checker{
		check.NewFormatGate(),
		check.NewAnalyzeGate(warningsAsErrors),
		check.NewGeneratedGate(formatters, runtime.GOMAXPROCS(0)),
		check.NewOrphanGate(cache).WithIgnores(migrateOwnedPatterns...),
		check.NewCacheGate(cache),
	}
}

func buildCheckFormatters(repoRoot string) (*format.Registry, error) {
	return format.Default(repoRoot)
}

func loadCheckCache(repoRoot string) *format.Cache {
	return format.LoadCache(repoRoot)
}

// exitCodeError carries a specific exit code back through cobra's error
// path. The Execute helper in root.go checks for this type before
// falling back to its generic os.Exit(1).
type exitCodeError struct {
	code int
	msg  string
}

func (e *exitCodeError) Error() string { return e.msg }
func (e *exitCodeError) ExitCode() int { return e.code }

// gateSet is a small helper for "is this gate name in the enabled
// subset?". Empty set means "all gates" (no --gates filter passed).
type gateSet struct {
	all bool
	in  map[string]struct{}
}

func wantedSet(gates []string) gateSet {
	if len(gates) == 0 {
		return gateSet{all: true}
	}
	in := make(map[string]struct{}, len(gates))
	for _, g := range gates {
		in[g] = struct{}{}
	}
	return gateSet{in: in}
}

func (s gateSet) has(name string) bool {
	if s.all {
		return true
	}
	_, ok := s.in[name]
	return ok
}
