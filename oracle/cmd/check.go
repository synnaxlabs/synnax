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
	"runtime"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// Flag names for the check command. Constants so test code can set them the same way
// the cobra binding does.
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
  cache        sync cache is internally consistent
  persistence  persisted shapes only reference persisted types
  versions     version files, live projections, and pins are consistent

Exit codes:
  0   all gates passed
  1   internal error
  10  format drift
  11  analyzer errors
  12  generated drift
  13  persistence violations
  14  cache incoherence
  15  versions drift

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
	// A failed gate is a normal exit - the user does not want cobra dumping the usage
	// block underneath the gate findings, nor a duplicate "Error: ..." line repeating
	// what the renderer already printed. The exit code (set by Execute via
	// exitCodeError) is the machine-readable signal; the rendered report is the human
	// one.
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.RunE = runCheck
	return cmd
}

func runCheck(cmd *cobra.Command, _ []string) (err error) {
	ctx := cmd.Context()

	gates := lo.Must(cmd.Flags().GetStringSlice(checkGatesFlag))
	outputFormat := lo.Must(cmd.Flags().GetString(checkFormatFlag))
	includeDiffs := lo.Must(cmd.Flags().GetBool(checkDiffFlag))
	warningsAsErrors := lo.Must(cmd.Flags().GetBool(checkWarningsAsErrorsFlag))
	verbose := viper.GetBool(verboseFlag)

	if outputFormat != string(check.FormatText) &&
		outputFormat != string(check.FormatJSON) {
		return errors.Newf(
			"invalid --format %q: must be 'text' or 'json'",
			outputFormat,
		)
	}

	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "check must be run within a Git repository")
	}

	text := outputFormat == string(check.FormatText)
	if text {
		printBanner()
	}
	schemas, err := discoverSchemas(repoRoot)
	if err != nil {
		return err
	}
	if text {
		printSchemaCount(len(schemas))
	}
	result, err := runPipeline(ctx, repoRoot, schemas)
	if err != nil {
		return err
	}

	wantedGates := wantedSet(gates)
	formatters, cache, err := loadGateResources(repoRoot, wantedGates)
	if err != nil {
		return err
	}
	shipped, err := loadShipped(ctx, repoRoot, wantedGates)
	if err != nil {
		return err
	}
	if shipped != nil {
		defer func() { err = errors.Combine(err, os.RemoveAll(shipped.Root)) }()
	}

	checkers := buildCheckers(formatters, cache, warningsAsErrors)
	report := check.Run(ctx, result, check.Env{
		RepoRoot:     repoRoot,
		IncludeDiffs: includeDiffs,
		Shipped:      shipped,
	}, checkers, gates)
	return renderReport(report, check.Format(outputFormat), verbose)
}

// discoverSchemas returns the repo-relative paths of all schema files, and fails when
// there are none.
func discoverSchemas(repoRoot string) ([]string, error) {
	schemas, err := pipeline.DiscoverSchemas(repoRoot)
	if err != nil {
		return nil, err
	}
	if len(schemas) == 0 {
		return nil, errors.New("no schema files found")
	}
	return schemas, nil
}

// runPipeline runs the full pipeline over schemas with every registered plugin.
func runPipeline(
	ctx context.Context,
	repoRoot string,
	schemas []string,
) (*pipeline.Result, error) {
	registry, err := buildPluginRegistry()
	if err != nil {
		return nil, err
	}
	result, err := pipeline.Run(ctx, pipeline.Options{
		RepoRoot: repoRoot,
		Schemas:  schemas,
		Plugins:  registry,
	})
	if err != nil {
		return nil, errors.Wrap(err, "pipeline")
	}
	return result, nil
}

// loadGateResources builds the formatter registry and loads the cache, each only when
// a wanted gate needs it. A resource that fails to build (e.g. a missing license
// template in a minimal test repo) must not fail gates that do not need it.
func loadGateResources(
	repoRoot string,
	wantedGates gateSet,
) (*format.Registry, *format.Cache, error) {
	var formatters *format.Registry
	if wantedGates.has("generated") {
		var err error
		if formatters, err = buildCheckFormatters(repoRoot); err != nil {
			return nil, nil, errors.Wrap(err, "build formatter registry")
		}
	}
	var cache *format.Cache
	if wantedGates.has("cache") {
		cache = loadCheckCache(repoRoot)
	}
	return formatters, cache, nil
}

// loadShipped extracts the last stable release when the versions gate runs. It returns
// nil otherwise. The caller removes the release root.
func loadShipped(
	ctx context.Context,
	repoRoot string,
	wantedGates gateSet,
) (*check.Release, error) {
	if !wantedGates.has("versions") {
		return nil, nil
	}
	release, err := extractRelease(ctx, repoRoot)
	if err != nil {
		return nil, err
	}
	return &release, nil
}

// renderReport prints the report to stdout and returns an exitCodeError when a gate
// failed.
func renderReport(report *check.Report, outputFormat check.Format, verbose bool) error {
	if err := check.Render(os.Stdout, report, outputFormat, verbose); err != nil {
		return errors.Wrap(err, "render report")
	}
	if code := report.FirstExitCode(); code != 0 {
		return &exitCodeError{
			code: code,
			msg:  fmt.Sprintf("%d gate(s) failed", report.TotalFailed),
		}
	}
	return nil
}

// buildCheckers wires the canonical gate set. The order matters and is part of the
// documented contract: format runs before analyze runs before generated, etc.
func buildCheckers(
	formatters *format.Registry,
	cache *format.Cache,
	warningsAsErrors bool,
) []check.Checker {
	return []check.Checker{
		check.NewFormatGate(),
		check.NewAnalyzeGate(warningsAsErrors),
		check.NewGeneratedGate(formatters, runtime.GOMAXPROCS(0)),
		check.NewCacheGate(cache),
		check.NewPersistenceGate(warningsAsErrors),
		check.NewVersionsGate(),
	}
}

func buildCheckFormatters(repoRoot string) (*format.Registry, error) {
	return format.Default(repoRoot, time.Now().Year(), nil)
}

func loadCheckCache(repoRoot string) *format.Cache {
	return format.LoadCache(repoRoot)
}

// exitCodeError carries a specific exit code back through cobra's error path. The
// Execute helper in root.go checks for this type before falling back to its generic
// os.Exit(1).
type exitCodeError struct {
	code int
	msg  string
}

func (e *exitCodeError) Error() string { return e.msg }

// gateSet is a small helper for "is this gate name in the enabled subset?". Empty set
// means "all gates" (no --gates filter passed).
type gateSet struct {
	all bool
	in  set.Set[string]
}

func wantedSet(gates []string) gateSet {
	if len(gates) == 0 {
		return gateSet{all: true}
	}
	return gateSet{in: set.New(gates...)}
}

func (s gateSet) has(name string) bool {
	if s.all {
		return true
	}
	return s.in.Contains(name)
}
