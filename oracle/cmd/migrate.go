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
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/freeze"
	gomigrate "github.com/synnaxlabs/oracle/plugin/go/migrate"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/snapshot"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/errors"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [resource...]",
		Short: "Mint or amend version files and generate migration code",
		Long: `With explicit version chains, migrate mints the next version file for the
named resources (or every drifted resource when none are named) from the live
persisted shapes, then syncs. --amend rewrites the current version file in
place instead — for versions that have never shipped in a release.

Without chains, the legacy snapshot-diffing flow runs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runMigrate(cmd, args); err != nil {
				printError(err.Error())
				return err
			}
			return nil
		},
	}
	cmd.Flags().Bool("amend", false,
		"rewrite the named resources' current version files in place")
	return cmd
}

// runChainMigrate mints or amends version files for the targeted chains, then
// syncs.
func runChainMigrate(
	cmd *cobra.Command,
	repoRoot string,
	chains map[string]versions.Chain,
	args []string,
	amend bool,
) error {
	ctx := cmd.Context()
	loader := analyzer.NewStandardFileLoader(repoRoot)
	resolver := versions.NewResolver(chains, loader)

	normalizedFiles, err := pipeline.DiscoverSchemas(repoRoot)
	if err != nil {
		return err
	}
	live, diag := analyzer.Analyze(ctx, normalizedFiles, loader)
	if diag != nil {
		printDiagnostics(diag.String())
		if !diag.Ok() {
			return errors.New("schema analysis failed")
		}
	}
	if err := resolver.Annotate(ctx, live); err != nil {
		return err
	}

	targets, err := resolveTargets(
		ctx, repoRoot, resolver, live, chains, args, amend,
	)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		printDim("no drifted resources; nothing to migrate")
		return nil
	}

	for _, chain := range targets {
		current := chain.Current()
		f, err := resolver.File(ctx, chain.LivePath(), current)
		if err != nil {
			return err
		}
		pins, docs, pinned := freeze.FileInput(f)
		in := freeze.Input{
			Live:     live,
			Resolver: resolver,
			Chain:    chain,
			Pinned:   pinned,
		}
		if amend {
			in.N, in.Pins, in.Docs = current, pins, docs
		} else {
			in.N = current + 1
		}
		out, err := freeze.Canonical(ctx, in)
		if err != nil {
			return err
		}
		target := filepath.Join(repoRoot, chain.FilePath(in.N)+".oracle")
		if err := writeFileIfChanged(target, []byte(out)); err != nil {
			return errors.Wrapf(err, "failed to write %s", target)
		}
		printDim(fmt.Sprintf("  ✏️  %s", chain.FilePath(in.N)+".oracle"))
	}
	printDim("running sync...")
	return runSync(cmd)
}

// resolveTargets maps resource arguments to chains; with no arguments it
// selects every chain whose live schema drifts from its current file.
func resolveTargets(
	ctx context.Context,
	repoRoot string,
	resolver *versions.Resolver,
	live *resolution.Table,
	chains map[string]versions.Chain,
	args []string,
	amend bool,
) ([]versions.Chain, error) {
	if len(args) > 0 {
		targets := make([]versions.Chain, 0, len(args))
		for _, arg := range args {
			var matches []versions.Chain
			for _, chain := range chains {
				if chain.Resource == arg || chain.LivePath() == arg {
					matches = append(matches, chain)
				}
			}
			switch len(matches) {
			case 1:
				targets = append(targets, matches[0])
			case 0:
				return nil, errors.Newf("no version chain matches %q", arg)
			default:
				return nil, errors.Newf(
					"%q is ambiguous; use a live path like %q",
					arg, matches[0].LivePath(),
				)
			}
		}
		return targets, nil
	}
	if amend {
		return nil, errors.New("--amend requires naming the resources to amend")
	}
	var drifted []versions.Chain
	for _, livePath := range slices.Sorted(maps.Keys(chains)) {
		chain := chains[livePath]
		f, err := resolver.File(ctx, livePath, chain.Current())
		if err != nil {
			return nil, err
		}
		pins, docs, pinned := freeze.FileInput(f)
		canonical, err := freeze.Canonical(ctx, freeze.Input{
			Live:     live,
			Resolver: resolver,
			Chain:    chain,
			N:        chain.Current(),
			Pins:     pins,
			Docs:     docs,
			Pinned:   pinned,
		})
		if err != nil {
			return nil, err
		}
		onDisk, err := os.ReadFile(
			filepath.Join(repoRoot, chain.FilePath(chain.Current())+".oracle"),
		)
		if err != nil {
			return nil, err
		}
		if string(onDisk) != canonical {
			drifted = append(drifted, chain)
		}
	}
	return drifted, nil
}

func runMigrate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	verbose := viper.GetBool(verboseFlag)
	printBanner()
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "migrate must be run within a git repository")
	}

	// Explicit version chains supersede the snapshot flow entirely.
	amend, err := cmd.Flags().GetBool("amend")
	if err != nil {
		return err
	}
	chains, err := versions.Discover(repoRoot)
	if err != nil {
		return err
	}
	if len(chains) > 0 {
		return runChainMigrate(cmd, repoRoot, chains, args, amend)
	}
	if len(args) > 0 || amend {
		return errors.New(
			"no version chains exist; resource arguments and --amend require them",
		)
	}

	normalizedFiles, err := pipeline.DiscoverSchemas(repoRoot)
	if err != nil {
		return err
	}
	if len(normalizedFiles) == 0 {
		return errors.New("no schema files found")
	}

	printSchemaCount(len(normalizedFiles))

	// Build a registry with only the migrate plugin.
	registry := plugin.NewRegistry()
	_ = registry.Register(gomigrate.New())

	// Load old snapshot if one exists.
	latestVersion, loadSnapshot, err := snapshot.TableLoader(ctx, repoRoot)
	if err != nil {
		return err
	}

	// Analyze current schemas.
	loader := analyzer.NewStandardFileLoader(repoRoot)
	table, diag := analyzer.Analyze(ctx, normalizedFiles, loader)
	if diag != nil {
		printDiagnostics(diag.String())
		if !diag.Ok() {
			return errors.New("schema analysis failed")
		}
	}

	// Read core version for migration numbering.
	coreVersion, err := readCoreVersion(repoRoot)
	if err != nil {
		return errors.Wrap(err, "failed to read core version")
	}

	// Build the plugin request.
	req := &plugin.Request{
		Resolutions:     table,
		RepoRoot:        repoRoot,
		SnapshotVersion: coreVersion,
		LoadSnapshot:    loadSnapshot,
	}

	// If we have a previous snapshot, load it for diffing. A snapshot the
	// current grammar can no longer analyze is historical text, not a
	// baseline — migrate proceeds without diffing rather than failing.
	if latestVersion > 0 {
		oldTable, err := loadSnapshot(latestVersion)
		if err != nil && !errors.Is(err, snapshot.ErrAnalysis) {
			return errors.Wrap(err, "failed to load latest snapshot")
		}
		switch {
		case err != nil:
			printDim(fmt.Sprintf(
				"snapshot v%d no longer parses under the current grammar; "+
					"migration diffing resumes at the next snapshot",
				latestVersion,
			))
		case oldTable != nil:
			req.OldResolutions = oldTable
			req.SnapshotVersion = latestVersion
			if gomigrate.SnapshotPreVersioning(oldTable) {
				printDim(fmt.Sprintf(
					"snapshot v%d predates @go version; migration diffing resumes at the next snapshot",
					latestVersion,
				))
			}
		}
	}

	// Run the migrate plugin.
	resp, err := registry.Get("go/migrate").Generate(req)
	if err != nil {
		return errors.Wrap(err, "migration generation failed")
	}

	formatters, err := format.Default(repoRoot)
	if err != nil {
		return errors.Wrap(err, "build formatter registry")
	}

	written := 0
	var templates []string
	for _, f := range resp.Files {
		fullPath := filepath.Join(repoRoot, f.Path)
		canonical, err := formatters.Format(ctx, f.Content, fullPath)
		if err != nil {
			return err
		}
		if err := writeFileIfChanged(fullPath, canonical); err != nil {
			return errors.Wrapf(err, "failed to write %s", f.Path)
		}
		if strings.HasSuffix(f.Path, "/migrate.go") {
			templates = append(templates, f.Path)
		}
		if verbose {
			printFileWritten("go/migrate", f.Path)
		}
		written++
	}
	if len(templates) > 0 {
		for _, t := range templates {
			printDim(fmt.Sprintf("  ✏️  %s ← edit this", t))
		}
	}
	// Delete files that were retargeted and moved.
	for _, d := range resp.Deletions {
		fullPath := filepath.Join(repoRoot, d)
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			return errors.Wrapf(err, "failed to delete retargeted file %s", d)
		}
		if verbose {
			printDim(fmt.Sprintf("  moved %s", d))
		}
	}

	printSyncedCount(written, len(resp.Files)-written)

	// Run oracle sync to update types/codecs.
	printDim("running sync...")
	if err := runSync(cmd); err != nil {
		return errors.Wrap(err, "sync failed after migration generation")
	}

	return nil
}

// readCoreVersion reads core/pkg/version/VERSION and returns the migration
// version number (major*1000 + minor). For "0.53.4" this returns 53.
func readCoreVersion(repoRoot string) (int, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "core", "pkg", "version", "VERSION"))
	if err != nil {
		return 0, errors.Wrap(err, "failed to read core VERSION file")
	}
	version := strings.TrimSpace(string(data))
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return 0, errors.Newf("invalid version format: %s", version)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, errors.Wrapf(err, "invalid major version: %s", parts[0])
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, errors.Wrapf(err, "invalid minor version: %s", parts[1])
	}
	return major*1000 + minor, nil
}

func writeFileIfChanged(path string, content []byte) error {
	existing, err := os.ReadFile(path)
	if err == nil && string(existing) == string(content) {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}
